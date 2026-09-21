package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Manager manages the lifecycle of multiple terminal sessions.
type Manager struct {
	mu           sync.RWMutex
	sessions     map[string]*Session
	counter      int
	defaultShell string
}

// NewManager creates a new Manager instance.
func NewManager(defaultShell string) *Manager {
	if defaultShell == "" {
		defaultShell = "/bin/bash"
	}
	return &Manager{
		sessions:     make(map[string]*Session),
		defaultShell: defaultShell,
	}
}

// generateID creates a short unique hex string (4 characters / 2 bytes).
func (m *Manager) generateID() string {
	var b [2]byte
	for {
		if _, err := rand.Read(b[:]); err != nil {
			return fmt.Sprintf("%04x", m.counter)
		}
		id := hex.EncodeToString(b[:])
		if _, exists := m.sessions[id]; !exists {
			return id
		}
	}
}

// CreateSession starts a new shell process in a PTY and registers the session.
func (m *Manager) CreateSession(shell string, cols, rows uint16) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if shell == "" {
		shell = m.defaultShell
	}

	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	m.counter++
	id := m.generateID()
	title := filepath.Base(shell)

	// Calculate cascaded window position
	offsetIndex := (m.counter - 1) % 10
	initialX := 40 + offsetIndex*28
	initialY := 40 + offsetIndex*24
	initialW := 720
	initialH := 440

	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
	)

	// Set working directory to user's home or current dir
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		cmd.Dir = home
	}

	ptyFile, err := StartPTY(cmd, rows, cols)
	if err != nil {
		return nil, fmt.Errorf("failed to start pty for shell %q: %w", shell, err)
	}

	session := NewSession(id, title, shell, cmd, ptyFile, rows, cols, initialX, initialY, initialW, initialH)
	m.sessions[id] = session

	return session, nil
}

// GetSession retrieves a session by ID.
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[id]
	return session, exists
}

// CloseSession closes and removes a session by ID.
func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	session, exists := m.sessions[id]
	if exists {
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	if !exists {
		return nil
	}
	return session.Close()
}

// ListSessions returns a slice of all active sessions.
func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		list = append(list, s)
	}
	return list
}

// ActiveCount returns the number of active sessions.
func (m *Manager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// CloseAll closes all active sessions.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	sessionsToClose := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessionsToClose = append(sessionsToClose, s)
	}
	m.sessions = make(map[string]*Session)
	m.mu.Unlock()

	for _, s := range sessionsToClose {
		_ = s.Close()
	}
}
