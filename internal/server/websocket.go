package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
)

// PTY buffer pool to eliminate heap allocations per connection
var ptyBufferPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 4096)
		return &b
	},
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:    2048,
	WriteBufferSize:   2048,
	EnableCompression: false, // Disabling compression saves CPU on fast terminal text streams
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// wsControlMessage represents client-sent control commands (e.g. resize).
type wsControlMessage struct {
	Type string `json:"type"`
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
	Data string `json:"data"`
}

// safeWSConn wraps websocket.Conn to serialize write operations safely.
type safeWSConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (s *safeWSConn) WriteMessage(messageType int, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.WriteMessage(messageType, data)
}

func (s *safeWSConn) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.conn.Close()
}

// handleTerminalWS handles WebSocket connections for a specific terminal session.
func (s *Server) handleTerminalWS(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	prefix := "/ws/terminal/"
	if len(path) <= len(prefix) {
		http.Error(w, "missing session ID", http.StatusBadRequest)
		return
	}
	sessionID := path[len(prefix):]

	session, exists := s.manager.GetSession(sessionID)
	if !exists {
		http.Error(w, "terminal session not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error for session %s: %v", sessionID, err)
		return
	}
	defer conn.Close()

	// Protect against memory exhaustion from oversized messages
	conn.SetReadLimit(64 * 1024)

	safeConn := &safeWSConn{conn: conn}

	done := make(chan struct{})
	var closeOnce sync.Once
	signalDone := func() {
		closeOnce.Do(func() {
			close(done)
		})
	}

	// Goroutine 1: PTY Output -> WebSocket
	go func() {
		defer signalDone()

		bufPtr := ptyBufferPool.Get().(*[]byte)
		buf := *bufPtr
		defer ptyBufferPool.Put(bufPtr)

		for {
			n, err := session.Read(buf)
			if n > 0 {
				if writeErr := safeConn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				var pathErr *os.PathError
				if errors.Is(err, io.EOF) || (errors.As(err, &pathErr) && errors.Is(pathErr.Err, syscall.EIO)) {
					_ = safeConn.WriteMessage(websocket.TextMessage, []byte("\r\n\x1b[33m[Process completed]\x1b[0m\r\n"))
				}
				return
			}
		}
	}()

	// Goroutine 2: WebSocket Input -> PTY
	go func() {
		defer signalDone()
		for {
			msgType, payload, err := conn.ReadMessage()
			if err != nil {
				return
			}

			if msgType == websocket.BinaryMessage {
				if _, err := session.Write(payload); err != nil {
					return
				}
				continue
			}

			if msgType == websocket.TextMessage {
				// Fast path: only unmarshal if payload starts with '{"type":'
				if bytes.HasPrefix(payload, []byte(`{"type":`)) {
					var ctrl wsControlMessage
					if err := json.Unmarshal(payload, &ctrl); err == nil && ctrl.Type != "" {
						switch ctrl.Type {
						case "resize":
							if ctrl.Cols > 0 && ctrl.Rows > 0 {
								_ = session.Resize(ctrl.Cols, ctrl.Rows)
							}
						case "input":
							if ctrl.Data != "" {
								_, _ = session.Write([]byte(ctrl.Data))
							}
						}
						continue
					}
				}

				// Fallback: raw text sent directly to PTY
				if _, err := session.Write(payload); err != nil {
					return
				}
			}
		}
	}()

	// Wait until either goroutine signals done or session closes
	select {
	case <-done:
	case <-session.WaitClosed():
	}

	_ = safeConn.Close()
	_ = s.manager.CloseSession(sessionID)
}
