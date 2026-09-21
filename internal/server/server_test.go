package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.lnoxsian.termweb"
	"github.lnoxsian.termweb/internal/config"
	"github.lnoxsian.termweb/internal/terminal"
)

func setupTestServer(t *testing.T) (*Server, *terminal.Manager) {
	cfg := config.DefaultConfig()
	cfg.Terminal.Shell = "/bin/sh"
	mgr := terminal.NewManager(cfg.Terminal.Shell)
	srv, err := NewServer(cfg, mgr, termweb.WebFS)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	return srv, mgr
}

func TestDesktopIndex(t *testing.T) {
	srv, _ := setupTestServer(t)
	mux := srv.setupRoutes()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Termino") {
		t.Error("expected 'Termino' in index HTML response")
	}
	if !strings.Contains(body, "xterm.js") {
		t.Error("expected 'xterm.js' in index HTML response")
	}
	if !strings.Contains(body, "htmx.min.js") {
		t.Error("expected 'htmx.min.js' in index HTML response")
	}
	if !strings.Contains(body, `id="desktop"`) {
		t.Error("expected '#desktop' in index HTML response")
	}
	if !strings.Contains(body, `id="context-menu"`) {
		t.Error("expected '#context-menu' in index HTML response")
	}
}

func TestTerminalLifecycle(t *testing.T) {
	srv, mgr := setupTestServer(t)
	mux := srv.setupRoutes()

	// 1. POST /terminal/new
	postReq := httptest.NewRequest("POST", "/terminal/new", nil)
	postRec := httptest.NewRecorder()
	mux.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on POST /terminal/new, got %d", postRec.Code)
	}

	postBody := postRec.Body.String()
	if !strings.Contains(postBody, "terminal-") {
		t.Errorf("expected terminal window element in response, got: %s", postBody)
	}

	if mgr.ActiveCount() != 1 {
		t.Fatalf("expected 1 active session in manager, got %d", mgr.ActiveCount())
	}

	sessions := mgr.ListSessions()
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	sessionID := sessions[0].ID

	// 2. GET /api/status
	statusReq := httptest.NewRequest("GET", "/api/status", nil)
	statusRec := httptest.NewRecorder()
	mux.ServeHTTP(statusRec, statusReq)

	if statusRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on /api/status, got %d", statusRec.Code)
	}
	var status map[string]interface{}
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil {
		t.Fatalf("failed to decode status json: %v", err)
	}
	if int(status["active_terminals"].(float64)) != 1 {
		t.Errorf("expected 1 active terminal in status, got %v", status["active_terminals"])
	}

	// 3. DELETE /terminal/{id}
	delReq := httptest.NewRequest("DELETE", fmt.Sprintf("/terminal/%s", sessionID), nil)
	delRec := httptest.NewRecorder()
	mux.ServeHTTP(delRec, delReq)

	if delRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on DELETE, got %d", delRec.Code)
	}

	if mgr.ActiveCount() != 0 {
		t.Errorf("expected 0 active sessions after delete, got %d", mgr.ActiveCount())
	}
}

func TestWebSocketTerminalInteraction(t *testing.T) {
	srv, mgr := setupTestServer(t)
	mux := srv.setupRoutes()
	server := httptest.NewServer(mux)
	defer server.Close()

	// Spawn a terminal session
	session, err := mgr.CreateSession("/bin/sh", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/terminal/" + session.ID

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to websocket %s: %v", wsURL, err)
	}
	defer ws.Close()

	// Send resize message
	resizeMsg, _ := json.Marshal(map[string]interface{}{
		"type": "resize",
		"cols": 100,
		"rows": 30,
	})
	if err := ws.WriteMessage(websocket.TextMessage, resizeMsg); err != nil {
		t.Fatalf("failed to send resize: %v", err)
	}

	// Give shell process a brief moment to initialize PTY
	time.Sleep(50 * time.Millisecond)

	// Send echo command
	cmdMsg := []byte("echo 'HELLO_WS_TERMINAL'\n")
	if err := ws.WriteMessage(websocket.BinaryMessage, cmdMsg); err != nil {
		t.Fatalf("failed to send command: %v", err)
	}

	found := false
	deadline := time.Now().Add(4 * time.Second)
	_ = ws.SetReadDeadline(deadline)

	for time.Now().Before(deadline) {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			break
		}
		if strings.Contains(string(msg), "HELLO_WS_TERMINAL") {
			found = true
			break
		}
	}

	if !found {
		t.Error("did not receive expected output from shell over websocket")
	}
}
