package account

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIdentities_Present(t *testing.T) {
	testCases := []struct {
		Identities  Identities
		WantPresent bool
	}{
		{
			Identities:  Identities{},
			WantPresent: false,
		},
		{
			Identities:  Identities{Address: "GD2UO6ADC3SY3UWL7BU32NFI3Q6MSP4BNVG2TTIGAJE44VTJ53EGNWOM"},
			WantPresent: true,
		},
		{
			Identities:  Identities{Email: "user@example.com"},
			WantPresent: true,
		},
		{
			Identities:  Identities{PhoneNumber: "+10000000000"},
			WantPresent: true,
		},
		{
			Identities: Identities{
				Address:     "GD2UO6ADC3SY3UWL7BU32NFI3Q6MSP4BNVG2TTIGAJE44VTJ53EGNWOM",
				Email:       "user@example.com",
				PhoneNumber: "+10000000000",
			},
			WantPresent: true,
		},
	}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			present := tc.Identities.Present()
			assert.Equal(t, tc.WantPresent, present)
		})
	}
}
