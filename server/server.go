// Package server provides the HTTP server for compass-dash.
// It binds to 127.0.0.1 only and serves both the API and the embedded frontend.
package server

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/jeffgeiser/compass-dash/config"
)

// Server holds the HTTP server and shared state.
type Server struct {
	mu  sync.RWMutex
	cfg config.Config
	srv *http.Server
}

// New constructs a Server with the given config and embedded frontend FS.
func New(cfg config.Config, frontendFS fs.FS) *Server {
	s := &Server{cfg: cfg}

	mux := http.NewServeMux()

	// Frontend — serve index.html for root; static files under /static/
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(frontendFS, "index.html")
		if err != nil {
			http.Error(w, "frontend not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
	mux.Handle("/static/", staticHandler(frontendFS))

	// API routes — dispatch by prefix and method
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/config", s.routeConfig)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/refinements", s.routeRefinements)
	mux.HandleFunc("/api/refinements/", s.routeRefinementAction)
	mux.HandleFunc("/api/files", s.handleListFiles)
	mux.HandleFunc("/api/files/", s.handleReadFile)
	mux.HandleFunc("/api/activity", s.handleActivity)

	handler := loggingMiddleware(localhostOnly(mux))

	s.srv = &http.Server{
		Handler: handler,
	}
	return s
}

// ListenAndServe binds to 127.0.0.1 at the configured port and serves.
func (s *Server) ListenAndServe() error {
	s.mu.RLock()
	port := s.cfg.Port
	s.mu.RUnlock()

	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", itoa(port)))
	if err != nil {
		return err
	}
	return s.srv.Serve(ln)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// Addr returns the configured listen address (before starting).
func (s *Server) Addr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return net.JoinHostPort("127.0.0.1", itoa(s.cfg.Port))
}

// routeConfig dispatches GET/PUT /api/config.
func (s *Server) routeConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetConfig(w, r)
	case http.MethodPut:
		s.handlePutConfig(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// routeRefinements dispatches GET /api/refinements.
func (s *Server) routeRefinements(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleListRefinements(w, r)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// routeRefinementAction dispatches POST /api/refinements/{id}/{action}.
func (s *Server) routeRefinementAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.Path // e.g. /api/refinements/2025-01-01-foo/accept
	switch {
	case strings.HasSuffix(path, "/preview"):
		s.handleRefinementPreview(w, r)
	case strings.HasSuffix(path, "/accept-edited"):
		s.handleRefinementAcceptEdited(w, r)
	case strings.HasSuffix(path, "/accept"):
		s.handleRefinementAccept(w, r)
	case strings.HasSuffix(path, "/reject"):
		s.handleRefinementReject(w, r)
	default:
		http.NotFound(w, r)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
