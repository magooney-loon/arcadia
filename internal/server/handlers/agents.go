package handlers

// API_SOURCE

import (
	"net/http"

	"arcadia/internal/repo"

	"github.com/pocketbase/pocketbase/core"
)

// API_DESC List all registered AI agents (ERC-8004)
// API_TAGS Agents
func agentsHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)
	records, err := repo.ListAgents(c.App, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	out := make([]map[string]any, len(records))
	for i, r := range records {
		out[i] = enrichAgentRecord(r)
	}
	total, _ := repo.CountAgents(c.App)
	return c.JSON(http.StatusOK, map[string]any{
		"agents": out,
		"count":  len(records),
		"total":  total,
	})
}

// API_DESC Single agent profile + job history
// API_TAGS Agents
func agentHandler(c *core.RequestEvent) error {
	address := c.Request.PathValue("address")
	if address == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "address required"})
	}

	agentRow, err := repo.AgentByAddress(c.App, address)
	if err != nil || agentRow == nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "agent not found"})
	}

	jobs, _ := repo.JobsByAddress(c.App, address, 50, 0)

	return c.JSON(http.StatusOK, map[string]any{
		"agent": enrichAgentRecord(agentRow),
		"jobs":  recordsToMaps(jobs),
	})
}

// API_DESC Agent job marketplace — filter by status or worker/employer
// API_TAGS Agents
func agentJobsHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	f := repo.JobFilter{}
	if status := qp(c, "status", ""); status != "" {
		f.Status = status
	}
	if employer := qp(c, "employer", ""); employer != "" {
		f.Employer = employer
	}
	if worker := qp(c, "worker", ""); worker != "" {
		f.Worker = worker
	}

	records, err := repo.ListJobs(c.App, f, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	total, _ := repo.CountJobs(c.App, f)
	return c.JSON(http.StatusOK, map[string]any{
		"jobs":  recordsToMaps(records),
		"count": len(records),
		"total": total,
	})
}

// API_DESC Agent leaderboard ranked by stablecoin volume transferred
// API_TAGS Agents
func analyticsAgentLeaderboardHandler(c *core.RequestEvent) error {
	cacheHeaders(c, 60)
	limit, offset := limitOffset(c)
	if limit > 100 {
		limit = 100
	}

	// Batch query: get job stats for ALL agents in a single SQL aggregation
	// instead of N+1 queries per agent.
	jobStats, _ := repo.AgentJobStats(c.App)

	statsMap := make(map[string]*repo.JobAgg, len(jobStats))
	for i := range jobStats {
		statsMap[jobStats[i].Addr] = &jobStats[i]
	}

	// Index-backed sort on the numeric mirror column avoids the old in-memory
	// sprintf+ParseFloat-per-agent pass and lets us cap the result set in SQL.
	agents, err := repo.AgentLeaderboard(c.App, limit, offset)
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

	total, _ := repo.CountAgents(c.App)

	// Aggregate totals across ALL agents (not just the current page) so the
	// page can show honest "total escrow" / "in-flight jobs" stats.
	var totalEscrow float64
	var totalJobs, totalPaid, totalRejected int
	for _, s := range jobStats {
		totalEscrow += s.TotalEscrow
		totalJobs += s.JobCount
		totalPaid += s.PaidJobs
		totalRejected += s.RejectedJobs
	}

	return c.JSON(http.StatusOK, map[string]any{
		"leaderboard": result,
		"count":       len(result),
		"total":       total,
		"summary": map[string]any{
			"total_escrow":   totalEscrow,
			"jobs_total":     totalJobs,
			"jobs_paid":      totalPaid,
			"jobs_rejected":  totalRejected,
			"jobs_in_flight": totalJobs - totalPaid - totalRejected,
		},
	})
}
