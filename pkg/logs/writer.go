package logs

import (
	"bufio"
	"bytes"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// LogWriter is a custom io.Writer that sends log lines to a Bubble Tea program.
type LogWriter struct {
	program *tea.Program
	buffer  bytes.Buffer
	mu      sync.Mutex
}

// NewLogWriter creates a new log writer that sends lines to the Bubble Tea program.
func NewLogWriter(program *tea.Program) *LogWriter {
	return &LogWriter{
		program: program,
	}
}

// Write implements io.Writer and sends complete lines to the Bubble Tea program.
func (w *LogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Write to buffer
	n, err = w.buffer.Write(p)
	if err != nil {
		return n, err
	}

	// Process complete lines
	scanner := bufio.NewScanner(&w.buffer)
	var remaining bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()
		if w.program != nil {
			w.program.Send(LogLineMsg{Line: line + "\n"})
		}
	}

	// Keep any incomplete line in the buffer
	if w.buffer.Len() > 0 {
		lastByte := w.buffer.Bytes()[w.buffer.Len()-1]
		if lastByte != '\n' {
			// Find the last newline
			data := w.buffer.Bytes()
			lastNewline := bytes.LastIndexByte(data, '\n')
			if lastNewline >= 0 {
				remaining.Write(data[lastNewline+1:])
			} else {
				remaining.Write(data)
			}
		}
	}

	w.buffer = remaining
	return n, nil
}

// Sync implements zapcore.WriteSyncer.
func (w *LogWriter) Sync() error {
	return nil
}
