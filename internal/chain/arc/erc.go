package arc

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

// TokenInfo describes an ERC-20/ERC-721/ERC-1155 token's display metadata.
type TokenInfo struct {
	Address      common.Address
	Symbol       string
	Name         string
	Decimals     uint8
	TokenType    string // "ERC-20", "ERC-721", "ERC-1155", or ""
	LookupFailed bool
}

// tokenInfoCache is an in-memory, process-wide cache of token metadata.
// evictionOrder tracks insertion order for FIFO eviction once the cache
// exceeds maxTokenCacheSize entries. The known stablecoins seeded by
// SeedKnownTokens are never evicted.
const maxTokenCacheSize = 5000

var (
	tokenInfoMu     sync.RWMutex
	tokenInfoCache  = map[common.Address]TokenInfo{}
	evictionOrder   []common.Address
	seededAddresses map[common.Address]struct{} // protected by tokenInfoMu
)

// rpcHTTPClient is a shared HTTP client with connection pooling for JSON-RPC calls.
var rpcHTTPClient = &http.Client{
	Timeout: 8 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 3,
		IdleConnTimeout:     60 * time.Second,
	},
}

// SeedKnownTokens primes the cache with the Arc stablecoins. Seeded entries
// are marked as non-evictable.
func SeedKnownTokens() {
	tokenInfoMu.Lock()
	defer tokenInfoMu.Unlock()
	seededAddresses = make(map[common.Address]struct{}, len(KnownTokens))
	for addr, sym := range KnownTokens {
		tokenInfoCache[addr] = TokenInfo{Address: addr, Symbol: sym, Decimals: 6, TokenType: "ERC-20"}
		seededAddresses[addr] = struct{}{}
	}
}

// LookupTokenInfo returns metadata for the given token address.
// Resolution order: in-memory cache → token_analytics collection → JSON-RPC fetch.
func LookupTokenInfo(app core.App, addr common.Address, firstSeenBlock uint64) TokenInfo {
	tokenInfoMu.RLock()
	if info, ok := tokenInfoCache[addr]; ok {
		tokenInfoMu.RUnlock()
		return info
	}
	tokenInfoMu.RUnlock()

	// Check token_analytics collection (single source of truth for token metadata).
	rows, _ := app.FindRecordsByFilter("token_analytics", "token_address = {:a}", "", 1, 0,
		map[string]any{"a": strings.ToLower(addr.Hex())})
	if len(rows) > 0 {
		r := rows[0]
		info := TokenInfo{
			Address:      addr,
			Symbol:       r.GetString("symbol"),
			Name:         r.GetString("name"),
			Decimals:     uint8(r.GetInt("decimals")),
			TokenType:    r.GetString("token_type"),
			LookupFailed: r.GetBool("lookup_failed"),
		}
		cacheTokenInfo(addr, info)
		return info
	}

	// Fetch from RPC (detects ERC-20, ERC-721, ERC-1155 automatically).
	info := fetchTokenInfoFromRPC(addr)
	info.Address = addr

	// Persist to token_analytics so future lookups skip RPC.
	c, err := app.FindCollectionByNameOrId("token_analytics")
	if err == nil {
		r := core.NewRecord(c)
		r.Set("token_address", strings.ToLower(addr.Hex()))
		r.Set("symbol", info.Symbol)
		r.Set("name", info.Name)
		r.Set("decimals", int(info.Decimals))
		r.Set("token_type", info.TokenType)
		r.Set("first_seen_block", firstSeenBlock)
		r.Set("lookup_failed", info.LookupFailed)
		_ = app.Save(r)
	}

	cacheTokenInfo(addr, info)
	return info
}

// cacheTokenInfo adds an entry to the in-memory cache and evicts the oldest
// non-seeded entry if the cache exceeds maxTokenCacheSize.
func cacheTokenInfo(addr common.Address, info TokenInfo) {
	tokenInfoMu.Lock()
	defer tokenInfoMu.Unlock()

	// If address is already cached, no need to add to eviction order.
	if _, exists := tokenInfoCache[addr]; !exists {
		evictionOrder = append(evictionOrder, addr)
	}
	tokenInfoCache[addr] = info

	// Evict oldest non-seeded entries until we're under the limit.
	for len(tokenInfoCache) > maxTokenCacheSize && len(evictionOrder) > 0 {
		oldest := evictionOrder[0]
		evictionOrder = evictionOrder[1:]
		if _, isSeeded := seededAddresses[oldest]; isSeeded {
			continue // never evict seeded stablecoins
		}
		delete(tokenInfoCache, oldest)
	}
}

// ── ERC-165 interface IDs ────────────────────────────────────────────────────

const (
	// supportsInterface(bytes4) selector
	sigSupportsInterface = "0x01ffc9a7"
	// ERC-721 metadata interface (does NOT include ERC-721Enumerable)
	erc721InterfaceID = "80ac58cd"
	// ERC-1155 core interface
	erc1155InterfaceID = "d9b67a26"
)

// ── RPC call helpers ─────────────────────────────────────────────────────────

const sigDecimals = "0x313ce567"
const sigSymbol = "0x95d89b41"
const sigName = "0x06fdde03"
const sigTotalSupply = "0x18160ddd"

// fetchTokenInfoFromRPC tries to classify a contract as ERC-20, ERC-721, or
// ERC-1155 using the following heuristic chain per RPC endpoint:
//
//  1. decimals() succeeds → ERC-20 (fetch symbol, name as well)
//  2. supportsInterface(ERC-721) → ERC-721
//  3. supportsInterface(ERC-1155) → ERC-1155
//  4. symbol() or name() work without decimals → assume ERC-721 (heuristic)
//
// Falls through to the next RPC on any transport error.
func fetchTokenInfoFromRPC(addr common.Address) TokenInfo {
	for _, rpcURL := range ArcRPCPool {
		// ── ERC-20 path ──
		dec, ok := callDecimals(rpcURL, addr)
		if ok {
			sym, _ := callSymbol(rpcURL, addr)
			nm, _ := callName(rpcURL, addr)
			return TokenInfo{Symbol: sym, Name: nm, Decimals: dec, TokenType: "ERC-20"}
		}

		// ── NFT detection via ERC-165 ──
		if callSupportsInterface(rpcURL, addr, erc721InterfaceID) {
			sym, _ := callSymbol(rpcURL, addr)
			nm, _ := callName(rpcURL, addr)
			return TokenInfo{Symbol: sym, Name: nm, TokenType: "ERC-721"}
		}
		if callSupportsInterface(rpcURL, addr, erc1155InterfaceID) {
			sym, _ := callSymbol(rpcURL, addr)
			nm, _ := callName(rpcURL, addr)
			return TokenInfo{Symbol: sym, Name: nm, TokenType: "ERC-1155"}
		}

		// ── Heuristic: no ERC-165 but symbol/name exist → likely ERC-721 ──
		sym, symOk := callSymbol(rpcURL, addr)
		nm, nmOk := callName(rpcURL, addr)
		if symOk || nmOk {
			return TokenInfo{Symbol: sym, Name: nm, TokenType: "ERC-721"}
		}
	}
	return TokenInfo{LookupFailed: true}
}

// FetchFullTokenInfo calls name(), symbol(), decimals(), totalSupply() on the
// contract and classifies it as ERC-20, ERC-721, or ERC-1155.
func FetchFullTokenInfo(addr common.Address) FullTokenInfo {
	for _, rpcURL := range ArcRPCPool {
		// ── ERC-20 path ──
		dec, ok := callDecimals(rpcURL, addr)
		if ok {
			sym, _ := callSymbol(rpcURL, addr)
			nm, _ := callName(rpcURL, addr)
			ts, _ := callTotalSupply(rpcURL, addr)
			return FullTokenInfo{
				Name: nm, Symbol: sym, Decimals: dec,
				TotalSupply: ts, TokenType: "ERC-20",
			}
		}

		// ── NFT detection ──
		if callSupportsInterface(rpcURL, addr, erc721InterfaceID) {
			sym, _ := callSymbol(rpcURL, addr)
			nm, _ := callName(rpcURL, addr)
			return FullTokenInfo{Name: nm, Symbol: sym, TokenType: "ERC-721"}
		}
		if callSupportsInterface(rpcURL, addr, erc1155InterfaceID) {
			sym, _ := callSymbol(rpcURL, addr)
			nm, _ := callName(rpcURL, addr)
			return FullTokenInfo{Name: nm, Symbol: sym, TokenType: "ERC-1155"}
		}

		// ── Heuristic fallback ──
		sym, symOk := callSymbol(rpcURL, addr)
		nm, nmOk := callName(rpcURL, addr)
		if symOk || nmOk {
			return FullTokenInfo{Name: nm, Symbol: sym, TokenType: "ERC-721"}
		}
	}
	return FullTokenInfo{LookupFailed: true}
}

// FullTokenInfo holds all ERC-20/721/1155 metadata retrieved from onchain calls.
type FullTokenInfo struct {
	Name         string
	Symbol       string
	Decimals     uint8
	TotalSupply  *big.Int
	TokenType    string // "ERC-20", "ERC-721", "ERC-1155"
	LookupFailed bool
}

// callSupportsInterface calls ERC-165 supportsInterface(bytes4) on the contract.
// interfaceIDHex is the 4-byte hex ID without "0x" prefix (e.g. "80ac58cd").
func callSupportsInterface(rpcURL string, addr common.Address, interfaceIDHex string) bool {
	// ABI: supportsInterface(bytes4) → bool
	// Calldata = selector (4 bytes) + padded arg (32 bytes)
	data := sigSupportsInterface + interfaceIDHex + "00000000000000000000000000000000000000000000000000000000"
	raw, err := ethCall(rpcURL, addr, data)
	if err != nil || len(raw) < 32 {
		return false
	}
	return raw[31] == 1
}

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

func ethCall(rpcURL string, addr common.Address, dataHex string) ([]byte, error) {
	body := fmt.Sprintf(
		`{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"to":%q,"data":%q},"latest"]}`,
		strings.ToLower(addr.Hex()), dataHex,
	)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := rpcHTTPClient.Do(req)
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
