package gotestify

import (
	"bytes"
	"encoding/json/v2"
	"strings"
)

// TableEqual renders data as a table (see PrintTable) and compares it line by
// line and cell by cell with expected. Leading and trailing whitespace on each
// expected line is ignored, as are blank lines, so the literal can be indented
// to match the surrounding code.
func TableEqual(t TestingT, expected string, data any, columns []string, jsonOpts ...json.Options) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	table, err := PrintTable(data, columns, jsonOpts...)
	if err != nil {
		t.Errorf("TableEqual failed to render table: %s", err)
		return
	}

	var expectedLines []string
	for line := range strings.Lines(expected) {
		if line = strings.TrimSpace(line); line != "" {
			expectedLines = append(expectedLines, line)
		}
	}
	tableLines := strings.Split(strings.TrimSpace(table), "\n")

	opts := DefaultConsoleOptions()
	var buf bytes.Buffer
	matched := diffLists(expectedLines, tableLines, "\n", &buf, opts, func(expected, actual string) bool {
		return matchLine(expected, actual, &buf, opts)
	})

	if !matched {
		t.Errorf("TableEqual failed:\n%s", buf.String())
	}
}

// matchLine compares one table line cell by cell, marking differing cells as
// changed.
func matchLine(expected, actual string, buf *bytes.Buffer, opts *Options) bool {
	a := strings.Split(strings.TrimSuffix(expected, "|"), "|")
	b := strings.Split(strings.TrimSuffix(actual, "|"), "|")

	matched := diffLists(a, b, "|", buf, opts, func(_, actual string) bool {
		buf.WriteString(opts.Changed.Begin + actual + opts.Changed.End + "|")
		return false
	})
	buf.WriteString("\n")

	return matched
}

// diffLists walks expected and actual in lockstep, writing entries only in
// expected with the Added tag, entries only in actual with the Removed tag,
// and delegating same-position mismatches to changed. Each entry is followed
// by sep. It reports whether every entry matched.
func diffLists(expected, actual []string, sep string, buf *bytes.Buffer, opts *Options, changed func(expected, actual string) bool) bool {
	matched := true
	for i := range max(len(expected), len(actual)) {
		switch {
		case i >= len(actual):
			buf.WriteString(opts.Added.Begin + expected[i] + opts.Added.End + sep)
			matched = false
		case i >= len(expected):
			buf.WriteString(opts.Removed.Begin + actual[i] + opts.Removed.End + sep)
			matched = false
		case expected[i] != actual[i]:
			if !changed(expected[i], actual[i]) {
				matched = false
			}
		default:
			buf.WriteString(expected[i] + sep)
		}
	}
	return matched
}
