import "Counter"

transaction {
    prepare(alice: &Account) {
        log("Alice says foo")
    }
}
