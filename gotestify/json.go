package gotestify

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// JSONArray is a JSON array of objects decoded into Go values.
type JSONArray []map[string]any

// Nice formats the given value as an indented JSON string. On failure it
// returns the error text, so it stays usable inside a failure message.
func Nice(val any, opts ...json.Options) string {
	b, err := MarshalIndent(val, "", "  ", opts...)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

// MarshalIndent marshals the given value to a JSON byte slice with indentation.
func MarshalIndent(v any, prefix, indent string, opts ...json.Options) ([]byte, error) {
	return json.Marshal(v, json.JoinOptions(opts...),
		jsontext.Multiline(true), jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
}

// ParseJSON parses the given JSON document into a value of type T.
func ParseJSON[T any](data []byte, opts ...json.Options) (T, error) {
	var v T
	if err := json.Unmarshal(data, &v, json.JoinOptions(opts...)); err != nil {
		return v, err
	}
	return v, nil
}
