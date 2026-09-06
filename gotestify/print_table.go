package gotestify

import (
	"encoding/json/v2"
	"fmt"
	"reflect"
	"strings"
	"text/tabwriter"
)

// PrintTable renders data as a text table with the given columns. Columns are
// JSON field names: see createTable.
func PrintTable(data any, columns []string, opts ...json.Options) (string, error) {
	const padding = 1
	var builder strings.Builder
	var res strings.Builder

	for _, col := range columns {
		builder.WriteString("{{." + col + "}}\t")
	}
	tmpl := builder.String()
	w := tabwriter.NewWriter(&res, 0, 2, padding, ' ', tabwriter.DiscardEmptyColumns|tabwriter.Debug)

	table, err := createTable(data, opts...)
	if err != nil {
		return "", err
	}

	for _, d := range table {
		line, err := CompileTemplate(tmpl, d)
		if err != nil {
			return "", err
		}

		_, _ = fmt.Fprintln(w, line)
	}

	if err := w.Flush(); err != nil {
		return "", err
	}

	return res.String(), nil
}

// createTable round-trips data through JSON into a JSONArray, so that every row
// is a uniform map keyed by JSON field name and any custom codecs in opts apply.
// A value that is not already a slice is wrapped in a single-element one.
func createTable(data any, opts ...json.Options) (JSONArray, error) {
	// data -> JSON -> map
	var jsonStr string
	switch {
	case data == nil:
		jsonStr = "[]"
	case isString(data):
		jsonStr = data.(string)
	case isSlice(data):
		jsonStr = Nice(data, opts...)
	default:
		jsonStr = Nice([]any{data}, opts...)
	}

	return ParseJSON[JSONArray]([]byte(jsonStr))
}

// isString checks if the provided data is a string using type assertion
func isString(data any) bool {
	_, ok := data.(string)
	return ok
}

// isSlice checks if the provided data is a slice
func isSlice(data any) bool {
	// Use reflect.TypeOf to get the type of the data
	t := reflect.TypeOf(data)
	// Check if the kind of the type is reflect.Slice
	return t.Kind() == reflect.Slice
}
