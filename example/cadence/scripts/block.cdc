access(all) fun main(): {String: AnyStruct} {
    let currentTime = getCurrentBlock().timestamp
    let currentBlock = getCurrentBlock().height

    return {
        "currentTimestamp": currentTime,
        "currentBlockHeight": currentBlock,
        "note": "Forte scheduled transactions are managed by the blockchain. Check emulator logs for execution."
    }
}
