package utils_test

import (
	"testing"

	"arcadia/internal/utils"
)

// ── utils.WindowToBlockCount ────────────────────────────────────────────────────────

func TestWindowToBlockCount(t *testing.T) {
	const avgBlockMs = 380
	tests := []struct {
		window string
		want   int
	}{
		{"1h", 3_600_000 / avgBlockMs},
		{"24h", 86_400_000 / avgBlockMs},
		{"7d", 7 * 86_400_000 / avgBlockMs},
		{"unknown", 86_400_000 / avgBlockMs}, // defaults to 24h
	}
	for _, tt := range tests {
		t.Run(tt.window, func(t *testing.T) {
			got := utils.WindowToBlockCount(tt.window)
			if got != tt.want {
				t.Errorf("utils.WindowToBlockCount(%q) = %d, want %d", tt.window, got, tt.want)
			}
		})
	}
}

// ── utils.PercentileFloat ───────────────────────────────────────────────────────────

func TestPercentileFloat(t *testing.T) {
	tests := []struct {
		name   string
		sorted []float64
		p      float64
		want   float64
	}{
		{"empty", nil, 50, 0},
		{"single", []float64{7.0}, 50, 7.0},
		{"p0", []float64{1, 2, 3, 4, 5}, 0, 1},
		{"p50", []float64{1, 2, 3, 4, 5}, 50, 3},
		{"p100", []float64{1, 2, 3, 4, 5}, 100, 5},
		{"p25", []float64{10, 20, 30, 40}, 25, 20},
		{"p75", []float64{10, 20, 30, 40}, 75, 40},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.PercentileFloat(tt.sorted, tt.p)
			if got != tt.want {
				t.Errorf("utils.PercentileFloat(%v, %v) = %v, want %v", tt.sorted, tt.p, got, tt.want)
			}
		})
	}
}

// ── utils.DomainName ────────────────────────────────────────────────────────────────

func TestDomainName(t *testing.T) {
	tests := []struct {
		id   int
		want string
	}{
		{0, "Ethereum"},
		{6, "Base"},
		{26, "Arc Testnet"},
		{999, "999"}, // unknown falls back to the numeric string
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := utils.DomainName(tt.id)
			if got != tt.want {
				t.Errorf("utils.DomainName(%d) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}
