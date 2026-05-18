package repo

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// TransferFilter holds optional filter criteria for transfers.
type TransferFilter struct {
	BlockNumber int64
	TokenSymbol string
	FromAddr    string
	ToAddr      string
	TokenAddr   string
	TxHash      string
}

// ListTransfers returns transfers matching the given filter.
func ListTransfers(app core.App, f TransferFilter, limit, offset int) ([]*core.Record, error) {
	filter, params := buildTransferFilter(f)
	return FindRecords(app, "transfers", filter, "-block_number", limit, offset, params)
}

// CountTransfers returns the total number of transfers matching the filter.
func CountTransfers(app core.App, f TransferFilter) (int, error) {
	filter, params := buildTransferFilter(f)
	return CountWithFilter(app, "transfers", filter, params)
}

// TransfersByTokenAddress returns transfers for a specific token.
func TransfersByTokenAddress(app core.App, tokenAddr string, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "transfers", "token_address = {:a}", "-block_number", limit, offset, map[string]any{"a": tokenAddr})
}

// TransfersByTxHash returns transfers for a specific transaction.
func TransfersByTxHash(app core.App, txHash string) ([]*core.Record, error) {
	return FindRecords(app, "transfers", "tx_hash = {:h}", "-block_number", 0, 0, map[string]any{"h": txHash})
}

// TransfersBySender returns transfers from an address.
func TransfersBySender(app core.App, addr string, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "transfers", "from_addr = {:a}", "-block_number", limit, offset, map[string]any{"a": addr})
}

// TransfersByReceiver returns transfers to an address.
func TransfersByReceiver(app core.App, addr string, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "transfers", "to_addr = {:a}", "-block_number", limit, offset, map[string]any{"a": addr})
}

func buildTransferFilter(f TransferFilter) (string, map[string]any) {
	parts := []string{}
	params := map[string]any{}
	if f.BlockNumber != 0 {
		parts = append(parts, "block_number = {:bn}")
		params["bn"] = f.BlockNumber
	}
	if f.TokenSymbol != "" {
		parts = append(parts, "token_symbol = {:sym}")
		params["sym"] = f.TokenSymbol
	}
	if f.FromAddr != "" {
		parts = append(parts, "from_addr = {:fa}")
		params["fa"] = f.FromAddr
	}
	if f.ToAddr != "" {
		parts = append(parts, "to_addr = {:ta}")
		params["ta"] = f.ToAddr
	}
	if f.TokenAddr != "" {
		parts = append(parts, "token_address = {:tka}")
		params["tka"] = f.TokenAddr
	}
	if f.TxHash != "" {
		parts = append(parts, "tx_hash = {:h}")
		params["h"] = f.TxHash
	}
	return strings.Join(parts, " && "), params
}
