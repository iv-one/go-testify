package gotestify_test

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/iv-one/go-testify"
)

// reporter stands in for *testing.T so the examples can print their outcome.
// In a real test you pass t directly.
type reporter struct{ cleanups []func() }

func (r *reporter) Errorf(format string, args ...any) { fmt.Printf(format+"\n", args...) }
func (r *reporter) Cleanup(f func())                  { r.cleanups = append(r.cleanups, f) }

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	TeamID    string    `json:"team_id"`
	CreatedAt time.Time `json:"created_at"`
}

func getUser() *User {
	return &User{
		ID:        "0193f2a1-4c3b-7a91-b8e2-1f5d9c7a3e40",
		Email:     "alice@example.com",
		TeamID:    "0193f2a1-4c3b-7a91-b8e2-1f5d9c7a3e41",
		CreatedAt: time.Now().UTC(),
	}
}

func ExampleJSONEqual() {
	var t reporter
	user := getUser()

	ok := gotestify.JSONEqual(&t, `{
		"id":         "{{uuid}}",
		"email":      "alice@example.com",
		"team_id":    "{{uuid}}",
		"created_at": "{{timestamp}}"
	}`, user)

	fmt.Println(ok)
	// Output: true
}

func ExampleJSONEqual_captureVariables() {
	var t reporter
	resp := `{
		"user":  {"id": "u_1", "name": "Alice"},
		"owner": {"id": "u_1"},
		"self":  "/users/u_1"
	}`

	// {{uid}} binds to the first value it meets and must match everywhere it
	// recurs; "/users/{{uid}}" is rendered from the captured value.
	ok := gotestify.JSONEqual(&t, `{
		"user":  {"id": "{{uid}}", "name": "Alice"},
		"owner": {"id": "{{uid}}"},
		"self":  "/users/{{uid}}"
	}`, resp)

	fmt.Println(ok)
	// Output: true
}

func ExampleJSONSubset() {
	var t reporter
	resp := `{"id": "0193f2a1-4c3b-7a91-b8e2-1f5d9c7a3e40", "name": "Alice", "internal": true}`

	// Extra properties on the actual side are fine.
	ok := gotestify.JSONSubset(&t, `{"id": "{{uuid}}", "name": "Alice"}`, resp)

	fmt.Println(ok)
	// Output: true
}

func ExampleRegisterMatcher() {
	var t reporter
	gotestify.RegisterMatcher("email", func(v any) bool {
		s, ok := v.(string)
		return ok && len(s) > 3 && s[0] != '@' && s[len(s)-1] != '@' && contains(s, "@")
	})

	ok := gotestify.JSONEqual(&t, `{"email": "{{email}}"}`, `{"email": "alice@example.com"}`)

	fmt.Println(ok)
	// Output: true
}

func ExampleCompare() {
	diff, rendered := gotestify.Compare(
		[]byte(`{"a": 1, "b": "{{any}}", "c": [1, 2]}`),
		[]byte(`{"a": 2, "b": null, "c": [1, 2, 3]}`),
		gotestify.JSONDiffOptions(),
	)

	fmt.Println(diff)
	fmt.Println(rendered)
	// Output:
	// NoMatch
	// {
	//     "a": {"changed":[1, 2]},
	//     "b": {"changed":["{{any}}", null]},
	//     "c": [
	//         1,
	//         2,
	//         "prop-added":{3}
	//     ]
	// }
}

func ExampleCollectVars() {
	vars, err := gotestify.CollectVars(
		`{"id": "{{uid}}", "team": {"id": "{{tid}}"}}`,
		`{"id": "u_1", "team": {"id": "t_9"}}`,
		gotestify.JSONDiffOptions(),
	)

	fmt.Println(err, vars["uid"], vars["tid"])
	// Output: <nil> u_1 t_9
}

func ExampleTableEqual() {
	var t reporter
	type Row struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	rows := []Row{{"Alice", 25}, {"Bob", 30}}

	// Columns are JSON field names; whitespace around each expected line is
	// ignored so the literal can be indented.
	ok := gotestify.TableEqual(&t, `
		Alice |25 |
		Bob   |30 |
	`, rows, []string{"name", "age"})

	fmt.Println(ok)
	// Output: true
}

func ExampleCaptureSlog() {
	var t reporter
	buf := gotestify.CaptureSlog(&t)

	slog.Info("work completed", "items", 3)

	fmt.Println(contains(buf.String(), "work completed"), contains(buf.String(), "items=3"))
	for _, f := range t.cleanups {
		f()
	}
	// Output: true true
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
