transaction {
    prepare(bob: auth(BorrowValue, SaveValue, IssueStorageCapabilityController, PublishCapability, GetStorageCapabilityController) &Account) {
        log("Bob says foo")
    }
}
