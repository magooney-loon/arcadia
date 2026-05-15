package indexer

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

func saveTransfer(app core.App, log *types.Log) (*big.Int, error) {
	if log.Topic1 == nil || log.Topic2 == nil || log.TransactionHash == nil || log.LogIndex == nil {
		return nil, nil
	}

	txHash := log.TransactionHash.Hex()
	logIdx := *log.LogIndex

	existing, err := app.FindRecordsByFilter("transfers",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": txHash, "i": logIdx})
	if err != nil {
		return nil, fmt.Errorf("find transfer %s/%d: %w", txHash, logIdx, err)
	}
	if len(existing) > 0 {
		return nil, nil
	}

	from := common.BytesToAddress(log.Topic1.Bytes()[12:])
	to := common.BytesToAddress(log.Topic2.Bytes()[12:])

	var amountRaw *big.Int
	if log.Data != nil && len(*log.Data) >= 32 {
		amountRaw = new(big.Int).SetBytes((*log.Data)[:32])
	} else {
		amountRaw = new(big.Int)
	}

	var firstSeenBlock uint64
	if log.BlockNumber != nil {
		firstSeenBlock = log.BlockNumber.Uint64()
	}
	info := utils.LookupTokenInfo(app, *log.Address, firstSeenBlock)

	isNFT := info.TokenType == "ERC-721" || info.TokenType == "ERC-1155"

	symbol := "OTHER"
	if s, ok := utils.KnownTokens[*log.Address]; ok {
		symbol = s
	}

	r := core.NewRecord(utils.MustCollection(app, "transfers"))
	r.Set("tx_hash", txHash)
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	r.Set("log_index", logIdx)
	r.Set("token_address", strings.ToLower(log.Address.Hex()))
	r.Set("token_symbol", symbol)
	r.Set("token_type", info.TokenType)
	r.Set("from_addr", from.Hex())
	r.Set("to_addr", to.Hex())
	r.Set("amount_raw", amountRaw.String())
	r.Set("decimals", info.Decimals)
	if info.Symbol != "" {
		r.Set("token_name", info.Symbol)
	}

	if isNFT {
		// For NFTs the value field is the token ID, not an amount — store as-is.
		// amount_human is not meaningful for NFT transfers.
	} else if !info.LookupFailed {
		r.Set("amount_human", utils.TokenAmountHuman(amountRaw, info.Decimals))
	}

	if err := app.Save(r); err != nil {
		return nil, fmt.Errorf("save transfer %s/%d: %w", txHash, logIdx, err)
	}

	// Only create wallet edges and return an aggregatable amount for fungible tokens.
	if isNFT {
		return nil, nil
	}

	if _, isStable := utils.KnownTokens[*log.Address]; isStable {
		if err := upsertWalletEdge(app, from.Hex(), to.Hex(), amountRaw, log.BlockNumber); err != nil {
			return nil, err
		}
	}

	return amountRaw, nil
}

func upsertWalletEdge(app core.App, from, to string, amount *big.Int, blockNumber *big.Int) error {
	existing, err := app.FindRecordsByFilter("wallet_edges",
		"from_wallet = {:f} && to_wallet = {:t}", "", 1, 0,
		map[string]any{"f": from, "t": to})
	if err != nil {
		return fmt.Errorf("find wallet edge %s -> %s: %w", from, to, err)
	}

	var r *core.Record
	if len(existing) > 0 {
		r = existing[0]
		prevTotal, _ := new(big.Int).SetString(r.GetString("total_usdc"), 10)
		if prevTotal == nil {
			prevTotal = new(big.Int)
		}
		newTotal := new(big.Int).Add(prevTotal, amount)
		r.Set("total_usdc", newTotal.String())
		r.Set("tx_count", r.GetInt("tx_count")+1)
		if blockNumber != nil {
			r.Set("last_seen_block", blockNumber.Uint64())
		}
	} else {
		r = core.NewRecord(utils.MustCollection(app, "wallet_edges"))
		r.Set("from_wallet", from)
		r.Set("to_wallet", to)
		r.Set("total_usdc", amount.String())
		r.Set("tx_count", 1)
		if blockNumber != nil {
			r.Set("last_seen_block", blockNumber.Uint64())
		}
	}

	if err := app.Save(r); err != nil {
		return fmt.Errorf("save wallet edge %s -> %s: %w", from, to, err)
	}
	return nil
}
