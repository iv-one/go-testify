package gotestify

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recorder struct {
	failed bool
	msg    string
}

func (r *recorder) Errorf(format string, args ...any) {
	r.failed = true
	r.msg = strconv.Quote(format)
}

func TestNumbersKeepLiteralText(t *testing.T) {
	diff, _ := CompareStr(`{"n":1}`, `{"n":1.0}`, JSONDiffOptions())
	assert.Equal(t, NoMatch, diff, "1 and 1.0 are different literals")

	big := `{"n":12345678901234567890}`
	diff, _ = CompareStr(big, big, JSONDiffOptions())
	assert.Equal(t, FullMatch, diff, "large integers must not go through float64")

	opts := JSONDiffOptions()
	opts.CompareNumbers = func(a, b Number) bool {
		fa, _ := strconv.ParseFloat(string(a), 64)
		fb, _ := strconv.ParseFloat(string(b), 64)
		return fa == fb
	}
	diff, _ = CompareStr(`{"n":1}`, `{"n":1.0}`, opts)
	assert.Equal(t, FullMatch, diff, "CompareNumbers can relax the literal comparison")
}

func TestInvalidDocumentsAreReported(t *testing.T) {
	diff, _ := CompareStr(`{"a":`, `{}`, JSONDiffOptions())
	assert.Equal(t, FirstArgIsInvalidJSON, diff)
	diff, _ = CompareStr(`{}`, `{"a":1,}`, JSONDiffOptions())
	assert.Equal(t, SecondArgIsInvalidJSON, diff)
	diff, _ = CompareStr(`[`, `]`, JSONDiffOptions())
	assert.Equal(t, BothArgsAreInvalidJSON, diff)
}

func TestRegisterMatcher(t *testing.T) {
	RegisterMatcher("even", func(v any) bool {
		n, ok := v.(Number)
		if !ok {
			return false
		}
		i, err := strconv.Atoi(string(n))
		return err == nil && i%2 == 0
	})

	var r recorder
	assert.True(t, JSONEqual(&r, `{"n":"{{even}}"}`, `{"n":4}`))
	assert.False(t, r.failed)

	assert.False(t, JSONEqual(&r, `{"n":"{{even}}"}`, `{"n":3}`))
	assert.True(t, r.failed)

	assert.Panics(t, func() { RegisterMatcher("", nil) })
}

func TestBadExpressionDoesNotPanic(t *testing.T) {
	diff, res := CompareStr(`{"a":"{{x}}:{{"}`, `{"a":"1:"}`, JSONDiffOptions())
	assert.Equal(t, ExpressionError, diff)
	assert.Contains(t, res, "expression")

	// CollectVars only renders expressions in object keys; values are
	// bound, not evaluated, on that pass.
	_, err := CollectVars(`{"{{x}}:{{": 1}`, `{"1:": 1}`, JSONDiffOptions())
	require.Error(t, err)
}

func TestAssertionsReportOutcome(t *testing.T) {
	var r recorder
	assert.True(t, JSONSubset(&r, `{"a":1}`, `{"a":1,"b":2}`))
	assert.False(t, JSONEqual(&r, `{"a":1}`, `{"a":1,"b":2}`))
	assert.True(t, TableEqual(&r, "x |\n", []map[string]any{{"c": "x"}}, []string{"c"}))
	assert.False(t, TableEqual(&r, "y |\n", []map[string]any{{"c": "x"}}, []string{"c"}))
}
