package gotestify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDifferenceString(t *testing.T) {
	cases := map[Difference]string{
		FullMatch:              "FullMatch",
		SupersetMatch:          "SupersetMatch",
		SubsetMatch:            "SubsetMatch",
		NoMatch:                "NoMatch",
		FirstArgIsInvalidJSON:  "FirstArgIsInvalidJSON",
		SecondArgIsInvalidJSON: "SecondArgIsInvalidJSON",
		BothArgsAreInvalidJSON: "BothArgsAreInvalidJSON",
		ExpressionError:        "ExpressionError",
		Difference(-1):         "Invalid",
	}
	for d, want := range cases {
		assert.Equal(t, want, d.String())
	}
}

func TestSkipMatchesCollapsesEqualValues(t *testing.T) {
	opts := ConsoleDiffOptions()
	opts.SkipMatches = true
	opts.Added = Tag{Begin: "+(", End: ")"}
	opts.Removed = Tag{Begin: "-(", End: ")"}
	opts.Changed = Tag{Begin: "~(", End: ")"}
	opts.Skipped = Tag{Begin: "s(", End: ")"}

	diff, out := CompareStr(
		`{"a":1,"b":2,"c":3,"arr":[1,2,3,4]}`,
		`{"a":1,"b":2,"c":4,"arr":[1,2,3,5]}`,
		opts,
	)

	assert.Equal(t, NoMatch, diff)
	assert.Equal(t, `{
  s(...skipped 1 object property...),
  "arr": [
    s(...skipped 3 array elements...),
    ~(4 => 5)
  ],
  s(...skipped 1 object property...),
  "c": ~(3 => 4)
}`, out)
}

func TestSkippedMessagesPluralize(t *testing.T) {
	assert.Equal(t, "...skipped 1 array element...", skippedArrayElement(1))
	assert.Equal(t, "...skipped 2 array elements...", skippedArrayElement(2))
	assert.Equal(t, "...skipped 1 object property...", skippedObjectProperty(1))
	assert.Equal(t, "...skipped 3 object properties...", skippedObjectProperty(3))
}

func TestPrintTypesAnnotatesValues(t *testing.T) {
	opts := JSONDiffOptions()
	opts.PrintTypes = true

	diff, out := CompareStr(
		`{"a":true,"n":null}`,
		`{"a":"x","n":null,"obj":{"k":[1,{}]},"e":[]}`,
		opts,
	)

	assert.Equal(t, NoMatch, diff)
	assert.Equal(t, `{
    "a": {"changed":[true (boolean), "x" (string)]},
    "prop-added":{"e": [] (array)},
    "n": null (null),
    "prop-added":{"obj": {}
        "prop-added":{"k": [}
            "prop-added":{1 (number),}
            "prop-added":{{} (object)}
        "prop-added":{] (array)}
    "prop-added":{} (object)}
} (object)`, out)
}
