import "Counter"

transaction {
    prepare(alice: &Account) {
        let block = getCurrentBlock()
        log("Alice says foo 2 at height=".concat(block.height.toString()))
    }
}
