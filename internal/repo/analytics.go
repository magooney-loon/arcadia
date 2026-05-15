package repo

import (
	"github.com/pocketbase/pocketbase/core"
)

// LatestSnapshot returns the most recent analytics snapshot for the given window.
func LatestSnapshot(app core.App, window string) (*core.Record, error) {
	return LatestRecord(app, "analytics_snapshots", "window = {:w}", "-snapshot_at", map[string]any{"w": window})
}

// SnapshotHistory returns all snapshots for a window, sorted by snapshot_at descending.
func SnapshotHistory(app core.App, window string, limit, offset int) ([]*core.Record, error) {
	return FindRecords(app, "analytics_snapshots", "window = {:w}", "-snapshot_at", limit, offset, map[string]any{"w": window})
}
