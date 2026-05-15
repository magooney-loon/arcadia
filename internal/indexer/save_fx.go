package indexer

import (
	"fmt"
	"math/big"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/chain"
	"arcadia/internal/utils"
)

func fxUpsert(app core.App, tradeID string, seen *batchSeen, update func(*core.Record)) error {
	r, ok := seen.fx[tradeID]
	if !ok {
		existing, err := app.FindRecordsByFilter("fx_swaps", "trade_id = {:id}", "", 1, 0, map[string]any{"id": tradeID})
		if err != nil {
			return fmt.Errorf("find fx trade %s: %w", tradeID, err)
		}
		if len(existing) > 0 {
			r = existing[0]
		} else {
			fxColl, ferr := utils.FindCollection(app, "fx_swaps")
			if ferr != nil {
				return ferr
			}
			r = core.NewRecord(fxColl)
			r.Set("trade_id", tradeID)
			r.Set("status", "created")
		}
		seen.fx[tradeID] = r
	}
	update(r)
	if err := app.Save(r); err != nil {
		return fmt.Errorf("save fx trade %s: %w", tradeID, err)
	}
	return nil
}

func saveFxEvent(app core.App, log *types.Log, seen *batchSeen) error {
	if log.Topic0 == nil || log.Topic1 == nil {
		return nil
	}

	tradeID := new(big.Int).SetBytes(log.Topic1.Bytes()).String()

	switch *log.Topic0 {
	case chain.TopicTradeRecorded:
		if log.Topic2 == nil {
			return nil
		}
		quoteID := log.Topic2.Hex()
		return fxUpsert(app, tradeID, seen, func(r *core.Record) {
			r.Set("quote_id", quoteID)
			r.Set("status", "created")
			if log.BlockNumber != nil {
				r.Set("block_number", log.BlockNumber.Uint64())
			}
			if log.TransactionHash != nil {
				r.Set("tx_hash", log.TransactionHash.Hex())
			}
		})

	case chain.TopicMakerFunded:
		if log.Topic2 == nil {
			return nil
		}
		maker := utils.AddressFromTopic(log.Topic2)
		return fxUpsert(app, tradeID, seen, func(r *core.Record) {
			r.Set("maker", maker)
			if r.GetString("status") == "taker_funded" {
				r.Set("status", "maker_funded")
			}
		})

	case chain.TopicTakerFunded:
		if log.Topic2 == nil {
			return nil
		}
		taker := utils.AddressFromTopic(log.Topic2)
		return fxUpsert(app, tradeID, seen, func(r *core.Record) {
			r.Set("taker", taker)
			r.Set("status", "taker_funded")
		})

	case chain.TopicTradeStatusChanged:
		if log.Data == nil || len(*log.Data) < 32 {
			return nil
		}
		statusCode := int(new(big.Int).SetBytes((*log.Data)[:32]).Int64())
		statusStr := "settled"
		if statusCode == 3 {
			statusStr = "cancelled"
		}
		return fxUpsert(app, tradeID, seen, func(r *core.Record) {
			r.Set("status_code", statusCode)
			r.Set("status", statusStr)
		})

	case chain.TopicFeesProcessed:
		if log.Data == nil || len(*log.Data) < 64 {
			return nil
		}
		takerFee := new(big.Int).SetBytes((*log.Data)[:32]).String()
		makerFee := new(big.Int).SetBytes((*log.Data)[32:64]).String()
		return fxUpsert(app, tradeID, seen, func(r *core.Record) {
			r.Set("taker_fee", takerFee)
			r.Set("maker_fee", makerFee)
		})
	}

	return nil
}
