package indexer

import (
	"fmt"
	"math/big"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

func routeLog(app core.App, log *types.Log) (*big.Int, error) {
	if log.Topic0 == nil || log.Address == nil {
		return nil, nil
	}

	addr := *log.Address
	switch *log.Topic0 {
	case utils.TopicTransfer:
		if addr == utils.AddrAgentRegistry {
			return nil, saveAgentRegistration(app, log)
		}
		return saveTransfer(app, log)

	case utils.TopicDepositForBurn:
		return nil, saveCCTPDepositForBurn(app, log)

	case utils.TopicMintAndWithdraw:
		return nil, saveCCTPMintAndWithdraw(app, log)

	case utils.TopicMessageReceived:
		return nil, saveCCTPMessageReceived(app, log)

	case utils.TopicGatewayDeposited:
		return nil, saveGatewayDeposited(app, log)

	case utils.TopicGatewayBurned:
		return nil, saveGatewayBurned(app, log)

	case utils.TopicAttestationUsed:
		return nil, saveAttestationUsed(app, log)

	case utils.TopicJobCreated:
		if addr == utils.AddrAgenticCommerce {
			return nil, saveAgentJobCreated(app, log)
		}
	case utils.TopicJobFunded:
		if addr == utils.AddrAgenticCommerce {
			return nil, saveAgentJobFunded(app, log)
		}
	case utils.TopicJobSubmitted:
		if addr == utils.AddrAgenticCommerce {
			return nil, saveAgentJobSubmitted(app, log)
		}
	case utils.TopicJobCompleted:
		if addr == utils.AddrAgenticCommerce {
			return nil, saveAgentJobCompleted(app, log)
		}
	case utils.TopicJobRejected:
		if addr == utils.AddrAgenticCommerce {
			return nil, saveAgentJobRejected(app, log)
		}
	case utils.TopicPaymentReleased:
		if addr == utils.AddrAgenticCommerce {
			return nil, saveAgentJobPaid(app, log)
		}
	case utils.TopicJobExpired:
		if addr == utils.AddrAgenticCommerce {
			return nil, saveAgentJobExpired(app, log)
		}

	case utils.TopicTradeRecorded, utils.TopicMakerFunded, utils.TopicTakerFunded, utils.TopicTradeStatusChanged, utils.TopicFeesProcessed:
		if addr == utils.AddrFxEscrow {
			return nil, saveFxEvent(app, log)
		}
	}
	return nil, nil
}

func saveTrace(app core.App, trace *types.Trace) error {
	r := core.NewRecord(utils.MustCollection(app, "traces"))
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
