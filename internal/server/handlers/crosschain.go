package handlers

// API_SOURCE

import (
	"net/http"

	"github.com/pocketbase/pocketbase/core"
)

// API_DESC Recent cross-chain events (CCTP + Gateway)
// API_TAGS CrossChain
func crosschainHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	addFilter := func(clause string) {
		if filter != "" {
			filter += " && "
		}
		filter += clause
	}

	if proto := qp(c, "protocol", ""); proto != "" {
		addFilter("protocol = {:p}")
		params["p"] = proto
	}
	if et := qp(c, "event_type", ""); et != "" {
		addFilter("event_type = {:et}")
		params["et"] = et
	}
	if sender := qp(c, "sender", ""); sender != "" {
		addFilter("sender = {:s}")
		params["s"] = sender
	}
	if recipient := qp(c, "recipient", ""); recipient != "" {
		addFilter("recipient = {:r}")
		params["r"] = recipient
	}
	// direction=inbound  → USDC arriving on Arc (destination_domain = 26)
	// direction=outbound → USDC leaving Arc   (source_domain = 26, destination != 26)
	switch qp(c, "direction", "") {
	case "inbound":
		addFilter("destination_domain = 26")
	case "outbound":
		addFilter("source_domain = 26 && destination_domain != 26")
	}

	records, err := c.App.FindRecordsByFilter("crosschain_events", filter, "-block_number", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"events": recordsToMaps(records),
		"count":  len(records),
	})
}

// API_DESC StableFX trades — filterable by status, maker, taker, or quote_id
// API_TAGS FX
func fxHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	if status := qp(c, "status", ""); status != "" {
		filter = "status = {:s}"
		params["s"] = status
	}
	if maker := qp(c, "maker", ""); maker != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "maker = {:m}"
		params["m"] = maker
	}
	if taker := qp(c, "taker", ""); taker != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "taker = {:t}"
		params["t"] = taker
	}
	if quoteID := qp(c, "quote_id", ""); quoteID != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "quote_id = {:q}"
		params["q"] = quoteID
	}

	records, err := c.App.FindRecordsByFilter("fx_swaps", filter, "-block_number", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"trades": recordsToMaps(records),
		"count":  len(records),
	})
}
