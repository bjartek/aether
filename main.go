package main

import (
	"os"
	"time"

	"github.com/bjartek/aether/pkg/flow"
	"github.com/rs/zerolog"
)

func main() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(output).With().Timestamp().Logger()

	err := flow.InitEmulator(&logger)
	if err != nil {
		panic(err)
	}
}
