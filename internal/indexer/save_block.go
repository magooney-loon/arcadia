package indexer

import (
	"fmt"
	"math/big"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

func saveBlock(app core.App, blk *types.Block, seen *batchSeen) error {
	bn := blk.Number.Uint64()
	if _, dup := seen.blocks[bn]; dup {
		return nil
	}

	coll, err := utils.FindCollection(app, "blocks")
	if err != nil {
		return err
	}
	r := core.NewRecord(coll)
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
		return fmt.Errorf("save block %d: %w", bn, err)
	}
	seen.blocks[bn] = struct{}{}
	seen.newBlocks[bn] = r
	return nil
}

func saveTransaction(app core.App, tx *types.Transaction, blockBaseFee *big.Int, seen *batchSeen) (*big.Int, error) {
	hash := tx.Hash.Hex()
	if _, dup := seen.txs[hash]; dup {
		return nil, nil
	}

	txColl, err := utils.FindCollection(app, "transactions")
	if err != nil {
		return nil, err
	}
	r := core.NewRecord(txColl)
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
		r.Set("fee_usdc", utils.WeiToUSDC(feeWei))
	}
	if tx.GasUsed != nil && tx.EffectiveGasPrice != nil && blockBaseFee != nil {
		priorityPerGas := new(big.Int).Sub(tx.EffectiveGasPrice, blockBaseFee)
		if priorityPerGas.Sign() < 0 {
			priorityPerGas.SetInt64(0)
		}
		priorityFeeWei := new(big.Int).Mul(new(big.Int).SetUint64(*tx.GasUsed), priorityPerGas)
		r.Set("priority_fee_per_gas", priorityPerGas.String())
		r.Set("priority_fee_usdc", utils.WeiToUSDC(priorityFeeWei))
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
		return nil, fmt.Errorf("save transaction %s: %w", hash, err)
	}
	seen.txs[hash] = struct{}{}
	return feeWei, nil
}
