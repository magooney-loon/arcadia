package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	hypersyncgo "github.com/enviodev/hypersync-client-go"
	"github.com/enviodev/hypersync-client-go/types"
	"github.com/ethereum/go-ethereum/common"

	arc "arcadia/internal/chain/arc"
)

// ── Query builder ─────────────────────────────────────────────────────────────

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
			{Topics: [][]common.Hash{{arc.TopicTransfer}}},
			{
				Address: []common.Address{arc.AddrCCTPTokenMessenger},
				Topics:  [][]common.Hash{{arc.TopicDepositForBurn, arc.TopicMintAndWithdraw}},
			},
			{
				Address: []common.Address{arc.AddrCCTPMessageTransmitter},
				Topics:  [][]common.Hash{{arc.TopicMessageReceived}},
			},
			{
				Address: []common.Address{arc.AddrGatewayWallet},
				Topics:  [][]common.Hash{{arc.TopicGatewayDeposited, arc.TopicGatewayBurned}},
			},
			{
				Address: []common.Address{arc.AddrGatewayMinter},
				Topics:  [][]common.Hash{{arc.TopicAttestationUsed}},
			},
			{Address: []common.Address{arc.AddrFxEscrow}},
			{Address: []common.Address{arc.AddrAgentRegistry}, Topics: [][]common.Hash{{arc.TopicAgentRegistered}}},
			{
				Address: []common.Address{arc.AddrAgenticCommerce},
				Topics: [][]common.Hash{{
					arc.TopicJobCreated,
					arc.TopicJobFunded,
					arc.TopicJobSubmitted,
					arc.TopicJobCompleted,
					arc.TopicJobRejected,
					arc.TopicPaymentReleased,
					arc.TopicJobExpired,
				}},
			},
		},
	}
}

// ── JSON response types ──────────────────────────────────────────────────────

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

// ── JSON parsing helpers ─────────────────────────────────────────────────────

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

func uint64Ptr(v uint64) *uint64 { return &v }
func uint8Ptr(v uint8) *uint8    { return &v }

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

// ── JSON → types conversion ──────────────────────────────────────────────────

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

// ── Batch fetching with retry ────────────────────────────────────────────────

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
