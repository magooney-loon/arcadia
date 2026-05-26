package collections

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// tokenAnalyticsCollection — Layer 1: chain skeleton.
func tokenAnalyticsCollection(app core.App) error {
	if collectionExists(app, "token_analytics") {
		return nil
	}
	c := core.NewBaseCollection("token_analytics")
	c.Fields.Add(&core.TextField{Name: "token_address", Required: true, Max: 42})
	c.Fields.Add(&core.TextField{Name: "symbol", Required: false, Max: 64})
	c.Fields.Add(&core.TextField{Name: "name", Required: false, Max: 128})
	c.Fields.Add(&core.NumberField{Name: "decimals"})
	c.Fields.Add(&core.SelectField{
		Name:   "token_type",
		Values: []string{"ERC-20", "ERC-721", "ERC-1155"},
	})
	c.Fields.Add(&core.TextField{Name: "total_supply_raw", Required: false, Max: 80})   // raw uint256 as string
	c.Fields.Add(&core.TextField{Name: "total_supply_human", Required: false, Max: 80}) // formatted
	c.Fields.Add(&core.NumberField{Name: "transfer_count"})
	c.Fields.Add(&core.NumberField{Name: "unique_senders"})
	c.Fields.Add(&core.NumberField{Name: "unique_receivers"})
	c.Fields.Add(&core.NumberField{Name: "first_seen_block"})
	c.Fields.Add(&core.NumberField{Name: "last_seen_block"})
	c.Fields.Add(&core.BoolField{Name: "lookup_failed"})
	c.AddIndex("idx_token_analytics_address", true, "token_address", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created token_analytics collection")
	return nil
}

// transfersCollection — Layer 3: all ERC-20 token transfers (USDC, EURC, USYC, …).
func transfersCollection(app core.App) error {
	if collectionExists(app, "transfers") {
		c, err := app.FindCollectionByNameOrId("transfers")
		if err != nil {
			return err
		}
		changed := false
		if c.Fields.GetByName("decimals") == nil {
			c.Fields.Add(&core.NumberField{Name: "decimals"})
			changed = true
		}
		if c.Fields.GetByName("token_name") == nil {
			c.Fields.Add(&core.TextField{Name: "token_name", Required: false, Max: 32})
			changed = true
		}
		// amount_num is an indexed numeric mirror of amount_human, so analytics
		// queries can ORDER BY / SUM without `CAST(... AS REAL)` table scans.
		backfillAmountNum := false
		if c.Fields.GetByName("amount_num") == nil {
			c.Fields.Add(&core.NumberField{Name: "amount_num"})
			changed = true
			backfillAmountNum = true
		}
		// Add token_symbol index for transfers filtering.
		hasSymIdx, hasAmountNumIdx, hasWindowIdx := false, false, false
		for _, idx := range c.Indexes {
			if strings.Contains(idx, "idx_transfers_token_symbol") {
				hasSymIdx = true
			}
			if strings.Contains(idx, "idx_transfers_amount_num") {
				hasAmountNumIdx = true
			}
			if strings.Contains(idx, "idx_transfers_window") {
				hasWindowIdx = true
			}
		}
		if !hasSymIdx {
			c.AddIndex("idx_transfers_token_symbol", false, "token_symbol", "")
			changed = true
		}
		if !hasAmountNumIdx {
			c.AddIndex("idx_transfers_amount_num", false, "amount_num", "")
			changed = true
		}
		// Composite (block_number, token_symbol) lets analytics_snapshot's
		// windowed GROUP BY token_symbol use a single index range scan.
		if !hasWindowIdx {
			c.AddIndex("idx_transfers_window", false, "block_number, token_symbol", "")
			changed = true
		}
		if changed {
			if err := app.Save(c); err != nil {
				return err
			}
		}
		if backfillAmountNum {
			if _, err := app.DB().NewQuery(
				`UPDATE transfers SET amount_num = CAST(amount_human AS REAL)
				 WHERE amount_num IS NULL AND amount_human IS NOT NULL AND amount_human != ''`).Execute(); err != nil {
				app.Logger().Warn("transfers.amount_num backfill failed", "error", err)
			}
		}
		return nil
	}
	c := core.NewBaseCollection("transfers")
	c.Fields.Add(&core.TextField{Name: "tx_hash", Required: true, Max: 66})
	c.Fields.Add(&core.NumberField{Name: "block_number"})
	c.Fields.Add(&core.NumberField{Name: "log_index"})
	c.Fields.Add(&core.TextField{Name: "token_address", Required: true, Max: 42})
	c.Fields.Add(&core.SelectField{
		Name:   "token_symbol",
		Values: []string{"USDC", "EURC", "USYC", "OTHER"},
	})
	c.Fields.Add(&core.TextField{Name: "token_name", Required: false, Max: 32}) // RPC symbol() result
	c.Fields.Add(&core.SelectField{
		Name:   "token_type",
		Values: []string{"ERC-20", "ERC-721", "ERC-1155"},
	})
	c.Fields.Add(&core.NumberField{Name: "decimals"}) // RPC decimals() result
	c.Fields.Add(&core.TextField{Name: "from_addr", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "to_addr", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "amount_raw", Required: false, Max: 80})
	// amount_human = amount_raw / 10^decimals (per-token, from on-chain decimals())
	c.Fields.Add(&core.TextField{Name: "amount_human", Required: false, Max: 40})
	// Indexed numeric mirror of amount_human, used by analytics SQL.
	c.Fields.Add(&core.NumberField{Name: "amount_num"})
	c.AddIndex("idx_transfers_unique", true, "tx_hash, log_index", "")
	c.AddIndex("idx_transfers_block", false, "block_number", "")
	c.AddIndex("idx_transfers_token", false, "token_address", "")
	c.AddIndex("idx_transfers_token_symbol", false, "token_symbol", "")
	c.AddIndex("idx_transfers_from", false, "from_addr", "")
	c.AddIndex("idx_transfers_to", false, "to_addr", "")
	c.AddIndex("idx_transfers_amount_num", false, "amount_num", "")
	c.AddIndex("idx_transfers_window", false, "block_number, token_symbol", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created transfers collection")
	return nil
}
