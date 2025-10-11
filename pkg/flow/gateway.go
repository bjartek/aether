package flow

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/onflow/flow-evm-gateway/bootstrap"
	gatewayConfig "github.com/onflow/flow-evm-gateway/config"
	flowsdk "github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go/fvm/evm/types"
	flowGo "github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flowkit/v2"
	"github.com/onflow/flowkit/v2/config"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"
)

type Gateway struct {
	ctx    context.Context
	cancel context.CancelFunc
	ready  chan struct{}
	done   chan struct{}
	dbPath string
	logger zerolog.Logger
}

// InitGateway initializes the EVM gateway with default configuration
func InitGateway(logWriter io.Writer, logLevel zerolog.Level) (*Gateway, gatewayConfig.Config, error) {
	loader := &afero.Afero{Fs: afero.NewOsFs()}
	state, err := flowkit.Load(config.DefaultPaths(), loader)
	if err != nil {
		return nil, gatewayConfig.Config{}, err
	}

	serviceAccount, err := state.EmulatorServiceAccount()
	if err != nil {
		return nil, gatewayConfig.Config{}, err
	}

	privateKey, err := serviceAccount.Key.PrivateKey()
	if err != nil {
		return nil, gatewayConfig.Config{}, err
	}

	pk := *privateKey

	// Default gateway configuration matching flow-cli defaults
	dbPath := "./evm-gateway-db"
	cfg := gatewayConfig.Config{
		DatabaseDir:       dbPath,
		AccessNodeHost:    "localhost:3569", // emulator gRPC port
		RPCPort:           3000,
		RPCHost:           "localhost",
		InitCadenceHeight: 0,
		FlowNetworkID:     flowGo.Emulator,
		EVMNetworkID:      types.FlowEVMPreviewNetChainID,                                        // Chain ID 646
		Coinbase:          gethCommon.HexToAddress("0x0000000000000000000000000000000000000000"), // use zero address as default
		GasPrice:          big.NewInt(1),
		COAAddress:        flowsdk.Address(serviceAccount.Address),
		COAKey:            pk,
		LogWriter:         logWriter,
		LogLevel:          logLevel,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create logger for gateway cleanup operations
	logger := zerolog.New(logWriter).With().Str("component", "evm-gateway").Timestamp().Logger().Level(logLevel)

	gateway := &Gateway{
		ctx:    ctx,
		cancel: cancel,
		ready:  make(chan struct{}),
		done:   make(chan struct{}),
		dbPath: cfg.DatabaseDir,
		logger: logger,
	}

	return gateway, cfg, nil
}

// Start starts the EVM gateway server
func (g *Gateway) Start(cfg gatewayConfig.Config) {
	go func() {
		defer close(g.done)
		defer func() {
			// Ensure ready is closed even on error
			select {
			case <-g.ready:
			default:
				close(g.ready)
			}
		}()

		// Recover from panics
		defer func() {
			if r := recover(); r != nil {
				g.logger.Error().
					Interface("panic", r).
					Msg("EVM gateway panicked during startup")
			}
		}()

		closeReady := func() {
			select {
			case <-g.ready:
			default:
				close(g.ready)
			}
		}

		g.logger.Info().Msg("Starting EVM gateway bootstrap...")
		err := bootstrap.Run(
			g.ctx,
			cfg,
			closeReady,
		)
		if err != nil && !errors.Is(err, context.Canceled) {
			g.logger.Error().
				Err(err).
				Str("database_dir", cfg.DatabaseDir).
				Str("rpc_host", fmt.Sprintf("%s:%d", cfg.RPCHost, cfg.RPCPort)).
				Str("access_node", cfg.AccessNodeHost).
				Msg("EVM gateway stopped with error")
		} else if errors.Is(err, context.Canceled) {
			g.logger.Info().Msg("EVM gateway stopped (context canceled)")
		}
	}()
}

// Ready returns a channel that will be closed when the gateway is ready
func (g *Gateway) Ready() <-chan struct{} {
	return g.ready
}

// Stop stops the gateway and cleans up the database
func (g *Gateway) Stop() {
	if g.cancel != nil {
		g.cancel()
	}
	// Wait for gateway to fully stop
	<-g.done

}
