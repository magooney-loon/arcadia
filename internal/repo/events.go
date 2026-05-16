package repo

import (
	"github.com/pocketbase/pocketbase/core"
)

// ErrorEventsSince returns indexer error events since the given timestamp.
func ErrorEventsSince(app core.App, since int64) ([]*core.Record, error) {
	return FindRecords(app, "indexer_events", "level = 'error' && created >= {:since}", "", 500, 0, map[string]any{"since": since})
}

// RecentBatchDones returns recent batch_done events sorted by created descending.
func RecentBatchDones(app core.App, since int64, limit int) ([]*core.Record, error) {
	return FindRecords(app, "indexer_events", "event = 'batch_done' && created >= {:since}", "-created", limit, 0, map[string]any{"since": since})
}

// DeleteEventsBefore deletes indexer events older than the given timestamp
// and returns the number of rows deleted.
func DeleteEventsBefore(app core.App, cutoff int64) (int64, error) {
	res, err := app.DB().NewQuery("DELETE FROM indexer_events WHERE timestamp < {:cutoff}").
		Bind(map[string]any{"cutoff": cutoff}).Execute()
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
