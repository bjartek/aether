package main

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
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
	configPath := flag.String("config", "", "Path to configuration file")
	network := flag.String("n", "", "Network to follow: emulator, testnet, or mainnet (overrides config)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Apply command-line overrides
	if *network != "" {
		cfg.Network = *network
	}

	// Create logger with or without file output based on config
	var logger zerolog.Logger
	var logWriter *logs.LogWriter

	if cfg.Logging.File.Enabled && cfg.Logging.File.Path != "" {
		// Create logger with file output
		var err2 error
		logger, logWriter, err2 = logs.NewLoggerWithFile(cfg.Logging.File.Path, cfg.Logging.File.BufferSize)
		if err2 != nil {
			fmt.Printf("Failed to create logger: %v\n", err2)
			os.Exit(1)
		}
	} else {
		// Create logger without file output (TUI only)
		logger, logWriter = logs.NewLogger(cfg.Logging.File.BufferSize)
	}

	// Set global log level from config
	logger = logger.Level(logs.ParseLogLevel(cfg.Logging.Level.Global))

	// Create component-specific loggers with their configured levels
	aetherLogger := logs.WithComponent(logger, "aether").Level(logs.ParseLogLevel(cfg.Logging.Level.Aether))
	emulatorLogger := logs.WithComponent(logger, "emulator").Level(logs.ParseLogLevel(cfg.Logging.Level.Emulator))
	walletLogger := logs.WithComponent(logger, "dev-wallet").Level(logs.ParseLogLevel(cfg.Logging.Level.DevWallet))
	gatewayLogger := logs.WithComponent(logger, "evm-gateway").Level(logs.ParseLogLevel(cfg.Logging.Level.EVMGateway))

	if cfg.Logging.File.Enabled {
		aetherLogger.Info().Str("file", cfg.Logging.File.Path).Msg("Logging to file for debugging")
	}

	// Log configuration source
	if *configPath != "" {
		aetherLogger.Info().Str("config", *configPath).Msg("Loaded configuration from file")
	} else {
		aetherLogger.Info().Msg("Using default configuration")
	}

	// Initialize components based on whether we're following a network or running locally
	var emu *server.EmulatorServer
	var dw *devWallet.Server
	var gateway *flow.Gateway
	var gatewayCfg gatewayConfig.Config
	var emulatorReady chan struct{}

	if cfg.Network == "emulator" {
		// Local mode: start emulator, dev wallet, and EVM gateway
		aetherLogger.Info().Msg("Initializing Flow emulator, dev wallet, and EVM gateway...")
		var err2 error
		emu, dw, err2 = flow.InitEmulator(&emulatorLogger, cfg)
		if err2 != nil {
			aetherLogger.Error().Err(err2).Msg("Failed to initialize Flow emulator & dev wallet")
			panic(err2)
		}

		aetherLogger.Info().Msg("Initializing EVM gateway...")
		var err3 error
		gateway, gatewayCfg, err3 = flow.InitGateway(gatewayLogger, cfg)
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
		aetherLogger.Info().Str("network", cfg.Network).Msg("Following network - no local services will be started")
		emulatorReady = make(chan struct{})
		close(emulatorReady) // Immediately ready since we're not starting an emulator
	}

	a := aether.Aether{
		Logger:  &aetherLogger,
		FclCdc:  fclCdc,
		Network: cfg.Network,
		Config:  cfg,
	}

	//this is the old way
	//model := ui.NewModelWithConfig(cfg)
	model := ui.NewTestModelWithConfig(cfg)

	// Now create the Bubble Tea program with config
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(), // Use alternate screen buffe
	)

	// Start EVM gateway after emulator is ready (only in local mode)
	if cfg.Network == "emulator" {
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
	if cfg.Network == "emulator" {
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
