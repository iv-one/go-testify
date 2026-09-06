# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`github.com/iv-one/go-testify` is a single root package, `gotestify`, extending
stretchr/testify with assertions on **JSON** documents and **tabular** output. It targets Go
1.27 and uses the stdlib `encoding/json/v2`, `encoding/json/jsontext` and `uuid` packages (all
build without `GOEXPERIMENT` on the go1.27.1 toolchain here). The import path and the package
name differ on purpose (`go-testify` / `gotestify`), the same convention as
`hashicorp/go-multierror` -> `multierror`. testify is the only module dependency and is used
only by the tests.

## Commands

```sh
go build ./...
go test -race ./...
go test -race . -run TestCompareAny -v   # single test
go test -race . -run Example             # the runnable examples in example_test.go
go vet ./...
gofmt -l .                               # CI fails on any output
go mod tidy
```

CI (`.github/workflows/test.yml`) runs gofmt, vet, and `go test -race` on every push to main
and every pull request.

## Architecture

Note the two distinct options types: `json.Options` (stdlib v2, controls how arguments are
_marshaled_ before comparison) and the package's own `*DiffOptions` (controls how a diff is
rendered and compared). Functions that take both name the former `jsonOpts`.

### JSON helpers (`json.go`, `template.go`)

Every exported helper that takes a value to encode accepts trailing `...json.Options` and
passes them straight through: `PrettyJSON`, `JSONEqual`, `JSONSubset`, `PrintTable`,
`TableEqual`. Keep that convention when adding helpers.

The unexported `toJSON` / `normalizeJSON[T]` pair in `json.go` is the shared round-trip: a
string argument is taken as an already-encoded document, anything else is marshaled, and
`normalizeJSON` applies the options on the unmarshal leg too. Route new "accept a value or a
JSON string" code through them rather than re-checking for `string`. `marshalIndent`,
`parseJSON`, `compileTemplate` and `compileTemplateJSON` are deliberately unexported: this is
an assertion package, not a JSON utility package.

### JSON diffing (`jsoneq.go`)

A fork of [nsf/jsondiff](https://github.com/nsf/jsondiff) extended with **template variables**.

- Entry points: `JSONEqual` / `JSONSubset` take a `TestingT`, return `bool`, and accept any two
  values (strings pass through verbatim; anything else is marshaled). Under them, `Compare` /
  `CompareStr` / `CompareStreams` return a `(Difference, renderedDiff)` pair - `FullMatch`,
  `SubsetMatch`, `SupersetMatch`, `NoMatch`, `ExpressionError`, or an invalid-JSON variant.
  `JSONSubset` accepts `FullMatch` or `SubsetMatch`; `JSONEqual` demands `FullMatch`.
- Decoding is a hand-written token walk over `jsontext.Decoder` (`decodeDocument` /
  `decodeValue`) producing `map[string]any`, `[]any`, `string`, `bool`, `nil`, and `Number` -
  the number's literal text, the package's own type. This replaced v1 `encoding/json` with
  `UseNumber`: plain v2 `Unmarshal` into `any` yields `float64` and there is no v2 `UseNumber`
  option or `SkipFunc` hook. Gotcha inside the walk: a `jsontext.Token` is voided by the next
  decoder call, so an object key must be stringified before recursing into its value.
- Comparison is a two-pass walk over the decoded trees:
  1. `collectVars` populates `ctx.vars` by binding `{{name}}` placeholders on the _expected_ side
     to the corresponding actual values.
  2. `printDiff` re-walks and renders, resolving placeholders through `ctx.val`.
- Placeholder forms, all distinguished by `varName` / `isFunc` / `isExpression`:
  - **Matchers** - `{{any}}`, `{{timestamp}}` (RFC 3339), `{{uuid}}` - live in the `matchers`
    map behind `matchersMu`; `RegisterMatcher` is the public way in. Matchers see numbers as
    `Number`. `{{any}}` does not match `null` (the nil branch in `printDiff` runs first).
  - **Capture variables** - `{{x}}` binds whatever the actual side holds, first binding wins
    (`putVar` never overwrites).
  - **Expressions** - anything containing `{{`/`}}` that is not a bare var, e.g. `"{{x}}:{{y}}"`,
    is rewritten to `{{.x}}:{{.y}}` and rendered with `compileTemplate` against the captured
    vars. A render failure is recorded once in `ctx.err` and surfaces as `ExpressionError` (or
    an error from `CollectVars`); it must never panic.
  - Map **keys** are substituted via `vkey` in `makeDualMapIterator`. `vkey` resolves against
    `ctx.vars`, so a variable used _only_ as a key never binds and renders as `<no value>`; it
    must also appear in a value position somewhere in the document. `CollectVars` evaluates
    expressions only in keys - values are bound, not rendered, on that pass.
- All `context` methods use pointer receivers. Two of them were value receivers in the fork,
  which silently dropped `ctx.err`; keep them pointers.
- Traversal is uniform over arrays and objects through the `dualIterator` interface
  (`dualSliceIterator`, `dualMapIterator`), which yields aligned `(a, b, present-flags)` tuples;
  `collectPairVars` and `printCollectionDiff` are its two consumers. Any structural change should
  go through that interface, not per-kind branches.
- `DiffOptions` controls rendering: `Tag{Begin,End}` pairs wrap added/removed/changed/skipped
  spans. `ConsoleDiffOptions()` uses ANSI colors (what the assertion helpers use);
  `JSONDiffOptions()` emits parseable-ish JSON, used by the examples because its output is
  stable. Rendered diffs are **not** valid JSON.
- `CollectVars` / `CollectVarsStream` expose pass 1 alone, for extracting ids out of a response.

### Table assertions (`print_table.go`, `table.go`)

- `PrintTable(data, columns, jsonOpts...)` parses one `text/template` row (`{{.Col}}\t...`) and
  executes it per row straight into a `tabwriter` with `DiscardEmptyColumns|tabwriter.Debug` -
  `Debug` is what produces the `|` separators the expected strings are written against. Rows
  are normalized by `createTable` through `normalizeJSON[jsonArray]`, so **column names are JSON
  field names**, not Go field names; a non-slice value is wrapped in a one-element slice and
  `nil` yields an empty table.
- `TableEqual(t, expected, data, columns, jsonOpts...)` renders the table, then diffs
  line-by-line and cell-by-cell (splitting on `|`) through one shared `diffLists` walk, trimming
  whitespace on the expected side so tests can indent the raw string literal. Failures are
  reported as one colorized diff via `t.Errorf`; the return value is the outcome.

### `slog.go`

`CaptureSlog(t)` swaps the process-global default logger for a buffer at `LevelDebug` and restores
it in `t.Cleanup`. Not safe under `t.Parallel`.

### `example_test.go`

Runnable `Example*` functions for every exported helper; they are what pkg.go.dev shows and
they run under `go test`. They use a tiny `reporter` stand-in for `*testing.T` so they can print
an outcome. Add one when adding an exported helper.

## Conventions

- Helpers accept the narrow local interfaces `TestingT` (just `Errorf`, compatible with
  testify's `assert.TestingT`) and `CleanupT`, never `*testing.T`, and call `Helper()` when the
  value implements `tHelper`.
- Assertion helpers report with `t.Errorf`, keep going, and return `bool` like testify's
  `assert` functions; they do not fatal and they do not panic on test input.
- Argument order is `(t, expected, actual)` - testify's, not jsonassert's.
- Keep the exported surface small; unexport utilities that are not assertions or their direct
  configuration.
- Exported identifiers carry doc comments - match that when adding to the package.
