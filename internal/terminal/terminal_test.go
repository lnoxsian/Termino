package terminal

import (
	"strings"
	"testing"
	"time"
)

func TestSessionAndManager(t *testing.T) {
	mgr := NewManager("/bin/sh")
	if mgr.ActiveCount() != 0 {
		t.Errorf("expected 0 active sessions, got %d", mgr.ActiveCount())
	}

	// Create session
	session, err := mgr.CreateSession("/bin/sh", 80, 24)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	if session.ID == "" {
		t.Error("expected non-empty session ID")
	}
	if mgr.ActiveCount() != 1 {
		t.Errorf("expected 1 active session, got %d", mgr.ActiveCount())
	}

	// Verify retrieval
	retrieved, exists := mgr.GetSession(session.ID)
	if !exists || retrieved != session {
		t.Error("failed to retrieve created session")
	}

	// Test write and read
	writeCmd := "echo 'RETRO_TEST_123'\n"
	if _, err := session.Write([]byte(writeCmd)); err != nil {
		t.Fatalf("failed to write to pty: %v", err)
	}

	readBuf := make([]byte, 1024)
	found := false
	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		n, err := session.Read(readBuf)
		if n > 0 && strings.Contains(string(readBuf[:n]), "RETRO_TEST_123") {
			found = true
			break
		}
		if err != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !found {
		t.Error("expected output 'RETRO_TEST_123' not found in PTY read")
	}

	// Test Resize
	if err := session.Resize(120, 40); err != nil {
		t.Errorf("failed to resize pty: %v", err)
	}
	if session.Cols != 120 || session.Rows != 40 {
		t.Errorf("expected 120x40, got %dx%d", session.Cols, session.Rows)
	}

	// Test Close
	if err := mgr.CloseSession(session.ID); err != nil {
		t.Errorf("error closing session: %v", err)
	}

	if mgr.ActiveCount() != 0 {
		t.Errorf("expected 0 active sessions after close, got %d", mgr.ActiveCount())
	}

	if !session.IsClosed() {
		t.Error("expected session to be marked closed")
	}
}
