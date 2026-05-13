package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	hypersyncgo "github.com/enviodev/hypersync-client-go"
	"github.com/enviodev/hypersync-client-go/options"
	"github.com/enviodev/hypersync-client-go/types"
	"github.com/enviodev/hypersync-client-go/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"
)

// ── Indexer entry point ───────────────────────────────────────────────────────

func StartIndexer(app core.App) {
	const startupDelay = 10 * time.Second
	log.Printf("[indexer] scheduled Arcadia HyperSync indexer startup in %s", startupDelay)
	go func() {
		time.Sleep(startupDelay)
		log.Println("[indexer] starting Arcadia HyperSync indexer")
		attempt := 0
		for {
			attempt++
			log.Printf("[indexer] run attempt #%d", attempt)
			recordIndexerEvent(app, "info", "run_start", "starting indexer run", indexerEventFields{"attempt": attempt})
			if err := runIndexer(app, attempt); err != nil {
				msg := err.Error()
				if strings.Contains(msg, "429") {
					log.Printf("[indexer] rate-limited (429) — waiting 30s before retry (attempt #%d)", attempt)
					recordIndexerEvent(app, "warn", "rate_limited", "HyperSync returned 429; backing off before retry", indexerEventFields{"attempt": attempt, "error": err})
					time.Sleep(30 * time.Second)
				} else {
					log.Printf("[indexer] crashed: %v — restarting in 5s (attempt #%d)", err, attempt)
					recordIndexerEvent(app, "error", "run_error", "indexer run failed; restarting", indexerEventFields{"attempt": attempt, "error": err})
					time.Sleep(5 * time.Second)
				}
			}
		}
	}()
}

type indexerEventFields map[string]any

func recordIndexerEvent(app core.App, level, event, message string, fields indexerEventFields) {
	c, err := app.FindCollectionByNameOrId("indexer_events")
	if err != nil {
		return
	}

	r := core.NewRecord(c)
	r.Set("timestamp", time.Now().Unix())
	r.Set("level", level)
	r.Set("event", event)
	r.Set("message", message)
	for key, val := range fields {
		switch key {
		case "attempt", "batch", "block", "tip", "lag", "duration_ms", "blocks", "transactions", "logs", "error":
			if key == "error" {
				if val != nil {
					r.Set("error", fmt.Sprint(val))
				}
				continue
			}
			r.Set(key, val)
		}
	}
	if err := app.Save(r); err != nil {
		log.Printf("[indexer] failed to persist indexer event %q: %v", event, err)
	}
}

func getChainTip(ctx context.Context, client interface {
	GetHeight(context.Context) (*big.Int, error)
}) (uint64, error) {
	height, err := client.GetHeight(ctx)
	if err != nil {
		return 0, err
	}
	if height == nil {
		return 0, fmt.Errorf("chain height response was nil")
	}
	return height.Uint64(), nil
}

func logIndexerHeartbeat(ctx context.Context, app core.App, client interface {
	GetHeight(context.Context) (*big.Int, error)
}, attempt int, batchCount, currentBlock uint64, lastBatchAt time.Time, processingBatch uint64, processingStartedAt time.Time, persist bool) {
	idleFor := time.Since(lastBatchAt).Round(time.Second)
	tipCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	tip, tipErr := getChainTip(tipCtx, client)
	cancel()
	lag := uint64(0)
	if tip > currentBlock {
		lag = tip - currentBlock
	}

	processingFor := time.Duration(0)
	if processingBatch > 0 && !processingStartedAt.IsZero() {
		processingFor = time.Since(processingStartedAt).Round(time.Second)
	}

	if tipErr != nil {
		if processingBatch > 0 {
			log.Printf("[indexer] heartbeat | processing batch #%d for %s | block %d | completed_batches=%d | tip=? err=%v", processingBatch, processingFor, currentBlock, batchCount, tipErr)
		} else {
			log.Printf("[indexer] heartbeat | idle %s | block %d | completed_batches=%d | tip=? err=%v", idleFor, currentBlock, batchCount, tipErr)
		}
		if persist {
			recordIndexerEvent(app, "warn", "heartbeat", "indexer heartbeat tip check failed", indexerEventFields{"attempt": attempt, "batch": batchCount, "block": currentBlock, "error": tipErr})
		}
		return
	}

	if processingBatch > 0 {
		log.Printf("[indexer] heartbeat | processing batch #%d for %s | block %d | tip %d | lag %d | completed_batches=%d", processingBatch, processingFor, currentBlock, tip, lag, batchCount)
	} else {
		log.Printf("[indexer] heartbeat | idle %s | block %d | tip %d | lag %d | batches=%d", idleFor, currentBlock, tip, lag, batchCount)
	}
	if persist {
		recordIndexerEvent(app, "info", "heartbeat", "indexer heartbeat", indexerEventFields{"attempt": attempt, "batch": batchCount, "block": currentBlock, "tip": tip, "lag": lag})
	}
}

func newIndexerQuery(fromBlock, toBlock uint64) *types.Query {
	return &types.Query{
		FromBlock:        new(big.Int).SetUint64(fromBlock),
		ToBlock:          new(big.Int).SetUint64(toBlock),
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
				"gas", "cumulative_gas_used", "max_fee_per_gas",
				"max_priority_fee_per_gas", "type", "status", "contract_address",
			},
			Log: []string{
				"block_number", "transaction_hash", "log_index",
				"address", "topic0", "topic1", "topic2", "topic3", "data",
			},
			Trace: []string{
				"block_number", "transaction_hash",
				"from", "to", "value", "call_type", "type", "gas_used", "error",
			},
		},
		Transactions: []types.TransactionSelection{{}},
		Traces:       []types.TraceSelection{{}},
		Logs: []types.LogSelection{
			{Topics: [][]common.Hash{{TopicTransfer}}},
			// CCTP: DepositForBurn (USDC exits Arc) + MintAndWithdraw (USDC arrives on Arc)
			{
				Address: []common.Address{AddrCCTPTokenMessenger},
				Topics:  [][]common.Hash{{TopicDepositForBurn, TopicMintAndWithdraw}},
			},
			// CCTP: MessageReceived (low-level transport event, captures source domain + nonce)
			{
				Address: []common.Address{AddrCCTPMessageTransmitter},
				Topics:  [][]common.Hash{{TopicMessageReceived}},
			},
			// Gateway: Deposited + GatewayBurned on GatewayWallet; AttestationUsed on GatewayMinter
			{
				Address: []common.Address{AddrGatewayWallet},
				Topics:  [][]common.Hash{{TopicGatewayDeposited, TopicGatewayBurned}},
			},
			{
				Address: []common.Address{AddrGatewayMinter},
				Topics:  [][]common.Hash{{TopicAttestationUsed}},
			},
			{Address: []common.Address{AddrFxEscrow}},
			{Address: []common.Address{AddrAgentRegistry}, Topics: [][]common.Hash{{TopicAgentRegistered}}},
			// ERC-8183 job lifecycle: creation + all state transitions
			{
				Address: []common.Address{AddrAgenticCommerce},
				Topics: [][]common.Hash{{
					TopicJobCreated,
					TopicJobFunded,
					TopicJobSubmitted,
					TopicJobCompleted,
					TopicJobRejected,
					TopicPaymentReleased,
					TopicJobExpired,
				}},
			},
		},
	}
}

type jsonQueryResponse struct {
	ArchiveHeight      json.RawMessage      `json:"archive_height"`
	NextBlock          json.RawMessage      `json:"next_block"`
	TotalExecutionTime uint64               `json:"total_execution_time"`
	Data               []jsonDataChunk      `json:"data"`
	RollbackGuard      *types.RollbackGuard `json:"rollback_guard"`
}

type jsonDataChunk struct {
	Blocks       []jsonBlock       `json:"blocks"`
	Transactions []jsonTransaction `json:"transactions"`
	Logs         []jsonLog         `json:"logs"`
	Traces       []jsonTrace       `json:"traces"`
}

type jsonBlock struct {
	Number        json.RawMessage `json:"number"`
	Hash          string          `json:"hash"`
	ParentHash    string          `json:"parent_hash"`
	Timestamp     json.RawMessage `json:"timestamp"`
	GasUsed       json.RawMessage `json:"gas_used"`
	GasLimit      json.RawMessage `json:"gas_limit"`
	BaseFeePerGas json.RawMessage `json:"base_fee_per_gas"`
	Miner         string          `json:"miner"`
	Size          json.RawMessage `json:"size"`
}

type jsonTransaction struct {
	Hash                 string          `json:"hash"`
	BlockNumber          json.RawMessage `json:"block_number"`
	TransactionIndex     json.RawMessage `json:"transaction_index"`
	From                 string          `json:"from"`
	To                   string          `json:"to"`
	Value                json.RawMessage `json:"value"`
	Nonce                json.RawMessage `json:"nonce"`
	Input                string          `json:"input"`
	Gas                  json.RawMessage `json:"gas"`
	GasPrice             json.RawMessage `json:"gas_price"`
	GasUsed              json.RawMessage `json:"gas_used"`
	CumulativeGasUsed    json.RawMessage `json:"cumulative_gas_used"`
	EffectiveGasPrice    json.RawMessage `json:"effective_gas_price"`
	MaxFeePerGas         json.RawMessage `json:"max_fee_per_gas"`
	MaxPriorityFeePerGas json.RawMessage `json:"max_priority_fee_per_gas"`
	Kind                 json.RawMessage `json:"type"`
	Status               json.RawMessage `json:"status"`
	ContractAddress      string          `json:"contract_address"`
}

type jsonLog struct {
	BlockNumber     json.RawMessage `json:"block_number"`
	TransactionHash string          `json:"transaction_hash"`
	LogIndex        json.RawMessage `json:"log_index"`
	Address         string          `json:"address"`
	Topic0          string          `json:"topic0"`
	Topic1          string          `json:"topic1"`
	Topic2          string          `json:"topic2"`
	Topic3          string          `json:"topic3"`
	Data            string          `json:"data"`
}

type jsonTrace struct {
	BlockNumber     json.RawMessage `json:"block_number"`
	TransactionHash string          `json:"transaction_hash"`
	From            string          `json:"from"`
	To              string          `json:"to"`
	Value           json.RawMessage `json:"value"`
	CallType        string          `json:"call_type"`
	Kind            string          `json:"type"`
	GasUsed         json.RawMessage `json:"gas_used"`
	Error           string          `json:"error"`
}

func parseJSONBig(raw json.RawMessage) (*big.Int, bool, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, false, nil
	}
	var s string
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, false, err
		}
	} else {
		s = string(raw)
	}
	if s == "" {
		return nil, false, nil
	}
	n, ok := new(big.Int).SetString(s, 0)
	if !ok {
		return nil, false, fmt.Errorf("invalid integer %q", s)
	}
	return n, true, nil
}

func parseJSONUint64(raw json.RawMessage) (*uint64, bool, error) {
	n, ok, err := parseJSONBig(raw)
	if err != nil || !ok {
		return nil, ok, err
	}
	return uint64Ptr(n.Uint64()), true, nil
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}

func uint8Ptr(v uint8) *uint8 {
	return &v
}

func jsonStringBytes(s string) *[]byte {
	if s == "" {
		return nil
	}
	b := common.FromHex(s)
	return &b
}

func jsonHash(s string) *common.Hash {
	if s == "" {
		return nil
	}
	h := common.HexToHash(s)
	return &h
}

func jsonAddress(s string) *common.Address {
	if s == "" {
		return nil
	}
	a := common.HexToAddress(s)
	return &a
}

func convertJSONQueryResponse(in *jsonQueryResponse) (*types.QueryResponse, error) {
	out := &types.QueryResponse{Data: types.DataResponse{}, RollbackGuard: in.RollbackGuard, TotalExecutionTime: in.TotalExecutionTime}
	if n, ok, err := parseJSONBig(in.ArchiveHeight); err != nil {
		return nil, fmt.Errorf("archive_height: %w", err)
	} else if ok {
		out.ArchiveHeight = n
	}
	if n, ok, err := parseJSONBig(in.NextBlock); err != nil {
		return nil, fmt.Errorf("next_block: %w", err)
	} else if ok {
		out.NextBlock = n
	}

	for _, chunk := range in.Data {
		for _, b := range chunk.Blocks {
			block := types.Block{
				Hash:       jsonHash(b.Hash),
				ParentHash: jsonHash(b.ParentHash),
				Miner:      jsonAddress(b.Miner),
			}
			if n, ok, err := parseJSONBig(b.Number); err != nil {
				return nil, fmt.Errorf("block.number: %w", err)
			} else if ok {
				block.Number = n
			}
			if n, ok, err := parseJSONUint64(b.Timestamp); err != nil {
				return nil, fmt.Errorf("block.timestamp: %w", err)
			} else if ok {
				ts := time.Unix(int64(*n), 0)
				block.Timestamp = &ts
			}
			if n, ok, err := parseJSONUint64(b.GasUsed); err != nil {
				return nil, fmt.Errorf("block.gas_used: %w", err)
			} else if ok {
				block.GasUsed = n
			}
			if n, ok, err := parseJSONUint64(b.GasLimit); err != nil {
				return nil, fmt.Errorf("block.gas_limit: %w", err)
			} else if ok {
				block.GasLimit = n
			}
			if n, ok, err := parseJSONBig(b.BaseFeePerGas); err != nil {
				return nil, fmt.Errorf("block.base_fee_per_gas: %w", err)
			} else if ok {
				block.BaseFeePerGas = n
			}
			if n, ok, err := parseJSONUint64(b.Size); err != nil {
				return nil, fmt.Errorf("block.size: %w", err)
			} else if ok {
				block.Size = n
			}
			out.Data.Blocks = append(out.Data.Blocks, block)
		}

		for _, t := range chunk.Transactions {
			tx := types.Transaction{
				Hash:            jsonHash(t.Hash),
				From:            jsonAddress(t.From),
				To:              jsonAddress(t.To),
				Input:           jsonStringBytes(t.Input),
				ContractAddress: jsonAddress(t.ContractAddress),
			}
			if n, ok, err := parseJSONBig(t.BlockNumber); err != nil {
				return nil, fmt.Errorf("tx.block_number: %w", err)
			} else if ok {
				tx.BlockNumber = n
			}
			if n, ok, err := parseJSONUint64(t.TransactionIndex); err != nil {
				return nil, fmt.Errorf("tx.transaction_index: %w", err)
			} else if ok {
				tx.TransactionIndex = n
			}
			if n, ok, err := parseJSONBig(t.Value); err != nil {
				return nil, fmt.Errorf("tx.value: %w", err)
			} else if ok {
				tx.Value = n
			}
			if n, ok, err := parseJSONUint64(t.Nonce); err != nil {
				return nil, fmt.Errorf("tx.nonce: %w", err)
			} else if ok {
				tx.Nonce = n
			}
			if n, ok, err := parseJSONUint64(t.Gas); err != nil {
				return nil, fmt.Errorf("tx.gas: %w", err)
			} else if ok {
				tx.Gas = n
			}
			if n, ok, err := parseJSONBig(t.GasPrice); err != nil {
				return nil, fmt.Errorf("tx.gas_price: %w", err)
			} else if ok {
				tx.GasPrice = n
			}
			if n, ok, err := parseJSONUint64(t.GasUsed); err != nil {
				return nil, fmt.Errorf("tx.gas_used: %w", err)
			} else if ok {
				tx.GasUsed = n
			}
			if n, ok, err := parseJSONUint64(t.CumulativeGasUsed); err != nil {
				return nil, fmt.Errorf("tx.cumulative_gas_used: %w", err)
			} else if ok {
				tx.CumulativeGasUsed = n
			}
			if n, ok, err := parseJSONBig(t.EffectiveGasPrice); err != nil {
				return nil, fmt.Errorf("tx.effective_gas_price: %w", err)
			} else if ok {
				tx.EffectiveGasPrice = n
			}
			if n, ok, err := parseJSONBig(t.MaxFeePerGas); err != nil {
				return nil, fmt.Errorf("tx.max_fee_per_gas: %w", err)
			} else if ok {
				tx.MaxFeePerGas = n
			}
			if n, ok, err := parseJSONBig(t.MaxPriorityFeePerGas); err != nil {
				return nil, fmt.Errorf("tx.max_priority_fee_per_gas: %w", err)
			} else if ok {
				tx.MaxPriorityFeePerGas = n
			}
			if n, ok, err := parseJSONUint64(t.Kind); err != nil {
				return nil, fmt.Errorf("tx.type: %w", err)
			} else if ok {
				tx.Kind = uint8Ptr(uint8(*n))
			}
			if n, ok, err := parseJSONUint64(t.Status); err != nil {
				return nil, fmt.Errorf("tx.status: %w", err)
			} else if ok {
				tx.Status = uint8Ptr(uint8(*n))
			}
			out.Data.Transactions = append(out.Data.Transactions, tx)
		}

		for _, l := range chunk.Logs {
			logRow := types.Log{
				TransactionHash: jsonHash(l.TransactionHash),
				Address:         jsonAddress(l.Address),
				Data:            jsonStringBytes(l.Data),
				Topic0:          jsonHash(l.Topic0),
				Topic1:          jsonHash(l.Topic1),
				Topic2:          jsonHash(l.Topic2),
				Topic3:          jsonHash(l.Topic3),
			}
			if n, ok, err := parseJSONBig(l.BlockNumber); err != nil {
				return nil, fmt.Errorf("log.block_number: %w", err)
			} else if ok {
				logRow.BlockNumber = n
			}
			if n, ok, err := parseJSONUint64(l.LogIndex); err != nil {
				return nil, fmt.Errorf("log.log_index: %w", err)
			} else if ok {
				logRow.LogIndex = n
			}
			out.Data.Logs = append(out.Data.Logs, logRow)
		}

		for _, t := range chunk.Traces {
			tr := types.Trace{
				TransactionHash: jsonHash(t.TransactionHash),
				From:            jsonAddress(t.From),
				To:              jsonAddress(t.To),
			}
			if t.CallType != "" {
				tr.CallType = &t.CallType
			}
			if t.Kind != "" {
				tr.Kind = &t.Kind
			}
			if t.Error != "" {
				tr.Error = &t.Error
			}
			if n, ok, err := parseJSONBig(t.BlockNumber); err != nil {
				return nil, fmt.Errorf("trace.block_number: %w", err)
			} else if ok {
				tr.BlockNumber = n
			}
			if n, ok, err := parseJSONBig(t.Value); err != nil {
				return nil, fmt.Errorf("trace.value: %w", err)
			} else if ok {
				tr.Value = n
			}
			if n, ok, err := parseJSONUint64(t.GasUsed); err != nil {
				return nil, fmt.Errorf("trace.gas_used: %w", err)
			} else if ok {
				tr.GasUsed = n
			}
			out.Data.Traces = append(out.Data.Traces, tr)
		}
	}
	return out, nil
}

func getIndexerBatch(ctx context.Context, client *hypersyncgo.Client, query *types.Query) (*types.QueryResponse, error) {
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		raw, err := hypersyncgo.DoQuery[*types.Query, jsonQueryResponse](ctx, client, http.MethodPost, query)
		if err == nil {
			res, convertErr := convertJSONQueryResponse(raw)
			if convertErr == nil {
				return res, nil
			}
			err = convertErr
		}
		lastErr = err

		select {
		case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, fmt.Errorf("failed to get JSON query data after retries: %w", lastErr)
}

func runIndexer(app core.App, attempt int) error {
	ctx := context.Background()

	apiToken := EnvioAPIToken()
	if apiToken == "" {
		return fmt.Errorf("ENVIO_API_TOKEN not set — get one at envio.dev")
	}

	rpc := NextRPCURL()
	log.Printf("[indexer] connecting — hypersync: %s  rpc: %s", ArcHyperSyncURL, rpc)

	hyper, err := hypersyncgo.NewHyper(ctx, options.Options{
		Blockchains: []options.Node{
			{
				Type:        utils.EthereumNetwork,
				NetworkId:   ArcNetworkID,
				Endpoint:    ArcHyperSyncURL,
				RpcEndpoint: rpc,
				ApiToken:    apiToken,
				// Fail fast on 429 so our outer backoff+rotation kicks in quickly.
				// Library retries: 3 attempts × ≤3s ceiling = ≤9s before error surfaces.
				MaxNumRetries:  3,
				RetryBaseMs:    500 * time.Millisecond,
				RetryBackoffMs: 500 * time.Millisecond,
				RetryCeilingMs: 3 * time.Second,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create HyperSync client: %w", err)
	}

	client, ok := hyper.GetClient(ArcNetworkID)
	if !ok {
		return fmt.Errorf("arc client not found in hyper")
	}

	fromBlock := resolveStartBlock(ctx, app, client)
	log.Printf("[indexer] streaming from block %d", fromBlock)
	recordIndexerEvent(app, "info", "stream_start", "streaming from saved start block", indexerEventFields{"attempt": attempt, "block": fromBlock})

	var currentBlock atomic.Uint64
	currentBlock.Store(fromBlock)
	var completedBatches atomic.Uint64
	var lastBatchAtUnixNano atomic.Int64
	lastBatchAtUnixNano.Store(time.Now().UnixNano())
	var processingBatch atomic.Uint64
	var processingStartedAtUnixNano atomic.Int64
	var lastPersistedHeartbeatUnixNano atomic.Int64

	heartbeatStop := make(chan struct{})
	defer close(heartbeatStop)
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				lastBatchAt := time.Unix(0, lastBatchAtUnixNano.Load())
				processingStartedAt := time.Time{}
				if started := processingStartedAtUnixNano.Load(); started > 0 {
					processingStartedAt = time.Unix(0, started)
				}
				activeBatch := processingBatch.Load()
				persistHeartbeat := false
				if activeBatch == 0 {
					lastPersisted := time.Unix(0, lastPersistedHeartbeatUnixNano.Load())
					if lastPersisted.IsZero() || time.Since(lastPersisted) >= time.Minute {
						persistHeartbeat = true
						lastPersistedHeartbeatUnixNano.Store(time.Now().UnixNano())
					}
				}
				// During a batch, SQLite may be inside a long write transaction,
				// so keep in-flight heartbeats to stdout.
				logIndexerHeartbeat(ctx, app, client, attempt, completedBatches.Load(), currentBlock.Load(), lastBatchAt, activeBatch, processingStartedAt, persistHeartbeat)
			case <-heartbeatStop:
				return
			}
		}
	}()

	log.Println("[indexer] explicit GetArrow loop running")

	for {
		start := currentBlock.Load()
		tipCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		tip, tipErr := getChainTip(tipCtx, client)
		cancel()
		if tipErr != nil {
			return fmt.Errorf("fetch chain tip before batch: %w", tipErr)
		}
		if start >= tip {
			time.Sleep(2 * time.Second)
			continue
		}

		toBlock := start + 200
		if toBlock <= start || toBlock > tip+1 {
			toBlock = tip + 1
		}

		nextBatch := completedBatches.Load() + 1
		batchStart := time.Now()
		processingBatch.Store(nextBatch)
		processingStartedAtUnixNano.Store(batchStart.UnixNano())
		log.Printf("[indexer] batch #%d fetching | range=[%d,%d) | tip=%d | lag=%d", nextBatch, start, toBlock, tip, tip-start)

		res, err := getIndexerBatch(ctx, client, newIndexerQuery(start, toBlock))
		if err != nil {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			return fmt.Errorf("fetch batch range [%d,%d): %w", start, toBlock, err)
		}

		nextBlock := "<nil>"
		if res.NextBlock != nil {
			nextBlock = res.NextBlock.String()
		}
		log.Printf("[indexer] batch #%d processing | current block %d | next_block=%s | blocks=%d txs=%d logs=%d",
			nextBatch, start, nextBlock, len(res.Data.Blocks), len(res.Data.Transactions), len(res.Data.Logs))
		recordIndexerEvent(app, "info", "batch_start", "started processing indexer batch", indexerEventFields{
			"attempt":      attempt,
			"batch":        nextBatch,
			"block":        start,
			"blocks":       len(res.Data.Blocks),
			"transactions": len(res.Data.Transactions),
			"logs":         len(res.Data.Logs),
		})
		if err := processBatch(app, res); err != nil {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			log.Printf("[indexer] batch processing error: %v", err)
			recordIndexerEvent(app, "error", "batch_error", "batch failed; cursor was not advanced", indexerEventFields{
				"attempt":      attempt,
				"batch":        nextBatch,
				"block":        start,
				"blocks":       len(res.Data.Blocks),
				"transactions": len(res.Data.Transactions),
				"logs":         len(res.Data.Logs),
				"error":        err,
			})
			return err
		}

		if res.NextBlock == nil {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			return fmt.Errorf("batch range [%d,%d) returned nil next_block", start, toBlock)
		}
		next := res.NextBlock.Uint64()
		if next <= start {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			return fmt.Errorf("batch range [%d,%d) did not advance next_block: %d", start, toBlock, next)
		}

		currentBlock.Store(next)
		if err := setLastIndexedBlock(app, next); err != nil {
			processingBatch.Store(0)
			processingStartedAtUnixNano.Store(0)
			recordIndexerEvent(app, "error", "cursor_error", "failed to persist cursor after batch", indexerEventFields{"attempt": attempt, "batch": nextBatch, "block": next, "error": err})
			return err
		}

		batchCount := completedBatches.Add(1)
		elapsed := time.Since(batchStart).Milliseconds()
		lastBatchAtUnixNano.Store(time.Now().UnixNano())
		processingBatch.Store(0)
		processingStartedAtUnixNano.Store(0)
		log.Printf("[indexer] batch #%d | block %d | range=[%d,%d) | blocks=%d txs=%d logs=%d | %dms",
			batchCount, currentBlock.Load(), start, toBlock,
			len(res.Data.Blocks), len(res.Data.Transactions), len(res.Data.Logs),
			elapsed,
		)
		recordIndexerEvent(app, "info", "batch_done", "finished processing indexer batch", indexerEventFields{
			"attempt":      attempt,
			"batch":        batchCount,
			"block":        currentBlock.Load(),
			"duration_ms":  elapsed,
			"blocks":       len(res.Data.Blocks),
			"transactions": len(res.Data.Transactions),
			"logs":         len(res.Data.Logs),
		})

		// Pace requests to avoid HyperSync free-tier burst throttling.
		time.Sleep(400 * time.Millisecond)
	}
}

// ── Batch processing ──────────────────────────────────────────────────────────

func processBatch(app core.App, res *types.QueryResponse) error {
	// aggregate per-block stats within this batch
	type blockAcc struct {
		txCount         int
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

	// per-address deltas for agent aggregation (raw values: wei for fees, raw ERC-20 for transfers)
	type agentDelta struct {
		feeWei      *big.Int
		transferred *big.Int
		txCount     int
	}
	agentDeltas := make(map[string]*agentDelta)

	getAgentDelta := func(addr string) *agentDelta {
		if agentDeltas[addr] == nil {
			agentDeltas[addr] = &agentDelta{feeWei: new(big.Int), transferred: new(big.Int)}
		}
		return agentDeltas[addr]
	}

	return app.RunInTransaction(func(txApp core.App) error {
		baseFeeByBlock := make(map[uint64]*big.Int)
		// 1. Blocks
		for _, blk := range res.Data.Blocks {
			if blk.Number == nil {
				continue
			}
			if blk.BaseFeePerGas != nil {
				baseFeeByBlock[blk.Number.Uint64()] = new(big.Int).Set(blk.BaseFeePerGas)
			}
			if err := saveBlock(txApp, &blk); err != nil {
				return err
			}
			getAcc(blk.Number.Uint64()) // ensure acc exists even for empty blocks
		}

		// 2. Transactions
		for _, tx := range res.Data.Transactions {
			if tx.Hash == nil || tx.BlockNumber == nil {
				continue
			}
			fee, err := saveTransaction(txApp, &tx, baseFeeByBlock[tx.BlockNumber.Uint64()])
			if err != nil {
				return err
			}
			bn := tx.BlockNumber.Uint64()
			acc := getAcc(bn)
			acc.txCount++
			if tx.From != nil {
				acc.uniqueSenders[tx.From.Hex()] = struct{}{}
				// accumulate for agent aggregation
				d := getAgentDelta(tx.From.Hex())
				d.txCount++
				if fee != nil {
					d.feeWei.Add(d.feeWei, fee)
				}
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

		// 3. Logs → transfers / crosschain / fx / agents / jobs
		for _, log := range res.Data.Logs {
			if log.BlockNumber == nil {
				continue
			}
			bn := log.BlockNumber.Uint64()
			acc := getAcc(bn)
			amount, err := routeLog(txApp, &log)
			if err != nil {
				return err
			}
			if amount != nil && log.Address != nil {
				switch *log.Address {
				case AddrUSDC:
					acc.totalUSDC.Add(acc.totalUSDC, amount)
					if amount.Cmp(acc.largestUSDC) > 0 {
						acc.largestUSDC.Set(amount)
					}
				case AddrEURC:
					acc.totalEURC.Add(acc.totalEURC, amount)
				case AddrUSYC:
					acc.totalUSYC.Add(acc.totalUSYC, amount)
				}
				// accumulate USDC sent by from_addr for agent aggregation
				if log.Topic0 != nil && *log.Topic0 == TopicTransfer &&
					log.Address != nil && *log.Address != AddrAgentRegistry &&
					log.Topic1 != nil {
					fromAddr := common.BytesToAddress(log.Topic1.Bytes()[12:]).Hex()
					getAgentDelta(fromAddr).transferred.Add(getAgentDelta(fromAddr).transferred, amount)
				}
			}
		}

		// 3b. Traces
		for _, trace := range res.Data.Traces {
			if trace.TransactionHash == nil || trace.BlockNumber == nil {
				continue
			}
			if err := saveTrace(txApp, &trace); err != nil {
				return err
			}
		}

		// 4. Block stats + tx_count back-fill onto blocks.
		//    Sort blocks ascending so we can compute consecutive block_time_ms.
		type blkTs struct {
			num uint64
			ts  int64
		}
		var sortedBlocks []blkTs
		for _, blk := range res.Data.Blocks {
			if blk.Number != nil && blk.Timestamp != nil {
				sortedBlocks = append(sortedBlocks, blkTs{blk.Number.Uint64(), blk.Timestamp.Unix()})
			}
		}
		// simple insertion sort (batches are small)
		for i := 1; i < len(sortedBlocks); i++ {
			for j := i; j > 0 && sortedBlocks[j].num < sortedBlocks[j-1].num; j-- {
				sortedBlocks[j], sortedBlocks[j-1] = sortedBlocks[j-1], sortedBlocks[j]
			}
		}

		// block_time_ms indexed by block number — look up prev from DB for the first block
		blockTimeMs := make(map[uint64]int64)
		for i, bt := range sortedBlocks {
			if i == 0 {
				prev, err := txApp.FindRecordsByFilter("blocks", "number = {:n}", "", 1, 0, map[string]any{"n": bt.num - 1})
				if err != nil {
					return fmt.Errorf("find previous block %d: %w", bt.num-1, err)
				}
				if len(prev) > 0 {
					prevTs := prev[0].GetInt("timestamp")
					blockTimeMs[bt.num] = (bt.ts - int64(prevTs)) * 1000
				}
			} else {
				blockTimeMs[bt.num] = (bt.ts - sortedBlocks[i-1].ts) * 1000
			}
		}

		for _, blk := range res.Data.Blocks {
			if blk.Number == nil || blk.Timestamp == nil {
				continue
			}
			bn := blk.Number.Uint64()
			acc := getAcc(bn)

			// back-fill tx_count onto the blocks record
			existingBlocks, err := txApp.FindRecordsByFilter("blocks", "number = {:n}", "", 1, 0, map[string]any{"n": bn})
			if err != nil {
				return fmt.Errorf("find block %d for stats backfill: %w", bn, err)
			}
			if len(existingBlocks) > 0 {
				existingBlocks[0].Set("tx_count", acc.txCount)
				if bms, ok := blockTimeMs[bn]; ok && bms > 0 {
					existingBlocks[0].Set("block_time_ms", bms)
				}
				if err := txApp.Save(existingBlocks[0]); err != nil {
					return fmt.Errorf("save block %d stats backfill: %w", bn, err)
				}
			}

			// skip block_stats if already persisted (indexer restart)
			existingStats, err := txApp.FindRecordsByFilter("block_stats", "block_number = {:n}", "", 1, 0, map[string]any{"n": bn})
			if err != nil {
				return fmt.Errorf("find block_stats %d: %w", bn, err)
			}
			if len(existingStats) > 0 {
				continue
			}

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

			bms := blockTimeMs[bn]
			var tps float64
			if bms > 0 {
				tps = float64(acc.txCount) / (float64(bms) / 1000.0)
			}

			stats := core.NewRecord(mustCollection(txApp, "block_stats"))
			stats.Set("block_number", bn)
			stats.Set("timestamp", blk.Timestamp.Unix())
			stats.Set("tx_count", acc.txCount)
			stats.Set("block_time_ms", bms)
			stats.Set("tps", tps)
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
				return fmt.Errorf("save block_stats %d: %w", bn, err)
			}
		}

		// 5. Agent aggregation — update tx_count, usdc_spent_fees, usdc_transferred
		//    for any agent address that had activity in this batch.
		//    Raw storage: wei string for fees, raw ERC-20 units string for transfers.
		for addr, delta := range agentDeltas {
			if delta.txCount == 0 && delta.feeWei.Sign() == 0 && delta.transferred.Sign() == 0 {
				continue
			}
			agentRows, err := txApp.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, map[string]any{"a": addr})
			if err != nil || len(agentRows) == 0 {
				continue // address is not a registered agent
			}
			r := agentRows[0]
			if delta.txCount > 0 {
				r.Set("tx_count", r.GetInt("tx_count")+delta.txCount)
			}
			if delta.feeWei.Sign() > 0 {
				prev, _ := new(big.Int).SetString(r.GetString("usdc_spent_fees"), 10)
				if prev == nil {
					prev = new(big.Int)
				}
				r.Set("usdc_spent_fees", new(big.Int).Add(prev, delta.feeWei).String())
			}
			if delta.transferred.Sign() > 0 {
				prev, _ := new(big.Int).SetString(r.GetString("usdc_transferred"), 10)
				if prev == nil {
					prev = new(big.Int)
				}
				r.Set("usdc_transferred", new(big.Int).Add(prev, delta.transferred).String())
			}
			if err := txApp.Save(r); err != nil {
				return fmt.Errorf("update agent %s stats: %w", addr, err)
			}
		}

		return nil
	})
}

// ── Individual record savers ──────────────────────────────────────────────────

func saveBlock(app core.App, blk *types.Block) error {
	// skip if already exists
	existing, err := app.FindRecordsByFilter("blocks", "number = {:n}", "", 1, 0, map[string]any{"n": blk.Number.Uint64()})
	if err != nil {
		return fmt.Errorf("find block %d: %w", blk.Number.Uint64(), err)
	}
	if len(existing) > 0 {
		return nil
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
		return fmt.Errorf("save block %d: %w", blk.Number.Uint64(), err)
	}

	return nil
}

// saveTransaction saves the transaction and returns the fee in wei (for stats accumulation).
func saveTransaction(app core.App, tx *types.Transaction, blockBaseFee *big.Int) (*big.Int, error) {
	existing, err := app.FindRecordsByFilter("transactions", "hash = {:h}", "", 1, 0, map[string]any{"h": tx.Hash.Hex()})
	if err != nil {
		return nil, fmt.Errorf("find transaction %s: %w", tx.Hash.Hex(), err)
	}
	if len(existing) > 0 {
		return nil, nil
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
	if tx.Gas != nil {
		r.Set("gas_limit", *tx.Gas)
	}
	if tx.GasUsed != nil {
		r.Set("gas_used", *tx.GasUsed)
	}
	if tx.CumulativeGasUsed != nil {
		r.Set("cumulative_gas_used", *tx.CumulativeGasUsed)
	}
	if tx.MaxFeePerGas != nil {
		r.Set("max_fee_per_gas", tx.MaxFeePerGas.String())
	}
	if tx.MaxPriorityFeePerGas != nil {
		r.Set("max_priority_fee_per_gas", tx.MaxPriorityFeePerGas.String())
	}

	var feeWei *big.Int
	if tx.GasUsed != nil && tx.EffectiveGasPrice != nil {
		feeWei = new(big.Int).Mul(new(big.Int).SetUint64(*tx.GasUsed), tx.EffectiveGasPrice)
		r.Set("effective_gas_price", tx.EffectiveGasPrice.String())
		r.Set("fee_usdc", weiToUSDC(feeWei))
	}
	if tx.GasUsed != nil && tx.EffectiveGasPrice != nil && blockBaseFee != nil {
		priorityPerGas := new(big.Int).Sub(tx.EffectiveGasPrice, blockBaseFee)
		if priorityPerGas.Sign() < 0 {
			priorityPerGas.SetInt64(0)
		}
		priorityFeeWei := new(big.Int).Mul(new(big.Int).SetUint64(*tx.GasUsed), priorityPerGas)
		r.Set("priority_fee_per_gas", priorityPerGas.String())
		r.Set("priority_fee_usdc", weiToUSDC(priorityFeeWei))
	}

	if tx.Kind != nil {
		r.Set("tx_type", *tx.Kind)
	}
	if tx.Status != nil {
		r.Set("status", *tx.Status)
	}
	isDeployment := tx.ContractAddress != nil
	if isDeployment {
		r.Set("contract_address", tx.ContractAddress.Hex())
	}
	r.Set("is_contract_deploy", isDeployment)

	if err := app.Save(r); err != nil {
		return nil, fmt.Errorf("save transaction %s: %w", tx.Hash.Hex(), err)
	}
	return feeWei, nil
}

// routeLog decodes a log and routes it to the right handler.
// Returns the transfer amount (in raw uint256) if this is an ERC-20 Transfer, else nil.
func routeLog(app core.App, log *types.Log) (*big.Int, error) {
	if log.Topic0 == nil || log.Address == nil {
		return nil, nil
	}

	addr := *log.Address
	switch *log.Topic0 {
	case TopicTransfer:
		if addr == AddrAgentRegistry {
			return nil, saveAgentRegistration(app, log)
		}
		return saveTransfer(app, log)

	case TopicDepositForBurn:
		return nil, saveCCTPDepositForBurn(app, log)

	case TopicMintAndWithdraw:
		return nil, saveCCTPMintAndWithdraw(app, log)

	case TopicMessageReceived:
		return nil, saveCCTPMessageReceived(app, log)

	case TopicGatewayDeposited:
		return nil, saveGatewayDeposited(app, log)

	case TopicGatewayBurned:
		return nil, saveGatewayBurned(app, log)

	case TopicAttestationUsed:
		return nil, saveAttestationUsed(app, log)

	case TopicJobCreated:
		if addr == AddrAgenticCommerce {
			return nil, saveAgentJobCreated(app, log)
		}
	case TopicJobFunded:
		if addr == AddrAgenticCommerce {
			return nil, saveAgentJobFunded(app, log)
		}
	case TopicJobSubmitted:
		if addr == AddrAgenticCommerce {
			return nil, saveAgentJobSubmitted(app, log)
		}
	case TopicJobCompleted:
		if addr == AddrAgenticCommerce {
			return nil, saveAgentJobCompleted(app, log)
		}
	case TopicJobRejected:
		if addr == AddrAgenticCommerce {
			return nil, saveAgentJobRejected(app, log)
		}
	case TopicPaymentReleased:
		if addr == AddrAgenticCommerce {
			return nil, saveAgentJobPaid(app, log)
		}
	case TopicJobExpired:
		if addr == AddrAgenticCommerce {
			return nil, saveAgentJobExpired(app, log)
		}

	case TopicTradeRecorded, TopicMakerFunded, TopicTakerFunded, TopicTradeStatusChanged, TopicFeesProcessed:
		if addr == AddrFxEscrow {
			return nil, saveFxEvent(app, log)
		}
	}
	return nil, nil
}

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

	symbol := "OTHER"
	if s, ok := KnownTokens[*log.Address]; ok {
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
		return nil, fmt.Errorf("save transfer %s/%d: %w", txHash, logIdx, err)
	}

	// update wallet graph edge
	if err := upsertWalletEdge(app, from.Hex(), to.Hex(), amountRaw, log.BlockNumber); err != nil {
		return nil, err
	}

	return amountRaw, nil
}

// saveCrosschain is the shared upsert helper for crosschain_events.
func saveCrosschain(app core.App, log *types.Log, fill func(*core.Record)) error {
	if log.TransactionHash == nil || log.LogIndex == nil {
		return nil
	}
	existing, err := app.FindRecordsByFilter("crosschain_events",
		"tx_hash = {:h} && log_index = {:i}", "", 1, 0,
		map[string]any{"h": log.TransactionHash.Hex(), "i": *log.LogIndex})
	if err != nil {
		return fmt.Errorf("find crosschain %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}
	if len(existing) > 0 {
		return nil
	}
	r := core.NewRecord(mustCollection(app, "crosschain_events"))
	r.Set("tx_hash", log.TransactionHash.Hex())
	r.Set("log_index", *log.LogIndex)
	if log.BlockNumber != nil {
		r.Set("block_number", log.BlockNumber.Uint64())
	}
	fill(r)
	if err := app.Save(r); err != nil {
		return fmt.Errorf("save crosschain %s/%d: %w", log.TransactionHash.Hex(), *log.LogIndex, err)
	}
	return nil
}

// readUint32 reads a uint32 from 32 ABI-padded bytes at offset in data.
func readUint32(data []byte, offset int) uint32 {
	if len(data) < offset+32 {
		return 0
	}
	return uint32(new(big.Int).SetBytes(data[offset : offset+32]).Uint64())
}

// readBig reads a *big.Int from 32 ABI bytes at offset in data.
func readBig(data []byte, offset int) *big.Int {
	if len(data) < offset+32 {
		return new(big.Int)
	}
	return new(big.Int).SetBytes(data[offset : offset+32])
}

// ── CCTP event handlers ───────────────────────────────────────────────────────

// DepositForBurn: USDC exits Arc via CCTP.
// t1=burnToken(indexed), t2=depositor(indexed), t3=minFinalityThreshold(indexed)
// data: [amount(u256), mintRecipient(bytes32), destinationDomain(u32), ...]
func saveCCTPDepositForBurn(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "cctp")
		r.Set("event_type", "burn")
		r.Set("source_domain", 26) // Arc is always the source here

		if log.Topic2 != nil {
			r.Set("sender", addressFromTopic(log.Topic2))
		}
		if log.Data != nil && len(*log.Data) >= 64 {
			d := *log.Data
			r.Set("amount_usdc", stablecoinHuman(readBig(d, 0)))
			// mintRecipient is bytes32; last 20 bytes = address
			r.Set("recipient", addressFromBytes32(d[32:64]))
			if len(d) >= 96 {
				r.Set("destination_domain", readUint32(d, 64))
			}
		}
	})
}

// MintAndWithdraw: USDC arrives on Arc from another chain.
// t1=mintRecipient(indexed), t2=mintToken(indexed)
// data: [amount(u256), feeCollected(u256)]
func saveCCTPMintAndWithdraw(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "cctp")
		r.Set("event_type", "mint")
		r.Set("destination_domain", 26)

		if log.Topic1 != nil {
			r.Set("recipient", addressFromTopic(log.Topic1))
		}
		if log.Data != nil && len(*log.Data) >= 32 {
			r.Set("amount_usdc", stablecoinHuman(readBig(*log.Data, 0)))
		}
	})
}

// MessageReceived: low-level CCTP message delivery on Arc.
// t1=caller(indexed), t2=nonce(bytes32 indexed), t3=finalityThreshold(indexed)
// data: [sourceDomain(u32), sender(bytes32), messageBody(bytes)]
func saveCCTPMessageReceived(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "cctp")
		r.Set("event_type", "mint")
		r.Set("destination_domain", 26)

		if log.Topic2 != nil {
			r.Set("nonce_val", log.Topic2.Hex())
		}
		if log.Data != nil && len(*log.Data) >= 64 {
			d := *log.Data
			r.Set("source_domain", readUint32(d, 0))
			// sender is bytes32; last 20 bytes = address
			r.Set("sender", addressFromBytes32(d[32:64]))
		}
	})
}

// ── Gateway event handlers ────────────────────────────────────────────────────

// Deposited: user deposits USDC into their unified balance on Arc.
// t1=token(indexed), t2=depositor(indexed), t3=sender(indexed)
// data: [value(u256)]
func saveGatewayDeposited(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "gateway")
		r.Set("event_type", "deposit")
		r.Set("source_domain", 26)
		r.Set("destination_domain", 26)

		if log.Topic2 != nil {
			r.Set("sender", addressFromTopic(log.Topic2))
		}
		if log.Topic3 != nil {
			r.Set("recipient", addressFromTopic(log.Topic3))
		}
		if log.Data != nil && len(*log.Data) >= 32 {
			r.Set("amount_usdc", stablecoinHuman(readBig(*log.Data, 0)))
		}
	})
}

// GatewayBurned: USDC leaves Arc via Gateway bridge.
// t1=token(indexed), t2=depositor(indexed), t3=transferSpecHash(bytes32 indexed)
// data: [destinationDomain(u32), destinationRecipient(bytes32), signer(addr), value(u256), ...]
func saveGatewayBurned(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "gateway")
		r.Set("event_type", "withdraw")
		r.Set("source_domain", 26)

		if log.Topic2 != nil {
			r.Set("sender", addressFromTopic(log.Topic2))
		}
		if log.Topic3 != nil {
			r.Set("nonce_val", log.Topic3.Hex()) // transferSpecHash
		}
		if log.Data != nil && len(*log.Data) >= 128 {
			d := *log.Data
			r.Set("destination_domain", readUint32(d, 0))
			r.Set("recipient", addressFromBytes32(d[32:64]))
			r.Set("amount_usdc", stablecoinHuman(readBig(d, 96)))
		}
	})
}

// AttestationUsed: USDC arrives on Arc from another chain via Gateway.
// t1=token(indexed), t2=recipient(indexed), t3=transferSpecHash(bytes32 indexed)
// data: [sourceDomain(u32), sourceDepositor(bytes32), sourceSigner(bytes32), value(u256)]
func saveAttestationUsed(app core.App, log *types.Log) error {
	return saveCrosschain(app, log, func(r *core.Record) {
		r.Set("protocol", "gateway")
		r.Set("event_type", "deposit")
		r.Set("destination_domain", 26)

		if log.Topic2 != nil {
			r.Set("recipient", addressFromTopic(log.Topic2))
		}
		if log.Topic3 != nil {
			r.Set("nonce_val", log.Topic3.Hex()) // transferSpecHash
		}
		if log.Data != nil && len(*log.Data) >= 128 {
			d := *log.Data
			r.Set("source_domain", readUint32(d, 0))
			r.Set("sender", addressFromBytes32(d[32:64]))
			r.Set("amount_usdc", stablecoinHuman(readBig(d, 96)))
		}
	})
}

// fxUpsert finds or creates the fx_swaps record for tradeID, then calls update(r) and saves it.
func fxUpsert(app core.App, tradeID string, update func(*core.Record)) error {
	existing, err := app.FindRecordsByFilter("fx_swaps", "trade_id = {:id}", "", 1, 0, map[string]any{"id": tradeID})
	if err != nil {
		return fmt.Errorf("find fx trade %s: %w", tradeID, err)
	}
	var r *core.Record
	if len(existing) > 0 {
		r = existing[0]
	} else {
		r = core.NewRecord(mustCollection(app, "fx_swaps"))
		r.Set("trade_id", tradeID)
		r.Set("status", "created")
	}
	update(r)
	if err := app.Save(r); err != nil {
		return fmt.Errorf("save fx trade %s: %w", tradeID, err)
	}
	return nil
}

func saveFxEvent(app core.App, log *types.Log) error {
	if log.Topic0 == nil || log.Topic1 == nil {
		return nil
	}

	// All FxEscrow events have trade ID as topic1 (indexed uint256)
	tradeID := new(big.Int).SetBytes(log.Topic1.Bytes()).String()

	switch *log.Topic0 {
	case TopicTradeRecorded:
		// TradeRecorded(uint256 indexed id, bytes32 indexed quoteId)
		if log.Topic2 == nil {
			return nil
		}
		quoteID := log.Topic2.Hex()
		return fxUpsert(app, tradeID, func(r *core.Record) {
			r.Set("quote_id", quoteID)
			r.Set("status", "created")
			if log.BlockNumber != nil {
				r.Set("block_number", log.BlockNumber.Uint64())
			}
			if log.TransactionHash != nil {
				r.Set("tx_hash", log.TransactionHash.Hex())
			}
		})

	case TopicMakerFunded:
		// MakerFunded(uint256 indexed id, address indexed maker)
		if log.Topic2 == nil {
			return nil
		}
		maker := addressFromTopic(log.Topic2)
		return fxUpsert(app, tradeID, func(r *core.Record) {
			r.Set("maker", maker)
			if r.GetString("status") == "taker_funded" {
				r.Set("status", "maker_funded")
			}
		})

	case TopicTakerFunded:
		// TakerFunded(uint256 indexed id, address indexed taker)
		if log.Topic2 == nil {
			return nil
		}
		taker := addressFromTopic(log.Topic2)
		return fxUpsert(app, tradeID, func(r *core.Record) {
			r.Set("taker", taker)
			r.Set("status", "taker_funded")
		})

	case TopicTradeStatusChanged:
		// TradeStatusChanged(uint256 indexed id, address indexed actor, uint8 newStatus)
		// newStatus is ABI-encoded in data: pad-left uint8
		if log.Data == nil || len(*log.Data) < 32 {
			return nil
		}
		statusCode := int(new(big.Int).SetBytes((*log.Data)[:32]).Int64())
		statusStr := "settled"
		if statusCode == 3 {
			statusStr = "cancelled"
		}
		return fxUpsert(app, tradeID, func(r *core.Record) {
			r.Set("status_code", statusCode)
			r.Set("status", statusStr)
		})

	case TopicFeesProcessed:
		// FeesProcessed(uint256 indexed id, uint256 takerFee, uint256 makerFee)
		if log.Data == nil || len(*log.Data) < 64 {
			return nil
		}
		takerFee := new(big.Int).SetBytes((*log.Data)[:32]).String()
		makerFee := new(big.Int).SetBytes((*log.Data)[32:64]).String()
		return fxUpsert(app, tradeID, func(r *core.Record) {
			r.Set("taker_fee", takerFee)
			r.Set("maker_fee", makerFee)
		})
	}

	return nil
}

func saveAgentRegistration(app core.App, log *types.Log) error {
	if log.Topic1 == nil || log.Topic2 == nil || log.TransactionHash == nil {
		return nil
	}
	zero := common.Hash{}
	if *log.Topic1 != zero {
		return nil
	}

	owner := addressFromTopic(log.Topic2)
	existing, err := app.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, map[string]any{"a": owner})
	if err != nil {
		return fmt.Errorf("find agent %s: %w", owner, err)
	}
	if len(existing) > 0 {
		return nil
	}

	r := core.NewRecord(mustCollection(app, "agents"))
	r.Set("agent_address", owner)
	if log.BlockNumber != nil {
		r.Set("registered_at_block", log.BlockNumber.Uint64())
	}
	r.Set("tx_hash", log.TransactionHash.Hex())
	r.Set("tx_count", 0)
	r.Set("usdc_spent_fees", "0")
	r.Set("usdc_transferred", "0")

	if err := app.Save(r); err != nil {
		return fmt.Errorf("save agent registration %s: %w", owner, err)
	}
	return nil
}

func saveAgentJobCreated(app core.App, log *types.Log) error {
	if log.Topic1 == nil || log.Topic2 == nil || log.Topic3 == nil || log.TransactionHash == nil {
		return nil
	}

	jobID := new(big.Int).SetBytes(log.Topic1.Bytes()).String()
	existing, err := app.FindRecordsByFilter("agent_jobs", "job_id = {:j}", "", 1, 0, map[string]any{"j": jobID})
	if err != nil {
		return fmt.Errorf("find agent job %s: %w", jobID, err)
	}
	if len(existing) > 0 {
		return nil
	}

	// topic2 = client (employer), topic3 = provider (worker)
	r := core.NewRecord(mustCollection(app, "agent_jobs"))
	r.Set("job_id", jobID)
	r.Set("employer_address", addressFromTopic(log.Topic2))
	r.Set("worker_address", addressFromTopic(log.Topic3))
	r.Set("status", "created")
	if log.BlockNumber != nil {
		r.Set("created_at_block", log.BlockNumber.Uint64())
	}
	r.Set("create_tx_hash", log.TransactionHash.Hex())

	if err := app.Save(r); err != nil {
		return fmt.Errorf("save agent job %s: %w", jobID, err)
	}
	return nil
}

// agentJobUpsert finds the agent_jobs record for jobID and calls update(r), then saves it.
// Creates a placeholder record if none exists (events can arrive out of order).
func agentJobUpsert(app core.App, log *types.Log, update func(*core.Record)) error {
	if log.Topic1 == nil {
		return nil
	}
	jobID := new(big.Int).SetBytes(log.Topic1.Bytes()).String()
	existing, err := app.FindRecordsByFilter("agent_jobs", "job_id = {:j}", "", 1, 0, map[string]any{"j": jobID})
	if err != nil {
		return fmt.Errorf("find agent job %s: %w", jobID, err)
	}
	var r *core.Record
	if len(existing) > 0 {
		r = existing[0]
	} else {
		r = core.NewRecord(mustCollection(app, "agent_jobs"))
		r.Set("job_id", jobID)
		r.Set("status", "created")
	}
	update(r)
	if err := app.Save(r); err != nil {
		return fmt.Errorf("save agent job %s: %w", jobID, err)
	}
	return nil
}

// JobFunded(uint256 indexed jobId, address indexed client, uint256 amount)
// Sets payment_usdc (raw ERC-20 6-decimal units) and advances status to "funded".
func saveAgentJobFunded(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "funded")
		if log.Data != nil && len(*log.Data) >= 32 {
			r.Set("payment_usdc", stablecoinHuman(readBig(*log.Data, 0)))
		}
	})
}

// JobSubmitted(uint256 indexed jobId, address indexed provider, bytes32 deliverable)
func saveAgentJobSubmitted(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "submitted")
	})
}

// JobCompleted(uint256 indexed jobId, address indexed evaluator, bytes32 reason)
func saveAgentJobCompleted(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "completed")
	})
}

// JobRejected(uint256 indexed jobId, address indexed rejector, bytes32 reason)
func saveAgentJobRejected(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "rejected")
	})
}

// PaymentReleased(uint256 indexed jobId, address indexed provider, uint256 amount)
// Terminal state: payment sent to provider.
func saveAgentJobPaid(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "paid")
		if log.BlockNumber != nil {
			r.Set("settled_at_block", log.BlockNumber.Uint64())
		}
		if log.TransactionHash != nil {
			r.Set("settle_tx_hash", log.TransactionHash.Hex())
		}
	})
}

// JobExpired(uint256 indexed jobId)
func saveAgentJobExpired(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "expired")
	})
}

func saveTrace(app core.App, trace *types.Trace) error {
	r := core.NewRecord(mustCollection(app, "traces"))
	if trace.TransactionHash != nil {
		r.Set("tx_hash", trace.TransactionHash.Hex())
	}
	if trace.BlockNumber != nil {
		r.Set("block_number", trace.BlockNumber.Uint64())
	}
	if trace.From != nil {
		r.Set("from_addr", trace.From.Hex())
	}
	if trace.To != nil {
		r.Set("to_addr", trace.To.Hex())
	}
	if trace.Value != nil {
		r.Set("value", trace.Value.String())
	}
	if trace.CallType != nil {
		r.Set("call_type", *trace.CallType)
	}
	if trace.Kind != nil {
		r.Set("trace_type", *trace.Kind)
	}
	if trace.GasUsed != nil {
		r.Set("gas_used", *trace.GasUsed)
	}
	if trace.Error != nil {
		r.Set("error_msg", *trace.Error)
	}
	if err := app.Save(r); err != nil {
		return fmt.Errorf("save trace: %w", err)
	}
	return nil
}

// upsertWalletEdge increments the edge (from→to) in the wallet graph.
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
		return fmt.Errorf("save wallet edge %s -> %s: %w", from, to, err)
	}
	return nil
}

// ── Cursor management ─────────────────────────────────────────────────────────

// arcBlocksPerDay is a conservative estimate based on Arc's ~1 second block time.
const arcBlocksPerDay = uint64(86_400)

// arcCatchupLookback is how far back we start on a fresh DB.
// 1 hour gives enough context data without blowing the free-tier rate limit
// during catch-up (18 batches of 200 blocks vs 3024 for 7 days).
const arcCatchupLookback = uint64(3_600)

// resolveStartBlock returns the block to stream from.
// If a cursor exists in the DB we resume from there. On a fresh start we fetch
// the current chain tip and walk back 7 days so we don't blow the free-tier
// Envio soft limits (100k events / 5GB) by replaying the entire chain history.
func resolveStartBlock(ctx context.Context, app core.App, client interface {
	GetHeight(context.Context) (*big.Int, error)
}) uint64 {
	last := getLastIndexedBlock(app)
	if last > 0 {
		log.Printf("[indexer] resuming from saved cursor %d", last)
		return last
	}

	height, err := client.GetHeight(ctx)
	if err != nil || height == nil {
		log.Printf("[indexer] could not fetch chain height, starting from block 0: %v", err)
		return 0
	}

	tip := height.Uint64()
	lookback := arcCatchupLookback
	start := uint64(0)
	if tip > lookback {
		start = tip - lookback
	}

	log.Printf("[indexer] fresh start | tip=%d from_block=%d lookback_blocks=%d", tip, start, lookback)
	return start
}

func getLastIndexedBlock(app core.App) uint64 {
	records, err := app.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	if err != nil || len(records) == 0 {
		return 0
	}
	val, _ := strconv.ParseUint(records[0].GetString("value"), 10, 64)
	return val
}

func setLastIndexedBlock(app core.App, block uint64) error {
	records, err := app.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	if err != nil {
		return fmt.Errorf("find lastBlock cursor: %w", err)
	}

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
		return fmt.Errorf("save lastBlock cursor %d: %w", block, err)
	}
	return nil
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

// addressFromBytes32 extracts an Ethereum address from a 32-byte ABI-padded slice (last 20 bytes).
func addressFromBytes32(b []byte) string {
	if len(b) < 32 {
		return ""
	}
	return common.BytesToAddress(b[12:32]).Hex()
}

// stripQuotes is a no-op helper kept for clarity when working with string values.
func stripQuotes(s string) string {
	return strings.Trim(s, `"`)
}
