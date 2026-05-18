package utils_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"arcadia/internal/utils"
)

// ── utils.WeiToUSDC ────────────────────────────────────────────────────────────────

func TestWeiToUSDC(t *testing.T) {
	tests := []struct {
		name string
		wei  *big.Int
		want string
	}{
		{"nil", nil, "0"},
		{"zero", big.NewInt(0), "0"},
		{"one_whole", new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), "1.00000000"},
		{"half", new(big.Int).Exp(big.NewInt(10), big.NewInt(17), nil), "0.10000000"},
		{"fractional", big.NewInt(1), "0.00000000"},
		{"large", func() *big.Int {
			v, _ := new(big.Int).SetString("1000000000000000000000", 10) // 1000 USDC
			return v
		}(), "1000.00000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.WeiToUSDC(tt.wei)
			if got != tt.want {
				t.Errorf("utils.WeiToUSDC(%v) = %q, want %q", tt.wei, got, tt.want)
			}
		})
	}
}

// ── utils.StablecoinHuman ───────────────────────────────────────────────────────────

func TestStablecoinHuman(t *testing.T) {
	tests := []struct {
		name string
		raw  *big.Int
		want string
	}{
		{"nil", nil, "0"},
		{"zero", big.NewInt(0), "0"},
		{"one_dollar", big.NewInt(1_000_000), "1.000000"},
		{"half_dollar", big.NewInt(500_000), "0.500000"},
		{"one_cent", big.NewInt(10_000), "0.010000"},
		{"smallest_unit", big.NewInt(1), "0.000001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.StablecoinHuman(tt.raw)
			if got != tt.want {
				t.Errorf("utils.StablecoinHuman(%v) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

// ── utils.TokenAmountHuman ──────────────────────────────────────────────────────────

func TestTokenAmountHuman(t *testing.T) {
	tests := []struct {
		name     string
		raw      *big.Int
		decimals uint8
		want     string
	}{
		{"nil", nil, 18, "0"},
		{"zero", big.NewInt(0), 18, "0"},
		{"18_decimals_one_whole", new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), 18, "1.00000000"},
		{"6_decimals_one_dollar", big.NewInt(1_000_000), 6, "1.000000"},
		{"2_decimals", big.NewInt(100), 2, "1.00"},
		{"0_decimals", big.NewInt(42), 0, "42"},
		// precision is capped at 8 even when decimals > 8
		{"18_decimals_precision_cap", big.NewInt(1), 18, "0.00000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.TokenAmountHuman(tt.raw, tt.decimals)
			if got != tt.want {
				t.Errorf("utils.TokenAmountHuman(%v, %d) = %q, want %q", tt.raw, tt.decimals, got, tt.want)
			}
		})
	}
}

// ── Float siblings ────────────────────────────────────────────────────────────

func TestWeiToUSDCFloat(t *testing.T) {
	if got := utils.WeiToUSDCFloat(nil); got != 0 {
		t.Errorf("nil: got %v, want 0", got)
	}
	if got := utils.WeiToUSDCFloat(big.NewInt(0)); got != 0 {
		t.Errorf("zero: got %v, want 0", got)
	}
	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	if got := utils.WeiToUSDCFloat(one); got != 1.0 {
		t.Errorf("1e18 wei: got %v, want 1.0", got)
	}
}

func TestStablecoinHumanFloat(t *testing.T) {
	if got := utils.StablecoinHumanFloat(nil); got != 0 {
		t.Errorf("nil: got %v, want 0", got)
	}
	if got := utils.StablecoinHumanFloat(big.NewInt(1_000_000)); got != 1.0 {
		t.Errorf("1_000_000: got %v, want 1.0", got)
	}
}

func TestTokenAmountHumanFloat(t *testing.T) {
	if got := utils.TokenAmountHumanFloat(nil, 6); got != 0 {
		t.Errorf("nil: got %v, want 0", got)
	}
	one := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	if got := utils.TokenAmountHumanFloat(one, 6); got != 1.0 {
		t.Errorf("1e6 / 6 decimals: got %v, want 1.0", got)
	}
}

// ── utils.AddressFromTopic ──────────────────────────────────────────────────────────

func TestAddressFromTopic(t *testing.T) {
	t.Run("nil_hash", func(t *testing.T) {
		if got := utils.AddressFromTopic(nil); got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("valid_hash", func(t *testing.T) {
		// Pad a known address into a 32-byte topic (12 zero bytes + 20-byte address).
		addr := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
		var h common.Hash
		copy(h[12:], addr.Bytes())
		got := utils.AddressFromTopic(&h)
		if !strings.EqualFold(got, addr.Hex()) {
			t.Errorf("got %q, want %q", got, addr.Hex())
		}
	})
}

// ── utils.AddressFromBytes32 ────────────────────────────────────────────────────────

func TestAddressFromBytes32(t *testing.T) {
	t.Run("too_short", func(t *testing.T) {
		if got := utils.AddressFromBytes32(make([]byte, 20)); got != "" {
			t.Errorf("got %q, want empty string", got)
		}
	})

	t.Run("valid_32_bytes", func(t *testing.T) {
		addr := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
		b := make([]byte, 32)
		copy(b[12:], addr.Bytes())
		got := utils.AddressFromBytes32(b)
		if !strings.EqualFold(got, addr.Hex()) {
			t.Errorf("got %q, want %q", got, addr.Hex())
		}
	})
}

// ── utils.ReadUint32 / utils.ReadBig ──────────────────────────────────────────────────────

func TestReadUint32(t *testing.T) {
	t.Run("too_short", func(t *testing.T) {
		if got := utils.ReadUint32(make([]byte, 10), 0); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("value_42", func(t *testing.T) {
		data := make([]byte, 32)
		data[31] = 42
		if got := utils.ReadUint32(data, 0); got != 42 {
			t.Errorf("got %d, want 42", got)
		}
	})
}

func TestReadBig(t *testing.T) {
	t.Run("too_short", func(t *testing.T) {
		got := utils.ReadBig(make([]byte, 10), 0)
		if got.Sign() != 0 {
			t.Errorf("got %v, want 0", got)
		}
	})

	t.Run("value_255", func(t *testing.T) {
		data := make([]byte, 32)
		data[31] = 0xff
		got := utils.ReadBig(data, 0)
		if got.Int64() != 255 {
			t.Errorf("got %v, want 255", got)
		}
	})
}
