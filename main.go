package main

import (
	"embed"
	"os"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/psiemens/graceland"
	"github.com/rs/zerolog"
)

//go:embed cadence/FCL.cdc
var fclCdc []byte

// just so that it does not complain
//
//go:embed cadence/*
var _ embed.FS

func main() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(output).With().Timestamp().Logger()

	dw, emu, err := flow.InitEmulator(&logger)
	if err != nil {
		panic(err)
	}

	gl := graceland.NewGroup()

	dw.AddToGroup(gl)
	gl.Add(emu)
	gl.Add(&aether.Aether{
		Logger: &logger,
		FclCdc: fclCdc,
	})

	err = gl.Start()
	if err != nil {
		logger.Error().Err(err).Msg("❗  Server error")
	}
	gl.Stop()
}
