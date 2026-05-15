package collections

import (
	"github.com/pocketbase/pocketbase/core"
)

// blocksCollection — Layer 1: chain skeleton.
func blocksCollection(app core.App) error {
	if collectionExists(app, "blocks") {
		return nil
	}
	c := core.NewBaseCollection("blocks")
	c.Fields.Add(&core.NumberField{Name: "number", Required: true})
	c.Fields.Add(&core.TextField{Name: "hash", Required: true, Max: 66})
	c.Fields.Add(&core.TextField{Name: "parent_hash", Required: false, Max: 66})
	c.Fields.Add(&core.TextField{Name: "miner", Required: false, Max: 42})
	c.Fields.Add(&core.NumberField{Name: "timestamp"})
	c.Fields.Add(&core.NumberField{Name: "gas_used"})
	c.Fields.Add(&core.NumberField{Name: "gas_limit"})
	// base_fee_per_gas stored as string — big.Int, sub-wei precision
	c.Fields.Add(&core.TextField{Name: "base_fee_per_gas", Required: false, Max: 80})
	c.Fields.Add(&core.NumberField{Name: "size"})
	// derived fields computed at index time
	c.Fields.Add(&core.NumberField{Name: "tx_count"})
	c.Fields.Add(&core.NumberField{Name: "block_time_ms"}) // ms since previous block
	c.Fields.Add(&core.NumberField{Name: "utilization_pct"})
	c.AddIndex("idx_blocks_number", true, "number", "")
	c.AddIndex("idx_blocks_hash", true, "hash", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created blocks collection")
	return nil
}

// transactionsCollection — Layer 2: every transaction.
func transactionsCollection(app core.App) error {
	if collectionExists(app, "transactions") {
		c, err := app.FindCollectionByNameOrId("transactions")
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
		addMissing(&core.NumberField{Name: "gas_limit"})
		addMissing(&core.NumberField{Name: "cumulative_gas_used"})
		addMissing(&core.TextField{Name: "max_fee_per_gas", Required: false, Max: 80})
		addMissing(&core.TextField{Name: "max_priority_fee_per_gas", Required: false, Max: 80})
		addMissing(&core.TextField{Name: "priority_fee_per_gas", Required: false, Max: 80})
		addMissing(&core.TextField{Name: "priority_fee_usdc", Required: false, Max: 40})
		addMissing(&core.NumberField{Name: "status"})
		if changed {
			return app.Save(c)
		}
		return nil
	}
	c := core.NewBaseCollection("transactions")
	c.Fields.Add(&core.TextField{Name: "hash", Required: true, Max: 66})
	c.Fields.Add(&core.NumberField{Name: "block_number"})
	c.Fields.Add(&core.NumberField{Name: "transaction_index"})
	c.Fields.Add(&core.TextField{Name: "from_addr", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "to_addr", Required: false, Max: 42})
	// value / amounts as strings to preserve uint256 precision
	c.Fields.Add(&core.TextField{Name: "value", Required: false, Max: 80})
	c.Fields.Add(&core.NumberField{Name: "nonce"})
	c.Fields.Add(&core.TextField{Name: "sighash", Required: false, Max: 10}) // first 4 bytes hex
	c.Fields.Add(&core.TextField{Name: "gas_price", Required: false, Max: 80})
	c.Fields.Add(&core.NumberField{Name: "gas_limit"})
	c.Fields.Add(&core.NumberField{Name: "gas_used"})
	c.Fields.Add(&core.NumberField{Name: "cumulative_gas_used"})
	c.Fields.Add(&core.TextField{Name: "effective_gas_price", Required: false, Max: 80})
	c.Fields.Add(&core.TextField{Name: "max_fee_per_gas", Required: false, Max: 80})
	c.Fields.Add(&core.TextField{Name: "max_priority_fee_per_gas", Required: false, Max: 80})
	c.Fields.Add(&core.TextField{Name: "priority_fee_per_gas", Required: false, Max: 80})
	// fee in USDC: gas_used * effective_gas_price / 1e18 (native USDC uses 18 decimals)
	c.Fields.Add(&core.TextField{Name: "fee_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "priority_fee_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.NumberField{Name: "tx_type"})
	c.Fields.Add(&core.NumberField{Name: "status"})
	c.Fields.Add(&core.TextField{Name: "contract_address", Required: false, Max: 42})
	c.Fields.Add(&core.BoolField{Name: "is_contract_deploy"})
	c.AddIndex("idx_tx_hash", true, "hash", "")
	c.AddIndex("idx_tx_block", false, "block_number", "")
	c.AddIndex("idx_tx_from", false, "from_addr", "")
	c.AddIndex("idx_tx_to", false, "to_addr", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created transactions collection")
	return nil
}

// tracesCollection — Layer 4: internal contract-to-contract calls.
func tracesCollection(app core.App) error {
	if collectionExists(app, "traces") {
		return nil
	}
	c := core.NewBaseCollection("traces")
	c.Fields.Add(&core.TextField{Name: "tx_hash", Required: true, Max: 66})
	c.Fields.Add(&core.NumberField{Name: "block_number"})
	c.Fields.Add(&core.TextField{Name: "from_addr", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "to_addr", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "value", Required: false, Max: 80})
	c.Fields.Add(&core.TextField{Name: "call_type", Required: false, Max: 20})
	c.Fields.Add(&core.TextField{Name: "trace_type", Required: false, Max: 20})
	c.Fields.Add(&core.NumberField{Name: "gas_used"})
	c.Fields.Add(&core.TextField{Name: "error_msg", Required: false, Max: 200})
	c.AddIndex("idx_traces_tx", false, "tx_hash", "")
	c.AddIndex("idx_traces_block", false, "block_number", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created traces collection")
	return nil
}
