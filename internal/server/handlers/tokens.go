package handlers

// API_SOURCE

import (
	"net/http"
	"strconv"
	"strings"

	"arcadia/internal/repo"

	"github.com/pocketbase/pocketbase/core"
)

// API_DESC Token transfers — filterable by block, token symbol, sender, or receiver
// API_TAGS Transfers
func transfersHandler(c *core.RequestEvent) error {
	limit, offset := limitOffset(c)

	f := repo.TransferFilter{}
	if block := qp(c, "block", ""); block != "" {
		if bn, err := strconv.ParseInt(block, 10, 64); err == nil {
			f.BlockNumber = bn
		}
	}
	if token := qp(c, "token", ""); token != "" {
		f.TokenSymbol = strings.ToUpper(token)
	}
	if from := qp(c, "from", ""); from != "" {
		f.FromAddr = from
	}
	if to := qp(c, "to", ""); to != "" {
		f.ToAddr = to
	}

	records, err := repo.ListTransfers(c.App, f, limit, offset)
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
	search := qp(c, "search", "")

	records, err := repo.ListTokens(c.App, search, limit, offset)
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

	tokenRow, err := repo.TokenByAddress(c.App, lower)
	if err != nil || tokenRow == nil {
		return c.JSON(http.StatusNotFound, map[string]any{"error": "token not found"})
	}

	tokenAddr := strings.ToLower(tokenRow.GetString("token_address"))

	// Fetch recent transfers for this token
	transfers, err := repo.TransfersByTokenAddress(c.App, tokenAddr, 50, 0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]any{
		"token":     tokenRow.PublicExport(),
		"transfers": recordsToMaps(transfers),
	})
}
