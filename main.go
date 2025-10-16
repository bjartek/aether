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
	devWallet "github.com/onflow/fcl-dev-wallet/go/wallet"
	"github.com/onflow/flow-emulator/server"
	gatewayConfig "github.com/onflow/flow-evm-gateway/config"
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
	logFile := flag.String("log-file", "", "Log to file (e.g. aether-debug.log)")
	network := flag.String("n", "", "Network to follow: testnet or mainnet")
	flag.Parse()

	// Create logger with or without file output
	var logger zerolog.Logger
	var logWriter *logs.LogWriter
	var err error

	if *logFile != "" {
		// Create logger with file output for debugging and channel buffering (1000 log lines)
		logger, logWriter, err = logs.NewLoggerWithFile(*logFile, 1000)
		if err != nil {
			fmt.Printf("Failed to create logger: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Create logger without file output (TUI only)
		logger, logWriter = logs.NewLogger(1000)
	}

	// Set log level based on verbose flag
	if *verbose {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	// Create component-specific loggers
	aetherLogger := logs.WithComponent(logger, "aether")
	emulatorLogger := logs.WithComponent(logger, "emulator")
	walletLogger := logs.WithComponent(logger, "dev-wallet")
	gatewayLogger := logs.WithComponent(logger, "evm-gateway").Level(zerolog.ErrorLevel)

	if *logFile != "" {
		aetherLogger.Info().Str("file", *logFile).Msg("Logging to file for debugging")
	}

	// Validate network flag if provided
	if *network != "" && *network != "testnet" && *network != "mainnet" {
		aetherLogger.Error().Str("network", *network).Msg("Invalid network specified. Use 'testnet' or 'mainnet'")
		fmt.Printf("Invalid network: %s. Use 'testnet' or 'mainnet'\n", *network)
		os.Exit(1)
	}

	// Initialize components based on whether we're following a network or running locally
	var emu *server.EmulatorServer
	var dw *devWallet.Server
	var gateway *flow.Gateway
	var gatewayCfg gatewayConfig.Config
	var emulatorReady chan struct{}

	if *network == "" {
		// Local mode: start emulator, dev wallet, and EVM gateway
		aetherLogger.Info().Msg("Initializing Flow emulator, dev wallet, and EVM gateway...")
		var err2 error
		emu, dw, err2 = flow.InitEmulator(&emulatorLogger)
		if err2 != nil {
			aetherLogger.Error().Err(err2).Msg("Failed to initialize Flow emulator & dev wallet")
			panic(err2)
		}

		aetherLogger.Info().Msg("Initializing EVM gateway...")
		var err3 error
		gateway, gatewayCfg, err3 = flow.InitGateway(gatewayLogger)
		if err3 != nil {
			aetherLogger.Error().Err(err3).Msg("Failed to initialize EVM gateway")
			panic(err3)
		}
		aetherLogger.Info().Msg("EVM gateway initialization complete")
		aetherLogger.Info().Msg("All initialization complete")

		// Channel to signal when emulator is ready
		emulatorReady = make(chan struct{})

		// Start emulator in background
		go func() {
			emulatorLogger.Info().Msg("Starting Flow emulator...")
			go func() {
				emu.Start()
				emulatorLogger.Info().Msg("Emulator stopped")
			}()

			// Wait for emulator to start listening
			time.Sleep(1 * time.Second)
			emulatorLogger.Info().Msg("Emulator is ready")
			close(emulatorReady)
		}()

		// Start dev wallet in background
		go func() {
			walletLogger.Info().Msg("Starting dev wallet...")
			if err := dw.Start(); err != nil {
				walletLogger.Error().Err(err).Msg("Dev wallet stopped with error")
			}
		}()
	} else {
		// Network mode: following testnet or mainnet
		aetherLogger.Info().Str("network", *network).Msg("Following network - no local services will be started")
		emulatorReady = make(chan struct{})
		close(emulatorReady) // Immediately ready since we're not starting an emulator
	}

	a := aether.Aether{
		Logger:  &aetherLogger,
		FclCdc:  fclCdc,
		Network: *network,
	}

	// Now create the Bubble Tea program
	p := tea.NewProgram(
		ui.NewModel(),
		tea.WithAltScreen(), // Use alternate screen buffer
	)

	// Start EVM gateway after emulator is ready (only in local mode)
	if *network == "" {
		go func() {
			gatewayLogger.Info().Msg("Waiting for emulator to be ready...")
			<-emulatorReady
			// Wait a few seconds for emulator to fully initialize
			time.Sleep(3 * time.Second)
			gatewayLogger.Info().Msg("Starting EVM gateway bootstrap...")
			gateway.Start(gatewayCfg)
			<-gateway.Ready()
		}()
	}

	// Start aether server after emulator is ready with tea program
	go func() {
		<-emulatorReady
		aetherLogger.Info().Msg("Starting aether server")
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
	aetherLogger.Info().Msg("Shutting down...")
	logWriter.Close()
	if *network == "" {
		// Only stop local services if they were started
		gatewayLogger.Info().Msg("Stopping EVM gateway...")
		gateway.Stop()
		gatewayLogger.Info().Msg("EVM gateway stopped")
		emulatorLogger.Info().Msg("Stopping emulator...")
		emu.Stop()
		walletLogger.Info().Msg("Stopping dev wallet...")
		dw.Stop()
	}
	aetherLogger.Info().Msg("Stopping aether server...")
	a.Stop()
	aetherLogger.Info().Msg("All services stopped cleanly")
}
