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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"github.com/monstercameron/earlcameron/internal/config"
	"github.com/monstercameron/earlcameron/internal/content"
	"github.com/monstercameron/earlcameron/proto/sitepb"
	"google.golang.org/grpc"
)

// Server owns the ingress lifecycle.
type Server struct {
	cfg    config.Config
	log    *slog.Logger
	grpc   *grpc.Server
	tunnel http.Handler
}

// New builds the gRPC server, registers the services, and wraps them in the GoGRPCBridge
// WebSocket tunnel. It returns an error if the tunnel handler cannot be constructed.
func New(cfg config.Config) (*Server, error) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	grpcSrv := grpc.NewServer()
	sitepb.RegisterContentServiceServer(grpcSrv, content.New())

	tunnel, err := grpctunnel.BuildBridgeHandler(grpcSrv, grpctunnel.BridgeConfig{
		// Dev: allow any WebSocket origin. TODO: restrict to the site origin in production.
		CheckOrigin: func(_ *http.Request) bool { return true },
	})
	if err != nil {
		return nil, err
	}
	return &Server{cfg: cfg, log: log, grpc: grpcSrv, tunnel: tunnel}, nil
}

// routes builds the request multiplexer.
func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	// Data plane: gRPC over WebSocket.
	mux.Handle("/socket", s.tunnel)
	mux.Handle("/socket/", s.tunnel)
	// Document plane (HTTP GET).
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/", s.ssrShell)
	return mux
}

// healthz reports liveness for the deploy health check and rollback logic.
func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

// ssrShell serves the standard site (SEO + no-WASM failsafe). TODO: server-render the GWC
// component tree here (SSR + hydrate); placeholder for now.
func (s *Server) ssrShell(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!doctype html><meta charset=utf-8><title>Earl Cameron</title><h1>earlcameron.com</h1>"))
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
