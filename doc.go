// Package gotestify extends github.com/stretchr/testify with assertions on
// JSON documents and tabular output. Its helpers take the same TestingT that
// testify's assert package accepts, follow the same (t, expected, actual)
// argument order, and return a bool the same way, so they sit naturally next
// to assert and require in a test.
//
// JSONEqual and JSONSubset compare two JSON documents and fail the test with
// a colorized diff. The expected side may contain {{...}} placeholders that
// match a value by shape ({{uuid}}, {{timestamp}}, {{any}}, or anything added
// with RegisterMatcher), capture a value for reuse, or interpolate previously
// captured values; see Compare for the details.
//
// PrintTable and TableEqual render a slice of values as a text table keyed
// by JSON field name and diff it cell by cell.
package gotestify
