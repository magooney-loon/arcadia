package utils

import (
	"fmt"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
)

// GetLastIndexedBlock reads the indexer cursor from the indexer_meta collection.
func GetLastIndexedBlock(app core.App) uint64 {
	records, err := app.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	if err != nil || len(records) == 0 {
		return 0
	}
	val, _ := strconv.ParseUint(records[0].GetString("value"), 10, 64)
	return val
}

// SetLastIndexedBlock persists the indexer cursor to the indexer_meta collection.
func SetLastIndexedBlock(app core.App, block uint64) error {
	records, err := app.FindRecordsByFilter("indexer_meta", "key = 'lastBlock'", "", 1, 0)
	if err != nil {
		return fmt.Errorf("find lastBlock cursor: %w", err)
	}

	var r *core.Record
	if len(records) > 0 {
		r = records[0]
	} else {
		c, ferr := FindCollection(app, "indexer_meta")
		if ferr != nil {
			return fmt.Errorf("find indexer_meta collection: %w", ferr)
		}
		r = core.NewRecord(c)
		r.Set("key", "lastBlock")
	}
	r.Set("value", strconv.FormatUint(block, 10))
	if err := app.Save(r); err != nil {
		return fmt.Errorf("save lastBlock cursor %d: %w", block, err)
	}
	return nil
}

// SetMetaValue upserts a key/value pair in the indexer_meta collection.
func SetMetaValue(app core.App, key, value string) error {
	records, err := app.FindRecordsByFilter("indexer_meta", "key = {:k}", "", 1, 0, map[string]any{"k": key})
	if err != nil {
		return fmt.Errorf("find meta key %q: %w", key, err)
	}
	var r *core.Record
	if len(records) > 0 {
		r = records[0]
	} else {
		c, ferr := FindCollection(app, "indexer_meta")
		if ferr != nil {
			return fmt.Errorf("find indexer_meta collection: %w", ferr)
		}
		r = core.NewRecord(c)
		r.Set("key", key)
	}
	r.Set("value", value)
	if err := app.Save(r); err != nil {
		return fmt.Errorf("save meta %q: %w", key, err)
	}
	return nil
}
