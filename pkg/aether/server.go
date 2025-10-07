package aether

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/overflow/v2"
	"github.com/enescakir/emoji"
	"github.com/rs/zerolog"
)

type Aether struct {
	Logger *zerolog.Logger
	FclCdc []byte
}

func (a *Aether) Start() error {
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

func (i *Aether) Stop() {
}
