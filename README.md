# go-testify

Assertion helpers for Go tests that deal with JSON — and, to a lesser extent, tables.

```sh
go get github.com/iv-one/go-testify
```

Requires Go 1.27 (uses the stdlib `encoding/json/v2` and `uuid` packages).

## The problem

You have a service method that returns a struct, and you want to assert on the whole thing:

```go
func (s *UserService) GetUser(ctx context.Context, id string) (*User, error)
```

Asserting field by field is verbose and, worse, it silently ignores the fields you forgot
to list — a new field with a wrong value slips through:

```go
u, err := svc.GetUser(ctx, id)
require.NoError(t, err)
assert.Equal(t, "alice@example.com", u.Email)
assert.Equal(t, "Alice", u.Name)
assert.NotZero(t, u.CreatedAt)  // and so on, forever
```

`assert.Equal` against a full literal struct is exhaustive, but then you have to
construct — and keep constructing — the server-generated values: ids, timestamps,
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

The actual value can be a struct, a pointer, a map, or a JSON string — anything that is
not a string is marshaled with `encoding/json/v2` first, so the assertion is written
against the same JSON your API actually serves.

The comparison is exhaustive: add a field to `User` and this test fails until you
acknowledge it. On failure you get one colorized diff of the whole document rather than a
list of unrelated assertion errors.

## Matchers

Placeholders go on the **expected** side.

| Placeholder     | Matches                                                   |
| --------------- | --------------------------------------------------------- |
| `{{any}}`       | any non-null value                                         |
| `{{timestamp}}` | a string parseable as RFC 3339                             |
| `{{uuid}}`      | a string parseable as a UUID                               |
| `{{name}}`      | anything — and binds the actual value to the variable `name` |

Add your own by registering an `Fn func(any) bool` in the `functions` map.

## Capturing and reusing values

A placeholder that is not a known matcher is a **capture variable**. It binds to whatever
the actual side holds, and every later use of that name must match the same value — which
is how you assert that two ids in a response refer to each other, without knowing either:

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
vars, err := gotestify.CollectVars(expected, actual, gotestify.DefaultJSONOptions())
require.NoError(t, err)
id := vars["uid"].(string)
```

## Partial matching

`IsSubsetJSON` accepts extra fields on the actual side — useful when you only care about
part of a large payload:

```go
gotestify.IsSubsetJSON(t, `{"id": "{{uuid}}", "name": "Alice"}`, resp)
```

For the raw result, `CompareStr` returns a `Difference` (`FullMatch`, `SubsetMatch`,
`SupersetMatch`, `NoMatch`) plus the rendered diff, which lets you write your own
assertion or inspect the comparison without failing a test.

## Custom encoding

Every helper that touches JSON takes trailing `...json.Options` (v2), so the codecs your
service uses apply to the value under test:

```go
rfc3339 := json.WithMarshalers(json.MarshalToFunc(
	func(enc *jsontext.Encoder, ts *timestamppb.Timestamp) error {
		return enc.WriteToken(jsontext.String(ts.AsTime().UTC().Format(time.RFC3339)))
	}))

gotestify.JSONEqual(t, expected, resp, rfc3339)
```

The same applies to `Nice`, `MarshalIndent`, `ParseJSON`, `CompileTemplateJSON`,
`IsSubsetJSON`, `PrintTable` and `TableEqual`.

## Compared to jsonassert

[`kinbiko/jsonassert`](https://github.com/kinbiko/jsonassert) solves the same core problem
and is the more mature, more focused library. The differences that matter when choosing:

- **Placeholders.** jsonassert has `<<PRESENCE>>` — the value exists, ignore it. This
  package adds *typed* matchers (`{{uuid}}`, `{{timestamp}}`), so a malformed id or a
  timestamp serialized in the wrong format fails instead of passing as "present".
- **Cross-field assertions.** Capture variables have no jsonassert equivalent. Asserting
  that two generated ids in a payload are the same id is the main reason to reach for
  this package.
- **Arrays.** jsonassert has `<<UNORDERED>>`; this package compares arrays strictly by
  index. If your payloads have non-deterministic array order, prefer jsonassert.
- **Formatting.** jsonassert builds the expected document with `Assertf` and
  `fmt.Sprintf` verbs. Here the expected document is a plain string and substitution
  happens through placeholders, which keeps `%` literals and `%d`-shaped content out of
  the picture.

## Gotchas

- `{{any}}` does **not** match `null`. Write `null` explicitly when you expect it.
- Raw `[]byte` is marshaled as a base64 string, not treated as a JSON document. Convert a
  response body with `string(b)` first.
- Numbers are compared by their literal representation, so `1` does not equal `1.0`. Set
  `Options.CompareNumbers` to change that.
- Arrays are compared by index; there is no unordered mode.
- A variable used as an object *key* must also be bound from a value position elsewhere
  in the document — a key-only variable resolves to nothing.
- Rendered diffs are meant to be read, not parsed. They are not valid JSON.
- `Options.SkipMatches` collapses the matching parts of a diff, which helps on large
  payloads.

## Tables

`TableEqual` renders a slice as a text table and diffs it cell by cell — handy for
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

## Credits

JSON diffing is based on [nsf/jsondiff](https://github.com/nsf/jsondiff), extended with
the placeholder and capture-variable machinery described above.

## License

MIT — see [LICENSE](LICENSE).
