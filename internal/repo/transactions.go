package repo

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// TransactionFilter holds optional filter criteria for transactions.
type TransactionFilter struct {
	BlockNumber int64
	FromAddr    string
	ToAddr      string
	Hash        string
}

// ListTransactions returns transactions matching the given filter, sorted by block number descending.
func ListTransactions(app core.App, f TransactionFilter, limit, offset int) ([]*core.Record, error) {
	filter, params := buildTxFilter(f)
	return FindRecords(app, "transactions", filter, "-block_number", limit, offset, params)
}

// CountTransactions returns the total number of transactions matching the filter.
func CountTransactions(app core.App, f TransactionFilter) (int, error) {
	filter, params := buildTxFilter(f)
	return CountWithFilter(app, "transactions", filter, params)
}

// TransactionByHash returns the transaction with the given hash, or nil if not found.
func TransactionByHash(app core.App, hash string) (*core.Record, error) {
	return LatestRecord(app, "transactions", "hash = {:h}", "", map[string]any{"h": hash})
}

// TransactionsByBlock returns all transactions for a given block number.
func TransactionsByBlock(app core.App, blockNumber int64) ([]*core.Record, error) {
	return FindRecords(app, "transactions", "block_number = {:n}", "", 0, 0, map[string]any{"n": blockNumber})
}

// TransactionsBySender returns transactions from an address, sorted by block number descending.
func TransactionsBySender(app core.App, addr string, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "transactions", "from_addr = {:a}", "-block_number", limit, offset, map[string]any{"a": addr})
}

// TransactionsByReceiver returns transactions to an address, sorted by block number descending.
func TransactionsByReceiver(app core.App, addr string, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "transactions", "to_addr = {:a}", "-block_number", limit, offset, map[string]any{"a": addr})
}

func buildTxFilter(f TransactionFilter) (string, map[string]any) {
	parts := []string{}
	params := map[string]any{}
	if f.Hash != "" {
		parts = append(parts, "hash = {:hash}")
		params["hash"] = f.Hash
	}
	if f.BlockNumber != 0 {
		parts = append(parts, "block_number = {:bn}")
		params["bn"] = f.BlockNumber
	}
	if f.FromAddr != "" {
		parts = append(parts, "from_addr = {:fa}")
		params["fa"] = f.FromAddr
	}
	if f.ToAddr != "" {
		parts = append(parts, "to_addr = {:ta}")
		params["ta"] = f.ToAddr
	}
	return strings.Join(parts, " && "), params
}
