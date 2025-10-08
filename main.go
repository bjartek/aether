package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/aether/pkg/logs"
	"github.com/bjartek/aether/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
)

//go:embed cadence/FCL.cdc
var fclCdc []byte

// just so that it does not complain
//
//go:embed cadence/*
var _ embed.FS

func main() {
	// Parse command line flags
	verbose := flag.Bool("verbose", false, "Enable verbose (debug) logging")
	flag.Parse()

	// Create logger with channel buffering (1000 log lines)
	// This allows logging before the Tea program starts
	logger, logWriter := logs.NewLogger(1000)

	// Set log level based on verbose flag
	if *verbose {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	logger.Info().Msg("Initializing Flow emulator and dev wallet...")
	emu, dw, err := flow.InitEmulator(&logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to initialize Flow emulator & dev wallet")
		panic(err)
	}
	logger.Info().Msg("Initialization complete")

	// Channel to signal when emulator is ready
	emulatorReady := make(chan struct{})

	// Start emulator in background
	go func() {
		logger.Info().Msg("Starting Flow emulator...")
		go func() {
			emu.Start()
			logger.Info().Msg("Emulator stopped")
		}()
		
		// Wait a moment for emulator to start listening
		time.Sleep(500 * time.Millisecond)
		logger.Info().Msg("Emulator is ready")
		close(emulatorReady)
	}()

	// Start dev wallet in background
	go func() {
		logger.Info().Msg("Starting dev wallet...")
		if err := dw.Start(); err != nil {
			logger.Error().Err(err).Msg("Dev wallet stopped with error")
		}
	}()

	a := aether.Aether{
		Logger: &logger,
		FclCdc: fclCdc,
	}

	// Now create the Bubble Tea program
	p := tea.NewProgram(
		ui.NewModel(),
		tea.WithAltScreen(), // Use alternate screen buffer
	)

	// Start aether server after emulator is ready with tea program
	go func() {
		<-emulatorReady
		logger.Info().Msg("Starting aether server")
		a.Start(p)
	}()

	// Attach the Tea program to the log writer
	// This will drain any buffered logs and start sending new logs to the UI
	logWriter.AttachProgram(p)


	// Run the program - this blocks until the user quits
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}

	// Clean up
	logWriter.Close()
	emu.Stop()
	dw.Stop()
	a.Stop()
}
