package server

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"time"

	"github.lnoxsian.termweb/internal/config"
	"github.lnoxsian.termweb/internal/terminal"
)

// Server represents the HTTP/WS web server.
type Server struct {
	cfg        *config.Config
	manager    *terminal.Manager
	httpServer *http.Server
	templates  *template.Template
	webFS      fs.FS
}

// NewServer initializes the server with routes, templates, and embedded filesystem.
func NewServer(cfg *config.Config, manager *terminal.Manager, webFS fs.FS) (*Server, error) {
	tmpl, err := template.ParseFS(webFS, "web/templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	s := &Server{
		cfg:       cfg,
		manager:   manager,
		templates: tmpl,
		webFS:     webFS,
	}

	mux := s.setupRoutes()
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1MB header limit
	}

	return s, nil
}

// Start begins listening and serving HTTP/WS requests.
func (s *Server) Start() error {
	addr := s.httpServer.Addr
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	log.Printf("[termino] server listening on http://%s", addr)
	return s.httpServer.Serve(ln)
}

// Addr returns the listening address of the server.
func (s *Server) Addr() string {
	if s.httpServer != nil {
		return s.httpServer.Addr
	}
	return ""
}

// Shutdown gracefully stops the server and closes all terminal sessions.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Printf("[termino] shutting down server and terminating sessions...")
	s.manager.CloseAll()
	return s.httpServer.Shutdown(ctx)
}
