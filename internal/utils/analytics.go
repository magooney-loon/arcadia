package utils

import (
	"strconv"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// WindowToBlockCount converts a time window string to an approximate block count.
// Arc L1 averages ~380 ms per block.
func WindowToBlockCount(window string) int {
	const avgBlockMs = 380
	switch window {
	case "1h":
		return 3_600_000 / avgBlockMs
	case "7d":
		return 7 * 86_400_000 / avgBlockMs
	default: // "24h"
		return 86_400_000 / avgBlockMs
	}
}

// WindowBlockFilter returns a PocketBase filter + params scoped to the last N blocks.
func WindowBlockFilter(app core.App, window string) (string, map[string]any) {
	blockCount := WindowToBlockCount(window)
	latest, _ := app.FindRecordsByFilter("block_stats", "", "-block_number", 1, 0)
	fromBlock := 0
	if len(latest) > 0 {
		fromBlock = latest[0].GetInt("block_number") - blockCount
		if fromBlock < 0 {
			fromBlock = 0
		}
	}
	return "block_number >= {:from}", map[string]any{"from": fromBlock}
}

// LoadFeeColumn returns avg_fee values for all block_stats rows in window.
func LoadFeeColumn(app core.App, fromBlock any) []float64 {
	type row struct {
		V float64 `db:"v"`
	}
	var rows []row
	_ = app.DB().NewQuery(
		`SELECT avg_fee_num AS v FROM block_stats WHERE block_number >= {:from}`).
		Bind(dbx.Params{"from": fromBlock}).All(&rows)
	out := make([]float64, len(rows))
	for i, r := range rows {
		out[i] = r.V
	}
	return out
}

// PercentileFloat computes the p-th percentile from a sorted float64 slice.
func PercentileFloat(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	idx := int(p / 100.0 * float64(n))
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}

// DomainNames maps CCTP domain IDs to human-readable chain names.
var DomainNames = map[int]string{
	0:  "Ethereum",
	1:  "Avalanche",
	2:  "OP Mainnet",
	3:  "Arbitrum",
	5:  "Solana",
	6:  "Base",
	7:  "Polygon PoS",
	10: "Unichain",
	11: "Linea",
	12: "Codex",
	13: "Sonic",
	14: "World Chain",
	15: "Monad",
	16: "Sei",
	17: "BNB Smart Chain",
	18: "XDC",
	19: "HyperEVM",
	21: "Ink",
	22: "Plume",
	25: "Starknet",
	26: "Arc Testnet",
	27: "Stellar",
	28: "EDGE",
	29: "Injective",
	30: "Morph",
	31: "Pharos",
}

// DomainName returns the human-readable name for a CCTP domain ID.
func DomainName(id int) string {
	if n, ok := DomainNames[id]; ok {
		return n
	}
	return strconv.Itoa(id)
}
