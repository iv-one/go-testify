package gotestify

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
)

// jsonArray is a JSON array of objects decoded into Go values.
type jsonArray []map[string]any

// PrettyJSON formats the given value as an indented JSON string. On failure it
// returns the error text, so it stays usable inside a failure message.
func PrettyJSON(val any, opts ...json.Options) string {
	b, err := marshalIndent(val, "", "  ", opts...)
	if err != nil {
		return err.Error()
	}
	return string(b)
}

// marshalIndent marshals the given value to a JSON byte slice with indentation.
func marshalIndent(v any, prefix, indent string, opts ...json.Options) ([]byte, error) {
	return json.Marshal(v, json.JoinOptions(opts...),
		jsontext.Multiline(true), jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
}

// parseJSON parses the given JSON document into a value of type T.
func parseJSON[T any](data []byte, opts ...json.Options) (T, error) {
	var v T
	err := json.Unmarshal(data, &v, opts...)
	return v, err
}

// toJSON returns v as a JSON document: a string is taken verbatim, anything
// else is marshaled with opts.
func toJSON(v any, opts ...json.Options) ([]byte, error) {
	if s, ok := v.(string); ok {
		return []byte(s), nil
	}
	return json.Marshal(v, opts...)
}

// normalizeJSON round-trips v through JSON into a T, applying opts to both the
// marshal and the unmarshal leg, so custom codecs shape the result and JSON
// field names become the keys. A string is treated as an already-encoded
// document.
func normalizeJSON[T any](v any, opts ...json.Options) (T, error) {
	b, err := toJSON(v, opts...)
	if err != nil {
		var zero T
		return zero, err
	}
	return parseJSON[T](b, opts...)
}
