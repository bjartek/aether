package logs

import (
	"bufio"
	"bytes"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// LogLineMsg is sent when a new log line is available
type LogLineMsg struct {
	Line string
	Err  error
}

// LogWriter is a custom io.Writer that sends log lines to a channel or Bubble Tea program.
// It buffers logs to a channel before the Tea program is ready, then drains them when attached.
type LogWriter struct {
	program *tea.Program
	buffer  bytes.Buffer
	mu      sync.Mutex
	logChan chan string
	logFile *os.File
}

// NewLogWriter creates a new log writer that sends lines to a buffered channel.
// The channel will buffer logs until AttachProgram is called.
func NewLogWriter(bufferSize int) *LogWriter {
	return &LogWriter{
		logChan: make(chan string, bufferSize),
	}
}

// SetLogFile sets the log file for persistent logging.
func (w *LogWriter) SetLogFile(file *os.File) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.logFile = file
}

// AttachProgram attaches a Bubble Tea program and drains any buffered logs.
// This should be called after the Tea program is created.
func (w *LogWriter) AttachProgram(program *tea.Program) {
	w.mu.Lock()
	w.program = program
	w.mu.Unlock()

	// Drain buffered logs from channel
	go func() {
		for line := range w.logChan {
			w.mu.Lock()
			if w.program != nil {
				w.program.Send(LogLineMsg{Line: line})
			}
			w.mu.Unlock()
		}
	}()
}

// Write implements io.Writer and sends complete lines to the channel or program.
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
		line := scanner.Text() + "\n"
		// Send to channel (non-blocking if program not attached yet)
		select {
		case w.logChan <- line:
		default:
			// Channel full, drop oldest log (shouldn't happen with large buffer)
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

// Close closes the log channel and file.
func (w *LogWriter) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.logChan != nil {
		close(w.logChan)
		w.logChan = nil
	}
	if w.logFile != nil {
		_ = w.logFile.Sync()
		w.logFile.Close()
		w.logFile = nil
	}
}

// Sync implements zapcore.WriteSyncer.
func (w *LogWriter) Sync() error {
	return nil
}
