package gotestify

import (
	"bytes"
	gojson "encoding/json"
	"encoding/json/v2"
	"fmt"
	"io"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"uuid"
)

// based on https://github.com/nsf/jsondiff

type tHelper interface {
	Helper()
}

// TestingT is the subset of *testing.T the assertion helpers need.
type TestingT interface {
	Fatalf(format string, args ...any)
	Errorf(format string, args ...any)
}

// JSONEqual fails t unless expected and actual are the same JSON document.
// Either argument may be a JSON string or any value marshalable with
// encoding/json/v2 and jsonOpts. The expected side may use {{...}}
// placeholders: see Compare.
func JSONEqual(t TestingT, expected, actual any, jsonOpts ...json.Options) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	assertJSON(t, expected, actual, "FullMatch", func(d Difference) bool {
		return d == FullMatch
	}, jsonOpts...)
}

// IsSubsetJSON is JSONEqual that also accepts extra properties on the actual
// side.
func IsSubsetJSON(t TestingT, expected, actual any, jsonOpts ...json.Options) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}
	assertJSON(t, expected, actual, "FullMatch | SubsetMatch", func(d Difference) bool {
		return d == FullMatch || d == SubsetMatch
	}, jsonOpts...)
}

func assertJSON(t TestingT, expected, actual any, want string, accept func(Difference) bool, jsonOpts ...json.Options) {
	if h, ok := t.(tHelper); ok {
		h.Helper()
	}

	a, err := toJSON(expected, jsonOpts...)
	if err != nil {
		t.Errorf("failed to marshal expected argument: %s", err)
		return
	}
	b, err := toJSON(actual, jsonOpts...)
	if err != nil {
		t.Errorf("failed to marshal actual argument: %s", err)
		return
	}

	diff, res := Compare(a, b, DefaultConsoleOptions())
	if !accept(diff) {
		t.Errorf("expected %s, got %s \n%s", want, diff, res)
	}
}

// Difference is the difference type.
type Difference int

const (
	// FullMatch means provided arguments are deeply equal.
	FullMatch Difference = iota
	// SupersetMatch means first argument is a superset of a second argument.
	SupersetMatch
	// SubsetMatch means first argument is a subset of a second argument.
	SubsetMatch
	// NoMatch means there is no match.
	NoMatch
	// FirstArgIsInvalidJSON means the first argument is invalid JSON.
	FirstArgIsInvalidJSON
	// SecondArgIsInvalidJSON means the second argument is invalid JSON.
	SecondArgIsInvalidJSON
	// BothArgsAreInvalidJSON means both arguments are invalid JSON.
	BothArgsAreInvalidJSON
)

// String returns the string representation of the difference type.
func (d Difference) String() string {
	switch d {
	case FullMatch:
		return "FullMatch"
	case SupersetMatch:
		return "SupersetMatch"
	case SubsetMatch:
		return "SubsetMatch"
	case NoMatch:
		return "NoMatch"
	case FirstArgIsInvalidJSON:
		return "FirstArgIsInvalidJSON"
	case SecondArgIsInvalidJSON:
		return "SecondArgIsInvalidJSON"
	case BothArgsAreInvalidJSON:
		return "BothArgsAreInvalidJSON"
	}
	return "Invalid"
}

// Tag wraps a span of diff output, e.g. with ANSI color codes.
type Tag struct {
	Begin string
	End   string
}

// Options controls how Compare renders a diff. It is unrelated to json.Options.
type Options struct {
	Normal                Tag
	Added                 Tag
	Removed               Tag
	Changed               Tag
	Skipped               Tag
	SkippedArrayElement   func(n int) string
	SkippedObjectProperty func(n int) string
	Prefix                string
	Indent                string
	PrintTypes            bool
	ChangedSeparator      string
	// When provided, this function will be used to compare two numbers. By default numbers are compared using their
	// literal representation byte by byte.
	CompareNumbers func(a, b gojson.Number) bool
	// When true, only differences will be printed. By default, it will print the full json.
	SkipMatches bool
}

// SkippedArrayElement returns the skipped array element string.
func SkippedArrayElement(n int) string {
	if n == 1 {
		return "...skipped 1 array element..."
	}

	return "...skipped " + strconv.Itoa(n) + " array elements..."
}

// SkippedObjectProperty returns the skipped object property string.
func SkippedObjectProperty(n int) string {
	if n == 1 {
		return "...skipped 1 object property..."
	}

	return "...skipped " + strconv.Itoa(n) + " object properties..."
}

// DefaultJSONOptions provides a set of options in JSON format that are fully parseable.
// It returns the default JSON options.
func DefaultJSONOptions() *Options {
	return &Options{
		Added:            Tag{Begin: "\"prop-added\":{", End: "}"},
		Removed:          Tag{Begin: "\"prop-removed\":{", End: "}"},
		Changed:          Tag{Begin: "{\"changed\":[", End: "]}"},
		ChangedSeparator: ", ",
		Indent:           "    ",
	}
}

// DefaultConsoleOptions provides a set of options that are well suited for console output. Options
// use ANSI foreground color escape sequences to highlight changes.
// It returns the default console options.
func DefaultConsoleOptions() *Options {
	return &Options{
		Added:                 Tag{Begin: "\033[0;32m", End: "\033[0m"},
		Removed:               Tag{Begin: "\033[0;31m", End: "\033[0m"},
		Changed:               Tag{Begin: "\033[0;33m", End: "\033[0m"},
		Skipped:               Tag{Begin: "\033[0;90m", End: "\033[0m"},
		SkippedArrayElement:   SkippedArrayElement,
		SkippedObjectProperty: SkippedObjectProperty,
		ChangedSeparator:      " => ",
		Indent:                "  ",
	}
}

type context struct {
	opts    *Options
	level   int
	lastTag *Tag
	diff    Difference
	vars    map[string]any
}

func (ctx *context) compareNumbers(a, b gojson.Number) bool {
	if ctx.opts.CompareNumbers != nil {
		return ctx.opts.CompareNumbers(a, b)
	}

	return a == b
}

func (ctx *context) terminateTag(buf *bytes.Buffer) {
	if ctx.lastTag != nil {
		buf.WriteString(ctx.lastTag.End)
		ctx.lastTag = nil
	}
}

func (ctx *context) newline(buf *bytes.Buffer, s string) {
	buf.WriteString(s)
	if ctx.lastTag != nil {
		buf.WriteString(ctx.lastTag.End)
	}
	buf.WriteString("\n")
	buf.WriteString(ctx.opts.Prefix)
	buf.WriteString(strings.Repeat(ctx.opts.Indent, ctx.level))
	if ctx.lastTag != nil {
		buf.WriteString(ctx.lastTag.Begin)
	}
}

func writeKey(buf *bytes.Buffer, k string) {
	buf.WriteString(strconv.Quote(k))
	buf.WriteString(": ")
}

func (ctx *context) writeValue(buf *bytes.Buffer, v any, full bool) {
	switch vv := v.(type) {
	case bool:
		buf.WriteString(strconv.FormatBool(vv))
	case gojson.Number:
		buf.WriteString(string(vv))
	case string:
		buf.WriteString(strconv.Quote(vv))
	case []any:
		if full {
			if len(vv) == 0 {
				buf.WriteString("[")
			} else {
				ctx.level++
				ctx.newline(buf, "[")
			}
			for i, v := range vv {
				ctx.writeValue(buf, v, true)
				if i != len(vv)-1 {
					ctx.newline(buf, ",")
				} else {
					ctx.level--
					ctx.newline(buf, "")
				}
			}
			buf.WriteString("]")
		} else {
			buf.WriteString("[]")
		}
	case map[string]any:
		if full {
			if len(vv) == 0 {
				buf.WriteString("{")
			} else {
				ctx.level++
				ctx.newline(buf, "{")
			}

			keys := slices.Sorted(maps.Keys(vv))

			i := 0
			for _, k := range keys {
				v := vv[k]
				writeKey(buf, k)
				ctx.writeValue(buf, v, true)
				if i != len(vv)-1 {
					ctx.newline(buf, ",")
				} else {
					ctx.level--
					ctx.newline(buf, "")
				}
				i++
			}
			buf.WriteString("}")
		} else {
			buf.WriteString("{}")
		}
	default:
		buf.WriteString("null")
	}

	ctx.writeTypeMaybe(buf, v)
}

func (ctx *context) writeTypeMaybe(buf *bytes.Buffer, v any) {
	if ctx.opts.PrintTypes {
		buf.WriteString(" ")
		ctx.writeType(buf, v)
	}
}

func (ctx *context) writeType(buf *bytes.Buffer, v any) {
	switch v.(type) {
	case bool:
		buf.WriteString("(boolean)")
	case gojson.Number:
		buf.WriteString("(number)")
	case string:
		buf.WriteString("(string)")
	case []any:
		buf.WriteString("(array)")
	case map[string]any:
		buf.WriteString("(object)")
	default:
		buf.WriteString("(null)")
	}
}

func (ctx *context) writeMismatch(buf *bytes.Buffer, a, b any) {
	ctx.writeValue(buf, a, false)
	buf.WriteString(ctx.opts.ChangedSeparator)
	ctx.writeValue(buf, b, false)
}

func (ctx *context) tag(buf *bytes.Buffer, tag *Tag) {
	if ctx.lastTag == tag {
		return
	} else if ctx.lastTag != nil {
		buf.WriteString(ctx.lastTag.End)
	}
	buf.WriteString(tag.Begin)
	ctx.lastTag = tag
}

func (ctx *context) result(d Difference) {
	switch {
	case d == NoMatch:
		ctx.diff = NoMatch
	case d == SubsetMatch && ctx.diff != NoMatch:
		ctx.diff = SubsetMatch
	case d == SupersetMatch && ctx.diff != NoMatch:
		ctx.diff = SupersetMatch
	case ctx.diff != NoMatch && ctx.diff != SupersetMatch:
		ctx.diff = FullMatch
	}
}

func (ctx *context) printMismatch(buf *bytes.Buffer, a, b any) {
	ctx.tag(buf, &ctx.opts.Changed)
	ctx.writeMismatch(buf, a, b)
}

func (ctx *context) printSkipped(buf *bytes.Buffer, n *int, strfunc func(n int) string, last bool) {
	if *n == 0 || strfunc == nil {
		return
	}
	ctx.tag(buf, &ctx.opts.Skipped)
	buf.WriteString(strfunc(*n))
	if !last {
		ctx.tag(buf, &ctx.opts.Normal)
		ctx.newline(buf, ",")
	}
	*n = 0
}

func (ctx *context) finalize(buf *bytes.Buffer) string {
	ctx.terminateTag(buf)
	return buf.String()
}

type collectionConfig struct {
	open    string
	close   string
	skipped func(n int) string
	value   any
}

type dualIterator interface {
	clone() dualIterator
	count() int
	next() (a any, aOK bool, b any, bOK bool, i int)
	key(buf *bytes.Buffer)
}

type dualSliceIterator struct {
	a       []any
	b       []any
	max     int
	current int
}

func (it *dualSliceIterator) clone() dualIterator {
	res := *it
	return &res
}

func (it *dualSliceIterator) count() int {
	return it.max
}

func (it *dualSliceIterator) next() (a any, aOK bool, b any, bOK bool, i int) {
	it.current++
	i = it.current
	if i <= it.max {
		if i < len(it.a) {
			a = it.a[i]
			aOK = true
		}
		if i < len(it.b) {
			b = it.b[i]
			bOK = true
		}
	} else {
		i = -1
	}
	return
}

func (it *dualSliceIterator) key(_ *bytes.Buffer) {
	// noop
}

type dualMapIterator struct {
	a       map[string]any
	b       map[string]any
	keys    []string
	current int
}

func (it *dualMapIterator) clone() dualIterator {
	res := *it
	return &res
}

func (it *dualMapIterator) count() int {
	return len(it.keys)
}

func (it *dualMapIterator) next() (a any, aOK bool, b any, bOK bool, i int) {
	it.current++
	i = it.current
	if i < len(it.keys) {
		key := it.keys[i]
		a, aOK = it.a[key]
		b, bOK = it.b[key]
	} else {
		i = -1
	}
	return
}

func (it *dualMapIterator) key(buf *bytes.Buffer) {
	writeKey(buf, it.keys[it.current])
}

func (ctx context) vkey(k string) string {
	kv := ctx.val(k)
	if s, ok := kv.(string); ok {
		return s
	}
	return k
}

func (ctx context) makeDualMapIterator(ax, bx map[string]any) dualIterator {
	a := make(map[string]any)
	b := make(map[string]any)

	for k, v := range ax {
		a[ctx.vkey(k)] = v
	}

	for k, v := range bx {
		b[ctx.vkey(k)] = v
	}

	keysMap := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		keysMap[k] = struct{}{}
	}
	for k := range b {
		keysMap[k] = struct{}{}
	}
	return &dualMapIterator{
		a:       a,
		b:       b,
		keys:    slices.Sorted(maps.Keys(keysMap)),
		current: -1,
	}
}

func makeDualSliceIterator(a, b []any) dualIterator {
	n := max(len(b), len(a))
	return &dualSliceIterator{
		a:       a,
		b:       b,
		max:     n,
		current: -1,
	}
}

func (ctx *context) collectDiffs(it dualIterator) (diffs []string, last int) {
	ctx.level++
	last = -1
	for {
		a, aok, b, bok, i := it.next()
		if i == -1 {
			break
		}
		var diff string
		if aok && bok {
			diff = ctx.printDiff(a, b)
		}
		if len(diff) > 0 || aok != bok {
			last = i
		}
		diffs = append(diffs, diff)
	}
	ctx.level--
	return
}

func (ctx *context) printCollectionDiff(cfg *collectionConfig, it dualIterator) string {
	var buf bytes.Buffer
	diffs, lastDiff := ctx.collectDiffs(it.clone())
	if ctx.opts.SkipMatches && lastDiff == -1 {
		// no diffs
		return ""
	}

	// some diffs or empty collection
	ctx.tag(&buf, &ctx.opts.Normal)
	if it.count() == 0 {
		buf.WriteString(cfg.open)
		buf.WriteString(cfg.close)
		ctx.writeTypeMaybe(&buf, cfg.value)
		return ctx.finalize(&buf)
	}
	// else
	ctx.level++
	ctx.newline(&buf, cfg.open)

	noDiffSpan := 0
	for {
		va, aok, vb, bok, i := it.next()
		equals := true
		switch {
		case aok && bok:
			diff := diffs[i]
			if len(diff) > 0 {
				equals = false
				ctx.printSkipped(&buf, &noDiffSpan, cfg.skipped, false)
				it.key(&buf)
				buf.WriteString(diff)
			}
		case aok:
			equals = false
			ctx.printSkipped(&buf, &noDiffSpan, cfg.skipped, false)
			ctx.tag(&buf, &ctx.opts.Removed)
			it.key(&buf)
			ctx.writeValue(&buf, va, true)
			ctx.result(SupersetMatch)
		case bok:
			equals = false
			ctx.printSkipped(&buf, &noDiffSpan, cfg.skipped, false)
			ctx.tag(&buf, &ctx.opts.Added)
			it.key(&buf)
			ctx.writeValue(&buf, vb, true)
			ctx.result(SubsetMatch)
		}
		if ctx.opts.SkipMatches && equals {
			noDiffSpan++
		}

		wroteItem := !ctx.opts.SkipMatches || !equals
		willWriteMoreItems :=
			(ctx.opts.SkipMatches && i < lastDiff) ||
				(ctx.opts.SkipMatches && cfg.skipped != nil && lastDiff < it.count()-1) ||
				(!ctx.opts.SkipMatches && i < it.count()-1)

		if wroteItem && willWriteMoreItems {
			ctx.tag(&buf, &ctx.opts.Normal)
			ctx.newline(&buf, ",")
		}
		if i == it.count()-1 {
			// we're done
			ctx.printSkipped(&buf, &noDiffSpan, cfg.skipped, true)
			ctx.level--
			ctx.tag(&buf, &ctx.opts.Normal)
			ctx.newline(&buf, "")
			break
		}
	}

	buf.WriteString(cfg.close)
	ctx.writeTypeMaybe(&buf, cfg.value)
	return ctx.finalize(&buf)
}

func (ctx *context) val(x any) any {
	if isFunc(x) {
		return x
	}

	if isExpression(x) {
		res, err := ctx.evalExpression(x)
		if err != nil {
			// TODO: improve error handling
			panic(err)
		}

		return res
	}
	return x
}

func (ctx *context) printDiff(ai, bi any) string {
	var buf bytes.Buffer
	a := ctx.val(ai)
	b := ctx.val(bi)

	if a == nil || b == nil {
		// either is nil, means there are just two cases:
		// 1. both are nil => match
		// 2. one of them is nil => mismatch
		if a == nil && b == nil {
			// match
			if !ctx.opts.SkipMatches {
				ctx.tag(&buf, &ctx.opts.Normal)
				ctx.writeValue(&buf, a, false)
				ctx.result(FullMatch)
			}
		} else {
			// mismatch
			ctx.printMismatch(&buf, a, b)
			ctx.result(NoMatch)
		}
		return ctx.finalize(&buf)
	}

	if isFunc(a) {
		if evalFunction(a, b) {
			ctx.tag(&buf, &ctx.opts.Normal)
			ctx.writeValue(&buf, a, true)
			return ctx.finalize(&buf)
		}
	}

	ka := reflect.TypeOf(a).Kind()
	kb := reflect.TypeOf(b).Kind()
	if ka != kb {
		// Go type does not match, this is definitely a mismatch since
		// we parse JSON into interface{}
		ctx.printMismatch(&buf, a, b)
		ctx.result(NoMatch)
		return ctx.finalize(&buf)
	}

	// big switch here handles type-specific mismatches and returns if that's the case
	// buf if control flow goes past through this switch, it's a match
	// NOTE: ka == kb at this point
	switch ka {
	case reflect.Bool:
		if a.(bool) != b.(bool) {
			ctx.printMismatch(&buf, a, b)
			ctx.result(NoMatch)
			return ctx.finalize(&buf)
		}
	case reflect.String:
		// string can be a json.Number here too (because it's a string type)
		switch aa := a.(type) {
		case gojson.Number:
			bb, ok := b.(gojson.Number)
			if !ok || !ctx.compareNumbers(aa, bb) {
				ctx.printMismatch(&buf, a, b)
				ctx.result(NoMatch)
				return ctx.finalize(&buf)
			}
		case string:
			bb, ok := b.(string)
			if !ok || aa != bb {
				ctx.printMismatch(&buf, a, b)
				ctx.result(NoMatch)
				return ctx.finalize(&buf)
			}
		}
	case reflect.Slice:
		sa, sb := a.([]any), b.([]any)
		return ctx.printCollectionDiff(&collectionConfig{
			open:    "[",
			close:   "]",
			skipped: ctx.opts.SkippedArrayElement,
			value:   a,
		}, makeDualSliceIterator(sa, sb))
	case reflect.Map:
		ma, mb := a.(map[string]any), b.(map[string]any)
		return ctx.printCollectionDiff(&collectionConfig{
			open:    "{",
			close:   "}",
			skipped: ctx.opts.SkippedObjectProperty,
			value:   a,
		}, ctx.makeDualMapIterator(ma, mb))
	}
	if !ctx.opts.SkipMatches {
		ctx.tag(&buf, &ctx.opts.Normal)
		ctx.writeValue(&buf, a, true)
		ctx.result(FullMatch)
	}
	return ctx.finalize(&buf)
}

// Compare compares two JSON documents using given options. Returns difference type and
// a string describing differences.
//
// *FullMatch* means provided arguments are deeply equal.
//
// *SupersetMatch* means first argument is a superset of a second argument. In
// this context being a superset means that for each object or array in the
// hierarchy which don't match exactly, it must be a superset of another one.
// For example:
//
//	{"a": 123, "b": 456, "c": [7, 8, 9]}
//
// Is a superset of:
//
//	{"a": 123, "c": [7, 8]}
//
// *SubsetMatch* means first argument is a subset of a second argument. In
// this context being a subset means that for each object or array in the
// hierarchy which don't match exactly, it must be a subset of another one.
// For example:
//
//	{"a": 123, "c": [7, 8]}
//
// Is a subset of:
//
//	{"a": 123, "b": 456, "c": [7, 8, 9]}
//
// *NoMatch* means there is no match.
//
// The rest of the difference types mean that one of or both JSON documents are
// invalid JSON.
//
// Returned string uses a format similar to pretty printed JSON to show the
// human-readable difference between provided JSON documents. It is important
// to understand that returned format is not a valid JSON and is not meant
// to be machine readable.
//
// Strings on the a side may be placeholders: {{any}}, {{timestamp}} and
// {{uuid}} match a value by shape; any other {{name}} captures the b value and
// must match it wherever the name recurs; "{{x}}:{{y}}" and similar are
// rendered with text/template against the captured values.
//
// Both documents are decoded with encoding/json (v1) so that numbers keep
// their literal text; json.Options passed to JSONEqual and friends affect
// only how the arguments are marshaled, not this comparison.
func Compare(a, b []byte, opts *Options) (Difference, string) {
	return CompareStreams(bytes.NewReader(a), bytes.NewReader(b), opts)
}

// CompareStr is Compare for string documents.
func CompareStr(a, b string, opts *Options) (Difference, string) {
	return CompareStreams(strings.NewReader(a), strings.NewReader(b), opts)
}

// CompareStreams compares two JSON documents streamed by the specified readers.
// See the documentation for `Compare` for a description of the input options and return values.
func CompareStreams(a, b io.Reader, opts *Options) (Difference, string) {
	av, bv, diff, err := decodePair(a, b)
	if err != nil {
		return diff, err.Error()
	}

	ctx := context{opts: opts}
	ctx.collectVars(av, bv)
	return ctx.diff, ctx.printDiff(av, bv)
}

// CollectVars collects the variables from two JSON documents using given options.
func CollectVars(a, b string, opts *Options) (map[string]any, error) {
	return CollectVarsStream(strings.NewReader(a), strings.NewReader(b), opts)
}

// CollectVarsStream collects the variables from two JSON documents streamed by the specified readers using given options.
func CollectVarsStream(a, b io.Reader, opts *Options) (map[string]any, error) {
	av, bv, _, err := decodePair(a, b)
	if err != nil {
		return nil, err
	}

	ctx := context{opts: opts}
	ctx.collectVars(av, bv)
	return ctx.vars, nil
}

// decodePair decodes two JSON documents into generic values, keeping numbers
// as their literal text. On failure it also reports which side was invalid.
func decodePair(a, b io.Reader) (av, bv any, diff Difference, err error) {
	da := gojson.NewDecoder(a)
	da.UseNumber()
	db := gojson.NewDecoder(b)
	db.UseNumber()
	errA := da.Decode(&av)
	errB := db.Decode(&bv)
	switch {
	case errA != nil && errB != nil:
		return nil, nil, BothArgsAreInvalidJSON, fmt.Errorf("invalid jsons:\na: %w\nb: %w", errA, errB)
	case errA != nil:
		return nil, nil, FirstArgIsInvalidJSON, fmt.Errorf("invalid json:\na: %w", errA)
	case errB != nil:
		return nil, nil, SecondArgIsInvalidJSON, fmt.Errorf("invalid json:\nb: %w", errB)
	}
	return av, bv, FullMatch, nil
}

func asStr(v any) string {
	s, _ := v.(string)
	return s
}

// varName returns the name of a variable placeholder. A variable is a string
// of the form {{name}} and nothing else: "{{a}}:{{b}}" is an expression, not
// a variable.
func varName(v any) (string, bool) {
	s := asStr(v)
	if len(s) < 4 || !strings.HasPrefix(s, "{{") || !strings.HasSuffix(s, "}}") {
		return "", false
	}
	if strings.Index(s, "}}") != len(s)-2 || strings.LastIndex(s, "{{") != 0 {
		return "", false
	}
	return s[2 : len(s)-2], true
}

func isVar(v any) bool {
	_, ok := varName(v)
	return ok
}

// Fn is a matcher: it reports whether an actual value satisfies a placeholder
// such as {{uuid}}.
type Fn func(x any) bool

var functions = map[string]Fn{
	"any":       anyFunc,
	"timestamp": isTimestamp,
	"uuid":      isUUID,
}

// isFunc reports whether v is a placeholder naming a registered matcher.
func isFunc(v any) bool {
	name, ok := varName(v)
	if !ok {
		return false
	}
	_, ok = functions[name]
	return ok
}

func isExpression(v any) bool {
	s := asStr(v)
	return strings.Contains(s, "{{") && strings.Contains(s, "}}")
}

// evalExpression renders an expression such as "{{a}}:{{b}}" against the
// variables collected so far.
func (ctx *context) evalExpression(v any) (string, error) {
	if !isExpression(v) {
		return "", fmt.Errorf("not an expression: %v", v)
	}

	return CompileTemplate(strings.ReplaceAll(asStr(v), "{{", "{{."), ctx.vars)
}

// evalFunction applies the matcher named by placeholder a to b.
func evalFunction(a, b any) bool {
	name, ok := varName(a)
	if !ok {
		return false
	}
	fn, ok := functions[name]
	return ok && fn(b)
}

func anyFunc(_ any) bool {
	return true
}

func isTimestamp(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	_, err := time.ParseInLocation(time.RFC3339, s, time.UTC)
	return err == nil
}

func isUUID(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	_, err := uuid.Parse(s)
	return err == nil
}

func (ctx *context) putVar(k string, v any) {
	if ctx.vars == nil {
		ctx.vars = make(map[string]any)
	}

	// put only if not exists
	if _, ok := ctx.vars[k]; !ok {
		ctx.vars[k] = v
	}
}

// parseAndCollectVar binds a to b when a is a capture variable (a placeholder
// that is not a matcher).
func (ctx *context) parseAndCollectVar(a, b any) {
	if name, ok := varName(a); ok && !isFunc(a) {
		ctx.putVar(name, b)
	}
}

func (ctx *context) collectVars(a, b any) {
	if a == nil || b == nil {
		if a == nil && b == nil {
			// match
			return
		}

		ctx.parseAndCollectVar(a, b)
		return
	}

	ka := reflect.TypeOf(a).Kind()
	kb := reflect.TypeOf(b).Kind()
	if ka != kb {
		ctx.parseAndCollectVar(a, b)
		return
	}

	// big switch here handles type-specific mismatches and returns if that's the case
	// buf if control flow goes past through this switch, it's a match
	// NOTE: ka == kb at this point
	switch ka {
	case reflect.String:
		ctx.parseAndCollectVar(a, b)
		return
	case reflect.Slice:
		sa, sb := a.([]any), b.([]any)
		ctx.collectPairVars(makeDualSliceIterator(sa, sb))
		return
	case reflect.Map:
		ma, mb := a.(map[string]any), b.(map[string]any)
		ctx.collectPairVars(ctx.makeDualMapIterator(ma, mb))
	}
}

// collectPairVars collects variables from every aligned pair the iterator
// yields, for arrays and objects alike.
func (ctx *context) collectPairVars(it dualIterator) {
	for {
		a, aok, b, bok, i := it.next()
		if i == -1 {
			break
		}

		if aok && bok {
			ctx.collectVars(a, b)
		}
	}
}
