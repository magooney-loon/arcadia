package indexer

import (
	"fmt"
	"math/big"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

func processBatch(app core.App, res *types.QueryResponse) error {
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
			getAcc(blk.Number.Uint64())
		}

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
				case utils.AddrUSDC:
					acc.totalUSDC.Add(acc.totalUSDC, amount)
					if amount.Cmp(acc.largestUSDC) > 0 {
						acc.largestUSDC.Set(amount)
					}
				case utils.AddrEURC:
					acc.totalEURC.Add(acc.totalEURC, amount)
				case utils.AddrUSYC:
					acc.totalUSYC.Add(acc.totalUSYC, amount)
				}
				if log.Topic0 != nil && *log.Topic0 == utils.TopicTransfer &&
					log.Address != nil && log.Topic1 != nil {
					if _, isStable := utils.KnownTokens[*log.Address]; isStable {
						fromAddr := common.BytesToAddress(log.Topic1.Bytes()[12:]).Hex()
						getAgentDelta(fromAddr).transferred.Add(getAgentDelta(fromAddr).transferred, amount)
					}
				}
			}
		}

		for _, trace := range res.Data.Traces {
			if trace.TransactionHash == nil || trace.BlockNumber == nil {
				continue
			}
			if err := saveTrace(txApp, &trace); err != nil {
				return err
			}
		}

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
		for i := 1; i < len(sortedBlocks); i++ {
			for j := i; j > 0 && sortedBlocks[j].num < sortedBlocks[j-1].num; j-- {
				sortedBlocks[j], sortedBlocks[j-1] = sortedBlocks[j-1], sortedBlocks[j]
			}
		}

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

			stats := core.NewRecord(utils.MustCollection(txApp, "block_stats"))
			stats.Set("block_number", bn)
			stats.Set("timestamp", blk.Timestamp.Unix())
			stats.Set("tx_count", acc.txCount)
			stats.Set("block_time_ms", bms)
			stats.Set("tps", tps)
			stats.Set("avg_fee_usdc", utils.WeiToUSDC(avgFee))
			stats.Set("total_fee_usdc", utils.WeiToUSDC(acc.totalFee))
			stats.Set("total_usdc_transferred", utils.StablecoinHuman(acc.totalUSDC))
			stats.Set("total_eurc_transferred", utils.StablecoinHuman(acc.totalEURC))
			stats.Set("total_usyc_transferred", utils.StablecoinHuman(acc.totalUSYC))
			stats.Set("unique_senders", len(acc.uniqueSenders))
			stats.Set("unique_receivers", len(acc.uniqueReceivers))
			stats.Set("new_contracts", acc.newContracts)
			stats.Set("largest_usdc_transfer", utils.StablecoinHuman(acc.largestUSDC))
			stats.Set("utilization_pct", utilPct)

			if err := txApp.Save(stats); err != nil {
				return fmt.Errorf("save block_stats %d: %w", bn, err)
			}
		}

		for addr, delta := range agentDeltas {
			if delta.txCount == 0 && delta.feeWei.Sign() == 0 && delta.transferred.Sign() == 0 {
				continue
			}
			agentRows, err := txApp.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, map[string]any{"a": addr})
			if err != nil || len(agentRows) == 0 {
				continue
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
