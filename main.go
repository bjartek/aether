package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/aether/pkg/logs"
	"github.com/bjartek/aether/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/psiemens/graceland"
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
	logger := logs.NewLogger(p)

	// Example: Start a goroutine that generates some log messages
	go func() {
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

		fmt.Println("lets goo")
		err = gl.Start()
		if err != nil {
			logger.Error().Err(err).Msg("❗  Server error")
		}
		fmt.Println("foobar")
		gl.Stop()
	}()

	// Run the program
	if _, err := p.Run(); err != nil {
		logger.Error().Err(err).Msg("Error running program")
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
