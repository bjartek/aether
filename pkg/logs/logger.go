package logs

import (
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
)

// NewLogger creates a zerolog logger that outputs only to the TUI logs view.
// This prevents logs from breaking the TUI display by writing to stdout.
func NewLogger(program *tea.Program) zerolog.Logger {
	// Create the TUI writer
	tuiWriter := NewLogWriter(program)

	// Create console writer for pretty output
	consoleWriter := zerolog.ConsoleWriter{
		Out:        tuiWriter,
		TimeFormat: "15:04:05",
		NoColor:    false,
	}

	// Create logger with console writer
	logger := zerolog.New(consoleWriter).
		With().
		Timestamp().
		Logger()

	return logger
}

// NewLoggerWithFile creates a zerolog logger that outputs to both a file and the TUI logs view.
// Use this if you want persistent logs while the TUI is running.
func NewLoggerWithFile(program *tea.Program, logFilePath string) (zerolog.Logger, error) {
	// Create the TUI writer
	tuiWriter := NewLogWriter(program)

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return zerolog.Logger{}, err
	}

	// Create multi-writer for both file and TUI
	multiWriter := io.MultiWriter(logFile, tuiWriter)

	// Create console writer for pretty output
	consoleWriter := zerolog.ConsoleWriter{
		Out:        multiWriter,
		TimeFormat: "15:04:05",
		NoColor:    false,
	}

	// Create logger
	logger := zerolog.New(consoleWriter).
		With().
		Timestamp().
		Logger()

	return logger, nil
}
