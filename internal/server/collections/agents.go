package collections

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// agentsCollection — Layer 5a: registered ERC-8004 AI agents.
func agentsCollection(app core.App) error {
	if collectionExists(app, "agents") {
		c, err := app.FindCollectionByNameOrId("agents")
		if err != nil {
			return err
		}
		needsBackfill := false
		if c.Fields.GetByName("usdc_transferred_num") == nil {
			c.Fields.Add(&core.NumberField{Name: "usdc_transferred_num"})
			needsBackfill = true
		}
		hasLeaderboardIdx := false
		for _, idx := range c.Indexes {
			if strings.Contains(idx, "usdc_transferred_num") {
				hasLeaderboardIdx = true
			}
		}
		if !hasLeaderboardIdx {
			c.AddIndex("idx_agents_leaderboard", false, "usdc_transferred_num", "")
		}
		if needsBackfill || !hasLeaderboardIdx {
			if err := app.Save(c); err != nil {
				return err
			}
		}
		if needsBackfill {
			// Raw column holds ERC-20 stablecoin units (6 decimals).
			if _, err := app.DB().NewQuery(
				`UPDATE agents
				 SET usdc_transferred_num = COALESCE(CAST(usdc_transferred AS REAL), 0) / 1000000.0
				 WHERE usdc_transferred_num IS NULL`).Execute(); err != nil {
				app.Logger().Warn("agents.usdc_transferred_num backfill failed", "error", err)
			}
		}
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
	// Indexed numeric mirror of usdc_transferred (decimals applied) for SQL leaderboard sort.
	c.Fields.Add(&core.NumberField{Name: "usdc_transferred_num"})
	c.AddIndex("idx_agents_address", true, "agent_address", "")
	c.AddIndex("idx_agents_leaderboard", false, "usdc_transferred_num", "")
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
