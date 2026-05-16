package repo

import (
	"github.com/pocketbase/pocketbase/core"
)

// MetaValue returns the value for a key in indexer_meta, or "" if not found.
func MetaValue(app core.App, key string) (string, error) {
	r, err := LatestRecord(app, "indexer_meta", "key = {:k}", "", map[string]any{"k": key})
	if err != nil {
		return "", err
	}
	if r == nil {
		return "", nil
	}
	return r.GetString("value"), nil
}

// AllMeta returns all key-value pairs from indexer_meta.
func AllMeta(app core.App) (map[string]string, error) {
	records, err := FindRecords(app, "indexer_meta", "key != ''", "", 0, 0)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(records))
	for _, r := range records {
		out[r.GetString("key")] = r.GetString("value")
	}
	return out, nil
}
