package handlers

// API_SOURCE

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/repo"
)

// API_DESC Recent cross-chain events (CCTP + Gateway)
// API_TAGS CrossChain
func crosschainHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	f := repo.CrosschainFilter{
		Protocol:  qp(c, "protocol", ""),
		EventType: qp(c, "event_type", ""),
		Sender:    qp(c, "sender", ""),
		Recipient: qp(c, "recipient", ""),
		Direction: qp(c, "direction", ""),
	}
	records, err := repo.ListCrosschainEvents(c.App, f, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	total, _ := repo.CountCrosschainEvents(c.App, f)
	return c.JSON(http.StatusOK, map[string]any{
		"events": recordsToMaps(records),
		"count":  len(records),
		"total":  total,
	})
}

// API_DESC StableFX trades — filterable by status, maker, taker, or quote_id
// API_TAGS FX
func fxHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	records, err := repo.ListFxSwaps(c.App, repo.FxSwapFilter{
		Status:  qp(c, "status", ""),
		Maker:   qp(c, "maker", ""),
		Taker:   qp(c, "taker", ""),
		QuoteID: qp(c, "quote_id", ""),
	}, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"trades": recordsToMaps(records),
		"count":  len(records),
	})
}
