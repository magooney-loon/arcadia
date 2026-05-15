package repo

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

// CrosschainFilter holds optional filter criteria for crosschain events.
type CrosschainFilter struct {
	Protocol  string
	EventType string
	Sender    string
	Recipient string
	Direction string // "inbound" or "outbound"
}

// ListCrosschainEvents returns crosschain events matching the filter.
func ListCrosschainEvents(app core.App, f CrosschainFilter, limit, offset int) ([]*core.Record, error) {
	parts := []string{}
	params := map[string]any{}
	if f.Protocol != "" {
		parts = append(parts, "protocol = {:p}")
		params["p"] = f.Protocol
	}
	if f.EventType != "" {
		parts = append(parts, "event_type = {:et}")
		params["et"] = f.EventType
	}
	if f.Sender != "" {
		parts = append(parts, "sender = {:s}")
		params["s"] = f.Sender
	}
	if f.Recipient != "" {
		parts = append(parts, "recipient = {:r}")
		params["r"] = f.Recipient
	}
	if f.Direction == "inbound" {
		parts = append(parts, "destination_domain = 26")
	} else if f.Direction == "outbound" {
		parts = append(parts, "source_domain = 26 && destination_domain != 26")
	}
	filter := strings.Join(parts, " && ")
	return FindRecords(app, "crosschain_events", filter, "-block_number", limit, offset, params)
}
