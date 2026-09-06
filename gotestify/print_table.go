package gotestify

import (
	"encoding/json/v2"
	"fmt"
	"reflect"
	"strings"
	"text/tabwriter"
	"text/template"
)

// PrintTable renders data as a text table with the given columns. Columns are
// JSON field names: see createTable.
func PrintTable(data any, columns []string, opts ...json.Options) (string, error) {
	var line strings.Builder
	for _, col := range columns {
		line.WriteString("{{." + col + "}}\t")
	}
	tmpl, err := template.New("row").Parse(line.String())
	if err != nil {
		return "", err
	}

	table, err := createTable(data, opts...)
	if err != nil {
		return "", err
	}

	const padding = 1
	var res strings.Builder
	w := tabwriter.NewWriter(&res, 0, 2, padding, ' ', tabwriter.DiscardEmptyColumns|tabwriter.Debug)

	for _, row := range table {
		if err := tmpl.Execute(w, row); err != nil {
			return "", err
		}
		_, _ = fmt.Fprintln(w)
	}

	if err := w.Flush(); err != nil {
		return "", err
	}

	return res.String(), nil
}

// createTable round-trips data through JSON into a JSONArray, so that every row
// is a uniform map keyed by JSON field name and any custom codecs in opts apply.
// A value that is neither a slice nor an encoded document is wrapped in a
// single-element slice; nil yields an empty table.
func createTable(data any, opts ...json.Options) (JSONArray, error) {
	_, isDocument := data.(string)
	if data != nil && !isDocument && reflect.ValueOf(data).Kind() != reflect.Slice {
		data = []any{data}
	}

	return normalizeJSON[JSONArray](data, opts...)
}
