package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bjartek/aether/pkg/flow"
	"github.com/bjartek/overflow/v2"
	"github.com/psiemens/graceland"
	"github.com/rs/zerolog"
	"github.com/sanity-io/litter"
)

//go:embed cadence/FCL.cdc
var fclCdc []byte

// just so that it does not complain
//
//go:embed cadence/*
var _ embed.FS

func main() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(output).With().Timestamp().Logger()

	dw, emu, err := flow.InitEmulator(&logger)
	if err != nil {
		panic(err)
	}

	gl := graceland.NewGroup()

	dw.AddToGroup(gl)
	gl.Add(emu)
	gl.Add(&Init{})

	err = gl.Start()
	if err != nil {
		logger.Error().Err(err).Msg("❗  Server error")
	}
	gl.Stop()
}

type Init struct{}

func (i *Init) Start() error {
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
		overflow.WithLogFull(),
		overflow.WithTransactionFolderName("aether"),
		overflow.WithPanicOnError(),
		overflow.WithPrintResults(),
		overflow.WithBasePath(basePath))

	_, err := o.CreateAccountsE(ctx)
	if err != nil {
		return err
	}
	o.InitializeContracts(ctx)
	err = flow.AddFclContract(o, fclCdc)
	if err != nil {
		return err
	}

	accounts := o.GetEmulatorAccounts()
	litter.Dump(accounts)

	err = flow.AddFclAccounts(o, accounts)
	if err != nil {
		return err
	}

	err = flow.RunInitTransactions(o, validPath)
	if err != nil {
		return err
	}
	return nil
}

func (i *Init) Stop() {
}
