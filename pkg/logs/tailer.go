package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// LogLineMsg is sent when a new log line is read from the file.
type LogLineMsg struct {
	Line string
	Err  error
}

// Tailer watches a file and sends new lines as they are written.
type Tailer struct {
	filePath string
	ctx      context.Context
	cancel   context.CancelFunc
	file     *os.File
	reader   *bufio.Reader
}

// NewTailer creates a new file tailer.
func NewTailer(filePath string) *Tailer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Tailer{
		filePath: filePath,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins tailing the file and returns a Bubble Tea command.
func (t *Tailer) Start() tea.Cmd {
	return func() tea.Msg {
		// Open the file on first start
		if t.file == nil {
			file, err := os.Open(t.filePath)
			if err != nil {
				return LogLineMsg{Err: fmt.Errorf("failed to open file: %w", err)}
			}
			t.file = file
			t.reader = bufio.NewReader(file)
		}
		return t.tail()
	}
}

// Stop stops the tailer.
func (t *Tailer) Stop() {
	if t.cancel != nil {
		t.cancel()
	}
	if t.file != nil {
		t.file.Close()
		t.file = nil
		t.reader = nil
	}
}

// tail reads the file and sends new lines as messages.
func (t *Tailer) tail() tea.Msg {
	// Check if context is cancelled
	select {
	case <-t.ctx.Done():
		return nil
	default:
	}

	// Try to read a line
	line, err := t.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			// No more data available, wait and try again
			time.Sleep(100 * time.Millisecond)
			return t.tail()
		}
		return LogLineMsg{Err: fmt.Errorf("error reading file: %w", err)}
	}

	// Send the line back to the program
	return LogLineMsg{Line: line}
}

// WaitForMoreLines returns a command that continues tailing.
func WaitForMoreLines(t *Tailer) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(100 * time.Millisecond)
		return t.tail()
	}
}
