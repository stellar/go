package xdr

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaimPredicateJSON(t *testing.T) {
	unconditional := &ClaimPredicate{
		Type: ClaimPredicateTypeClaimPredicateUnconditional,
	}
	relBefore := Int64(12)
	absBefore := Int64(1598440539)

	source := ClaimPredicate{
		Type: ClaimPredicateTypeClaimPredicateAnd,
		AndPredicates: &[]ClaimPredicate{
			{
				Type: ClaimPredicateTypeClaimPredicateOr,
				OrPredicates: &[]ClaimPredicate{
					{
						Type:      ClaimPredicateTypeClaimPredicateBeforeRelativeTime,
						RelBefore: &relBefore,
					},
					{
						Type:      ClaimPredicateTypeClaimPredicateBeforeAbsoluteTime,
						AbsBefore: &absBefore,
					},
				},
			},
			{
				Type:         ClaimPredicateTypeClaimPredicateNot,
				NotPredicate: &unconditional,
			},
		},
	}

	serialized, err := json.Marshal(source)
	assert.NoError(t, err)
	assert.JSONEq(
		t,
		`{"and":[{"or":[{"rel_before":"12"},{"abs_before":"2020-08-26T11:15:39Z"}]},{"not":{"unconditional":true}}]}`,
		string(serialized),
	)

	var parsed ClaimPredicate
	assert.NoError(t, json.Unmarshal(serialized, &parsed))

	var serializedBase64, parsedBase64 string
	serializedBase64, err = MarshalBase64(source)
	assert.NoError(t, err)

	parsedBase64, err = MarshalBase64(parsed)
	assert.NoError(t, err)

	assert.Equal(t, serializedBase64, parsedBase64)
}

func TestAbsBeforeTimestamps(t *testing.T) {
	const year = 365 * 24 * 60 * 60
	for _, testCase := range []struct {
		unix     int64
		expected string
	}{
		{
			0,
			`{"abs_before":"1970-01-01T00:00:00Z"}`,
		},
		{
			900 * year,
			`{"abs_before":"2869-05-27T00:00:00Z"}`,
		},
		{
			math.MaxInt64,
			`{"abs_before":"292277026596-12-04T15:30:07Z"}`,
		},
		{
			-10,
			`{"abs_before":"1969-12-31T23:59:50Z"}`,
		},
		{
			math.MinInt64,
			// this serialization doesn't make sense but at least it doesn't crash the marshaller
			`{"abs_before":"292277026596-12-04T15:30:08Z"}`,
		},
	} {
		xdrSec := Int64(testCase.unix)
		source := ClaimPredicate{
			Type:      ClaimPredicateTypeClaimPredicateBeforeAbsoluteTime,
			AbsBefore: &xdrSec,
		}

		serialized, err := json.Marshal(source)
		assert.NoError(t, err)
		assert.JSONEq(t, testCase.expected, string(serialized))

		var parsed ClaimPredicate
		assert.NoError(t, json.Unmarshal(serialized, &parsed))
		assert.Equal(t, *parsed.AbsBefore, *source.AbsBefore)
	}
}
