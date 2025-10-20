package aether

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bjartek/aether/pkg/config"
	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/overflow/v2"
	"github.com/bjartek/underflow"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/enescakir/emoji"
	"github.com/rs/zerolog"
)

type Aether struct {
	Logger          *zerolog.Logger
	FclCdc          []byte
	Overflow        *overflow.OverflowState
	AccountRegistry *AccountRegistry
	Network         string // "testnet", "mainnet", or "emulator"
	Config          *config.Config
}

// BlockTransactionMsg is sent when a transaction is processed
type BlockTransactionMsg struct {
	BlockHeight     uint64
	BlockID         string
	Transaction     overflow.OverflowTransaction
	AccountRegistry *AccountRegistry
}

// OverflowReadyMsg is sent when overflow is initialized and ready
type OverflowReadyMsg struct {
	Overflow        *overflow.OverflowState
	AccountRegistry *AccountRegistry
}

func (a *Aether) Start(teaProgram *tea.Program) error {
	ctx := context.Background()

	// Define the two possible paths
	path1 := "aether"
	path2 := filepath.Join("cadence", "aether")

	// Check which path exists
	var validPath string
	basePath := ""
	if _, err := os.Stat(path1); err == nil {
		validPath = path1
	} else if _, err := os.Stat(path2); err == nil {
		validPath = path2
		basePath = "cadence"
	} else {
		return fmt.Errorf("neither %q nor %q exists", path1, path2)
	}

	// Configure underflow options for human-friendly output
	underflowOptions := underflow.Options{
		ByteArrayAsHex:             true,
		ShowUnixTimestampsAsString: true,
		TimestampFormat:            "2006-01-02 15:04:05 UTC",
	}

	// Initialize overflow based on network mode
	var o *overflow.OverflowState
	if a.Network == "emulator" {
		// Local emulator mode
		o = overflow.Overflow(
			overflow.WithExistingEmulator(),
			overflow.WithLogNone(),
			overflow.WithReturnErrors(),
			overflow.WithTransactionFolderName("aether"),
			overflow.WithBasePath(basePath),
			overflow.WithUnderflowOptions(underflowOptions),
			overflow.WithFlowForNewUsers(a.Config.Flow.NewUserBalance))
	} else {
		// Network mode (testnet or mainnet)
		a.Logger.Info().Str("network", a.Network).Msg("Initializing overflow for network")
		o = overflow.Overflow(
			overflow.WithNetwork(a.Network),
			overflow.WithLogNone(),
			overflow.WithReturnErrors(),
			overflow.WithTransactionFolderName("aether"),
			overflow.WithBasePath(basePath),
			overflow.WithUnderflowOptions(underflowOptions))
	}

	// Only create accounts in local mode
	if a.Network == "emulator" {
		a.Logger.Info().Str("network", o.Network.Name).Msg("emulator")
		_, err := o.CreateAccountsE(ctx)
		if err != nil {
			return err
		}
	}
	a.Overflow = o

	// Initialize account registry after accounts are created
	a.AccountRegistry = NewAccountRegistry(o)
	dump := a.AccountRegistry.DebugDump()
	a.Logger.Info().
		Int("accounts", len(a.AccountRegistry.addressToName)).
		Interface("registry", dump).
		Msg("Initialized account registry")

	// Create second overflow instance for runner view with same underflow options
	var oR *overflow.OverflowState
	if a.Network == "emulator" {
		oR = overflow.Overflow(
			overflow.WithExistingEmulator(),
			overflow.WithLogNone(),
			overflow.WithReturnErrors(),
			overflow.WithBasePath(basePath),
			overflow.WithUnderflowOptions(underflowOptions))
	} else {
		oR = overflow.Overflow(
			overflow.WithNetwork(a.Network),
			overflow.WithLogNone(),
			overflow.WithReturnErrors(),
			overflow.WithBasePath(basePath),
			overflow.WithUnderflowOptions(underflowOptions))
	}

	// Send overflow ready message to UI
	if teaProgram != nil {
		teaProgram.Send(OverflowReadyMsg{
			Overflow:        oR,
			AccountRegistry: a.AccountRegistry,
		})
	}

	overflowChannel := make(chan flow.BlockResult)

	// Determine starting block height and polling interval based on network mode
	var startHeight uint64
	pollInterval := a.Config.Indexer.PollingInterval

	if a.Network == "emulator" {
		// Local emulator mode - start from block 1
		startHeight = 1
	} else {
		// Network mode - start from latest block
		latestBlock, err := o.GetLatestBlock(ctx)
		if err != nil {
			a.Logger.Error().Err(err).Str("network", a.Network).Msg("Failed to get latest block")
			return err
		}
		startHeight = latestBlock.Height

		a.Logger.Info().
			Str("network", a.Network).
			Uint64("startHeight", startHeight).
			Dur("pollInterval", pollInterval).
			Msg("Starting to stream from latest block")
	}

	go func() {
		a.Logger.Info().Msg("Started streaming")

		err := flow.StreamTransactions(ctx, o, pollInterval, startHeight, a.Logger, overflowChannel)
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				a.Logger.Info().Msg("Streaming stopped due to context cancellation")
			} else {
				a.Logger.Warn().Err(err).Msg("Streaming encountered an error")
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				a.Logger.Info().Msg("Block receiver goroutine stopping due to context cancellation")
				return
			case br, ok := <-overflowChannel:
				if !ok {
					a.Logger.Info().Msg("Channel has been closed. Crawler stopped completely.")
					return
				}

				l := br.Logger

				if br.Error != nil {
					l.Warn().Err(br.Error).Msg("Failed fetching block")
					continue
				}

				// Send transactions directly to UI if there are any
				if len(br.Transactions) > 0 && teaProgram != nil {
					for _, tx := range br.Transactions {
						teaProgram.Send(BlockTransactionMsg{
							BlockHeight:     br.Block.Height,
							BlockID:         br.Block.ID.String(),
							Transaction:     tx,
							AccountRegistry: a.AccountRegistry,
						})
					}
				}

				// Log the block processing
				txCount := len(br.Transactions)

				if txCount > 0 {
					l.Info().
						Uint64("height", br.Block.Height).
						Int("txCount", txCount).
						Msg("Processed block")
				}
			}
		}
	}()

	// Only perform local setup in emulator mode
	if a.Network == "emulator" {
		a.Logger.Info().Msgf("%v Created accounts for emulator users in flow.json", emoji.Person)
		o.InitializeContracts(ctx)

		a.Logger.Info().Msgf("%v  Deployed contracts specified in emulator deployment block", emoji.Envelope)
		if err := flow.AddFclContract(o, a.FclCdc); err != nil {
			return err
		}

		accounts := o.GetEmulatorAccounts()
		a.Logger.Debug().Int("filtered_accounts", len(accounts)).Interface("accounts", accounts).Msg("Filtered emulator accounts")

		if len(accounts) > 0 {
			a.Logger.Info().Int("count", len(accounts)).Interface("accounts", accounts).Msgf("%v Adding accounts to FCL", emoji.Person)
			if err := flow.AddFclAccounts(o, accounts); err != nil {
				return err
			}
			a.Logger.Info().Msgf("%v Successfully added %d accounts to FCL", emoji.Person, len(accounts))
		} else {
			a.Logger.Warn().Msg("No accounts to add to FCL")
		}

		// Use same overflow state for both .cdc and .json files
		// Both point to same state since JSON configs reference the same transaction files
		if err := flow.RunInitTransactions(o, oR, validPath, a.Logger); err != nil {
			return err
		}
	} else {
		a.Logger.Info().Str("network", a.Network).Msg("Following network - skipping local setup steps")
	}
	return nil
}

func (a *Aether) Stop() {
}
