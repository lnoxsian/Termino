package terminal

import (
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Session represents an active terminal instance backed by a PTY.
type Session struct {
	ID            string
	Title         string
	Shell         string
	Cmd           *exec.Cmd
	Pty           *os.File
	Rows          uint16
	Cols          uint16
	InitialX      int
	InitialY      int
	InitialWidth  int
	InitialHeight int
	CreatedAt     time.Time

	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
	closeChan chan struct{}
}

// NewSession creates and initializes a new Session struct.
func NewSession(id, title, shell string, cmd *exec.Cmd, ptyFile *os.File, rows, cols uint16, x, y, w, h int) *Session {
	return &Session{
		ID:            id,
		Title:         title,
		Shell:         shell,
		Cmd:           cmd,
		Pty:           ptyFile,
		Rows:          rows,
		Cols:          cols,
		InitialX:      x,
		InitialY:      y,
		InitialWidth:  w,
		InitialHeight: h,
		CreatedAt:     time.Now(),
		closeChan:     make(chan struct{}),
	}
}

// Read reads output from the PTY.
func (s *Session) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	ptyFile := s.Pty
	closed := s.closed
	s.mu.Unlock()

	if closed || ptyFile == nil {
		return 0, io.EOF
	}
	return ptyFile.Read(p)
}

// Write writes input into the PTY.
func (s *Session) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	ptyFile := s.Pty
	closed := s.closed
	s.mu.Unlock()

	if closed || ptyFile == nil {
		return 0, io.EOF
	}
	return ptyFile.Write(p)
}

// Resize updates the terminal dimensions on the PTY.
func (s *Session) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed || s.Pty == nil {
		return nil
	}

	s.Cols = cols
	s.Rows = rows
	return ResizePTY(s.Pty, rows, cols)
}

// Close terminates the shell process and closes the PTY.
func (s *Session) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		close(s.closeChan)
		ptyFile := s.Pty
		cmd := s.Cmd
		s.mu.Unlock()

		if ptyFile != nil {
			_ = ptyFile.Close()
		}

		if cmd != nil && cmd.Process != nil {
			pid := cmd.Process.Pid
			// Signal the entire process group to avoid zombie / orphaned child processes
			_ = syscall.Kill(-pid, syscall.SIGTERM)
			_ = cmd.Process.Signal(syscall.SIGTERM)

			// Wait briefly for exit, then force kill process group
			done := make(chan struct{})
			go func() {
				_ = cmd.Wait()
				close(done)
			}()

			select {
			case <-done:
			case <-time.After(150 * time.Millisecond):
				_ = syscall.Kill(-pid, syscall.SIGKILL)
				_ = cmd.Process.Kill()
			}
		}
	})
	return err
}

// IsClosed reports whether the session has been closed.
func (s *Session) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// WaitClosed returns a channel that is closed when the session closes.
func (s *Session) WaitClosed() <-chan struct{} {
	return s.closeChan
}
