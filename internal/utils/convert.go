package utils

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"
)

// WeiToUSDC converts a fee in native USDC wei (18 decimals) to a human-readable string.
func WeiToUSDC(wei *big.Int) string {
	if wei == nil || wei.Sign() == 0 {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	quot := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(divisor))
	return quot.Text('f', 8)
}

// StablecoinHuman converts an ERC-20 stablecoin amount (6 decimals) to a human-readable string.
func StablecoinHuman(raw *big.Int) string {
	if raw == nil || raw.Sign() == 0 {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	quot := new(big.Float).Quo(new(big.Float).SetInt(raw), new(big.Float).SetInt(divisor))
	return quot.Text('f', 6)
}

// TokenAmountHuman converts a raw uint256 amount to a human-readable string using
// the supplied decimals. Output precision is min(decimals, 8).
func TokenAmountHuman(raw *big.Int, decimals uint8) string {
	if raw == nil || raw.Sign() == 0 {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	quot := new(big.Float).Quo(new(big.Float).SetInt(raw), new(big.Float).SetInt(divisor))
	prec := int(decimals)
	if prec > 8 {
		prec = 8
	}
	return quot.Text('f', prec)
}

// MustCollection fetches a collection by name and panics if missing.
func MustCollection(app core.App, name string) *core.Collection {
	c, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		panic(fmt.Sprintf("collection %q not found: %v", name, err))
	}
	return c
}

// AddressFromTopic extracts an Ethereum address from a 32-byte topic (last 20 bytes).
func AddressFromTopic(h *common.Hash) string {
	if h == nil {
		return ""
	}
	return common.BytesToAddress(h.Bytes()[12:]).Hex()
}

// AddressFromBytes32 extracts an Ethereum address from a 32-byte ABI-padded slice (last 20 bytes).
func AddressFromBytes32(b []byte) string {
	if len(b) < 32 {
		return ""
	}
	return common.BytesToAddress(b[12:32]).Hex()
}

// ReadUint32 reads a uint32 from 32 ABI-padded bytes at offset in data.
func ReadUint32(data []byte, offset int) uint32 {
	if len(data) < offset+32 {
		return 0
	}
	return uint32(new(big.Int).SetBytes(data[offset : offset+32]).Uint64())
}

// ReadBig reads a *big.Int from 32 ABI bytes at offset in data.
func ReadBig(data []byte, offset int) *big.Int {
	if len(data) < offset+32 {
		return new(big.Int)
	}
	return new(big.Int).SetBytes(data[offset : offset+32])
}
