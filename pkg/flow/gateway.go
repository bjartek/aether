package flow

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"

	gethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/onflow/flow-evm-gateway/bootstrap"
	gatewayConfig "github.com/onflow/flow-evm-gateway/config"
	flowsdk "github.com/onflow/flow-go-sdk"
	flowCrypto "github.com/onflow/flow-go-sdk/crypto"
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
func InitGateway(logger zerolog.Logger) (*Gateway, gatewayConfig.Config, error) {
	loader := &afero.Afero{Fs: afero.NewOsFs()}
	state, err := flowkit.Load(config.DefaultPaths(), loader)
	if err != nil {
		return nil, gatewayConfig.Config{}, err
	}

	serviceAccount, err := state.EmulatorServiceAccount()
	if err != nil {
		return nil, gatewayConfig.Config{}, err
	}

	// Use the standard EVM gateway private key (same as flow-cli)
	// This key is used for both COA operations and wallet API
	gatewayKeyHex := "2619878f0e2ff438d17835c2a4561cb87b4d24d72d12ec34569acd0dd4af7c21"

	// Parse as ECDSA private key for wallet
	evmPrivateKey, err := gethCrypto.HexToECDSA(gatewayKeyHex)
	if err != nil {
		return nil, gatewayConfig.Config{}, fmt.Errorf("failed to parse EVM private key: %w", err)
	}

	// Parse as Flow private key for COA
	flowPrivateKeyBytes, err := gethCrypto.HexToECDSA(gatewayKeyHex)
	if err != nil {
		return nil, gatewayConfig.Config{}, fmt.Errorf("failed to parse Flow private key: %w", err)
	}

	// Convert ECDSA key to Flow crypto format
	flowPrivateKey, err := flowCrypto.DecodePrivateKeyHex(flowCrypto.ECDSA_P256, gatewayKeyHex)
	if err != nil {
		return nil, gatewayConfig.Config{}, fmt.Errorf("failed to decode Flow private key: %w", err)
	}

	// Derive the EVM address from the private key
	evmAddress := gethCrypto.PubkeyToAddress(flowPrivateKeyBytes.PublicKey)

	// Default gateway configuration matching flow-cli defaults
	dbPath := "./evm-gateway-db"

	// Create logger for gateway operations
	logger.Info().Str("evmAddress", evmAddress.Hex()).Msg("Using EVM gateway key")

	// Clean up old database to ensure fresh start
	if _, err := os.Stat(dbPath); err == nil {
		logger.Info().Str("path", dbPath).Msg("Removing old EVM gateway database")
		if err := os.RemoveAll(dbPath); err != nil {
			logger.Warn().Err(err).Str("path", dbPath).Msg("Failed to remove old database directory")
		} else {
			logger.Info().Str("path", dbPath).Msg("Old database removed successfully")
		}
	}

	cfg := gatewayConfig.Config{
		DatabaseDir:       dbPath,
		AccessNodeHost:    "localhost:3569", // emulator gRPC port
		RPCPort:           8545,
		RPCHost:           "localhost",
		InitCadenceHeight: 1,
		FlowNetworkID:     flowGo.Emulator,
		EVMNetworkID:      types.FlowEVMPreviewNetChainID, // Chain ID 646
		Coinbase:          evmAddress,                     // Use derived address from private key
		WalletEnabled:     true,
		WalletKey:         evmPrivateKey, // ECDSA private key for wallet API
		GasPrice:          big.NewInt(1),
		COAAddress:        flowsdk.Address(serviceAccount.Address),
		COAKey:            flowPrivateKey, // Flow private key for COA operations
		Logger:            &logger,
		TxStateValidation: "local-index",
		ProfilerEnabled:   true,
		ProfilerPort:      6060,
	}

	ctx, cancel := context.WithCancel(context.Background())

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

		// CRITICAL: Recover from panics FIRST before any other defer
		// This must be the outermost defer to catch everything
		defer func() {
			if r := recover(); r != nil {
				g.logger.Error().
					Interface("panic", r).
					Msg("RECOVERED: EVM gateway panicked - continuing aether execution")
			}
		}()

		defer func() {
			// Ensure ready is closed even on error
			select {
			case <-g.ready:
			default:
				close(g.ready)
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
				Msg("EVM gateway stopped with error - aether continues running")
		} else if errors.Is(err, context.Canceled) {
			g.logger.Info().Msg("EVM gateway stopped (context canceled)")
		} else if err == nil {
			g.logger.Info().Msg("EVM gateway stopped successfully")
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
