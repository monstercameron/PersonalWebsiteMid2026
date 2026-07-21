// Package server wires the single ingress HTTP server.
//
// It will host BOTH the gRPC-over-WebSocket tunnel (the app data plane, added with the first
// service) and the document plane (wasm assets, SSR pages, RSS, PDFs). Browser<->server app
// comms are gRPC/WS only; there is deliberately no ad-hoc REST API. Today it serves /healthz,
// static files, and an SSR placeholder so the foundation is runnable and verifiable.
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

	"github.com/monstercameron/earlcameron/internal/config"
)

// Server owns the ingress lifecycle.
type Server struct {
	cfg config.Config
	log *slog.Logger
}

// New constructs a Server from configuration.
func New(cfg config.Config) *Server {
	return &Server{cfg: cfg, log: slog.New(slog.NewJSONHandler(os.Stdout, nil))}
}

// routes builds the request multiplexer.
func (s *Server) routes() *http.ServeMux {
	mux := http.NewServeMux()
	// Document plane (HTTP GET). TODO: /socket tunnel, RSS, résumé PDF, /apps/cashflux.
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/", s.ssrShell)
	return mux
}

// healthz reports liveness for the deploy health check and rollback logic.
func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("ok"))
}

// ssrShell server-renders the standard site (SEO + no-WASM failsafe). The GWC/WASM terminal
// will later enhance over this markup. TODO: render real content per detected locale.
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
