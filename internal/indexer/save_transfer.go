package indexer

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/chain"
	"arcadia/internal/rpc"
	"arcadia/internal/utils"
)

func saveTransfer(app core.App, log *types.Log, seen *batchSeen, edges map[edgeKey]*edgeDelta) (*big.Int, error) {
	if log.Topic1 == nil || log.Topic2 == nil || log.TransactionHash == nil || log.LogIndex == nil {
		return nil, nil
	}

	txHash := log.TransactionHash.Hex()
	logIdx := *log.LogIndex

	if _, dup := seen.transfers[txLogKey{txHash, logIdx}]; dup {
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
	info := rpc.LookupTokenInfo(app, *log.Address, firstSeenBlock)

	isNFT := info.TokenType == "ERC-721" || info.TokenType == "ERC-1155"

	symbol := "OTHER"
	if s, ok := chain.KnownTokens[*log.Address]; ok {
		symbol = s
	}

	coll, err := utils.FindCollection(app, "transfers")
	if err != nil {
		return nil, err
	}
	r := core.NewRecord(coll)
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
		r.Set("amount_num", utils.TokenAmountHumanFloat(amountRaw, info.Decimals))
	}

	if err := app.Save(r); err != nil {
		return nil, fmt.Errorf("save transfer %s/%d: %w", txHash, logIdx, err)
	}
	seen.transfers[txLogKey{txHash, logIdx}] = struct{}{}

	// Only accumulate wallet edges for fungible stablecoins.
	if isNFT {
		return nil, nil
	}

	if _, isStable := chain.KnownTokens[*log.Address]; isStable {
		key := edgeKey{from.Hex(), to.Hex()}
		d, ok := edges[key]
		if !ok {
			d = &edgeDelta{total: new(big.Int)}
			edges[key] = d
		}
		d.total.Add(d.total, amountRaw)
		d.count++
		if log.BlockNumber != nil {
			if bn := log.BlockNumber.Uint64(); bn > d.lastSeen {
				d.lastSeen = bn
			}
		}
		if _, isAgent := seen.agents[key.from]; isAgent {
			d.fromIsAgent = true
		}
		if _, isAgent := seen.agents[key.to]; isAgent {
			d.toIsAgent = true
		}
	}

	return amountRaw, nil
}
