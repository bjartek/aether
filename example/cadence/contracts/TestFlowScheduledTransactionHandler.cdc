import "FlowTransactionScheduler"
import "FlowToken"
import "FungibleToken"
import "MetadataViews"

access(all) contract TestFlowScheduledTransactionHandler {

    access(all) let HandlerStoragePath: StoragePath
    access(all) let HandlerPublicPath: PublicPath

    access(all) resource Handler: FlowTransactionScheduler.TransactionHandler {

        access(all) view fun getViews(): [Type] {
            return [Type<MetadataViews.Display>()]
        }

        access(all) fun resolveView(_ view: Type): AnyStruct? {
            switch view {
            case Type<MetadataViews.Display>():
                return MetadataViews.Display(
                    name: "A test handler",
                    description: "a Test handler desciprtion", 
                    thumbnail: MetadataViews.HTTPFile(
                        url: ""
                    )
                )
            default:
                return nil
            }
        }

        /// executeTransaction executes the transaction logic
        /// This executeTransaction function only exists to test the transaction scheduler
        /// and doesn't do anything meaningful besides having a few different cases for testing
        /// The regular success case simply appends the transaction ID to the succeededTransactions array
        access(FlowTransactionScheduler.Execute) 
        fun executeTransaction(id: UInt64, data: AnyStruct?) {
            panic("foo")
        }
    }

    access(all) fun createHandler(): @Handler {
        return <- create Handler()
    }

    access(all) init() {

        self.HandlerStoragePath = /storage/testTransactionHandler
        self.HandlerPublicPath = /public/testTransactionHandler
    }

}
