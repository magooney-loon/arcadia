package indexer

import (
	"fmt"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

func saveCrosschain(app core.App, log *types.Log, fill func(*core.Record)) error {
	if log.TransactionHash == nil || log.LogIndex == nil {
		return nil
	}
	existing, err := app.FindRecordsByFilter("crosschain_events",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": log.TransactionHash.Hex(), "i": *log.LogIndex})
	if err != nil {
		return fmt.Errorf("find crosschain %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}
	if len(existing) > 0 {
		return nil
	}
	r := core.NewRecord(utils.MustCollection(app, "crosschain_events"))
	r.Set("tx_hash", log.TransactionHash.Hex())
	r.Set("log_index", *log.LogIndex)
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	fill(r)
	if err := app.Save(r); err != nil {
		return fmt.Errorf("save crosschain %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}
	return nil
}

func saveCCTPDepositForBurn(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "cctp")
		r.Set("event_type", "burn")
		r.Set("source_domain", 26)

		if log.Topic2 != nil {
			r.Set("sender", utils.AddressFromTopic(log.Topic2))
		}
		if log.Data != nil && len(*log.Data) >= 64 {
			d := *log.Data
			r.Set("amount_usdc", utils.StablecoinHuman(utils.ReadBig(d, 0)))
			r.Set("recipient", utils.AddressFromBytes32(d[32:64]))
			if len(d) >= 96 {
				r.Set("destination_domain", utils.ReadUint32(d, 64))
			}
		}
	})
}

func saveCCTPMintAndWithdraw(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "cctp")
		r.Set("event_type", "mint")
		r.Set("destination_domain", 26)

		if log.Topic1 != nil {
			r.Set("recipient", utils.AddressFromTopic(log.Topic1))
		}
		if log.Data != nil && len(*log.Data) >= 32 {
			r.Set("amount_usdc", utils.StablecoinHuman(utils.ReadBig(*log.Data, 0)))
		}
	})
}

func saveCCTPMessageReceived(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "cctp")
		r.Set("event_type", "mint")
		r.Set("destination_domain", 26)

		if log.Topic2 != nil {
			r.Set("nonce_val", log.Topic2.Hex())
		}
		if log.Data != nil && len(*log.Data) >= 64 {
			d := *log.Data
			r.Set("source_domain", utils.ReadUint32(d, 0))
			r.Set("sender", utils.AddressFromBytes32(d[32:64]))
		}
	})
}

func saveGatewayDeposited(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "gateway")
		r.Set("event_type", "deposit")
		r.Set("source_domain", 26)
		r.Set("destination_domain", 26)

		if log.Topic2 != nil {
			r.Set("sender", utils.AddressFromTopic(log.Topic2))
		}
		if log.Topic3 != nil {
			r.Set("recipient", utils.AddressFromTopic(log.Topic3))
		}
		if log.Data != nil && len(*log.Data) >= 32 {
			r.Set("amount_usdc", utils.StablecoinHuman(utils.ReadBig(*log.Data, 0)))
		}
	})
}

func saveGatewayBurned(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "gateway")
		r.Set("event_type", "withdraw")
		r.Set("source_domain", 26)

		if log.Topic2 != nil {
			r.Set("sender", utils.AddressFromTopic(log.Topic2))
		}
		if log.Topic3 != nil {
			r.Set("nonce_val", log.Topic3.Hex())
		}
		if log.Data != nil && len(*log.Data) >= 128 {
			d := *log.Data
			r.Set("destination_domain", utils.ReadUint32(d, 0))
			r.Set("recipient", utils.AddressFromBytes32(d[32:64]))
			r.Set("amount_usdc", utils.StablecoinHuman(utils.ReadBig(d, 96)))
		}
	})
}

func saveAttestationUsed(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "gateway")
		r.Set("event_type", "deposit")
		r.Set("destination_domain", 26)

		if log.Topic2 != nil {
			r.Set("recipient", utils.AddressFromTopic(log.Topic2))
		}
		if log.Topic3 != nil {
			r.Set("nonce_val", log.Topic3.Hex())
		}
		if log.Data != nil && len(*log.Data) >= 128 {
			d := *log.Data
			r.Set("source_domain", utils.ReadUint32(d, 0))
			r.Set("sender", utils.AddressFromBytes32(d[32:64]))
			r.Set("amount_usdc", utils.StablecoinHuman(utils.ReadBig(d, 96)))
		}
	})
}
