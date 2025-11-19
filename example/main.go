package main

import (
	"context"
	"fmt"

	"github.com/bjartek/overflow/v2"
)

func main() {
	o := overflow.Overflow(overflow.WithBasePath("cadence"))
	// we start the emulator and everything
	o.Tx("message",
		overflow.WithSigner("bob"),
		overflow.WithArg("message", "Oh yeah"),
	).Print()

	ctx := context.Background()

	/*
		arg := os.Args[1]


		// Convert to uint64
		v, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			fmt.Println("invalid uint64:", err)
			return
		}

	*/
	b, _ := o.GetLatestBlock(ctx)

	t, tr, _ := o.GetTransactionsByBlockId(ctx, b.ID)

	for i, rx := range t {

		tx := *rx
		result := *tr[i]

		fmt.Println("ID", tx.ID().String())
		fmt.Println("Collection ", result.CollectionID.String())
		fmt.Println("Body", string(tx.Script))
		fmt.Println("EVENTS")
		for _, ev := range result.Events {
			fmt.Println("  ", ev.Type)
		}

		fmt.Println()
		fmt.Println()
	}
}
