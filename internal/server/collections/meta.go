package collections

import (
	"github.com/pocketbase/pocketbase/core"
)

// metaCollection stores indexer cursor state (key/value pairs).
func metaCollection(app core.App) error {
	if collectionExists(app, "indexer_meta") {
		return nil
	}
	c := core.NewBaseCollection("indexer_meta")
	c.Fields.Add(&core.TextField{Name: "key", Required: true, Max: 100})
	c.Fields.Add(&core.TextField{Name: "value", Required: false, Max: 500})
	c.AddIndex("idx_meta_key", true, "key", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created indexer_meta collection")
	return nil
}

// indexerEventsCollection stores durable indexer lifecycle, progress, and error events.
func indexerEventsCollection(app core.App) error {
	if collectionExists(app, "indexer_events") {
		return nil
	}
	c := core.NewBaseCollection("indexer_events")
	c.Fields.Add(&core.NumberField{Name: "timestamp", Required: true})
	c.Fields.Add(&core.SelectField{
		Name:   "level",
		Values: []string{"debug", "info", "warn", "error"},
	})
	c.Fields.Add(&core.TextField{Name: "event", Required: true, Max: 80})
	c.Fields.Add(&core.TextField{Name: "message", Required: false, Max: 500})
	c.Fields.Add(&core.NumberField{Name: "attempt"})
	c.Fields.Add(&core.NumberField{Name: "batch"})
	c.Fields.Add(&core.NumberField{Name: "block"})
	c.Fields.Add(&core.NumberField{Name: "tip"})
	c.Fields.Add(&core.NumberField{Name: "lag"})
	c.Fields.Add(&core.NumberField{Name: "duration_ms"})
	c.Fields.Add(&core.NumberField{Name: "blocks"})
	c.Fields.Add(&core.NumberField{Name: "transactions"})
	c.Fields.Add(&core.NumberField{Name: "logs"})
	c.Fields.Add(&core.TextField{Name: "error", Required: false, Max: 1000})
	c.AddIndex("idx_indexer_events_ts", false, "timestamp", "")
	c.AddIndex("idx_indexer_events_level", false, "level", "")
	c.AddIndex("idx_indexer_events_event", false, "event", "")
	c.ViewRule = nil
	if err := app.Save(c); err != nil {
		return err
	}
	app.Logger().Info("Created indexer_events collection")
	return nil
}
