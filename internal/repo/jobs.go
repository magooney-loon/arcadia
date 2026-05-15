package repo

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// JobFilter holds optional filter criteria for agent jobs.
type JobFilter struct {
	Status   string
	Employer string
	Worker   string
	Addr     string // matches either employer or worker
}

// ListJobs returns agent jobs matching the given filter.
func ListJobs(app core.App, f JobFilter, limit, offset int) ([]*core.Record, error) {
	parts := []string{}
	params := map[string]any{}
	if f.Status != "" {
		parts = append(parts, "status = {:s}")
		params["s"] = f.Status
	}
	if f.Employer != "" {
		parts = append(parts, "employer_address = {:e}")
		params["e"] = f.Employer
	}
	if f.Worker != "" {
		parts = append(parts, "worker_address = {:w}")
		params["w"] = f.Worker
	}
	filter := strings.Join(parts, " && ")
	return FindRecords(app, "agent_jobs", filter, "-created_at_block", limit, offset, params)
}

// JobsByAddress returns jobs where the address is either employer or worker.
func JobsByAddress(app core.App, addr string, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "agent_jobs", "employer_address = {:a} || worker_address = {:a}", "-created_at_block", limit, offset, map[string]any{"a": addr})
}
