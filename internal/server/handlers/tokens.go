package handlers

// API_SOURCE

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// API_DESC Token transfers — filterable by block, token symbol, sender, or receiver
// API_TAGS Transfers
func transfersHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	if block := qp(c, "block", ""); block != "" {
		filter = "block_number = {:b}"
		params["b"] = block
	}
	if token := qp(c, "token", ""); token != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "token_symbol = {:sym}"
		params["sym"] = strings.ToUpper(token)
	}
	if from := qp(c, "from", ""); from != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "from_addr = {:f}"
		params["f"] = from
	}
	if to := qp(c, "to", ""); to != "" {
		if filter != "" {
			filter += " && "
		}
		filter += "to_addr = {:t}"
		params["t"] = to
	}

	records, err := c.App.FindRecordsByFilter("transfers", filter, "-block_number", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{
		"transfers": recordsToMaps(records),
		"count":     len(records),
	})
}

// API_DESC All discovered ERC-20 tokens with aggregated analytics
// API_TAGS Tokens
func tokensHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	filter := ""
	params := map[string]any{}

	if search := qp(c, "search", ""); search != "" {
		filter = "(LOWER(symbol) LIKE {:s} OR LOWER(name) LIKE {:s} OR LOWER(token_address) LIKE {:s})"
		params["s"] = "%" + strings.ToLower(search) + "%"
	}

	records, err := c.App.FindRecordsByFilter("token_analytics", filter, "-transfer_count", limit, offset, params)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"tokens": recordsToMaps(records),
		"count":  len(records),
	})
}

// API_DESC Token detail with recent transfers
// API_TAGS Tokens
func tokenDetailHandler(c *core.RequestEvent) error {
	address := c.Request.PathValue("address")
	if address == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "address required"})
	}

	lower := strings.ToLower(address)

	tokenRows, err := c.App.FindRecordsByFilter("token_analytics",
		"token_address = {:a}", "", 1, 0,
		map[string]any{"a": lower})
	if err != nil || len(tokenRows) == 0 {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "token not found"})
	}

	tokenAddr := strings.ToLower(tokenRows[0].GetString("token_address"))

	// Fetch recent transfers for this token
	transfers, err := c.App.FindRecordsByFilter("transfers",
		"token_address = {:a}", "-block_number", 50, 0,
		map[string]any{"a": tokenAddr})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"token":     tokenRows[0].PublicExport(),
		"transfers": recordsToMaps(transfers),
	})
}
