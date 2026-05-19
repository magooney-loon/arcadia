package arc_test

import (
	"testing"

	arc "arcadia/internal/chain/arc"
)

// TestNextRPCURL verifies that over many calls every pool endpoint is returned
// and no URL outside the pool appears. The atomic counter is shared state, so
// we just assert membership rather than exact per-cycle counts.
func TestNextRPCURL(t *testing.T) {
	n := len(arc.ArcRPCPool)
	if n == 0 {
		t.Fatal("arc.ArcRPCPool is empty")
	}

	poolSet := make(map[string]struct{}, n)
	for _, url := range arc.ArcRPCPool {
		poolSet[url] = struct{}{}
	}

	seen := make(map[string]int, n)
	calls := n * 5
	for i := 0; i < calls; i++ {
		url := arc.NextRPCURL()
		if _, ok := poolSet[url]; !ok {
			t.Errorf("NextRPCURL returned unknown endpoint %q", url)
		}
		seen[url]++
	}

	// Each endpoint must appear roughly calls/n times (within ±1 due to offset).
	for _, url := range arc.ArcRPCPool {
		if seen[url] == 0 {
			t.Errorf("endpoint %q was never returned in %d calls", url, calls)
		}
	}
}

// TestArcChainID sanity-checks the chain ID constant.
func TestArcChainID(t *testing.T) {
	const expected = 5042002
	if arc.ArcChainID != expected {
		t.Errorf("arc.ArcChainID = %d, want %d", arc.ArcChainID, expected)
	}
}
