package debug

import (
	"os"

	"github.com/rs/zerolog"
)

var Logger zerolog.Logger

func init() {
	// Create or open debug log file
	file, err := os.OpenFile("aether-debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// Fallback to stderr if file can't be opened
		Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
		return
	}
	
	Logger = zerolog.New(file).With().Timestamp().Logger()
}
