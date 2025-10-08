import "Counter"

transaction {
    prepare(alice: &Account) {
        let block = getCurrentBlock()
        log("Alice says foo at height=".concat(block.height.toString()))
    }
}
