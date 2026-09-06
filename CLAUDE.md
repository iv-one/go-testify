# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`github.com/iv-one/go-testify` is a single-package library (`gotestify`) of testify-style assertion
helpers for **JSON** and **tabular** test output. It targets Go 1.27 and uses the new stdlib
`encoding/json/v2` + `encoding/json/jsontext` and the stdlib `uuid` package (both build without
`GOEXPERIMENT` on the go1.27.1 toolchain here).

## Commands

```sh
go build ./...
go test ./...
go test ./gotestify -run TestCompareAny -v   # single test
go vet ./...
go mod tidy
```

## Architecture

Everything lives in `gotestify/`. Note the two distinct options types: `json.Options` (stdlib v2,
controls encoding) and the package's own `*Options` (controls how a diff is rendered). Functions
that take both name the former `jsonOpts`.

### JSON helpers (`json.go`, `template.go`)

Every helper that touches JSON takes trailing `...json.Options` (v2) and forwards them via
`json.JoinOptions`, so a caller's custom codecs apply all the way down: `Nice`, `MarshalIndent`,
`ParseJSON[T]`, `CompileTemplateJSON`, `JSONEqual`, `IsSubsetJSON`, `PrintTable`, `TableEqual`.
Keep that convention when adding helpers. `CompileTemplate` is the one exception — it renders a
value that is already a decoded map, so it needs no options.

### JSON diffing (`jsoneq.go`)

A fork of [nsf/jsondiff](https://github.com/nsf/jsondiff) extended with **template variables**.

- Entry points: `JSONEqual` / `IsSubsetJSON` take a `TestingT` and any two values (strings pass
  through verbatim; anything else is marshaled via `encoding/json/v2`). Under them,
  `Compare` / `CompareStr` / `CompareStreams` return a `(Difference, renderedDiff)` pair —
  `FullMatch`, `SubsetMatch`, `SupersetMatch`, `NoMatch`, or an invalid-JSON variant.
  `IsSubsetJSON` accepts `FullMatch` or `SubsetMatch`; `JSONEqual` demands `FullMatch`.
- Comparison is a two-pass walk over `any` trees decoded with `UseNumber()`:
  1. `collectVars` populates `ctx.vars` by binding `{{name}}` placeholders on the _expected_ side
     to the corresponding actual values.
  2. `printDiff` re-walks and renders, resolving placeholders through `ctx.val`.
- Placeholder forms, all distinguished by `isVar` / `isFunc` / `isExpression`:
  - **Matcher functions** — `{{any}}`, `{{timestamp}}` (RFC3339), `{{uuid}}` — registered in the
    `functions` map; add new matchers there as `Fn func(any) bool`.
  - **Capture variables** — `{{x}}` binds whatever the actual side holds, first binding wins
    (`putVar` never overwrites).
  - **Expressions** — anything containing `{{`/`}}` that is not a bare var, e.g. `"{{x}}:{{y}}"`,
    is rewritten to `{{.x}}:{{.y}}` and rendered with `text/template` against the captured vars.
  - Map **keys** are substituted via `vkey` in `makeDualMapIterator`, so a captured id can be used
    as an object key. Note the ordering constraint: `vkey` resolves against `ctx.vars`, so a
    variable used *only* as a key never binds and renders as `<no value>`. It must also appear in
    a value position somewhere in the document.
- Traversal is uniform over arrays and objects through the `dualIterator` interface
  (`dualSliceIterator`, `dualMapIterator`), which yields aligned `(a, b, present-flags)` tuples.
  Any structural change should go through that interface, not per-kind branches.
- `Options` controls rendering: `Tag{Begin,End}` pairs wrap added/removed/changed/skipped spans.
  `DefaultConsoleOptions()` uses ANSI colors (what the assertion helpers use);
  `DefaultJSONOptions()` emits parseable-ish JSON. Rendered diffs are **not** valid JSON.
- `CollectVars` / `CollectVarsStream` expose pass 1 alone, for extracting ids out of a response.
- `ctx.val` **panics** on a bad expression (marked TODO in the source).

### Table assertions (`print_table.go`, `table.go`)

- `PrintTable(data, columns, jsonOpts...)` builds a `text/template` line (`{{.Col}}\t...`) per row and runs it
  through `tabwriter` with `DiscardEmptyColumns|tabwriter.Debug` — `Debug` is what produces the
  `|` separators the expected strings are written against. Rows are normalized by round-tripping
  the input through JSON into `JSONArray` (`ParseJSON[JSONArray]`), so **column names are JSON
  field names**, not Go field names; a non-slice value is wrapped in a one-element slice.
- `TableEqual(t, expected, data, columns, jsonOpts...)` renders the table, then diffs line-by-line and
  cell-by-cell (splitting on `|`), trimming whitespace on the expected side so tests can indent
  the raw string literal. Failures are reported as one colorized diff via `t.Errorf`.

### `slog.go`

`CaptureSlog(t)` swaps the process-global default logger for a buffer at `LevelDebug` and restores
it in `t.Cleanup`. Not safe under `t.Parallel`.

## Conventions

- Helpers accept the narrow local interfaces `TestingT` (`Errorf`/`Fatalf`) and `CleanupT`, never
  `*testing.T`, and call `Helper()` when the value implements `tHelper`.
- Assertion helpers report with `t.Errorf` and keep going; they do not fatal.
- Exported identifiers carry doc comments — match that when adding to the package.
