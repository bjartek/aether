package main

import (
	"embed"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/bjartek/aether/pkg/aether"
	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/aether/pkg/frontend"
	"github.com/bjartek/aether/pkg/logs"
	"github.com/bjartek/aether/pkg/tabbedtui"
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

// isPortAvailable checks if a port is available for binding
func isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// checkRequiredPorts validates that all required ports are available
func checkRequiredPorts(cfg *config.Config) error {
	if cfg.Network != "emulator" {
		// Only check ports in emulator mode
		return nil
	}

	type portCheck struct {
		port int
		name string
	}

	portsToCheck := []portCheck{
		{cfg.Ports.Emulator.GRPC, "Emulator gRPC"},
		{cfg.Ports.Emulator.REST, "Emulator REST"},
		{cfg.Ports.Emulator.Admin, "Emulator Admin"},
		{cfg.Ports.Emulator.Debugger, "Emulator Debugger"},
		{cfg.Ports.DevWallet, "Dev Wallet"},
		{cfg.Ports.EVM.RPC, "EVM Gateway RPC"},
		{cfg.Ports.EVM.Profiler, "EVM Gateway Profiler"},
		{cfg.Ports.EVM.Metrics, "EVM Gateway Metrics"},
	}

	var unavailablePorts []string
	for _, pc := range portsToCheck {
		if !isPortAvailable(pc.port) {
			unavailablePorts = append(unavailablePorts, fmt.Sprintf("%s (port %d)", pc.name, pc.port))
		}
	}

	if len(unavailablePorts) > 0 {
		return fmt.Errorf("the following ports are already in use:\n  - %s\n\nPlease stop the services using these ports or configure different ports in your config file", 
			joins(unavailablePorts, "\n  - "))
	}

	return nil
}

// joins is a simple helper to join strings with a separator
func joins(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	network := flag.String("n", "", "Network to follow: emulator, testnet, or mainnet (overrides config)")
	debugFlag := flag.Bool("debug", false, "Enable debug logging to aether-debug.log")
	flag.BoolVar(debugFlag, "d", false, "Enable debug logging to aether-debug.log (shorthand)")
	flag.Parse()

	// Create debug logger if --debug/-d flag is set
	var debugLogger zerolog.Logger
	if *debugFlag {
		// Delete existing debug log file
		_ = os.Remove("aether-debug.log")

		// Create new debug log file
		debugLogFile, err := os.OpenFile("aether-debug.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Printf("Failed to create debug log file: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = debugLogFile.Close() }()

		debugLogger = zerolog.New(debugLogFile).
			With().
			Timestamp().
			Logger().
			Level(zerolog.DebugLevel)

		debugLogger.Info().Msg("Debug logging enabled")
	} else {
		// No debug logging - use no-op logger
		debugLogger = zerolog.Nop()
	}

	// Load configuration with debug logger
	cfg, err := config.Load(*configPath, debugLogger)
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

	// Check if required ports are available before starting services
	if err := checkRequiredPorts(cfg); err != nil {
		fmt.Printf("\nError: %v\n\n", err)
		os.Exit(1)
	}

	// Initialize components based on whether we're following a network or running locally
	var emu *server.EmulatorServer
	var dw *devWallet.Server
	var gateway *flow.Gateway
	var gatewayCfg gatewayConfig.Config
	var emulatorReady chan struct{}
	var frontendManager *frontend.FrontendManager

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

	// Create views externally for better composability
	dashboardView := ui.NewDashboardViewWithConfig(cfg, debugLogger, &a)
	txView := ui.NewTransactionsViewWithConfig(cfg, debugLogger)
	eventsView := ui.NewEventsViewWithConfig(cfg, debugLogger)
	runnerView := ui.NewRunnerViewWithConfig(cfg, debugLogger)
	logsView := ui.NewLogsViewWithConfig(cfg, debugLogger)

	// Create model with pre-created views using new tabbedtui package
	tabs := []tabbedtui.TabbedModelPage{dashboardView, txView, eventsView, runnerView, logsView}
	model := tabbedtui.NewModel(tabs,
		tabbedtui.WithStyles(ui.GetTabbedStyles()),
	)

	// Now create the Bubble Tea program with config
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(), // Use alternate screen buffer
	)

	// Start EVM gateway after emulator is ready (only in local mode)
	if cfg.Network == "emulator" {
		go func() {
			<-emulatorReady // Wait for emulator to be ready
			gatewayLogger.Info().Msg("Starting EVM gateway...")
			gateway.Start(gatewayCfg)
		}()
	}

	// Start frontend process if configured (after emulator is ready)
	if cfg.FrontendCommand != "" {
		go func() {
			<-emulatorReady // Wait for emulator to be ready
			frontendManager = frontend.NewFrontendManager(cfg.FrontendCommand, logger)
			if err := frontendManager.Start(p); err != nil {
				logger.Error().Err(err).Msg("Failed to start frontend process")
			}
		}()
	}

	// Start aether server after emulator is ready with tea program
	go func() {
		<-emulatorReady
		aetherLogger.Info().Msg("Starting aether server")
		if err := a.Start(p); err != nil {
			aetherLogger.Error().Err(err).Msg("Failed to start aether server")
		}
	}()

	// Attach the Tea program to the log writer
	// This will drain any buffered logs and start sending new logs to the UI
	logWriter.AttachProgram(p)

	// Start the Bubble Tea program
	if _, err := p.Run(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run Bubble Tea program")
	}

	aetherLogger.Info().Msg("Shutting down...")

	// Cleanup (deferred functions will run here)
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
	if cfg.FrontendCommand != "" {
		if err := frontendManager.Stop(); err != nil {
			aetherLogger.Error().Err(err).Msg("Failed to stop frontend process")
		} else {
			aetherLogger.Info().Msg("Frontend process stopped")
		}
	}
	aetherLogger.Info().Msg("Stopping aether server...")
	a.Stop()
	aetherLogger.Info().Msg("All services stopped cleanly")
}
