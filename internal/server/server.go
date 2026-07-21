// Package server wires the single ingress HTTP server.
//
// One listener hosts BOTH the gRPC-over-WebSocket tunnel (the app data plane, at /socket, via
// GoGRPCBridge) and the document plane (static assets, SSR pages — SSR is added next as GWC
// components). Browser<->server app comms are gRPC/WS only; there is no ad-hoc REST API.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"github.com/monstercameron/earlcameron/internal/admin"
	"github.com/monstercameron/earlcameron/internal/anime"
	"github.com/monstercameron/earlcameron/internal/budget"
	"github.com/monstercameron/earlcameron/internal/rss"
	"github.com/monstercameron/earlcameron/internal/config"
	"github.com/monstercameron/earlcameron/internal/contact"
	"github.com/monstercameron/earlcameron/internal/content"
	"github.com/monstercameron/earlcameron/internal/site"
	"github.com/monstercameron/earlcameron/internal/store"
	"github.com/monstercameron/earlcameron/proto/sitepb"
	"google.golang.org/grpc"
)

// Server owns the ingress lifecycle.
type Server struct {
	cfg    config.Config
	log    *slog.Logger
	grpc   *grpc.Server
	tunnel   http.Handler
	store    *store.Store
	page     []byte // the standard site, server-rendered once at startup
	anime    *anime.Service
	sessions *admin.Sessions
}

// New opens the store, builds the gRPC server, registers the services, and wraps them in the
// GoGRPCBridge WebSocket tunnel. It returns an error if the store or tunnel cannot be built.
func New(cfg config.Config) (*Server, error) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}
	// Seed the default QOTD prompts on a fresh database (no-op once prompts exist).
	if _, err := rss.SeedPrompts(context.Background(), st, time.Now()); err != nil {
		log.Warn("qotd seed failed", "err", err)
	}

	cs := content.New()
	animeSvc := anime.New(st)
	sessions := admin.NewSessions(cfg.AdminUsername, cfg.AdminPassword, cfg.AdminSecret)

	// resolveOpenAI reads the effective OpenAI config: a DB setting (settings page) overrides the env.
	resolveOpenAI := func(ctx context.Context) (string, string) {
		key, model := cfg.OpenAIKey, cfg.OpenAIModel
		if v, _ := st.GetSetting(ctx, store.SettingOpenAIKey); v != "" {
			key = v
		}
		if v, _ := st.GetSetting(ctx, store.SettingOpenAIModel); v != "" {
			model = v
		}
		return key, model
	}

	// The auth interceptor gates AdminService methods (all but Login) on a valid session token in
	// gRPC metadata; the public services (Content/Contact) pass through.
	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(sessions.UnaryAuthInterceptor()))
	sitepb.RegisterContentServiceServer(grpcSrv, cs)
	sitepb.RegisterContactServiceServer(grpcSrv, contact.New(st))
	sitepb.RegisterAdminServiceServer(grpcSrv, admin.NewService(animeSvc, sessions, st, resolveOpenAI))

	tunnel, err := grpctunnel.BuildBridgeHandler(grpcSrv, grpctunnel.BridgeConfig{
		CheckOrigin: originChecker(cfg.AllowedOrigins),
	})
	if err != nil {
		_ = st.Close()
		return nil, err
	}

	// The standard site is static content, so render it once here (avoids per-request work and
	// the process-global CSS sink accumulating across concurrent renders).
	page, err := site.RenderHTML(cs.About(), cs.Projects())
	if err != nil {
		_ = st.Close()
		return nil, err
	}
	return &Server{cfg: cfg, log: log, grpc: grpcSrv, tunnel: tunnel, store: st, page: []byte(page), anime: animeSvc, sessions: sessions}, nil
}

// originChecker returns a WebSocket upgrade origin validator that prevents cross-site WebSocket
// hijacking: it allows requests with no Origin header (non-browser clients), same-origin
// requests (Origin host matches the request Host), and any origin in the allow-list. Everything
// else — i.e. a browser page on another site — is rejected.
func originChecker(allowed []string) func(*http.Request) bool {
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		set[o] = struct{}{}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		if strings.EqualFold(u.Host, r.Host) {
			return true
		}
		_, ok := set[origin]
		return ok
	}
}

// routes builds the request multiplexer.
func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	// Data plane: gRPC over WebSocket.
	mux.Handle("/socket", s.tunnel)
	mux.Handle("/socket/", s.tunnel)
	// Document plane (HTTP GET).
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	// CashFlux, hosted as a managed budgeting app (its own WASM SPA under web/cashflux).
	mux.Handle("/budget/", http.StripPrefix("/budget/", budget.Handler()))
	mux.HandleFunc("/healthz", s.healthz)
	s.registerAdminRoutes(mux)
	s.registerResumeRoutes(mux)
	mux.HandleFunc("/", s.ssrShell)
	return mux
}

// healthz reports liveness for the deploy health check and rollback logic.
func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

// ssrShell serves the standard site (SEO + no-WASM failsafe), rendered once at startup. The
// GWC/WASM terminal will later hydrate over this markup.
func (s *Server) ssrShell(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(s.page)
}

// Run starts the ingress server and blocks until SIGINT/SIGTERM, then shuts down gracefully.
func (s *Server) Run() error {
	srv := &http.Server{Addr: s.cfg.Addr, Handler: s.routes(), ReadHeaderTimeout: 10 * time.Second}

	go func() {
		s.log.Info("ingress up", "addr", s.cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("serve failed", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	s.grpc.GracefulStop()
	_ = s.store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
