package flow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bjartek/overflow/v2"
)

// TODO: add logger
func RunInitTransactions(o *overflow.OverflowState, validPath string) error {
	err := filepath.Walk(validPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Process the file
		fmt.Printf("Processing file: %sn", info.Name())

		res := o.Tx(strings.TrimSuffix(info.Name(), ".cdc"), overflow.WithAutoSigner())
		if res.Err != nil {
			return res.Err
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
