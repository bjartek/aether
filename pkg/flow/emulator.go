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

	"github.com/onflow/flow-emulator/server"
	"github.com/onflow/flowkit/v2"
	"github.com/onflow/flowkit/v2/config"
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

	serverConf := &server.Config{
		GRPCPort:                    3569,
		GRPCDebug:                   false,
		AdminPort:                   8080,
		DebuggerPort:                2345,
		RESTPort:                    8888,
		RESTDebug:                   false,
		HTTPHeaders:                 nil,
		BlockTime:                   1 * time.Second,
		ServicePublicKey:            pk.PublicKey(),
		ServicePrivateKey:           pk,
		ServiceKeySigAlgo:           serviceAccount.Key.SigAlgo(),
		ServiceKeyHashAlgo:          serviceAccount.Key.HashAlgo(),
		Persist:                     false,
		Snapshot:                    false,
		DBPath:                      "./flowdb",
		GenesisTokenSupply:          cadence.UFix64(1000000000.0),
		TransactionMaxGasLimit:      9999,
		ScriptGasLimit:              100000,
		TransactionExpiry:           10,
		StorageLimitEnabled:         true,
		StorageMBPerFLOW:            fvm.DefaultStorageMBPerFLOW,
		MinimumStorageReservation:   fvm.DefaultMinimumStorageReservation,
		TransactionFeesEnabled:      true,
		WithContracts:               true,
		SkipTransactionValidation:   false,
		SimpleAddressesEnabled:      false,
		Host:                        "",
		ChainID:                     flowgo.Emulator,
		RedisURL:                    "",
		ContractRemovalEnabled:      true,
		SqliteURL:                   "",
		CoverageReportingEnabled:    false,
		StartBlockHeight:            0,
		RPCHost:                     "",
		CheckpointPath:              "",
		StateHash:                   "",
		ComputationReportingEnabled: false,
		SetupEVMEnabled:             true,
		SetupVMBridgeEnabled:        true,
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
