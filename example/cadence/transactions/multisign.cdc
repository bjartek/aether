transaction(message:String) {
    prepare(user: &Account, user2: &Account) {
        log(message.concat(" from ").concat(user.address.toString()))
        log(message.concat(" from ").concat(user2.address.toString()))
    }
}
