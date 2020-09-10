package xdr

import (
	"encoding/json"
	"fmt"
	"time"
)

type claimPredicateJSON struct {
	And           *[]claimPredicateJSON `json:"and,omitempty"`
	Or            *[]claimPredicateJSON `json:"or,omitempty"`
	Not           *claimPredicateJSON   `json:"not,omitempty"`
	Unconditional bool                  `json:"unconditional,omitempty"`
	AbsBefore     *time.Time            `json:"absBefore,omitempty"`
	RelBefore     *int64                `json:"relBefore,omitempty"`
}

func convertPredicatesToXDR(input []claimPredicateJSON) ([]ClaimPredicate, error) {
	parts := make([]ClaimPredicate, len(input))
	for i, pred := range input {
		converted, err := pred.toXDR()
		if err != nil {
			return parts, err
		}
		parts[i] = converted
	}
	return parts, nil
}

func (c claimPredicateJSON) toXDR() (ClaimPredicate, error) {
	var result ClaimPredicate
	var err error

	switch {
	case c.Unconditional:
		result.Type = ClaimPredicateTypeClaimPredicateUnconditional
	case c.RelBefore != nil:
		relBefore := Int64(*c.RelBefore)
		result.Type = ClaimPredicateTypeClaimPredicateBeforeRelativeTime
		result.RelBefore = &relBefore
	case c.AbsBefore != nil:
		absBefore := Int64((*c.AbsBefore).UTC().Unix())
		result.Type = ClaimPredicateTypeClaimPredicateBeforeAbsoluteTime
		result.AbsBefore = &absBefore
	case c.Not != nil:
		if inner, innerErr := c.Not.toXDR(); innerErr != nil {
			err = innerErr
		} else {
			result.Type = ClaimPredicateTypeClaimPredicateNot
			result.NotPredicate = new(*ClaimPredicate)
			*result.NotPredicate = &inner
		}
	case c.And != nil:
		if inner, innerErr := convertPredicatesToXDR(*c.And); innerErr != nil {
			err = innerErr
		} else {
			result.Type = ClaimPredicateTypeClaimPredicateAnd
			result.AndPredicates = &inner
		}
	case c.Or != nil:
		if inner, innerErr := convertPredicatesToXDR(*c.Or); innerErr != nil {
			err = innerErr
		} else {
			result.Type = ClaimPredicateTypeClaimPredicateOr
			result.OrPredicates = &inner
		}
	}

	return result, err
}

func convertPredicatesToJSON(input []ClaimPredicate) ([]claimPredicateJSON, error) {
	parts := make([]claimPredicateJSON, len(input))
	for i, pred := range input {
		converted, err := pred.toJSON()
		if err != nil {
			return parts, err
		}
		parts[i] = converted
	}
	return parts, nil
}

func (c ClaimPredicate) toJSON() (claimPredicateJSON, error) {
	var payload claimPredicateJSON
	var err error

	switch c.Type {
	case ClaimPredicateTypeClaimPredicateAnd:
		payload.And = new([]claimPredicateJSON)
		*payload.And, err = convertPredicatesToJSON(c.MustAndPredicates())
	case ClaimPredicateTypeClaimPredicateOr:
		payload.Or = new([]claimPredicateJSON)
		*payload.Or, err = convertPredicatesToJSON(c.MustOrPredicates())
	case ClaimPredicateTypeClaimPredicateUnconditional:
		payload.Unconditional = true
	case ClaimPredicateTypeClaimPredicateNot:
		payload.Not = new(claimPredicateJSON)
		*payload.Not, err = c.MustNotPredicate().toJSON()
	case ClaimPredicateTypeClaimPredicateBeforeAbsoluteTime:
		payload.AbsBefore = new(time.Time)
		*payload.AbsBefore = time.Unix(int64(c.MustAbsBefore()), 0).UTC()
	case ClaimPredicateTypeClaimPredicateBeforeRelativeTime:
		payload.RelBefore = new(int64)
		*payload.RelBefore = int64(c.MustRelBefore())
	default:
		err = fmt.Errorf("invalid predicate type: " + c.Type.String())
	}
	return payload, err
}

func (c ClaimPredicate) MarshalJSON() ([]byte, error) {
	payload, err := c.toJSON()
	if err != nil {
		return nil, err
	}
	return json.Marshal(payload)
}

func (c *ClaimPredicate) UnmarshalJSON(b []byte) error {
	var payload claimPredicateJSON
	if err := json.Unmarshal(b, &payload); err != nil {
		return err
	}

	parsed, err := payload.toXDR()
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}
