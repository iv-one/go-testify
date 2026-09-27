# go-testify

[![Go Quality score](https://raw.githubusercontent.com/iv-one/go-testify/quality-history/badges/score.svg)](https://github.com/iv-one/go-testify/blob/quality-history/report.txt)
[![Go Quality grade](https://raw.githubusercontent.com/iv-one/go-testify/quality-history/badges/grade.svg)](https://github.com/iv-one/go-testify/blob/quality-history/report.txt)

An extension of [`stretchr/testify`](https://github.com/stretchr/testify) for asserting on
JSON documents - and, to a lesser extent, tables.

```sh
go get github.com/iv-one/go-testify
```

```go
import "github.com/iv-one/go-testify" // package gotestify
```

Requires Go 1.27 (uses the stdlib `encoding/json/v2` and `uuid` packages). The only
dependency is testify itself, and only for the test suite.

The helpers accept the same `TestingT` that testify's `assert` package does, take arguments in
the same `(t, expected, actual)` order, and return a `bool` the same way, so they drop in next
to `assert` and `require` without ceremony.

## The problem

You have a service method that returns a struct, and you want to assert on the whole thing:

```go
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error)
```

Asserting field by field is verbose and, worse, it silently ignores the fields you forgot
to list - a new field with a wrong value slips through:

```go
u, err := svc.GetUser(ctx, id)
require.NoError(t, err)
assert.Equal(t, "alice@example.com", u.Email)
assert.Equal(t, "Alice", u.Name)
assert.NotZero(t, u.CreatedAt)  // and so on, forever
```

`assert.Equal` against a full literal struct is exhaustive, but then you have to
construct - and keep constructing - the server-generated values: ids, timestamps,
tenant references. Those change on every run.

## The fix

Write the expected value as the JSON you actually expect, and use a placeholder wherever
the value is generated rather than fixed:

```go
func TestGetUser(t *testing.T) {
	u, err := svc.GetUser(ctx, id)
	require.NoError(t, err)

	gotestify.JSONEqual(t, `{
		"id":         "{{uuid}}",
		"email":      "alice@example.com",
		"name":       "Alice",
		"team_id":    "{{uuid}}",
		"created_at": "{{timestamp}}",
		"updated_at": "{{timestamp}}"
	}`, u)
}
```

The actual value can be a struct, a pointer, a map, or a JSON string - anything that is
not a string is marshaled with `encoding/json/v2` first, so the assertion is written
against the same JSON your API actually serves.

The comparison is exhaustive: add a field to `User` and this test fails until you
acknowledge it. On failure you get one colorized diff of the whole document rather than a
list of unrelated assertion errors.

Runnable examples for every helper are in
[`example_test.go`](example_test.go) and on
[pkg.go.dev](https://pkg.go.dev/github.com/iv-one/go-testify).

## Matchers

Placeholders go on the **expected** side.

| Placeholder     | Matches                                                      |
| --------------- | ------------------------------------------------------------ |
| `{{any}}`       | any non-null value                                            |
| `{{timestamp}}` | a string parseable as RFC 3339                                |
| `{{uuid}}`      | a string parseable as a UUID                                  |
| `{{name}}`      | anything - and binds the actual value to the variable `name` |

Add your own with `RegisterMatcher`, typically from an `init` or `TestMain`:

```go
gotestify.RegisterMatcher("email", func(v any) bool {
	s, ok := v.(string)
	return ok && strings.Contains(s, "@")
})

gotestify.JSONEqual(t, `{"email": "{{email}}"}`, resp)
```

Numbers reach a matcher as `gotestify.Number`, the literal text of the JSON number.

## Capturing and reusing values

A placeholder that is not a registered matcher is a **capture variable**. It binds to
whatever the actual side holds, and every later use of that name must match the same
value - which is how you assert that two ids in a response refer to each other, without
knowing either:

```go
gotestify.JSONEqual(t, `{
	"user":  {"id": "{{uid}}", "name": "Alice"},
	"owner": {"id": "{{uid}}"},
	"self":  "/users/{{uid}}"
}`, resp)
```

That passes when `user.id`, `owner.id` and the `self` link agree, and fails when they
don't. `"{{x}}:{{y}}"` and similar are rendered with `text/template` against the captured
variables, so you can assert on composed strings.

To pull a generated id out of one response and feed it into the next request, use
`CollectVars`:

```go
vars, err := gotestify.CollectVars(expected, actual, gotestify.JSONDiffOptions())
require.NoError(t, err)
id := vars["uid"].(string)
```

## Partial matching

`JSONSubset` accepts extra fields on the actual side - useful when you only care about
part of a large payload:

```go
gotestify.JSONSubset(t, `{"id": "{{uuid}}", "name": "Alice"}`, resp)
```

For the raw result, `Compare` returns a `Difference` (`FullMatch`, `SubsetMatch`,
`SupersetMatch`, `NoMatch`, ...) plus the rendered diff, which lets you write your own
assertion or inspect the comparison without failing a test.

## Custom encoding

`JSONEqual`, `JSONSubset`, `PrintTable` and `TableEqual` take trailing `...json.Options`
(v2), so the codecs your service uses apply to the value under test. Say your API renders a
`Timestamp` type as RFC 3339 rather than its default encoding:

```go
rfc3339 := json.WithMarshalers(json.MarshalToFunc(
	func(enc *jsontext.Encoder, ts Timestamp) error {
		return enc.WriteToken(jsontext.String(ts.Time().UTC().Format(time.RFC3339)))
	}))

gotestify.JSONEqual(t, expected, resp, rfc3339)
```

Options shape how the arguments are _marshaled_ before comparison. The comparison itself
works on the resulting JSON text and is not affected by them.

## Gotchas

- `{{any}}` does **not** match `null`. Write `null` explicitly when you expect it.
- Raw `[]byte` is marshaled as a base64 string, not treated as a JSON document. Convert a
  response body with `string(b)` first.
- Numbers are compared by their literal text, so `1` does not equal `1.0` and large
  integers keep full precision. Set `DiffOptions.CompareNumbers` to relax that.
- Arrays are compared by index; there is no unordered mode.
- A variable used as an object _key_ must also be bound from a value position elsewhere
  in the document - a key-only variable resolves to nothing.
- A malformed `{{...}}` expression yields `ExpressionError` with the template error in the
  diff; it never panics.
- Rendered diffs are meant to be read, not parsed. They are not valid JSON.
- `DiffOptions.SkipMatches` collapses the matching parts of a diff, which helps on large
  payloads.

## Tables

`TableEqual` renders a slice as a text table and diffs it cell by cell - handy for
asserting on report or listing output. Columns are **JSON** field names:

```go
rows := []Row{{Name: "Alice", Age: 25}, {Name: "Bob", Age: 30}}

gotestify.TableEqual(t, `
	Alice |25 |
	Bob   |30 |
`, rows, []string{"name", "age"})
```

Leading and trailing whitespace on the expected side is trimmed, so the literal can be
indented to match the surrounding code. `PrintTable` returns the rendered table if you
want to assert on it yourself.

## Capturing logs

`CaptureSlog` redirects the default `slog` logger into a buffer for the duration of a test
and restores it on cleanup. It swaps a process-global, so it is not safe under
`t.Parallel`:

```go
buf := gotestify.CaptureSlog(t)
svc.DoWork(ctx)
assert.Contains(t, buf.String(), "work completed")
```

## Alternatives

- [`kinbiko/jsonassert`](https://github.com/kinbiko/jsonassert) - semantic JSON equality with
  `<<PRESENCE>>` and `<<UNORDERED>>` directives. Prefer it when array order is
  non-deterministic; this package compares arrays by index.
- [`swaggest/assertjson`](https://github.com/swaggest/assertjson) - testify-style JSON
  equality built on gojsondiff, with an `"<ignore-diff>"` placeholder and custom comparers.
- [`google/go-cmp`](https://github.com/google/go-cmp) - general Go value comparison with
  readable diffs and fine-grained options. Not JSON-aware; the right tool when you are
  comparing Go values rather than what they serialize to.
- [`alecthomas/assert`](https://github.com/alecthomas/assert) - a small, generics-based
  general assertion library that uses go-cmp for diffs, positioned as a reduced-surface
  alternative to testify itself.

What this package adds over the JSON-specific ones is typed matchers (`{{uuid}}`,
`{{timestamp}}`, your own via `RegisterMatcher`) and capture variables for asserting that two
generated values in a payload agree.

## Credits

JSON diffing is based on [nsf/jsondiff](https://github.com/nsf/jsondiff), extended with
the placeholder and capture-variable machinery described above.

## License

MIT - see [LICENSE](LICENSE).
