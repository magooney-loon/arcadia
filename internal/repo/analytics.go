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

// DeleteSnapshotsBefore deletes analytics snapshot rows whose snapshot_at is
// earlier than the cutoff (Unix seconds). Returns the number of rows deleted.
func DeleteSnapshotsBefore(app core.App, cutoff int64) (int64, error) {
	res, err := app.DB().NewQuery("DELETE FROM analytics_snapshots WHERE snapshot_at < {:cutoff}").
		Bind(map[string]any{"cutoff": cutoff}).Execute()
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
