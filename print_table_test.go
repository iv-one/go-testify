package gotestify

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintTable(t *testing.T) {
	type Row struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	rows := []Row{
		{"Alice", 25},
		{"Bob", 30},
	}

	table, err := PrintTable(rows, []string{"name", "age"})
	require.NoError(t, err)
	expected := "Alice |25 |\nBob   |30 |\n"
	assert.Equal(t, expected, table)
}

func TestPrintTableNonSlice(t *testing.T) {
	type Row struct {
		Name string `json:"name"`
	}

	table, err := PrintTable(Row{"Alice"}, []string{"name"})
	require.NoError(t, err)
	assert.Equal(t, "Alice |\n", table)
}

func TestPrintTableAppliesJSONOptions(t *testing.T) {
	type Row struct {
		Name string `json:"name"`
	}

	redact := json.WithMarshalers(json.MarshalToFunc(
		func(enc *jsontext.Encoder, s string) error {
			return enc.WriteToken(jsontext.String("***"))
		}))

	table, err := PrintTable([]Row{{Name: "Alice"}}, []string{"name"}, redact)
	require.NoError(t, err)
	assert.Equal(t, "*** |\n", table)
}
