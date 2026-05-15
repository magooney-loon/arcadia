package repo

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// TraceFilter holds optional filter criteria for traces.
type TraceFilter struct {
	TxHash   string
	FromAddr string
	ToAddr   string
}

// ListTraces returns traces matching the given filter.
func ListTraces(app core.App, f TraceFilter, limit, offset int) ([]*core.Record, error) {
	parts := []string{}
	params := map[string]any{}
	if f.TxHash != "" {
		parts = append(parts, "tx_hash = {:h}")
		params["h"] = f.TxHash
	}
	if f.FromAddr != "" {
		parts = append(parts, "from_addr = {:fa}")
		params["fa"] = f.FromAddr
	}
	if f.ToAddr != "" {
		parts = append(parts, "to_addr = {:ta}")
		params["ta"] = f.ToAddr
	}
	filter := strings.Join(parts, " && ")
	return FindRecords(app, "traces", filter, "-block_number", limit, offset, params)
}

// TracesByTxHash returns traces for a specific transaction.
func TracesByTxHash(app core.App, txHash string) ([]*core.Record, error) {
	return FindRecords(app, "traces", "tx_hash = {:h}", "", 0, 0, map[string]any{"h": txHash})
}
