package flow

import (
	"context"
	"strings"
	"time"

	"github.com/bjartek/overflow/v2"
	"github.com/bjartek/underflow"
	"github.com/cockroachdb/errors"
	"github.com/onflow/flow-go-sdk"
	"github.com/rs/zerolog"
)

type BlockResult struct {
	Block        flow.Block
	Transactions []overflow.OverflowTransaction
	Error        error
	Logger       zerolog.Logger
	View         uint64
	StartTime    time.Time
}

func StreamTransactions(ctx context.Context, o *overflow.OverflowState, poll time.Duration, height uint64, logger *zerolog.Logger, channel chan<- BlockResult) error {
	logger.Info().Msg("StreamTransactions started")
	latestKnownBlock, err := o.GetLatestBlock(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get latest block")
		return err
	}
	logger.Info().Uint64("height", latestKnownBlock.Height).Uint64("heightToStartAt", height).Dur("poll", poll).Msg("Starting to stream from latest block")

	sleep := poll
	for {
		select {
		case <-time.After(sleep):

			start := time.Now()
			sleep = poll
			nextBlockToProcess := height + 1
			if height == uint64(0) {
				nextBlockToProcess = latestKnownBlock.Height
				height = latestKnownBlock.Height
			}
			logg := logger.With().Uint64("height", nextBlockToProcess).Uint64("latestKnownBlock", latestKnownBlock.Height).Logger()
			logg.Debug().Msg("tick")

			var block *flow.Block
			if nextBlockToProcess < latestKnownBlock.Height {
				logg.Debug().Msg("next block is smaller then latest known block")
				// we are still processing historical blocks
				block, err = o.GetBlockAtHeight(ctx, nextBlockToProcess)
				if err != nil {
					// things can be wrapped
					if strings.Contains(err.Error(), "context canceled") {
						return nil
					}
					logg.Info().Err(err).Str("raw error", err.Error()).Msg("error fetching old block")
					continue
				}
			} else if nextBlockToProcess != latestKnownBlock.Height {
				logg.Debug().Msg("next block is not equal to latest block")
				block, err = o.GetLatestBlock(ctx)
				if err != nil {
					logg.Info().Err(err).Msg("error fetching latest block, retrying")
					continue
				}

				if block == nil || block.Height == latestKnownBlock.Height {
					continue
				}
				latestKnownBlock = block
				// we just continue the next iteration in the loop here
				sleep = time.Millisecond
				// the reason we just cannot process here is that the latestblock might not be the next block we should process
				continue
			} else {
				logg.Debug().Msg("we are on the block")
				block = latestKnownBlock
			}
			readDur := time.Since(start)
			logg.Debug().Uint64("block", block.Height).Uint64("latestBlock", latestKnownBlock.Height).Float64("readDur", readDur.Seconds()).Msg("block read")

			logg.Debug().Uint64("height", block.Height).Str("blockID", block.ID.String()).Msg("Fetching transactions for block...")
			transactions, err := GetOverflowTransactionsForBlockID(ctx, o, block.ID, logg)
			if err != nil {
				if strings.Contains(err.Error(), "context canceled") {
					return nil
				}
				if strings.Contains(err.Error(), "could not retrieve collection: key not found") {
					continue
				}

				logg.Debug().Err(err).Msg("getting transaction")

				select {
				case channel <- BlockResult{Block: *block, Error: errors.Wrap(err, "getting transactions"), Logger: logg, View: 0, StartTime: start}:
					height = nextBlockToProcess
				case <-ctx.Done():
					close(channel)
					return ctx.Err()
				}
				continue
			}
			logg = logg.With().Int("tx", len(transactions)).Logger()
			logg.Debug().Msg("fetched transactions")

			blockResult := BlockResult{
				Block:        *block,
				Transactions: transactions,
				Logger:       logg,
				View:         0,
				StartTime:    start,
			}

			logg.Debug().Uint64("height", block.Height).Int("txCount", len(transactions)).Msg("Sending block to channel")

			select {
			case channel <- blockResult:
				logg.Debug().Uint64("height", block.Height).Msg("Block sent to channel successfully")
				height = nextBlockToProcess
			case <-ctx.Done():
				close(channel)
				return ctx.Err()
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func GetOverflowTransactionsForBlockID(ctx context.Context, o overflow.OverflowClient, id flow.Identifier, logg zerolog.Logger) ([]overflow.OverflowTransaction, error) {
	transactions := []overflow.OverflowTransaction{}

	logg.Debug().Str("blockId", id.String()).Msg("Fetching transactions for block")
	tx, txR, err := o.GetTransactionsByBlockId(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting transaction results")
	}

	logg.Info().Str("blockId", id.String()).Int("tx", len(tx)).Int("txR", len(txR)).Msg("Fetched tx")
	for i, rp := range txR {

		t := *tx[i]
		r := *rp
		if r.CollectionID == flow.EmptyID {
			logg.Info().Msg("System transaction")
			if len(r.Events) > 0 {
				for _, e := range r.Events {
					cjson, err := underflow.CadenceValueToJsonString(e.Value)
					if err != nil {
						logg.Info().Msg(err.Error())
					}
					logg.Info().Str("TransactionID", e.TransactionID.Hex()).Str("event", cjson).Msg(e.Type)
				}
			}
			continue

		}
		logg.Debug().Str("collection", r.CollectionID.Hex()).Int("txIndex", i).Msg("Processing transaction")

		txLogger := logg.With().Str("txid", r.TransactionID.Hex()).Logger()
		ot, err := o.CreateOverflowTransaction(id.String(), r, t, i)
		if err != nil {
			txLogger.Error().Err(err).Msg("Failed to create overflow transaction - skipping this transaction")
			// Don't panic - just skip this transaction and continue processing
			continue
		}

		if ot.ProposalKey.Address.String() == "0x0000000000000000" && len(ot.Events) == 0 {
			txLogger.Info().Msg("skipping empty schedule tx process")
			continue
		}
		transactions = append(transactions, *ot)
	}

	return transactions, nil
}
