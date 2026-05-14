package main

import (
	"github.com/pocketbase/pocketbase/core"
)

func registerCollections(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		for _, fn := range []func(core.App) error{
			metaCollection,
			indexerEventsCollection,
			tokensCollection,
			blocksCollection,
			transactionsCollection,
			transfersCollection,
			tracesCollection,
			crosschainEventsCollection,
			fxSwapsCollection,
			agentsCollection,
			agentJobsCollection,
			blockStatsCollection,
			walletEdgesCollection,
			analyticsSnapshotsCollection,
		} {
			if err := fn(e.App); err != nil {
				app.Logger().Error("Collection setup error", "error", err)
			}
		}
		return e.Next()
	})
}

func collectionExists(app core.App, name string) bool {
	c, _ := app.FindCollectionByNameOrId(name)
	return c != nil
}

// metaCollection stores indexer cursor state (key/value pairs).
func metaCollection(app core.App) error {
	if collectionExists(app, "indexer_meta") {
		return nil
	}
	c := core.NewBaseCollection("indexer_meta")
	c.Fields.Add(&core.TextField{Name: "key", Required: true, Max: 100})
	c.Fields.Add(&core.TextField{Name: "value", Required: false, Max: 500})
	c.AddIndex("idx_meta_key", true, "key", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created indexer_meta collection")
	return nil
}

// indexerEventsCollection stores durable indexer lifecycle, progress, and error events.
func indexerEventsCollection(app core.App) error {
	if collectionExists(app, "indexer_events") {
		return nil
	}
	c := core.NewBaseCollection("indexer_events")
	c.Fields.Add(&core.NumberField{Name: "timestamp", Required: true})
	c.Fields.Add(&core.SelectField{
		Name:   "level",
		Values: []string{"debug", "info", "warn", "error"},
	})
	c.Fields.Add(&core.TextField{Name: "event", Required: true, Max: 80})
	c.Fields.Add(&core.TextField{Name: "message", Required: false, Max: 500})
	c.Fields.Add(&core.NumberField{Name: "attempt"})
	c.Fields.Add(&core.NumberField{Name: "batch"})
	c.Fields.Add(&core.NumberField{Name: "block"})
	c.Fields.Add(&core.NumberField{Name: "tip"})
	c.Fields.Add(&core.NumberField{Name: "lag"})
	c.Fields.Add(&core.NumberField{Name: "duration_ms"})
	c.Fields.Add(&core.NumberField{Name: "blocks"})
	c.Fields.Add(&core.NumberField{Name: "transactions"})
	c.Fields.Add(&core.NumberField{Name: "logs"})
	c.Fields.Add(&core.TextField{Name: "error", Required: false, Max: 1000})
	c.AddIndex("idx_indexer_events_ts", false, "timestamp", "")
	c.AddIndex("idx_indexer_events_level", false, "level", "")
	c.AddIndex("idx_indexer_events_event", false, "event", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created indexer_events collection")
	return nil
}

// tokensCollection — Layer 3b: token metadata cache (decimals + symbol).
// Populated lazily by the indexer on first sighting of an ERC-20 contract.
// Known stables (USDC/EURC/USYC) are also stored so the lookup path is uniform.
func tokensCollection(app core.App) error {
	if collectionExists(app, "tokens") {
		return nil
	}
	c := core.NewBaseCollection("tokens")
	c.Fields.Add(&core.TextField{Name: "address", Required: true, Max: 42})
	c.Fields.Add(&core.TextField{Name: "symbol", Required: false, Max: 32})
	c.Fields.Add(&core.NumberField{Name: "decimals"})
	c.Fields.Add(&core.NumberField{Name: "first_seen_block"})
	c.Fields.Add(&core.BoolField{Name: "lookup_failed"}) // true if RPC didn't return decimals
	c.AddIndex("idx_tokens_address", true, "address", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created tokens collection")
	return nil
}

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
		if changed {
			return app.Save(c)
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
	c.Fields.Add(&core.NumberField{Name: "decimals"})                            // RPC decimals() result
	c.Fields.Add(&core.TextField{Name: "from_addr", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "to_addr", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "amount_raw", Required: false, Max: 80})
	// amount_human = amount_raw / 10^decimals (per-token, from on-chain decimals())
	c.Fields.Add(&core.TextField{Name: "amount_human", Required: false, Max: 40})
	c.AddIndex("idx_transfers_unique", true, "tx_hash, log_index", "")
	c.AddIndex("idx_transfers_block", false, "block_number", "")
	c.AddIndex("idx_transfers_token", false, "token_address", "")
	c.AddIndex("idx_transfers_from", false, "from_addr", "")
	c.AddIndex("idx_transfers_to", false, "to_addr", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created transfers collection")
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
	c.Fields.Add(&core.TextField{Name: "quote_id", Required: false, Max: 66})  // bytes32 from TradeRecorded
	c.Fields.Add(&core.TextField{Name: "maker", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "taker", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "taker_fee", Required: false, Max: 40}) // raw wei
	c.Fields.Add(&core.TextField{Name: "maker_fee", Required: false, Max: 40}) // raw wei
	c.Fields.Add(&core.NumberField{Name: "status_code"})                        // raw uint8 from TradeStatusChanged
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

// agentsCollection — Layer 5a: registered ERC-8004 AI agents.
func agentsCollection(app core.App) error {
	if collectionExists(app, "agents") {
		return nil
	}
	c := core.NewBaseCollection("agents")
	c.Fields.Add(&core.TextField{Name: "agent_address", Required: true, Max: 42})
	c.Fields.Add(&core.TextField{Name: "metadata_uri", Required: false, Max: 500})
	c.Fields.Add(&core.NumberField{Name: "registered_at_block"})
	c.Fields.Add(&core.TextField{Name: "tx_hash", Required: false, Max: 66})
	// aggregated stats updated by indexer
	c.Fields.Add(&core.NumberField{Name: "tx_count"})
	c.Fields.Add(&core.TextField{Name: "usdc_spent_fees", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "usdc_transferred", Required: false, Max: 40})
	c.AddIndex("idx_agents_address", true, "agent_address", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created agents collection")
	return nil
}

// agentJobsCollection — Layer 5b: ERC-8183 job lifecycle events.
// Status lifecycle: created → funded → submitted → completed|rejected|expired → paid
func agentJobsCollection(app core.App) error {
	if collectionExists(app, "agent_jobs") {
		return nil
	}
	c := core.NewBaseCollection("agent_jobs")
	c.Fields.Add(&core.TextField{Name: "job_id", Required: true, Max: 80})
	c.Fields.Add(&core.TextField{Name: "employer_address", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "worker_address", Required: false, Max: 42})
	c.Fields.Add(&core.TextField{Name: "payment_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.SelectField{
		Name:   "status",
		Values: []string{"created", "funded", "submitted", "completed", "rejected", "expired", "paid"},
	})
	c.Fields.Add(&core.NumberField{Name: "created_at_block"})
	c.Fields.Add(&core.NumberField{Name: "settled_at_block"})
	c.Fields.Add(&core.TextField{Name: "create_tx_hash", Required: false, Max: 66})
	c.Fields.Add(&core.TextField{Name: "settle_tx_hash", Required: false, Max: 66})
	c.AddIndex("idx_jobs_id", true, "job_id", "")
	c.AddIndex("idx_jobs_employer", false, "employer_address", "")
	c.AddIndex("idx_jobs_worker", false, "worker_address", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created agent_jobs collection")
	return nil
}

// blockStatsCollection — Layer 7: pre-aggregated per-block metrics.
func blockStatsCollection(app core.App) error {
	if collectionExists(app, "block_stats") {
		return nil
	}
	c := core.NewBaseCollection("block_stats")
	c.Fields.Add(&core.NumberField{Name: "block_number", Required: true})
	c.Fields.Add(&core.NumberField{Name: "timestamp"})
	c.Fields.Add(&core.NumberField{Name: "tps"})
	c.Fields.Add(&core.NumberField{Name: "tx_count"})
	c.Fields.Add(&core.NumberField{Name: "failed_tx_count"})
	c.Fields.Add(&core.TextField{Name: "avg_fee_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "total_fee_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "total_usdc_transferred", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "total_eurc_transferred", Required: false, Max: 40})
	c.Fields.Add(&core.TextField{Name: "total_usyc_transferred", Required: false, Max: 40})
	c.Fields.Add(&core.NumberField{Name: "unique_senders"})
	c.Fields.Add(&core.NumberField{Name: "unique_receivers"})
	c.Fields.Add(&core.NumberField{Name: "new_contracts"})
	c.Fields.Add(&core.TextField{Name: "largest_usdc_transfer", Required: false, Max: 40})
	c.Fields.Add(&core.NumberField{Name: "utilization_pct"})
	c.Fields.Add(&core.NumberField{Name: "block_time_ms"})
	c.AddIndex("idx_bstats_number", true, "block_number", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created block_stats collection")
	return nil
}

// analyticsSnapshotsCollection — Layer 9: pre-aggregated window analytics.
// One row per window per snapshot time. Handlers read the latest row; history
// endpoint returns all rows for time-series charting.
func analyticsSnapshotsCollection(app core.App) error {
	if collectionExists(app, "analytics_snapshots") {
		return nil
	}
	c := core.NewBaseCollection("analytics_snapshots")
	c.Fields.Add(&core.NumberField{Name: "snapshot_at", Required: true})
	c.Fields.Add(&core.NumberField{Name: "block_number"})
	c.Fields.Add(&core.TextField{Name: "window", Required: true, Max: 10})
	// transfers / volume
	c.Fields.Add(&core.NumberField{Name: "transfers_count"})
	c.Fields.Add(&core.NumberField{Name: "transfer_volume"})
	c.Fields.Add(&core.NumberField{Name: "largest_transfer"})
	c.Fields.Add(&core.NumberField{Name: "largest_transfer_block"})
	c.Fields.Add(&core.NumberField{Name: "usdc_volume"})
	c.Fields.Add(&core.NumberField{Name: "eurc_volume"})
	c.Fields.Add(&core.NumberField{Name: "usyc_volume"})
	c.Fields.Add(&core.NumberField{Name: "usdc_count"})
	c.Fields.Add(&core.NumberField{Name: "eurc_count"})
	c.Fields.Add(&core.NumberField{Name: "usyc_count"})
	c.Fields.Add(&core.NumberField{Name: "whale_transfers"})
	c.Fields.Add(&core.NumberField{Name: "unique_senders"})
	c.Fields.Add(&core.NumberField{Name: "unique_receivers"})
	c.Fields.Add(&core.NumberField{Name: "total_transfers"})
	// fees / tx stats
	c.Fields.Add(&core.NumberField{Name: "fees_total"})
	c.Fields.Add(&core.NumberField{Name: "fee_p25"})
	c.Fields.Add(&core.NumberField{Name: "fee_p50"})
	c.Fields.Add(&core.NumberField{Name: "fee_p75"})
	c.Fields.Add(&core.NumberField{Name: "fee_p95"})
	c.Fields.Add(&core.NumberField{Name: "failed_tx_ratio"})
	c.Fields.Add(&core.NumberField{Name: "total_txs"})
	c.Fields.Add(&core.NumberField{Name: "failed_txs"})
	c.Fields.Add(&core.NumberField{Name: "avg_block_time_ms"})
	c.Fields.Add(&core.NumberField{Name: "block_count"})
	// bridge
	c.Fields.Add(&core.NumberField{Name: "bridge_inbound_vol"})
	c.Fields.Add(&core.NumberField{Name: "bridge_inbound_count"})
	c.Fields.Add(&core.NumberField{Name: "bridge_outbound_vol"})
	c.Fields.Add(&core.NumberField{Name: "bridge_outbound_count"})
	c.Fields.Add(&core.NumberField{Name: "bridge_net_flow"})
	c.Fields.Add(&core.TextField{Name: "bridge_by_chain", Required: false, Max: 8000})
	// agents
	c.Fields.Add(&core.NumberField{Name: "agent_count"})
	c.AddIndex("idx_snapshots_window_at", false, "window, snapshot_at", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created analytics_snapshots collection")
	return nil
}

// walletEdgesCollection — Layer 8: wallet graph (nodes + edges for 3D viz).
func walletEdgesCollection(app core.App) error {
	if collectionExists(app, "wallet_edges") {
		return nil
	}
	c := core.NewBaseCollection("wallet_edges")
	c.Fields.Add(&core.TextField{Name: "from_wallet", Required: true, Max: 42})
	c.Fields.Add(&core.TextField{Name: "to_wallet", Required: true, Max: 42})
	c.Fields.Add(&core.TextField{Name: "total_usdc", Required: false, Max: 40})
	c.Fields.Add(&core.NumberField{Name: "tx_count"})
	c.Fields.Add(&core.NumberField{Name: "last_seen_block"})
	c.Fields.Add(&core.BoolField{Name: "from_is_agent"})
	c.Fields.Add(&core.BoolField{Name: "to_is_agent"})
	c.AddIndex("idx_edges_unique", true, "from_wallet, to_wallet", "")
	c.AddIndex("idx_edges_from", false, "from_wallet", "")
	c.AddIndex("idx_edges_to", false, "to_wallet", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created wallet_edges collection")
	return nil
}
