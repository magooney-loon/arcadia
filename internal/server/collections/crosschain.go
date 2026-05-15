package collections

import (
	"github.com/pocketbase/pocketbase/core"
)

// crosschainEventsCollection — Layer 6: CCTP burns/mints and Gateway deposits/withdrawals.
func crosschainEventsCollection(app core.App) error {
	if collectionExists(app, "crosschain_events") {
		return nil
	}
	c := core.NewBaseCollection("crosschain_events")
	c.Fields.Add(&core.TextField{Name: "tx_hash", Required: true, Max: 66})
	c.Fields.Add(&core.NumberField{Name: "block_number"})
	c.Fields.Add(&core.NumberField{Name: "log_index"})
	c.Fields.Add(&core.SelectField{
		Name:   "protocol",
		Values: []string{"cctp", "gateway"},
	})
	c.Fields.Add(&core.SelectField{
		Name:   "event_type",
		Values: []string{"burn", "mint", "deposit", "withdraw"},
	})
	c.Fields.Add(&core.NumberField{Name: "source_domain"})
	c.Fields.Add(&core.NumberField{Name: "destination_domain"})
	c.Fields.Add(&core.TextField{Name: "sender", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "recipient", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "amount_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "nonce_val", Required: false, Max: 80})
	c.AddIndex("idx_crosschain_unique", true, "tx_hash, log_index", "")
	c.AddIndex("idx_crosschain_block", false, "block_number", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created crosschain_events collection")
	return nil
}

// fxSwapsCollection — Layer 6b: StableFX trade lifecycle (FxEscrow contract).
// Each record represents one trade, keyed by trade_id. Multiple events
// (TradeRecorded, MakerFunded, TakerFunded, TradeStatusChanged, FeesProcessed)
// are upserted into the same row.
func fxSwapsCollection(app core.App) error {
	if collectionExists(app, "fx_swaps") {
		c, err := app.FindCollectionByNameOrId("fx_swaps")
		if err != nil {
			return err
		}
		changed := false
		addMissing := func(field core.Field) {
			if c.Fields.GetByName(field.GetName()) == nil {
				c.Fields.Add(field)
				changed = true
			}
		}
		addMissing(&core.TextField{Name: "trade_id", Required: false, Max: 80})
		addMissing(&core.TextField{Name: "quote_id", Required: false, Max: 66})
		addMissing(&core.TextField{Name: "taker_fee", Required: false, Max: 40})
		addMissing(&core.TextField{Name: "maker_fee", Required: false, Max: 40})
		addMissing(&core.NumberField{Name: "status_code"})
		if changed {
			return app.Save(c)
		}
		return nil
	}
	c := core.NewBaseCollection("fx_swaps")
	// trade_id links all events for the same trade (uint256 as string)
	c.Fields.Add(&core.TextField{Name: "trade_id", Required: true, Max: 80})
	c.Fields.Add(&core.TextField{Name: "quote_id", Required: false, Max: 66}) // bytes32 from TradeRecorded
	c.Fields.Add(&core.TextField{Name: "maker", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "taker", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "taker_fee", Required: false, Max: 40}) // raw wei
	c.Fields.Add(&core.TextField{Name: "maker_fee", Required: false, Max: 40}) // raw wei
	c.Fields.Add(&core.NumberField{Name: "status_code"})                       // raw uint8 from TradeStatusChanged
	c.Fields.Add(&core.SelectField{
		Name:   "status",
		Values: []string{"created", "taker_funded", "maker_funded", "settled", "cancelled"},
	})
	c.Fields.Add(&core.NumberField{Name: "block_number"})
	c.Fields.Add(&core.TextField{Name: "tx_hash", Required: false, Max: 66}) // tx of TradeRecorded
	c.AddIndex("idx_fx_trade_id", true, "trade_id", "")
	c.AddIndex("idx_fx_block", false, "block_number", "")
	c.AddIndex("idx_fx_maker", false, "maker", "")
	c.AddIndex("idx_fx_taker", false, "taker", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created fx_swaps collection")
	return nil
}
