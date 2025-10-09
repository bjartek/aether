package flow

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bjartek/overflow/v2"
	"github.com/enescakir/emoji"
	"github.com/rs/zerolog"
)

// RunInitTransactions runs initialization transactions from both .cdc and .json files
// cdcOverflow is used for .cdc files, jsonOverflow is used for .json config files
func RunInitTransactions(cdcOverflow *overflow.OverflowState, jsonOverflow *overflow.OverflowState, validPath string, logger *zerolog.Logger) error {
	err := filepath.Walk(validPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(info.Name())
		
		// Handle .json configuration files
		if ext == ".json" {
			config, err := LoadTransactionConfig(path)
			if err != nil {
				logger.Error().Err(err).Str("file", info.Name()).Msg("Failed to load transaction config")
				return err
			}

			// Build overflow options from config
			var opts []overflow.OverflowInteractionOption
			
			// Add signers - first signer uses WithSigner, rest use WithPayloadSigner
			for i, signer := range config.Signers {
				if i == 0 {
					opts = append(opts, overflow.WithSigner(signer))
				} else {
					opts = append(opts, overflow.WithPayloadSigner(signer))
				}
			}
			
			// Add arguments
			for argName, argValue := range config.Arguments {
				opts = append(opts, overflow.WithArg(argName, argValue))
			}

			// Execute transaction using jsonOverflow state
			res := jsonOverflow.Tx(config.Name, opts...)
			if res.Err != nil {
				logger.Error().Err(res.Err).Str("config", info.Name()).Str("transaction", config.Name).Msg("Failed to run transaction from config")
				return res.Err
			}
			
			logger.Info().Str("config", info.Name()).Str("transaction", config.Name).Msgf("%v Ran init transaction from config", emoji.Scroll)
			return nil
		}
		
		// Handle .cdc files (original behavior)
		if ext == ".cdc" {
			fileName := strings.TrimSuffix(info.Name(), ".cdc")
			res := cdcOverflow.Tx(fileName, overflow.WithAutoSigner())
			if res.Err != nil {
				return res.Err
			}
			logger.Info().Str("file", fileName).Msgf("%v Ran init transaction", emoji.Scroll)
			return nil
		}

		return nil
	})
	return err
}

func AddFclContract(o *overflow.OverflowState, contract []byte) error {
	res := o.Tx(`
transaction(code: [UInt8]) {
  prepare(acct: auth(Contracts) &Account) {
    let labels: [String]=["Service Account"]
    let key = acct.keys.get(keyIndex:0)!
    //this is a mess!
    let hash: UInt8=3
    let sign: UInt8=1
    let pubk= String.encodeHex(key.publicKey.publicKey)
    acct.contracts.add(name: "FCL", code: code, publicKey: pubk, hashAlgorithm: hash, signAlgorithm: sign, initAccountsLabels: labels)
  }
}`,
		overflow.WithSignerServiceAccount(),
		overflow.WithArg("code", contract),
	)

	return res.Err
}

func AddFclAccounts(o *overflow.OverflowState, accounts map[string]string) error {
	res := o.Tx(`
    import FCL from  0xf8d6e0586b0a20c7
    transaction(accounts: {String:Address}) {
  prepare(acct: auth(Storage) &Account) {

    let root = acct.storage.borrow<&FCL.Root>(from: FCL.storagePath)!
    let scopes:[String]=[]

    for name in accounts.keys {
      let address=accounts[name]!
      root.add(FCL.FCLAccount(address: address, label: name, scopes: scopes))

    }
  }
}`,
		overflow.WithSignerServiceAccount(),
		overflow.WithArg("accounts", accounts),
	)

	return res.Err
}
