import "EVM"

// Note that this is a simplified example & will not handle cases where the COA already exists
transaction() {
    prepare(signer: auth(SaveValue, IssueStorageCapabilityController, PublishCapability) &Account) {
        let storagePath = /storage/evm
        let publicPath = /public/evm

        // Create account & save to storage
        let coa: @EVM.CadenceOwnedAccount <- EVM.createCadenceOwnedAccount()
        signer.storage.save(<-coa, to: storagePath)

        // Publish a public capability to the COA
        let cap = signer.capabilities.storage.issue<&EVM.CadenceOwnedAccount>(storagePath)
        signer.capabilities.publish(cap, at: publicPath)
    }
}
