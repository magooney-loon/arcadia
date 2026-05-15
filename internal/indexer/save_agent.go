package indexer

import (
	"fmt"
	"math/big"

	"github.com/enviodev/hypersync-client-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/utils"
)

func saveAgentRegistration(app core.App, log *types.Log) error {
	if log.Topic1 == nil || log.Topic2 == nil || log.TransactionHash == nil {
		return nil
	}
	zero := common.Hash{}
	if *log.Topic1 != zero {
		return nil
	}

	owner := utils.AddressFromTopic(log.Topic2)
	existing, err := app.FindRecordsByFilter("agents", "agent_address = {:a}", "", 1, 0, map[string]any{"a": owner})
	if err != nil {
		return fmt.Errorf("find agent %s: %w", owner, err)
	}
	if len(existing) > 0 {
		return nil
	}

	r := core.NewRecord(utils.MustCollection(app, "agents"))
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

	r := core.NewRecord(utils.MustCollection(app, "agent_jobs"))
	r.Set("job_id", jobID)
	r.Set("employer_address", utils.AddressFromTopic(log.Topic2))
	r.Set("worker_address", utils.AddressFromTopic(log.Topic3))
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
		r = core.NewRecord(utils.MustCollection(app, "agent_jobs"))
		r.Set("job_id", jobID)
		r.Set("status", "created")
	}
	update(r)
	if err := app.Save(r); err != nil {
		return fmt.Errorf("save agent job %s: %w", jobID, err)
	}
	return nil
}

func saveAgentJobFunded(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "funded")
		if log.Data != nil && len(*log.Data) >= 32 {
			r.Set("payment_usdc", utils.StablecoinHuman(utils.ReadBig(*log.Data, 0)))
		}
	})
}

func saveAgentJobSubmitted(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "submitted")
	})
}

func saveAgentJobCompleted(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "completed")
	})
}

func saveAgentJobRejected(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "rejected")
	})
}

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

func saveAgentJobExpired(app core.App, log *types.Log) error {
	return agentJobUpsert(app, log, func(r *core.Record) {
		r.Set("status", "expired")
	})
}
