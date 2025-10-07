package flow

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/onflow/cadence"
	devWallet "github.com/onflow/fcl-dev-wallet/go/wallet"
	"github.com/onflow/flow-go/fvm"
	flowgo "github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-emulator/cmd/emulator/start"
	"github.com/onflow/flow-emulator/server"
	"github.com/onflow/flowkit/v2"
	"github.com/onflow/flowkit/v2/config"
	"github.com/psiemens/sconfig"
	"github.com/spf13/afero"
)

func InitEmulator(logger *zerolog.Logger) (*server.EmulatorServer, *devWallet.Server, error) {
	loader := &afero.Afero{Fs: afero.NewOsFs()}
	state, err := flowkit.Load(config.DefaultPaths(), loader)
	if err != nil {
		if errors.Is(err, config.ErrDoesNotExist) {
			return nil, nil, errors.New("🙏 Configuration (flow.json) is missing, are you in the correct directory? If you are trying to create a new project, initialize it with 'flow init' and then rerun this command")
		} else {
			return nil, nil, err
		}
	}

	serviceAccount, err := state.EmulatorServiceAccount()
	if err != nil {
		return nil, nil, err
	}

	privateKey, err := serviceAccount.Key.PrivateKey()
	if err != nil {
		return nil, nil, err
	}

	pk := *privateKey

	var conf start.Config

	// we use sconfig to bind with as they do in flow-emulator
	err = sconfig.New(&conf).
		FromEnvironment("AETHER").
		Parse()
	if err != nil {
		panic(err)
	}

	serverConf := &server.Config{
		GRPCPort:                    conf.Port,
		GRPCDebug:                   true,
		AdminPort:                   conf.AdminPort,
		DebuggerPort:                conf.DebuggerPort,
		RESTPort:                    conf.RestPort,
		RESTDebug:                   true,
		HTTPHeaders:                 nil,
		BlockTime:                   1 * time.Second,
		ServicePublicKey:            pk.PublicKey(),
		ServicePrivateKey:           pk,
		ServiceKeySigAlgo:           serviceAccount.Key.SigAlgo(),
		ServiceKeyHashAlgo:          serviceAccount.Key.HashAlgo(),
		Persist:                     conf.Persist,
		Snapshot:                    conf.Snapshot,
		DBPath:                      conf.DBPath,
		GenesisTokenSupply:          cadence.UFix64(1000000000.0),
		TransactionMaxGasLimit:      uint64(conf.TransactionMaxGasLimit),
		ScriptGasLimit:              uint64(conf.ScriptGasLimit),
		TransactionExpiry:           uint(conf.TransactionExpiry),
		StorageLimitEnabled:         conf.StorageLimitEnabled,
		StorageMBPerFLOW:            fvm.DefaultStorageMBPerFLOW,
		MinimumStorageReservation:   fvm.DefaultMinimumStorageReservation,
		TransactionFeesEnabled:      true,
		WithContracts:               conf.Contracts,
		SkipTransactionValidation:   conf.SkipTxValidation,
		SimpleAddressesEnabled:      conf.SimpleAddresses,
		Host:                        conf.Host,
		ChainID:                     flowgo.Emulator,
		RedisURL:                    conf.RedisURL,
		ContractRemovalEnabled:      conf.ContractRemovalEnabled,
		SqliteURL:                   conf.SqliteURL,
		CoverageReportingEnabled:    conf.CoverageReportingEnabled,
		StartBlockHeight:            conf.StartBlockHeight,
		RPCHost:                     conf.RPCHost,
		CheckpointPath:              conf.CheckpointPath,
		StateHash:                   conf.StateHash,
		ComputationReportingEnabled: conf.ComputationReportingEnabled,
		//		ScheduledTransactionsEnabled: true, //not sure why this deploys it 2 times
	}

	emu := server.NewEmulatorServer(logger, serverConf)

	devWalletConfig := &devWallet.FlowConfig{
		Address:    fmt.Sprintf("0x%s", serviceAccount.Address.String()),
		PrivateKey: strings.TrimPrefix(pk.String(), "0x"),
		PublicKey:  strings.TrimPrefix(pk.PublicKey().String(), "0x"),
		AccessNode: "http://localhost:8888",
	}

	dw, err := devWallet.NewHTTPServer(8701, devWalletConfig)
	if err != nil {
		return nil, nil, err
	}

	return emu, dw, nil
}
