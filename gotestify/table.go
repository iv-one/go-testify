package gotestify

import (
	"bytes"
	"encoding/json/v2"
	"strings"

	"github.com/stretchr/testify/assert"
)

// TableEqual compares a table of data with an expected table.
func TableEqual(t TestingT, expected string, data any, columns []string, jsonOpts ...json.Options) {
	table, err := PrintTable(data, columns, jsonOpts...)

	if err != nil {
		assert.NoError(t, err)
		return
	}

	opts := DefaultConsoleOptions()
	var buf bytes.Buffer

	tableLines := strings.Split(strings.TrimSpace(table), "\n")
	expectedLinesRaw := strings.Split(expected, "\n")
	expectedLines := []string{}

	for _, line := range expectedLinesRaw {
		l := strings.TrimSpace(line)
		if l != "" {
			expectedLines = append(expectedLines, l)
		}
	}

	matched := true
	n := MaxInt(len(tableLines), len(expectedLines))
	for i := range n {
		if i >= len(tableLines) {
			// missing line
			buf.WriteString(opts.Added.Begin + expectedLines[i] + opts.Added.End)
			matched = false
			continue
		}

		if i >= len(expectedLines) {
			// extra line
			buf.WriteString(opts.Removed.Begin + tableLines[i] + opts.Removed.End + "\n")
			matched = false
			continue
		}

		if tableLines[i] != expectedLines[i] {
			if !matchLine(expectedLines[i], tableLines[i], &buf, opts) {
				matched = false
			}
		} else {
			buf.WriteString(tableLines[i])
			buf.WriteString("\n")
		}
	}

	if !matched {
		t.Errorf("TableEqual failed:\n%s", buf.String())
	}
}

func matchLine(expected, actual string, buf *bytes.Buffer, opts *Options) bool {
	a := strings.Split(strings.TrimSuffix(expected, "|"), "|")
	b := strings.Split(strings.TrimSuffix(actual, "|"), "|")
	n := MaxInt(len(a), len(b))

	matched := true
	for i := range n {
		if i >= len(a) {
			// missing row
			buf.WriteString(opts.Removed.Begin + b[i] + opts.Removed.End + "|")
			matched = false
			continue
		}

		if i >= len(b) {
			// extra row
			buf.WriteString(opts.Added.Begin + a[i] + opts.Added.End + "|")
			matched = false
			continue
		}

		if a[i] != b[i] {
			buf.WriteString(opts.Changed.Begin + b[i] + opts.Changed.End + "|")
			matched = false
		} else {
			buf.WriteString(a[i] + "|")
		}
	}

	buf.WriteString("\n")

	return matched
}

// MaxInt returns the maximum of two integers.
func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
