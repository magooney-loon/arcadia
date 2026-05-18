package repo

import (
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// FindRecords is a thin wrapper around PocketBase's FindRecordsByFilter.
// It returns an error if the underlying call fails; callers decide how
// to handle empty results (len==0 is not an error).
func FindRecords(app core.App, collection, filter, sort string, limit, offset int, params ...map[string]any) ([]*core.Record, error) {
	var p map[string]any
	if len(params) > 0 {
		p = params[0]
	}
	records, err := app.FindRecordsByFilter(collection, filter, sort, limit, offset, p)
	if err != nil {
		return nil, fmt.Errorf("%s query: %w", collection, err)
	}
	return records, nil
}

// LatestRecord returns the single most recent record matching the filter,
// or nil if none found.
func LatestRecord(app core.App, collection, filter, sort string, params ...map[string]any) (*core.Record, error) {
	records, err := FindRecords(app, collection, filter, sort, 1, 0, params...)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil
}

// RecordMaps converts a slice of records to public-exported maps.
func RecordMaps(records []*core.Record) []map[string]any {
	out := make([]map[string]any, len(records))
	for i, r := range records {
		out[i] = r.PublicExport()
	}
	return out
}

// RowCount returns the number of rows in a table using COUNT(*).
func RowCount(app core.App, table string) (int, error) {
	var row struct {
		N int `db:"n"`
	}
	if err := app.DB().NewQuery("SELECT COUNT(*) AS n FROM " + table).One(&row); err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return row.N, nil
}

// CountWithFilter returns the number of rows in a table matching the supplied
// PocketBase-style filter. The filter string uses the same syntax accepted by
// FindRecordsByFilter (e.g. "block_number = {:bn} && from_addr = {:fa}") and
// the same {:param} placeholders. The "&&" and "||" operators are rewritten to
// SQL "AND"/"OR" before executing the COUNT(*) so the filter can be authored
// once and used by both list and count queries.
func CountWithFilter(app core.App, table, filter string, params map[string]any) (int, error) {
	sql := "SELECT COUNT(*) AS n FROM " + table
	if filter != "" {
		sql += " WHERE " + strings.ReplaceAll(strings.ReplaceAll(filter, "&&", "AND"), "||", "OR")
	}
	var row struct {
		N int `db:"n"`
	}
	q := app.DB().NewQuery(sql)
	if len(params) > 0 {
		q = q.Bind(dbx.Params(params))
	}
	if err := q.One(&row); err != nil {
		return 0, fmt.Errorf("count %s: %w", table, err)
	}
	return row.N, nil
}
