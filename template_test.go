package gotestify

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileTemplate(t *testing.T) {
	out, err := compileTemplate("{{.Name}} is {{.Age}}", map[string]any{"Name": "Alice", "Age": 25})
	require.NoError(t, err)
	assert.Equal(t, "Alice is 25", out)
}

func TestCompileTemplateJSON(t *testing.T) {
	type row struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	// Fields are addressed by their JSON names, not the Go ones.
	out, err := compileTemplateJSON("{{.name}} is {{.age}}", row{Name: "Alice", Age: 25})
	require.NoError(t, err)
	assert.Equal(t, "Alice is 25", out)

	upper := json.WithMarshalers(json.MarshalToFunc(
		func(enc *jsontext.Encoder, s string) error {
			return enc.WriteToken(jsontext.String("<" + s + ">"))
		}))

	out, err = compileTemplateJSON("{{.name}}", row{Name: "Alice"}, upper)
	require.NoError(t, err)
	assert.Equal(t, "<Alice>", out)
}
