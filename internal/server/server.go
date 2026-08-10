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

	cashfluxembed "github.com/monstercameron/CashFlux/pkg/embed"
	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"github.com/monstercameron/earlcameron/internal/admin"
	"github.com/monstercameron/earlcameron/internal/anime"
	"github.com/monstercameron/earlcameron/internal/budget"
	"github.com/monstercameron/earlcameron/internal/config"
	"github.com/monstercameron/earlcameron/internal/contact"
	"github.com/monstercameron/earlcameron/internal/content"
	"github.com/monstercameron/earlcameron/internal/site"
	"github.com/monstercameron/earlcameron/internal/store"
	"github.com/monstercameron/earlcameron/internal/system"
	"github.com/monstercameron/earlcameron/proto/sitepb"
	"google.golang.org/grpc"
)

// Server owns the ingress lifecycle.
type Server struct {
	cfg    config.Config
	log    *slog.Logger
	grpc   *grpc.Server
	tunnel http.Handler
	store  *store.Store
	page   []byte // the standard site, server-rendered once at startup
	// projectPages holds the Flux case-study pages (/projects/<slug>), keyed by slug and
	// server-rendered once at startup for the same reason as page.
	projectPages map[string][]byte
	anime        *anime.Service
	sessions     *admin.Sessions
	adminSvc     *admin.Service // for the scheduled daily Slack post
	// budgetGate is the password door in front of the CashFlux app at /budget/ (guest bypass grants a
	// local-only session). Disabled (pass-through) when no BudgetPassword is configured.
	budgetGate *budget.Gate
	// cashfluxSync is the embedded CashFlux data-sync bridge — SyncService + AuthService +
	// BlobService exposed over its own gRPC-over-WebSocket tunnel (not the full CashFlux site),
	// backed by an encrypted server-side SQLite store. nil when disabled. cashfluxClose releases
	// its store at shutdown. cashfluxAdmin is the client-management handle admin.Service's
	// CashFlux RPCs call into directly (also nil when disabled).
	cashfluxSync  http.Handler
	cashfluxClose func() error
	cashfluxAdmin admin.CashFluxAdmin
	// system serves the terminal's `stats`/`uptime` and the boot-log latency probe, and counts
	// served requests via the countRequests middleware.
	system *system.Service
}

// New opens the store, builds the gRPC server, registers the services, and wraps them in the
// GoGRPCBridge WebSocket tunnel. It returns an error if the store or tunnel cannot be built.
func New(cfg config.Config) (*Server, error) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		return nil, err
	}

	cs := content.New()
	animeSvc := anime.New(st)
	sessions := admin.NewSessions(st, cfg.AdminUsername, cfg.AdminPassword, cfg.AdminSecret, cfg.AdminRecoveryToken, cfg.AdminSetupToken)
	// Surface two auth footguns loudly at boot rather than silently:
	//   - first-run setup is open to whoever reaches the site first when no setup token gates it;
	//   - a stored owner account makes ADMIN_PASSWORD inert, so setting it looks like it changed the
	//     password when it did nothing.
	if sessions.NeedsSetup(context.Background()) && cfg.AdminSetupToken == "" {
		log.Warn("owner setup is OPEN — the first visitor to complete setup claims the account; set ADMIN_SETUP_TOKEN to gate it on an internet-reachable deploy")
	}
	if has, _ := st.HasCredential(context.Background()); has && cfg.AdminPassword != "" {
		log.Warn("ADMIN_PASSWORD is set but a stored owner account exists — the env password is inert; log in with the stored account, or reset it via the recovery phrase")
	}

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

	// Embed CashFlux's per-person sync engine in-process (owns its encrypted server-side SQLite
	// store) when enabled: SyncService + AuthService + BlobService over its own gRPC-over-WebSocket
	// bridge, not the full CashFlux site (no billing/portal/OAuth/admin surfaces). AuthService gives
	// real per-person identity (phone/SMS sign-in) instead of one shared static token, so this site
	// and a small, manually-invited set of people can each have their own account on one server.
	// New-account creation is gated by CASHFLUX_SERVER_SETUP_CODE (read directly from the process
	// environment by CashFlux's own Config.FromEnv — nothing to plumb through here): set it to a
	// value only Cam knows, hand it to an invitee once, and their phone number can enroll; a
	// returning, already-verified phone number never needs it again. Built before admin.NewService
	// below so its CashFlux client-management RPCs can call straight into cashfluxAdmin.
	var cashfluxSync http.Handler
	var cashfluxClose func() error
	var cashfluxAdmin admin.CashFluxAdmin
	if cfg.CashFluxDataDir != "" {
		// The frontend is served same-origin (/budget/), so its WebSocket Origin equals our BaseURL;
		// seed CashFlux's app-origin from it (unless explicitly overridden) so the bridge's origin
		// check accepts the frontend in production too, not just on loopback. Reduce BaseURL to a bare
		// scheme://host — CashFlux rejects an origin carrying any path (e.g. a trailing slash), which
		// would silently disable the whole sync engine.
		if os.Getenv("CASHFLUX_SERVER_APP_ORIGIN") == "" {
			if origin := schemeHostOrigin(cfg.BaseURL); origin != "" {
				_ = os.Setenv("CASHFLUX_SERVER_APP_ORIGIN", origin)
			}
		}
		bridge, err := cashfluxembed.NewSyncAndAuthBridge(cfg.CashFluxDataDir)
		if err != nil {
			// A degraded managed service is not a silent condition: the site advertises CashFlux sync,
			// so surface the failure loudly (common cause: a BASE_URL that isn't a bare origin).
			log.Error("embedded CashFlux sync engine disabled", "err", err)
		} else {
			cashfluxSync, cashfluxClose, cashfluxAdmin = bridge.Handler, bridge.Close, bridge.Admin
			log.Info("embedded CashFlux sync engine", "data_dir", cfg.CashFluxDataDir, "setup_code_gated", os.Getenv("CASHFLUX_SERVER_SETUP_CODE") != "")
			if bridge.Token != "" {
				// The token was auto-generated (no CASHFLUX_SERVER_TOKEN pinned). Surface it once so it
				// can be entered as the CashFlux frontend's "server token"; otherwise sync is
				// unauthenticated. It changes every restart until pinned via env.
				log.Warn("CashFlux sync access token (auto-generated; enter in CashFlux settings; set CASHFLUX_SERVER_TOKEN to keep it stable)", "token", bridge.Token)
			}
		}
	}

	// The auth interceptor gates AdminService methods (all but Login) on a valid session token in
	// gRPC metadata; the public services (Content/Contact) pass through.
	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(sessions.UnaryAuthInterceptor()))
	sitepb.RegisterContentServiceServer(grpcSrv, cs)
	sitepb.RegisterContactServiceServer(grpcSrv, contact.New(st))
	sys := system.New(st)
	sitepb.RegisterSystemServiceServer(grpcSrv, sys)
	adminSvc := admin.NewService(animeSvc, sessions, st, cfg.BaseURL, resolveOpenAI, cashfluxAdmin)
	sitepb.RegisterAdminServiceServer(grpcSrv, adminSvc)

	tunnel, err := grpctunnel.BuildBridgeHandler(grpcSrv, grpctunnel.BridgeConfig{
		CheckOrigin: originChecker(cfg.AllowedOrigins),
	})
	if err != nil {
		_ = st.Close()
		return nil, err
	}

	// The standard site is static content, so render it once here (avoids per-request work and
	// the process-global CSS sink accumulating across concurrent renders).
	page, err := site.RenderHTML(cs.About(), cs.Projects(), cfg.BaseURL)
	if err != nil {
		_ = st.Close()
		return nil, err
	}
	// The Flux case-study pages are static for the same reason and are rendered here too. A
	// failure is fatal at boot rather than a blank page discovered by a visitor.
	projectPages, err := buildProjectPages(cfg.BaseURL)
	if err != nil {
		_ = st.Close()
		return nil, err
	}
	gate := budget.NewGate(cfg.BudgetPassword, cfg.AdminSecret)
	// Let the gate recognise a visitor arriving with a live activation code — one
	// the admin console minted for them moments earlier, from an authenticated
	// session. Only wired when CashFlux embedding is configured; without it the gate
	// behaves exactly as before.
	if cashfluxAdmin != nil {
		gate.SetActivationChecker(cashfluxAdmin.ActivationCodeIsValid)
	}
	return &Server{cfg: cfg, log: log, grpc: grpcSrv, tunnel: tunnel, store: st, page: []byte(page), projectPages: projectPages, anime: animeSvc, sessions: sessions, adminSvc: adminSvc, budgetGate: gate, cashfluxSync: cashfluxSync, cashfluxClose: cashfluxClose, cashfluxAdmin: cashfluxAdmin, system: sys}, nil
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

// schemeHostOrigin reduces a base URL to a bare "scheme://host" origin, dropping any path, query,
// or fragment. It returns "" when raw is empty or lacks a scheme/host. CashFlux's WebSocket origin
// check rejects an origin that carries a path (e.g. a BASE_URL with a trailing slash), so the
// app-origin seeded from BaseURL must be normalized to this shape.
func schemeHostOrigin(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// routes builds the request multiplexer.
func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	// Data plane: gRPC over WebSocket.
	mux.Handle("/socket", s.tunnel)
	mux.Handle("/socket/", s.tunnel)
	// Document plane (HTTP GET).
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	// CashFlux, hosted as a managed budgeting app: the WASM SPA (web/cashflux) plus its per-person
	// sync/auth engine served in-process over gRPC-over-WebSocket at /grpc (the bridge path the
	// CashFlux frontend dials), backed by our encrypted server-side SQLite store — so multi-device
	// sync persists on our backend. Point CashFlux's "server URL" at this origin.
	mux.Handle("/budget/", http.StripPrefix("/budget/", s.budgetGate.Wrap(budget.Handler())))
	if s.cashfluxSync != nil {
		// /grpc and /v1/version are deliberately NOT behind s.budgetGate: real access control now
		// lives in AuthService itself (a bearer token from phone/SMS sign-in, gated at account
		// creation by CASHFLUX_SERVER_SETUP_CODE — see the embed call site above), not in keeping
		// this path secret. The embedded engine serves the /grpc WebSocket tunnel and the
		// /v1/version discovery handshake the CashFlux frontend probes before it will connect
		// ("Test connection").
		mux.Handle("/grpc", s.cashfluxSync)
		mux.Handle("/v1/version", s.cashfluxSync)
	}
	mux.HandleFunc("/healthz", s.healthz)
	s.registerAdminRoutes(mux)
	s.registerResumeRoutes(mux)
	s.registerProjectRoutes(mux)
	mux.HandleFunc("/", s.ssrShell)
	return mux
}

// countRequests increments the served-request counter that the terminal's `stats` reports. It
// records a count and nothing else — no path, no client address, no timing — because the only
// question it answers is "is this process actually serving traffic", and a visitor watching that
// number move is the whole point.
func (s *Server) countRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.system.CountRequest()
		next.ServeHTTP(w, r)
	})
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
	srv := &http.Server{Addr: s.cfg.Addr, Handler: s.countRequests(s.routes()), ReadHeaderTimeout: 10 * time.Second}

	go func() {
		s.log.Info("ingress up", "addr", s.cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Error("serve failed", "err", err)
		}
	}()

	// Daily scheduled Slack discussion post (only acts when the owner enabled it).
	schedCtx, schedCancel := context.WithCancel(context.Background())
	go s.runSlackScheduler(schedCtx)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	schedCancel()
	s.grpc.GracefulStop()
	_ = s.store.Close()
	if s.cashfluxClose != nil {
		_ = s.cashfluxClose()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

// runSlackScheduler checks once a minute whether the configured daily post is due and, if so,
// generates and publishes it (RSS always; Slack best-effort). The service guards against
// double-posting within a day; a failed attempt skips the day rather than retrying every tick.
// The outcome message is logged verbatim so a skipped/failed Slack delivery is visible in the
// server log, not just in the manual path's Ack. Stops when ctx is cancelled (shutdown).
func (s *Server) runSlackScheduler(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if msg, posted, err := s.adminSvc.PostScheduledIfDue(ctx, time.Now()); err != nil {
				s.log.Warn("scheduled daily post failed", "err", err)
			} else if posted {
				s.log.Info("scheduled daily post", "outcome", msg)
			}
		}
	}
}
