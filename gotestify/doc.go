// Package gotestify provides testify-style assertion helpers for tests that
// compare JSON documents or tabular output.
//
// JSONEqual and IsSubsetJSON diff two JSON documents and report a colorized
// difference. The expected side may contain {{...}} placeholders that match a
// value by shape, capture it for reuse, or interpolate previously captured
// values; see CompareStr.
//
// PrintTable and TableEqual render a slice of values as a text table keyed by
// JSON field name and diff it cell by cell.

package gotestify
