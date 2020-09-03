package xdr

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
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
		`{"and":[{"or":[{"relBefore":12},{"absBefore":"2020-08-26T11:15:39Z"}]},{"not":{"unconditional":true}}]}`,
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
