package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	hypersyncgo "github.com/enviodev/hypersync-client-go"
	"github.com/enviodev/hypersync-client-go/options"
	"github.com/enviodev/hypersync-client-go/types"
	"github.com/enviodev/hypersync-client-go/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"
)

// Arc Testnet — HyperSync endpoint and internal network ID (just a lookup key, not Arc's chain ID).
const (
	arcEndpoint    = "https://arc-testnet.hypersync.xyz"
	arcRPCEndpoint = "https://rpc.arc.network" // confirm from docs.arc.network/arc/references/connect-to-arc.md
)

var arcNetworkID = utils.NetworkID(99999)

// ── Contract addresses ────────────────────────────────────────────────────────

var (
	addrUSDC = common.HexToAddress("0x3600000000000000000000000000000000000000")
	addrEURC = common.HexToAddress("0x89B50855Aa3bE2F677cD6303Cec089B5F319D72a")
	addrUSYC = common.HexToAddress("0xe9185F0c5F296Ed1797AaE4238D26CCaBEadb86C")

	addrCCTPMessenger     = common.HexToAddress("0x8FE6B999Dc680CcFDD5Bf7EB0974218be2542DAA")
	addrCCTPTransmitter   = common.HexToAddress("0xE737e5cEBEEBa77EFE34D4aa090756590b1CE275")
	addrGatewayWallet     = common.HexToAddress("0x0077777d7EBA4688BDeF3E311b846F25870A19B9")
	addrGatewayMinter     = common.HexToAddress("0x0022222ABE238Cc2C7Bb1f21003F0a260052475B")
	addrFxEscrow          = common.HexToAddress("0x867650F5eAe8df91445971f14d89fd84F0C9a9f8")
)

// ── Event topics ──────────────────────────────────────────────────────────────

var (
	topicTransfer       = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	topicDepositForBurn = common.HexToHash("0x2fa9ca894982930190727e75500a97d8dc500233a5065e0f3126c48fbe0343c0")
	topicMsgReceived    = common.HexToHash("0x58200b4c34ae05ee816d710053fff3ad1bcea173d0113462f6fd5162ab9adca5")
)

// knownTokens maps address → symbol for the three Arc stablecoins.
var knownTokens = map[common.Address]string{
	addrUSDC: "USDC",
	addrEURC: "EURC",
	addrUSYC: "USYC",
}

// ── Indexer entry point ───────────────────────────────────────────────────────

func StartIndexer(app core.App) {
	app.Logger().Info("Starting Arcadia HyperSync indexer")
	go func() {
		for {
			if err := runIndexer(app); err != nil {
				app.Logger().Error("Indexer crashed, restarting in 5s", "error", err)
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

func runIndexer(app core.App) error {
	ctx := context.Background()

	hyper, err := hypersyncgo.NewHyper(ctx, options.Options{
		Blockchains: []options.Node{
			{
				Type:        utils.EthereumNetwork,
				NetworkId:   arcNetworkID,
				Endpoint:    arcEndpoint,
				RpcEndpoint: arcRPCEndpoint,
				ApiToken:    os.Getenv("ENVIO_API_TOKEN"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create HyperSync client: %w", err)
	}

	client, ok := hyper.GetClient(arcNetworkID)
	if !ok {
		return fmt.Errorf("arc client not found in hyper")
	}

	fromBlock := getLastIndexedBlock(app)
	app.Logger().Info("Resuming indexer", "from_block", fromBlock)

	query := &types.Query{
		FromBlock:        new(big.Int).SetUint64(fromBlock),
		IncludeAllBlocks: true,
		FieldSelection: types.FieldSelection{
			Block: []string{
				"number", "hash", "parent_hash", "timestamp",
				"gas_used", "gas_limit", "base_fee_per_gas", "miner", "size",
			},
			Transaction: []string{
				"hash", "block_number", "transaction_index",
				"from", "to", "value", "nonce", "input",
				"gas_price", "gas_used", "effective_gas_price",
				"type", "contract_address",
			},
			Log: []string{
				"block_number", "transaction_hash", "log_index",
				"address", "topic0", "topic1", "topic2", "topic3", "data",
			},
		},
		Logs: []types.LogSelection{
			// All ERC-20 Transfer events (USDC, EURC, USYC, …)
			{Topics: [][]common.Hash{{topicTransfer}}},
			// CCTP: DepositForBurn on TokenMessengerV2
			{
				Address: []common.Address{addrCCTPMessenger},
				Topics:  [][]common.Hash{{topicDepositForBurn}},
			},
			// CCTP: MessageReceived on MessageTransmitterV2
			{
				Address: []common.Address{addrCCTPTransmitter},
				Topics:  [][]common.Hash{{topicMsgReceived}},
			},
			// Gateway: all events from GatewayWallet + GatewayMinter
			{Address: []common.Address{addrGatewayWallet, addrGatewayMinter}},
			// StableFX: all events from FxEscrow
			{Address: []common.Address{addrFxEscrow}},
		},
	}

	stream, err := client.Stream(ctx, query, options.DefaultStreamOptions())
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	if err := stream.Subscribe(); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer stream.Unsubscribe()

	for {
		select {
		case err := <-stream.Err():
			return fmt.Errorf("stream error: %w", err)

		case res := <-stream.Channel():
			if err := processBatch(app, res); err != nil {
				app.Logger().Error("Batch processing error", "error", err)
			} else if res.NextBlock != nil {
				setLastIndexedBlock(app, res.NextBlock.Uint64())
				app.Logger().Info("Batch indexed", "next_block", res.NextBlock.Uint64(),
					"blocks", len(res.Data.Blocks),
					"txs", len(res.Data.Transactions),
					"logs", len(res.Data.Logs),
				)
			}
			stream.Ack()

		case <-stream.Done():
			app.Logger().Info("Stream complete, reconnecting")
			return nil // outer loop restarts
		}
	}
}

// ── Batch processing ──────────────────────────────────────────────────────────

func processBatch(app core.App, res *types.QueryResponse) error {
	// aggregate per-block stats within this batch
	type blockAcc struct {
		txCount      int
		uniqueSenders   map[string]struct{}
		uniqueReceivers map[string]struct{}
		newContracts    int
		totalFee        *big.Int
		totalUSDC       *big.Int
		totalEURC       *big.Int
		totalUSYC       *big.Int
		largestUSDC     *big.Int
	}
	perBlock := make(map[uint64]*blockAcc)

	getAcc := func(blockNum uint64) *blockAcc {
		if _, ok := perBlock[blockNum]; !ok {
			perBlock[blockNum] = &blockAcc{
				uniqueSenders:   make(map[string]struct{}),
				uniqueReceivers: make(map[string]struct{}),
				totalFee:        new(big.Int),
				totalUSDC:       new(big.Int),
				totalEURC:       new(big.Int),
				totalUSYC:       new(big.Int),
				largestUSDC:     new(big.Int),
			}
		}
		return perBlock[blockNum]
	}

	return app.RunInTransaction(func(txApp core.App) error {
		// 1. Blocks
		for _, blk := range res.Data.Blocks {
			if blk.Number == nil {
				continue
			}
			saveBlock(txApp, &blk)
			getAcc(blk.Number.Uint64()) // ensure acc exists even for empty blocks
		}

		// 2. Transactions
		for _, tx := range res.Data.Transactions {
			if tx.Hash == nil || tx.BlockNumber == nil {
				continue
			}
			fee := saveTransaction(txApp, &tx)
			bn := tx.BlockNumber.Uint64()
			acc := getAcc(bn)
			acc.txCount++
			if tx.From != nil {
				acc.uniqueSenders[tx.From.Hex()] = struct{}{}
			}
			if tx.To != nil {
				acc.uniqueReceivers[tx.To.Hex()] = struct{}{}
			}
			if tx.ContractAddress != nil {
				acc.newContracts++
			}
			if fee != nil {
				acc.totalFee.Add(acc.totalFee, fee)
			}
		}

		// 3. Logs → transfers / crosschain / fx
		for _, log := range res.Data.Logs {
			if log.BlockNumber == nil {
				continue
			}
			bn := log.BlockNumber.Uint64()
			acc := getAcc(bn)
			amount := routeLog(txApp, &log)
			if amount != nil && log.Address != nil {
				switch *log.Address {
				case addrUSDC:
					acc.totalUSDC.Add(acc.totalUSDC, amount)
					if amount.Cmp(acc.largestUSDC) > 0 {
						acc.largestUSDC.Set(amount)
					}
				case addrEURC:
					acc.totalEURC.Add(acc.totalEURC, amount)
				case addrUSYC:
					acc.totalUSYC.Add(acc.totalUSYC, amount)
				}
			}
		}

		// 4. Block stats — one upsert per block in this batch
		for _, blk := range res.Data.Blocks {
			if blk.Number == nil || blk.Timestamp == nil {
				continue
			}
			bn := blk.Number.Uint64()
			acc := getAcc(bn)

			avgFee := new(big.Int)
			if acc.txCount > 0 && acc.totalFee.Sign() > 0 {
				avgFee.Div(acc.totalFee, big.NewInt(int64(acc.txCount)))
			}

			gasUsed := uint64(0)
			gasLimit := uint64(1)
			if blk.GasUsed != nil {
				gasUsed = *blk.GasUsed
			}
			if blk.GasLimit != nil && *blk.GasLimit > 0 {
				gasLimit = *blk.GasLimit
			}
			utilPct := float64(gasUsed) / float64(gasLimit) * 100

			stats := core.NewRecord(mustCollection(txApp, "block_stats"))
			stats.Set("block_number", bn)
			stats.Set("timestamp", blk.Timestamp.Unix())
			stats.Set("tx_count", acc.txCount)
			stats.Set("avg_fee_usdc", weiToUSDC(avgFee))
			stats.Set("total_fee_usdc", weiToUSDC(acc.totalFee))
			stats.Set("total_usdc_transferred", stablecoinHuman(acc.totalUSDC))
			stats.Set("total_eurc_transferred", stablecoinHuman(acc.totalEURC))
			stats.Set("total_usyc_transferred", stablecoinHuman(acc.totalUSYC))
			stats.Set("unique_senders", len(acc.uniqueSenders))
			stats.Set("unique_receivers", len(acc.uniqueReceivers))
			stats.Set("new_contracts", acc.newContracts)
			stats.Set("largest_usdc_transfer", stablecoinHuman(acc.largestUSDC))
			stats.Set("utilization_pct", utilPct)

			if err := txApp.Save(stats); err != nil {
				txApp.Logger().Error("Failed to save block_stats", "block", bn, "error", err)
			}
		}

		return nil
	})
}

// ── Individual record savers ──────────────────────────────────────────────────

func saveBlock(app core.App, blk *types.Block) {
	// skip if already exists
	existing, _ := app.FindRecordsByFilter("blocks", "number = {:n}", "", 1, 0, map[string]any{"n": blk.Number.Uint64()})
	if len(existing) > 0 {
		return
	}

	r := core.NewRecord(mustCollection(app, "blocks"))
	r.Set("number", blk.Number.Uint64())
	if blk.Hash != nil {
		r.Set("hash", blk.Hash.Hex())
	}
	if blk.ParentHash != nil {
		r.Set("parent_hash", blk.ParentHash.Hex())
	}
	if blk.Miner != nil {
		r.Set("miner", blk.Miner.Hex())
	}
	if blk.Timestamp != nil {
		r.Set("timestamp", blk.Timestamp.Unix())
	}
	if blk.GasUsed != nil {
		r.Set("gas_used", *blk.GasUsed)
	}
	if blk.GasLimit != nil {
		r.Set("gas_limit", *blk.GasLimit)
		if blk.GasUsed != nil && *blk.GasLimit > 0 {
			r.Set("utilization_pct", float64(*blk.GasUsed)/float64(*blk.GasLimit)*100)
		}
	}
	if blk.BaseFeePerGas != nil {
		r.Set("base_fee_per_gas", blk.BaseFeePerGas.String())
	}
	if blk.Size != nil {
		r.Set("size", *blk.Size)
	}

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save block", "number", blk.Number.Uint64(), "error", err)
	}
}

// saveTransaction saves the transaction and returns the fee in wei (for stats accumulation).
func saveTransaction(app core.App, tx *types.Transaction) *big.Int {
	existing, _ := app.FindRecordsByFilter("transactions", "hash = {:h}", "", 1, 0, map[string]any{"h": tx.Hash.Hex()})
	if len(existing) > 0 {
		return nil
	}

	r := core.NewRecord(mustCollection(app, "transactions"))
	r.Set("hash", tx.Hash.Hex())
	if tx.BlockNumber != nil {
		r.Set("block_number", tx.BlockNumber.Uint64())
	}
	if tx.TransactionIndex != nil {
		r.Set("transaction_index", *tx.TransactionIndex)
	}
	if tx.From != nil {
		r.Set("from_addr", tx.From.Hex())
	}
	if tx.To != nil {
		r.Set("to_addr", tx.To.Hex())
	}
	if tx.Value != nil {
		r.Set("value", tx.Value.String())
	}
	if tx.Nonce != nil {
		r.Set("nonce", *tx.Nonce)
	}
	if tx.Input != nil && len(*tx.Input) >= 4 {
		r.Set("sighash", fmt.Sprintf("0x%x", (*tx.Input)[:4]))
	}
	if tx.GasPrice != nil {
		r.Set("gas_price", tx.GasPrice.String())
	}
	if tx.GasUsed != nil {
		r.Set("gas_used", *tx.GasUsed)
	}

	var feeWei *big.Int
	if tx.GasUsed != nil && tx.EffectiveGasPrice != nil {
		feeWei = new(big.Int).Mul(new(big.Int).SetUint64(*tx.GasUsed), tx.EffectiveGasPrice)
		r.Set("effective_gas_price", tx.EffectiveGasPrice.String())
		r.Set("fee_usdc", weiToUSDC(feeWei))
	}

	if tx.Kind != nil {
		r.Set("tx_type", *tx.Kind)
	}
	isDeployment := tx.ContractAddress != nil
	if isDeployment {
		r.Set("contract_address", tx.ContractAddress.Hex())
	}
	r.Set("is_contract_deploy", isDeployment)

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save transaction", "hash", tx.Hash.Hex(), "error", err)
	}
	return feeWei
}

// routeLog decodes a log and routes it to the right handler.
// Returns the transfer amount (in raw uint256) if this is an ERC-20 Transfer, else nil.
func routeLog(app core.App, log *types.Log) *big.Int {
	if log.Topic0 == nil || log.Address == nil {
		return nil
	}

	switch *log.Topic0 {
	case topicTransfer:
		return saveTransfer(app, log)

	case topicDepositForBurn:
		saveCCTPEvent(app, log, "cctp", "burn")

	case topicMsgReceived:
		saveCCTPEvent(app, log, "cctp", "mint")

	default:
		addr := *log.Address
		if addr == addrGatewayWallet || addr == addrGatewayMinter {
			saveGatewayEvent(app, log)
		} else if addr == addrFxEscrow {
			saveFxEvent(app, log)
		}
	}
	return nil
}

func saveTransfer(app core.App, log *types.Log) *big.Int {
	if log.Topic1 == nil || log.Topic2 == nil || log.TransactionHash == nil || log.LogIndex == nil {
		return nil
	}

	txHash := log.TransactionHash.Hex()
	logIdx := *log.LogIndex

	existing, _ := app.FindRecordsByFilter("transfers",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": txHash, "i": logIdx})
	if len(existing) > 0 {
		return nil
	}

	from := common.BytesToAddress(log.Topic1.Bytes()[12:])
	to := common.BytesToAddress(log.Topic2.Bytes()[12:])

	var amountRaw *big.Int
	if log.Data != nil && len(*log.Data) >= 32 {
		amountRaw = new(big.Int).SetBytes((*log.Data)[:32])
	} else {
		amountRaw = new(big.Int)
	}

	symbol := "OTHER"
	if s, ok := knownTokens[*log.Address]; ok {
		symbol = s
	}

	r := core.NewRecord(mustCollection(app, "transfers"))
	r.Set("tx_hash", txHash)
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	r.Set("log_index", logIdx)
	r.Set("token_address", log.Address.Hex())
	r.Set("token_symbol", symbol)
	r.Set("from_addr", from.Hex())
	r.Set("to_addr", to.Hex())
	r.Set("amount_raw", amountRaw.String())
	r.Set("amount_human", stablecoinHuman(amountRaw))

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save transfer", "tx", txHash, "error", err)
		return nil
	}

	// update wallet graph edge
	upsertWalletEdge(app, from.Hex(), to.Hex(), amountRaw, log.BlockNumber)

	return amountRaw
}

func saveCCTPEvent(app core.App, log *types.Log, protocol, eventType string) {
	if log.TransactionHash == nil || log.LogIndex == nil {
		return
	}
	existing, _ := app.FindRecordsByFilter("crosschain_events",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": log.TransactionHash.Hex(), "i": *log.LogIndex})
	if len(existing) > 0 {
		return
	}

	r := core.NewRecord(mustCollection(app, "crosschain_events"))
	r.Set("tx_hash", log.TransactionHash.Hex())
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	r.Set("log_index", *log.LogIndex)
	r.Set("protocol", protocol)
	r.Set("event_type", eventType)
	r.Set("destination_domain", 26) // Arc testnet domain

	// DepositForBurn: topic1 = nonce, topic2 = burnToken, data contains amount+recipient
	// Full ABI decode can be added once the exact ABI is confirmed.

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save cctp event", "error", err)
	}
}

func saveGatewayEvent(app core.App, log *types.Log) {
	if log.TransactionHash == nil || log.LogIndex == nil {
		return
	}
	existing, _ := app.FindRecordsByFilter("crosschain_events",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": log.TransactionHash.Hex(), "i": *log.LogIndex})
	if len(existing) > 0 {
		return
	}

	r := core.NewRecord(mustCollection(app, "crosschain_events"))
	r.Set("tx_hash", log.TransactionHash.Hex())
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	r.Set("log_index", *log.LogIndex)
	r.Set("protocol", "gateway")
	r.Set("event_type", "deposit") // refine once Gateway ABI is confirmed

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save gateway event", "error", err)
	}
}

func saveFxEvent(app core.App, log *types.Log) {
	if log.TransactionHash == nil || log.LogIndex == nil {
		return
	}
	existing, _ := app.FindRecordsByFilter("fx_swaps",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": log.TransactionHash.Hex(), "i": *log.LogIndex})
	if len(existing) > 0 {
		return
	}

	r := core.NewRecord(mustCollection(app, "fx_swaps"))
	r.Set("tx_hash", log.TransactionHash.Hex())
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	r.Set("log_index", *log.LogIndex)
	r.Set("status", "created") // refine once FxEscrow ABI is confirmed

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to save fx event", "error", err)
	}
}

// upsertWalletEdge increments the edge (from→to) in the wallet graph.
func upsertWalletEdge(app core.App, from, to string, amount *big.Int, blockNumber *big.Int) {
	existing, _ := app.FindRecordsByFilter("wallet_edges",
		"from_wallet = {:f} && to_wallet = {:t}", "", 1, 0,
		map[string]any{"f": from, "t": to})

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
		r = core.NewRecord(mustCollection(app, "wallet_edges"))
		r.Set("from_wallet", from)
		r.Set("to_wallet", to)
		r.Set("total_usdc", amount.String())
		r.Set("tx_count", 1)
		if blockNumber != nil {
			r.Set("last_seen_block", blockNumber.Uint64())
		}
	}

	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to upsert wallet edge", "from", from, "to", to, "error", err)
	}
}

// ── Cursor management ─────────────────────────────────────────────────────────

func getLastIndexedBlock(app core.App) uint64 {
	records, err := app.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	if err != nil || len(records) == 0 {
		return 0
	}
	val, _ := strconv.ParseUint(records[0].GetString("value"), 10, 64)
	return val
}

func setLastIndexedBlock(app core.App, block uint64) {
	records, _ := app.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)

	var r *core.Record
	if len(records) > 0 {
		r = records[0]
	} else {
		c := mustCollection(app, "indexer_meta")
		r = core.NewRecord(c)
		r.Set("key", "lastBlock")
	}
	r.Set("value", strconv.FormatUint(block, 10))
	if err := app.Save(r); err != nil {
		app.Logger().Error("Failed to persist lastBlock cursor", "error", err)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// weiToUSDC converts a fee in native USDC wei (18 decimals) to a human-readable string.
func weiToUSDC(wei *big.Int) string {
	if wei == nil || wei.Sign() == 0 {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	quot := new(big.Float).Quo(new(big.Float).SetInt(wei), new(big.Float).SetInt(divisor))
	return quot.Text('f', 8)
}

// stablecoinHuman converts an ERC-20 stablecoin amount (6 decimals) to a human-readable string.
func stablecoinHuman(raw *big.Int) string {
	if raw == nil || raw.Sign() == 0 {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
	quot := new(big.Float).Quo(new(big.Float).SetInt(raw), new(big.Float).SetInt(divisor))
	return quot.Text('f', 6)
}

// mustCollection fetches a collection by name and panics if missing — collections are
// registered at startup so absence here is a programming error.
func mustCollection(app core.App, name string) *core.Collection {
	c, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		panic(fmt.Sprintf("collection %q not found: %v", name, err))
	}
	return c
}

// addressFromTopic extracts an Ethereum address from a 32-byte topic (last 20 bytes).
func addressFromTopic(h *common.Hash) string {
	if h == nil {
		return ""
	}
	return common.BytesToAddress(h.Bytes()[12:]).Hex()
}

// stripQuotes is a no-op helper kept for clarity when working with string values.
func stripQuotes(s string) string {
	return strings.Trim(s, `"`)
}
