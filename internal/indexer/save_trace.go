package indexer

import (
	"fmt"
	"math/big"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/chain"
	"arcadia/internal/utils"
)

func routeLog(app core.App, log *types.Log, seen *batchSeen, edges map[edgeKey]*edgeDelta) (*big.Int, error) {
	if log.Topic0 == nil || log.Address == nil {
		return nil, nil
	}

	addr := *log.Address
	switch *log.Topic0 {
	case chain.TopicTransfer:
		if addr == chain.AddrAgentRegistry {
			return nil, saveAgentRegistration(app, log, seen)
		}
		return saveTransfer(app, log, seen, edges)

	case chain.TopicDepositForBurn:
		return nil, saveCCTPDepositForBurn(app, log, seen)

	case chain.TopicMintAndWithdraw:
		return nil, saveCCTPMintAndWithdraw(app, log, seen)

	case chain.TopicMessageReceived:
		return nil, saveCCTPMessageReceived(app, log, seen)

	case chain.TopicGatewayDeposited:
		return nil, saveGatewayDeposited(app, log, seen)

	case chain.TopicGatewayBurned:
		return nil, saveGatewayBurned(app, log, seen)

	case chain.TopicAttestationUsed:
		return nil, saveAttestationUsed(app, log, seen)

	case chain.TopicJobCreated:
		if addr == chain.AddrAgenticCommerce {
			return nil, saveAgentJobCreated(app, log, seen)
		}
	case chain.TopicJobFunded:
		if addr == chain.AddrAgenticCommerce {
			return nil, saveAgentJobFunded(app, log, seen)
		}
	case chain.TopicJobSubmitted:
		if addr == chain.AddrAgenticCommerce {
			return nil, saveAgentJobSubmitted(app, log, seen)
		}
	case chain.TopicJobCompleted:
		if addr == chain.AddrAgenticCommerce {
			return nil, saveAgentJobCompleted(app, log, seen)
		}
	case chain.TopicJobRejected:
		if addr == chain.AddrAgenticCommerce {
			return nil, saveAgentJobRejected(app, log, seen)
		}
	case chain.TopicPaymentReleased:
		if addr == chain.AddrAgenticCommerce {
			return nil, saveAgentJobPaid(app, log, seen)
		}
	case chain.TopicJobExpired:
		if addr == chain.AddrAgenticCommerce {
			return nil, saveAgentJobExpired(app, log, seen)
		}

	case chain.TopicTradeRecorded, chain.TopicMakerFunded, chain.TopicTakerFunded, chain.TopicTradeStatusChanged, chain.TopicFeesProcessed:
		if addr == chain.AddrFxEscrow {
			return nil, saveFxEvent(app, log, seen)
		}
	}
	return nil, nil
}

func saveTrace(app core.App, trace *types.Trace) error {
	coll, err := utils.FindCollection(app, "traces")
	if err != nil {
		return err
	}
	r := core.NewRecord(coll)
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
