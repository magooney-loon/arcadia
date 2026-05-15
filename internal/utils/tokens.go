package utils

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"
)

// TokenInfo describes an ERC-20 token's display metadata.
type TokenInfo struct {
	Address      common.Address
	Symbol       string
	Decimals     uint8
	LookupFailed bool
}

// tokenInfoCache is an in-memory, process-wide cache of token metadata.
var (
	tokenInfoMu    sync.RWMutex
	tokenInfoCache = map[common.Address]TokenInfo{}
)

// SeedKnownTokens primes the cache with the three Arc stablecoins.
func SeedKnownTokens() {
	tokenInfoMu.Lock()
	defer tokenInfoMu.Unlock()
	for addr, sym := range KnownTokens {
		tokenInfoCache[addr] = TokenInfo{Address: addr, Symbol: sym, Decimals: 6}
	}
}

// LookupTokenInfo returns metadata for the given token address.
// Resolution order: in-memory → DB cache (tokens collection) → JSON-RPC fetch.
func LookupTokenInfo(app core.App, addr common.Address, firstSeenBlock uint64) TokenInfo {
	tokenInfoMu.RLock()
	if info, ok := tokenInfoCache[addr]; ok {
		tokenInfoMu.RUnlock()
		return info
	}
	tokenInfoMu.RUnlock()

	rows, _ := app.FindRecordsByFilter("tokens", "address = {:a}", "", 1, 0, map[string]any{"a": addr.Hex()})
	if len(rows) > 0 {
		r := rows[0]
		info := TokenInfo{
			Address:      addr,
			Symbol:       r.GetString("symbol"),
			Decimals:     uint8(r.GetInt("decimals")),
			LookupFailed: r.GetBool("lookup_failed"),
		}
		tokenInfoMu.Lock()
		tokenInfoCache[addr] = info
		tokenInfoMu.Unlock()
		return info
	}

	info := fetchTokenInfoFromRPC(addr)
	info.Address = addr

	c, err := app.FindCollectionByNameOrId("tokens")
	if err == nil {
		r := core.NewRecord(c)
		r.Set("address", addr.Hex())
		r.Set("symbol", info.Symbol)
		r.Set("decimals", info.Decimals)
		r.Set("first_seen_block", firstSeenBlock)
		r.Set("lookup_failed", info.LookupFailed)
		_ = app.Save(r)
	}

	tokenInfoMu.Lock()
	tokenInfoCache[addr] = info
	tokenInfoMu.Unlock()
	return info
}

func fetchTokenInfoFromRPC(addr common.Address) TokenInfo {
	for _, rpcURL := range arcRPCPool {
		dec, ok := callDecimals(rpcURL, addr)
		if !ok {
			continue
		}
		sym, _ := callSymbol(rpcURL, addr)
		return TokenInfo{Symbol: sym, Decimals: dec}
	}
	return TokenInfo{LookupFailed: true}
}

const sigDecimals = "0x313ce567"
const sigSymbol = "0x95d89b41"

func callDecimals(rpcURL string, addr common.Address) (uint8, bool) {
	raw, err := ethCall(rpcURL, addr, sigDecimals)
	if err != nil || len(raw) == 0 {
		return 0, false
	}
	n := new(big.Int).SetBytes(raw)
	if !n.IsUint64() || n.Uint64() > 36 {
		return 0, false
	}
	return uint8(n.Uint64()), true
}

func callSymbol(rpcURL string, addr common.Address) (string, bool) {
	raw, err := ethCall(rpcURL, addr, sigSymbol)
	if err != nil || len(raw) == 0 {
		return "", false
	}
	s := decodeABIString(raw)
	if s == "" || len(s) > 32 {
		s = strings.TrimRight(string(bytes.Trim(raw, "\x00")), "\x00")
	}
	s = strings.TrimSpace(s)
	if len(s) > 32 {
		s = s[:32]
	}
	return s, s != ""
}

const sigName = "0x06fdde03"
const sigTotalSupply = "0x18160ddd"

func callName(rpcURL string, addr common.Address) (string, bool) {
	raw, err := ethCall(rpcURL, addr, sigName)
	if err != nil || len(raw) == 0 {
		return "", false
	}
	s := decodeABIString(raw)
	if s == "" || len(s) > 64 {
		s = strings.TrimRight(string(bytes.Trim(raw, "\x00")), "\x00")
	}
	s = strings.TrimSpace(s)
	if len(s) > 64 {
		s = s[:64]
	}
	return s, s != ""
}

func callTotalSupply(rpcURL string, addr common.Address) (*big.Int, bool) {
	raw, err := ethCall(rpcURL, addr, sigTotalSupply)
	if err != nil || len(raw) == 0 {
		return nil, false
	}
	n := new(big.Int).SetBytes(raw)
	return n, true
}

// FetchFullTokenInfo calls name(), symbol(), decimals(), totalSupply() on the contract.
func FetchFullTokenInfo(addr common.Address) FullTokenInfo {
	for _, rpcURL := range arcRPCPool {
		dec, ok := callDecimals(rpcURL, addr)
		if !ok {
			continue
		}
		sym, _ := callSymbol(rpcURL, addr)
		nm, _ := callName(rpcURL, addr)
		ts, _ := callTotalSupply(rpcURL, addr)
		return FullTokenInfo{
			Name:        nm,
			Symbol:      sym,
			Decimals:    dec,
			TotalSupply: ts,
		}
	}
	return FullTokenInfo{LookupFailed: true}
}

// FullTokenInfo holds all ERC-20 metadata retrieved from onchain calls.
type FullTokenInfo struct {
	Name         string
	Symbol       string
	Decimals     uint8
	TotalSupply  *big.Int
	LookupFailed bool
}

func ethCall(rpcURL string, addr common.Address, dataHex string) ([]byte, error) {
	body := fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"to":%q,"data":%q},"latest"]}`,
		strings.ToLower(addr.Hex()), dataHex,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var out struct {
		Result string          `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if len(out.Error) > 0 {
		return nil, fmt.Errorf("rpc error: %s", string(out.Error))
	}
	hexStr := strings.TrimPrefix(out.Result, "0x")
	if hexStr == "" {
		return nil, nil
	}
	return hex.DecodeString(hexStr)
}

// decodeABIString decodes the ABI-encoded `string` return value.
func decodeABIString(raw []byte) string {
	if len(raw) < 64 {
		return ""
	}
	length := new(big.Int).SetBytes(raw[32:64]).Uint64()
	if length == 0 || 64+length > uint64(len(raw)) {
		return ""
	}
	return string(raw[64 : 64+length])
}
