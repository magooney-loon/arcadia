package repo

import (
	"github.com/pocketbase/pocketbase/core"
)

// EdgesByFromWallet returns edges originating from a wallet, sorted by tx count descending.
func EdgesByFromWallet(app core.App, addr string, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "wallet_edges", "from_wallet = {:a}", "-tx_count", limit, offset, map[string]any{"a": addr})
}

// EdgesByToWallet returns edges going to a wallet, sorted by tx count descending.
func EdgesByToWallet(app core.App, addr string, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "wallet_edges", "to_wallet = {:a}", "-tx_count", limit, offset, map[string]any{"a": addr})
}

// EdgesByWallet returns edges where the wallet is either source or destination.
func EdgesByWallet(app core.App, addr string, limit, offset int) ([]*core.Record, error) {
	if addr == "" {
		return FindRecords(app, "wallet_edges", "", "-tx_count", limit, offset)
	}
	return FindRecords(app, "wallet_edges", "from_wallet = {:w} || to_wallet = {:w}", "-tx_count", limit, offset, map[string]any{"w": addr})
}
