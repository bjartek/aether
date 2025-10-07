package logs

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// NewLogger creates a zerolog logger that outputs to a channel-buffered writer.
// Call AttachProgram on the returned writer after creating the Tea program.
func NewLogger(bufferSize int) (zerolog.Logger, *LogWriter) {
	// Create the TUI writer with channel buffering
	tuiWriter := NewLogWriter(bufferSize)

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

	return logger, tuiWriter
}

// NewLoggerWithFile creates a zerolog logger that outputs to both a file and a channel-buffered writer.
// Call AttachProgram on the returned writer after creating the Tea program.
func NewLoggerWithFile(logFilePath string, bufferSize int) (zerolog.Logger, *LogWriter, error) {
	// Create the TUI writer with channel buffering
	tuiWriter := NewLogWriter(bufferSize)

	// Open log file
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return zerolog.Logger{}, nil, err
	}

	// Store the file handle in the writer for proper cleanup
	tuiWriter.SetLogFile(logFile)

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

	return logger, tuiWriter, nil
}
