import "EVM"
import "FungibleToken"
import "FlowToken"

transaction(addr:String) {

    prepare(signer: auth(Storage) &Account) {
        let vaultRef = signer.storage.borrow<auth(FungibleToken.Withdraw) &FlowToken.Vault>(
            from: /storage/flowTokenVault
        ) ?? panic("Could not borrow reference to the owner's Vault!")

        let eoaAddressBytes: [UInt8; 20] = addr.decodeHex().toConstantSized<[UInt8; 20]>()!
        let eoaAddress: EVM.EVMAddress = EVM.EVMAddress(bytes: eoaAddressBytes)
        let fundVault <- vaultRef.withdraw(amount: 1.0) as! @FlowToken.Vault

        eoaAddress.deposit(from: <- fundVault)
    }

}
