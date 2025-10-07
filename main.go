package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/aether/pkg/logs"
	"github.com/bjartek/aether/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

//go:embed cadence/FCL.cdc
var fclCdc []byte

// just so that it does not complain
//
//go:embed cadence/*
var _ embed.FS

func main() {
	// Create the Bubble Tea program with the UI model
	p := tea.NewProgram(
		ui.NewModel(),
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Create a zerolog logger that outputs to the TUI logs view
	// This is safe to create before p.Run() because it just sets up the writer
	logger, err := logs.NewLoggerWithFile(p, "aether.log")
	if err != nil {
		panic(err)
	}

	// Start initialization in background, but wait a moment for TUI to be ready
	go func() {
		// Small delay to ensure TUI event loop is running
		// The TUI needs to process the initial WindowSizeMsg before it can handle log messages
		// In practice, this happens almost instantly

		// Initialize emulator and dev wallet
		logger.Info().Msg("Initializing Flow emulator and dev wallet...")
		emu, dw, err := flow.InitEmulator(&logger)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to initialize Flow emulator & dev wallet")
			return
		}
		logger.Info().Msg("Initialization complete")

		// Start emulator
		go func() {
			logger.Info().Msg("Starting Flow emulator...")
			emu.Start()
			logger.Info().Msg("Emulator stopped")
		}()

		// Start dev wallet
		go func() {
			logger.Info().Msg("Starting dev wallet...")
			if err := dw.Start(); err != nil {
				logger.Error().Err(err).Msg("Dev wallet stopped with error")
			}
		}()
	}()

	// Run the program - this blocks until the user quits
	// Once this is called, the TUI event loop starts and can receive log messages
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
