package flow

import (
	"errors"
	"fmt"
	"strings"

	aetherConfig "github.com/bjartek/aether/pkg/config"
	"github.com/onflow/cadence"
	devWallet "github.com/onflow/fcl-dev-wallet/go/wallet"
	"github.com/onflow/flow-go/fvm"
	flowgo "github.com/onflow/flow-go/model/flow"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-emulator/server"
	"github.com/onflow/flowkit/v2"
	"github.com/onflow/flowkit/v2/config"
	"github.com/spf13/afero"
)

func InitEmulator(logger *zerolog.Logger, cfg *aetherConfig.Config) (*server.EmulatorServer, *devWallet.Server, error) {
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

	serverConf := &server.Config{
		GRPCPort:                     cfg.Ports.Emulator.GRPC,
		GRPCDebug:                    false,
		AdminPort:                    cfg.Ports.Emulator.Admin,
		DebuggerPort:                 cfg.Ports.Emulator.Debugger,
		RESTPort:                     cfg.Ports.Emulator.REST,
		RESTDebug:                    false,
		HTTPHeaders:                  nil,
		BlockTime:                    cfg.Flow.BlockTime,
		ServicePublicKey:             pk.PublicKey(),
		ServicePrivateKey:            pk,
		ServiceKeySigAlgo:            serviceAccount.Key.SigAlgo(),
		ServiceKeyHashAlgo:           serviceAccount.Key.HashAlgo(),
		Persist:                      false,
		Snapshot:                     false,
		DBPath:                       "./flowdb",
		GenesisTokenSupply:           cadence.UFix64(1000000000.0),
		TransactionMaxGasLimit:       9999,
		ScriptGasLimit:               100000,
		TransactionExpiry:            10,
		StorageLimitEnabled:          true,
		StorageMBPerFLOW:             fvm.DefaultStorageMBPerFLOW,
		MinimumStorageReservation:    fvm.DefaultMinimumStorageReservation,
		TransactionFeesEnabled:       true,
		WithContracts:                true,
		SkipTransactionValidation:    false,
		SimpleAddressesEnabled:       false,
		Host:                         "",
		ChainID:                      flowgo.Emulator,
		RedisURL:                     "",
		ContractRemovalEnabled:       true,
		SqliteURL:                    "",
		CoverageReportingEnabled:     false,
		StartBlockHeight:             0,
		RPCHost:                      "",
		CheckpointPath:               "",
		StateHash:                    "",
		ComputationReportingEnabled:  true,
		SetupEVMEnabled:              true,
		SetupVMBridgeEnabled:         true,
		ScheduledTransactionsEnabled: true,
	}

	emu := server.NewEmulatorServer(logger, serverConf)

	devWalletConfig := &devWallet.FlowConfig{
		Address:    fmt.Sprintf("0x%s", serviceAccount.Address.String()),
		PrivateKey: strings.TrimPrefix(pk.String(), "0x"),
		PublicKey:  strings.TrimPrefix(pk.PublicKey().String(), "0x"),
		AccessNode: fmt.Sprintf("http://localhost:%d", cfg.Ports.Emulator.REST),
	}

	dw, err := devWallet.NewHTTPServer(uint(cfg.Ports.DevWallet), devWalletConfig)
	if err != nil {
		return nil, nil, err
	}

	return emu, dw, nil
}
