package main

import (
	"context"
	"fmt"

	"github.com/bjartek/overflow/v2"
)

func main() {
	ctx := context.Background()
	// this has a normal flow transaction
	// testnetBlockId := "dcda8ae35c30d2d4535f2150a9f792340dfc3fcbfcb23e1601b4202cd26b84a8"
	// o := overflow.Overflow(overflow.WithNetwork("testnet"))
	// block, err := o.GetBlockById(ctx, testnetBlockId)
	o := overflow.Overflow(overflow.WithExistingEmulator())
	block, err := o.GetBlockAtHeight(ctx, 98)
	//	block, err := o.GetLatestBlock(ctx)
	if err != nil {
		panic(err)
	}

	tx, txR, err := o.Flowkit.GetTransactionsByBlockID(ctx, block.ID)
	if err != nil {
		panic(err)
	}

	for i, rawTx := range tx {

		t := rawTx
		tr := txR[i]

		fmt.Println("\n=== Transaction", i, "===")
		fmt.Println("ID:", t.ID().String())
		fmt.Println("BlockID:", tr.BlockID.String())
		fmt.Println("BlockHeight:", tr.BlockHeight)
		fmt.Println("CollectionID:", tr.CollectionID.String())
		fmt.Println("Payer:", t.Payer.String())
		fmt.Println("Proposer:", t.ProposalKey.Address.String)
		fmt.Println("Script:")
		fmt.Println(string(t.Script))
		fmt.Println()

		fmt.Println("Events count:", len(tr.Events))
		for j, event := range tr.Events {
			fmt.Printf("  Event[%d]: %s (txIndex=%d, eventIndex=%d)\n", j, event.Type, event.TransactionIndex, event.EventIndex)
		}

	}
}
