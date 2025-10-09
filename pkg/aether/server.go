package aether

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/overflow/v2"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/enescakir/emoji"
	"github.com/rs/zerolog"
)

type Aether struct {
	Logger          *zerolog.Logger
	FclCdc          []byte
	Overflow        *overflow.OverflowState
	AccountRegistry *AccountRegistry
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

	o := overflow.Overflow(
		overflow.WithExistingEmulator(),
		overflow.WithLogNone(),
		overflow.WithTransactionFolderName("aether"),
		overflow.WithPanicOnError(),
		overflow.WithBasePath(basePath))

	_, err := o.CreateAccountsE(ctx)
	if err != nil {
		return err
	}
	a.Overflow = o

	// Initialize account registry after accounts are created
	a.AccountRegistry = NewAccountRegistry(o)
	dump := a.AccountRegistry.DebugDump()
	a.Logger.Info().
		Int("accounts", len(a.AccountRegistry.addressToName)).
		Interface("registry", dump).
		Msg("Initialized account registry")

	// Send overflow ready message to UI
	if teaProgram != nil {
		teaProgram.Send(OverflowReadyMsg{
			Overflow:        o,
			AccountRegistry: a.AccountRegistry,
		})
	}

	overflowChannel := make(chan flow.BlockResult)

	go func() {
		a.Logger.Info().Msg("Started streaming")

		err := flow.StreamTransactions(ctx, o, 200*time.Millisecond, 1, a.Logger, overflowChannel)
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

	a.Logger.Info().Msgf("%v Created accounts for emulator users in flow.json", emoji.Person)
	o.InitializeContracts(ctx)

	a.Logger.Info().Msgf("%v  Deployed contracts specified in emulator deployment block", emoji.Envelope)
	err = flow.AddFclContract(o, a.FclCdc)
	if err != nil {
		return err
	}

	accounts := o.GetEmulatorAccounts()

	err = flow.AddFclAccounts(o, accounts)
	if err != nil {
		return err
	}
	a.Logger.Info().Dict("accounts", zerolog.Dict().Fields(accounts)).Msgf("%v Added accounts to FCL", emoji.Person)

	err = flow.RunInitTransactions(o, validPath, a.Logger)
	if err != nil {
		return err
	}
	return nil
}

func (a *Aether) Stop() {
}
