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
	logger, err := logs.NewLoggerWithFile(p, "aether.log")
	if err != nil {
		panic(err)
	}

	fmt.Println("test1")
	// Initialize emulator and dev wallet
	emu, dw, err := flow.InitEmulator(&logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to initialize Flow emulator & dev wallet")
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("test2")

	// Start emulator in background
	go func() {
		logger.Info().Msg("Starting Flow emulator...")
		emu.Start()
		logger.Info().Msg("Emulator stopped")
	}()
	fmt.Println("test3")

	// Start dev wallet in background
	go func() {
		logger.Info().Msg("Starting dev wallet...")
		if err := dw.Start(); err != nil {
			logger.Error().Err(err).Msg("Dev wallet stopped with error")
		}
	}()

	fmt.Println("test4")
	// Run the program
	if _, err := p.Run(); err != nil {
		logger.Error().Err(err).Msg("Error running program")
		fmt.Printf("Error: %v\n", err)
		emu.Stop()
		dw.Stop()
		os.Exit(1)
	}

	fmt.Println("test5")

	// Clean shutdown
	logger.Info().Msg("Shutting down services...")
	emu.Stop()
	dw.Stop()
}
