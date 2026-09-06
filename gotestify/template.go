package gotestify

import (
	"encoding/json/v2"
	"strings"
	"text/template"
)

// CompileTemplate renders the given text/template against data.
func CompileTemplate(tmpl string, data any) (string, error) {
	t, err := template.New("gotestify").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// CompileTemplateJSON renders the given text/template against data normalized
// through JSON, so template fields are the JSON field names rather than the Go
// ones and any custom codecs in opts apply.
func CompileTemplateJSON(tmpl string, data any, opts ...json.Options) (string, error) {
	normalized, err := normalizeJSON[any](data, opts...)
	if err != nil {
		return "", err
	}

	return CompileTemplate(tmpl, normalized)
}
