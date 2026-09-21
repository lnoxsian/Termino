package terminal

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// StartPTY starts a command with a pseudo-terminal of the specified size.
func StartPTY(cmd *exec.Cmd, rows, cols uint16) (*os.File, error) {
	ws := &pty.Winsize{
		Rows: rows,
		Cols: cols,
	}
	return pty.StartWithSize(cmd, ws)
}

// ResizePTY resizes an active PTY to the specified dimensions.
func ResizePTY(f *os.File, rows, cols uint16) error {
	if f == nil {
		return nil
	}
	ws := &pty.Winsize{
		Rows: rows,
		Cols: cols,
	}
	return pty.Setsize(f, ws)
}
