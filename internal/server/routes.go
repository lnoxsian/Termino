package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
)

// DesktopViewData contains template rendering context.
type DesktopViewData struct {
	Theme      string
	Shell      string
	FontSize   int
	Scrollback int
}

// setupRoutes configures all HTTP and WebSocket endpoints on http.ServeMux.
func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Sub-filesystem rooted at "web"
	subFS, err := fs.Sub(s.webFS, "web")
	if err != nil {
		log.Fatalf("failed to create sub filesystem: %v", err)
	}

	// Static & Vendor asset file servers with caching to minimize HTTP requests
	fileServer := http.FileServer(http.FS(subFS))
	cachedAssets := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		fileServer.ServeHTTP(w, r)
	})
	mux.Handle("GET /static/", cachedAssets)
	mux.Handle("GET /vendor/", cachedAssets)

	// Main Desktop Page
	mux.HandleFunc("GET /", s.handleDesktopIndex)

	// Terminal Lifecycle Endpoints (HTMX)
	mux.HandleFunc("POST /terminal/new", s.handleTerminalNew)
	mux.HandleFunc("DELETE /terminal/{id}", s.handleTerminalDelete)

	// WebSocket Endpoint
	mux.HandleFunc("GET /ws/terminal/{id}", s.handleTerminalWS)

	// API Status Endpoint
	mux.HandleFunc("GET /api/status", s.handleAPIStatus)

	return mux
}

// handleDesktopIndex renders the main 90s desktop layout.
func (s *Server) handleDesktopIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := DesktopViewData{
		Theme:      s.cfg.Desktop.Theme,
		Shell:      s.cfg.Terminal.Shell,
		FontSize:   s.cfg.Terminal.FontSize,
		Scrollback: s.cfg.Terminal.Scrollback,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("[http] error rendering index.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleTerminalNew creates a new terminal session and returns the window HTML.
func (s *Server) handleTerminalNew(w http.ResponseWriter, r *http.Request) {
	session, err := s.manager.CreateSession("", 80, 24)
	if err != nil {
		log.Printf("[terminal] error creating session: %v", err)
		http.Error(w, fmt.Sprintf("Failed to spawn terminal: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, "terminal.html", session); err != nil {
		log.Printf("[http] error rendering terminal.html: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleTerminalDelete terminates a terminal session and returns HTML to remove the window.
func (s *Server) handleTerminalDelete(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	if sessionID == "" {
		http.Error(w, "missing session ID", http.StatusBadRequest)
		return
	}

	_ = s.manager.CloseSession(sessionID)

	w.WriteHeader(http.StatusOK)
}

// handleAPIStatus returns JSON status of server and active terminal sessions.
func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"active_terminals": s.manager.ActiveCount(),
		"default_shell":    s.cfg.Terminal.Shell,
		"theme":            s.cfg.Desktop.Theme,
		"font_size":        s.cfg.Terminal.FontSize,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(status)
}
