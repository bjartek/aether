transaction(message:String) {
    prepare(user: &Account) {
        log(message)
    }
}
