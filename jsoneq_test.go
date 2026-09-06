package gotestify

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompare(t *testing.T) {
	opts := JSONDiffOptions()
	diff, _ := Compare([]byte(`{"a":1}`), []byte(`{"a":1}`), opts)
	assert.Equal(t, FullMatch, diff)
}

func TestCompareAny(t *testing.T) {
	opts := JSONDiffOptions()
	a := `{"a":"{{any}}"}`
	b := `{"a":1}`
	diff, _ := CompareStr(a, b, opts)
	assert.Equal(t, FullMatch, diff)

	a = `{"a":"{{any}}"}`
	b = `{"a":1, "b":2}`
	diff, _ = CompareStr(a, b, opts)
	assert.Equal(t, SubsetMatch, diff)
}

func TestEqual(t *testing.T) {
	type test struct {
		X string `json:"x"`
	}

	a := test{X: "a"}
	b := test{X: "a"}

	JSONEqual(t, a, b)
}

func TestCollectVars(t *testing.T) {
	opts := JSONDiffOptions()
	vars, err := CollectVars(`{"a":"{{x}}"}`, `{"a":"1"}`, opts)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"x": "1",
	}, vars)
}

func TestEqualWithFunctions(t *testing.T) {
	a := `
	{
		"account": {
				"apps_id": "{{apps_id}}",
				"id": "{{account_id}}",
				"name": "Example",
				"created_at": "{{timestamp}}",
				"updated_at": "{{timestamp}}"
		},
		"app": {
				"{{app_id1}}": {
						"created_at": "{{timestamp}}",
						"id": "{{app_id1}}",
						"instances": {
								"0x1": "0x1"
						},
						"name": "Admin",
						"updated_at": "{{timestamp}}"
				},
				"{{app_id2}}": {
						"created_at": "{{timestamp}}",
						"id": "{{app_id2}}",
						"instances": {
								"0x2": "0x2"
						},
						"name": "XYZ",
						"updated_at": "{{timestamp}}"
				}
		},
		"apps": {
				"apps": {
						"{{app_id1}}": "{{app_id1}}",
						"{{app_id2}}": "{{app_id2}}"
				},
				"created_at": "{{timestamp}}",
				"id": "{{apps_id}}",
				"updated_at": "{{timestamp}}"
		},
		"domains": {
				"app-xyz": {
						"domain_kind": 1,
						"host": "https://app-xyz.example.tech",
						"id": "app-xyz",
						"tenant_id": "0x2"
				},
				"admin": {
						"host": "https://portal.example.tech",
						"id": "admin",
						"tenant_id": "0x1"
				}
		},
		"email": null,
		"google_id": null,
		"setup": {
				"account_id": "{{account_id}}",
				"created_at": "{{timestamp}}",
				"id": "",
				"is_init": true,
				"updated_at": "{{timestamp}}"
		},
		"tenant": {
				"app_id": "{{app_id1}}",
				"app_url": "https://portal.example.tech",
				"trace_id": "{{uuid}}",
				"created_at": "{{timestamp}}",
				"domain": "admin",
				"env": {
						"name": "Production",
						"type": 1
				},
				"id": "0x1",
				"is_admin": true
		},
		"tenant_xyz": {
				"app_id": "{{app_id2}}",
				"app_url": "http://localhost:3000",
				"trace_id": "{{uuid}}",
				"created_at": "{{timestamp}}",
				"domain": "app-xyz",
				"env": {
						"name": "Development"
				},
				"id": "0x2"
		},
		"user": null
	}
`

	b := `
	{
		"account": {
				"apps_id": "ci66nuogg3gmpmjl6dd0",
				"created_at": "2023-06-16T14:04:43Z",
				"id": "ci66nuogg3gmpmjl6dc0",
				"name": "Example",
				"updated_at": "2023-06-16T14:04:43Z"
		},
		"app": {
				"ci66nuogg3gmpmjl6db0": {
						"created_at": "2023-06-16T14:04:43Z",
						"id": "ci66nuogg3gmpmjl6db0",
						"instances": {
								"0x1": "0x1"
						},
						"name": "Admin",
						"updated_at": "2023-06-16T14:04:43Z"
				},
				"ci66nuogg3gmpmjl6dcg": {
						"created_at": "2023-06-16T14:04:43Z",
						"id": "ci66nuogg3gmpmjl6dcg",
						"instances": {
								"0x2": "0x2"
						},
						"name": "XYZ",
						"updated_at": "2023-06-16T14:04:43Z"
				}
		},
		"apps": {
				"apps": {
						"ci66nuogg3gmpmjl6db0": "ci66nuogg3gmpmjl6db0",
						"ci66nuogg3gmpmjl6dcg": "ci66nuogg3gmpmjl6dcg"
				},
				"created_at": "2023-06-16T14:04:43Z",
				"id": "ci66nuogg3gmpmjl6dd0",
				"updated_at": "2023-06-16T14:04:43Z"
		},
		"domains": {
				"app-xyz": {
						"domain_kind": 1,
						"host": "https://app-xyz.example.tech",
						"id": "app-xyz",
						"tenant_id": "0x2"
				},
				"admin": {
						"host": "https://portal.example.tech",
						"id": "admin",
						"tenant_id": "0x1"
				}
		},
		"email": null,
		"google_id": null,
		"setup": {
				"account_id": "ci66nuogg3gmpmjl6dc0",
				"created_at": "2023-06-16T14:04:43Z",
				"id": "",
				"is_init": true,
				"updated_at": "2023-06-16T14:04:43Z"
		},
		"tenant": {
				"app_id": "ci66nuogg3gmpmjl6db0",
				"app_url": "https://portal.example.tech",
				"trace_id": "65a7d0b2-a0a5-4c31-b57e-dc3371de93c4",
				"created_at": "2023-06-16T14:04:43Z",
				"domain": "admin",
				"env": {
						"name": "Production",
						"type": 1
				},
				"id": "0x1",
				"is_admin": true
		},
		"tenant_xyz": {
				"app_id": "ci66nuogg3gmpmjl6dcg",
				"app_url": "http://localhost:3000",
				"trace_id": "1538f684-2184-41b2-9439-ecbb5b30464d",
				"created_at": "2023-06-16T14:04:43Z",
				"domain": "app-xyz",
				"env": {
						"name": "Development"
				},
				"id": "0x2"
		},
		"user": null
	}
`

	JSONEqual(t, a, b)
}

func TestEqualWithFunctionsBetterExpressions(t *testing.T) {
	a := `
		{
			"f": "{{any}}",
			"x": "{{x}}",
			"y": "{{y}}",
			"z": "{{x}}:{{y}}"
		}
	`

	b := `
		{
			"f": "does not matter",
			"x": "1",
			"y": "2",
			"z": "1:2"
		}
	`

	JSONEqual(t, a, b)
}

func TestSubsetMatch(t *testing.T) {
	a := `
		{
			"x": "{{x}}",
			"y": "{{y}}"
		}
	`

	b := `
		{
			"f": "does not matter",
			"x": "1",
			"y": "2",
			"z": "1:2"
		}
	`

	JSONSubset(t, a, b)
}

// epoch stands in for a type such as a protobuf Timestamp: its default
// encoding (a number) is not how the API serves it, so a codec is needed.
type epoch int64

// JSONEqual marshals non-string arguments with the caller's json.Options, so
// a service's custom codecs apply to the value under test.
func TestJSONEqualAppliesJSONOptions(t *testing.T) {
	type doc struct {
		ID string `json:"id"`
		At epoch  `json:"at"`
	}

	rfc3339 := json.WithMarshalers(json.MarshalToFunc(
		func(enc *jsontext.Encoder, e epoch) error {
			return enc.WriteToken(jsontext.String(time.Unix(int64(e), 0).UTC().Format(time.RFC3339)))
		}))

	v := &doc{
		ID: "cjd0keldrb6jdafhr860",
		At: epoch(time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC).Unix()),
	}
	expected := `{"id":"cjd0keldrb6jdafhr860","at":"2024-05-06T07:08:09Z"}`

	JSONEqual(t, expected, v, rfc3339)

	// Without the codec the timestamp marshals as a number and no longer matches.
	diff, _ := CompareStr(expected, PrettyJSON(v), JSONDiffOptions())
	assert.Equal(t, NoMatch, diff)
}
