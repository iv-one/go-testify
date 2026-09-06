package gotestify

import (
	"bytes"
	"encoding/json/v2"
	"text/template"
)

// CompileTemplate renders the given text/template against data.
func CompileTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("gotestify").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// CompileTemplateJSON renders the given text/template against data normalized
// through JSON, so template fields are the JSON field names rather than the Go
// ones and any custom codecs in opts apply.
func CompileTemplateJSON(tmpl string, data any, opts ...json.Options) (string, error) {
	b, err := json.Marshal(data, json.JoinOptions(opts...))
	if err != nil {
		return "", err
	}

	normalized, err := ParseJSON[any](b)
	if err != nil {
		return "", err
	}

	return CompileTemplate(tmpl, normalized)
}
