package repo

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ListAgents returns all agents sorted by registration block descending.
func ListAgents(app core.App, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "agents", "", "-registered_at_block", limit, offset)
}

// AgentByAddress returns the agent with the given address, or nil.
func AgentByAddress(app core.App, addr string) (*core.Record, error) {
	return LatestRecord(app, "agents", "agent_address = {:a}", "", map[string]any{"a": addr})
}

// AgentLeaderboard returns agents sorted by USDC transferred descending.
func AgentLeaderboard(app core.App, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "agents", "", "-usdc_transferred_num", limit, offset)
}

// CountAgents returns the total number of registered agents.
func CountAgents(app core.App) (int, error) {
	return CountWithFilter(app, "agents", "", nil)
}

// SearchAgents searches agents by address. For 0x-prefixed queries it uses an
// anchored LIKE so the address index is hit; otherwise it falls back to
// PocketBase's `~` (contains).
func SearchAgents(app core.App, q string, limit int) ([]*core.Record, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return FindRecords(app, "agents", "", "-usdc_transferred_num", limit, 0)
	}
	if strings.HasPrefix(strings.ToLower(q), "0x") {
		var records []*core.Record
		err := app.RecordQuery("agents").
			AndWhere(dbx.NewExp("agent_address LIKE {:p}", dbx.Params{"p": strings.ToLower(q) + "%"})).
			OrderBy("usdc_transferred_num DESC").
			Limit(int64(limit)).
			All(&records)
		return records, err
	}
	return FindRecords(app, "agents", "agent_address ~ {:s}", "-usdc_transferred_num", limit, 0,
		map[string]any{"s": q})
}

// JobAgg holds aggregated job statistics per agent address.
type JobAgg struct {
	Addr         string  `db:"addr"`
	JobCount     int     `db:"job_count"`
	TotalEscrow  float64 `db:"total_escrow"`
	PaidJobs     int     `db:"paid_jobs"`
	RejectedJobs int     `db:"rejected_jobs"`
}

// AgentJobStats returns per-address aggregated job statistics.
func AgentJobStats(app core.App) ([]JobAgg, error) {
	var stats []JobAgg
	err := app.DB().NewQuery(`
		SELECT addr, SUM(job_count) as job_count, SUM(total_escrow) as total_escrow,
		       SUM(paid_jobs) as paid_jobs, SUM(rejected_jobs) as rejected_jobs
		FROM (
		    SELECT employer_address as addr, COUNT(*) as job_count,
		           COALESCE(SUM(CAST(payment_usdc AS REAL)), 0) as total_escrow,
		           SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) as paid_jobs,
		           SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) as rejected_jobs
		    FROM agent_jobs GROUP BY employer_address
		    UNION ALL
		    SELECT worker_address as addr, COUNT(*) as job_count,
		           COALESCE(SUM(CAST(payment_usdc AS REAL)), 0) as total_escrow,
		           SUM(CASE WHEN status = 'paid' THEN 1 ELSE 0 END) as paid_jobs,
		           SUM(CASE WHEN status = 'rejected' THEN 1 ELSE 0 END) as rejected_jobs
		    FROM agent_jobs GROUP BY worker_address
		) GROUP BY addr`).All(&stats)
	if err != nil {
		return nil, err
	}
	return stats, nil
}
