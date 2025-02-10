import "Counter"

transaction {
    prepare(bob: &Account) {
        log("Bob says foo")
    }
}
