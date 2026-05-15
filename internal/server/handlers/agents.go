package handlers

// API_SOURCE

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// API_DESC List all registered AI agents (ERC-8004)
// API_TAGS Agents
func agentsHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)
	records, err := c.App.FindRecordsByFilter("agents", "", "-registered_at_block", limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	out := make([]map[string]any, len(records))
	for i, r := range records {
		out[i] = enrichAgentRecord(r)
	}
	return c.JSON(http.StatusOK, map[string]any{
		"agents": out,
		"count":  len(records),
	})
}

// API_DESC Single agent profile + job history
// API_TAGS Agents
func agentHandler(c *core.RequestEvent) error {
	address := c.Request.PathValue("address")
	if address == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "address required"})
	}

	agentRows, _ := c.App.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, map[string]any{"a": address})
	if len(agentRows) == 0 {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "agent not found"})
	}

	jobs, _ := c.App.FindRecordsByFilter("agent_jobs",
		"employer_address = {:a} || worker_address = {:a}", "-created_at_block", 50, 0,
		map[string]any{"a": address})

	return c.JSON(http.StatusOK, map[string]any{
		"agent": enrichAgentRecord(agentRows[0]),
		"jobs":  recordsToMaps(jobs),
	})
}

// API_DESC Agent job marketplace — filter by status or worker/employer
// API_TAGS Agents
func agentJobsHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}
	if status := qp(c, "status", ""); status != "" {
		filter = "status = {:s}"
		params["s"] = status
	}
	if employer := qp(c, "employer", ""); employer != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "employer_address = {:e}"
		params["e"] = employer
	}
	if worker := qp(c, "worker", ""); worker != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "worker_address = {:w}"
		params["w"] = worker
	}

	records, err := c.App.FindRecordsByFilter("agent_jobs", filter, "-created_at_block", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"jobs":  recordsToMaps(records),
		"count": len(records),
	})
}

// API_DESC Agent leaderboard ranked by stablecoin volume transferred
// API_TAGS Agents
func analyticsAgentLeaderboardHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 60)
	limit, _ := limitOffset(c)
	if limit > 100 {
		limit = 100
	}

	// Batch query: get job stats for ALL agents in a single SQL aggregation
	// instead of N+1 queries per agent.
	type jobAgg struct {
		Addr         string  `db:"addr"`
		JobCount     int     `db:"job_count"`
		TotalEscrow  float64 `db:"total_escrow"`
		PaidJobs     int     `db:"paid_jobs"`
		RejectedJobs int     `db:"rejected_jobs"`
	}
	var jobStats []jobAgg
	_ = c.App.DB().NewQuery(`
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
		) GROUP BY addr`).All(&jobStats)

	statsMap := make(map[string]*jobAgg, len(jobStats))
	for i := range jobStats {
		statsMap[jobStats[i].Addr] = &jobStats[i]
	}

	// Index-backed sort on the numeric mirror column avoids the old in-memory
	// sprintf+ParseFloat-per-agent pass and lets us cap the result set in SQL.
	agents, err := c.App.FindRecordsByFilter("agents", "", "-usdc_transferred_num", limit, 0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	result := make([]map[string]any, 0, len(agents))
	for _, a := range agents {
		entry := enrichAgentRecord(a)
		addr := a.GetString("agent_address")
		if s, ok := statsMap[addr]; ok {
			entry["job_count"] = s.JobCount
			entry["total_escrow"] = s.TotalEscrow
			entry["paid_jobs"] = s.PaidJobs
			entry["rejected_jobs"] = s.RejectedJobs
		} else {
			entry["job_count"] = 0
			entry["total_escrow"] = 0.0
			entry["paid_jobs"] = 0
			entry["rejected_jobs"] = 0
		}
		result = append(result, entry)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"leaderboard": result,
		"count":       len(result),
	})
}
