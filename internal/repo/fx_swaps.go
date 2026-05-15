package repo

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// FxSwapFilter holds optional filter criteria for FX swaps.
type FxSwapFilter struct {
	Status  string
	Maker   string
	Taker   string
	QuoteID string
}

// ListFxSwaps returns FX swaps matching the filter.
func ListFxSwaps(app core.App, f FxSwapFilter, limit, offset int) ([]*core.Record, error) {
	parts := []string{}
	params := map[string]any{}
	if f.Status != "" {
		parts = append(parts, "status = {:s}")
		params["s"] = f.Status
	}
	if f.Maker != "" {
		parts = append(parts, "maker = {:m}")
		params["m"] = f.Maker
	}
	if f.Taker != "" {
		parts = append(parts, "taker = {:t}")
		params["t"] = f.Taker
	}
	if f.QuoteID != "" {
		parts = append(parts, "quote_id = {:q}")
		params["q"] = f.QuoteID
	}
	filter := strings.Join(parts, " && ")
	return FindRecords(app, "fx_swaps", filter, "-block_number", limit, offset, params)
}
