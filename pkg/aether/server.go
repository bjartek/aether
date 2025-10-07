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
	"github.com/enescakir/emoji"
	"github.com/rs/zerolog"
)

type Aether struct {
	Logger   *zerolog.Logger
	FclCdc   []byte
	Overflow *overflow.OverflowState
	Store    *Store
}

func (a *Aether) Start() error {
	ctx := context.Background()
	
	// Initialize store if not provided
	if a.Store == nil {
		a.Store = NewStore()
	}
	
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
	overflowChannel := make(chan flow.BlockResult)

	go func() {
		err := flow.StreamTransactions(ctx, o, 1*time.Second, 1, a.Logger, overflowChannel)
		if err != nil {
			if strings.Contains(err.Error(), "context canceled") {
				a.Logger.Info().Msg("Streaming stopped due to context cancellation")
			} else {
				a.Logger.Warn().Msg("Streaming encountered an error")
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				a.Logger.Info().Msg("Context done, stopping overflow stream")
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

				// Store the block result
				a.Store.Add(br)

				// Log the block processing
				totalDur := time.Since(br.StartTime)
				txCount := len(br.Transactions)

				l.Info().
					Uint64("height", br.Block.Height).
					Int64("durationMs", totalDur.Milliseconds()).
					Int("txCount", txCount).
					Int("totalBlocks", a.Store.Count()).
					Msg("Processed and stored block")
			}
		}
	}()

	return nil
}

func (i *Aether) Stop() {
}
