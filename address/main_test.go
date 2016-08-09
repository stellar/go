package address

import (
	"testing"

	"github.com/stellar/go/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestSplit(t *testing.T) {
	cases := []struct {
		CaseName       string
		Address        string
		ExpectedName   string
		ExpectedDomain string
		ExpectedError  error
	}{
		{"happy path", "scott*stellar.org", "scott", "stellar.org", nil},
		{"blank", "", "", "", ErrInvalidAddress},
		{"blank name", "*stellar.org", "", "", ErrInvalidName},
		{"blank domain", "scott*", "", "", ErrInvalidDomain},
		{"invalid domain", "scott*--3.com", "", "", ErrInvalidDomain},
	}

	for _, c := range cases {
		name, domain, err := Split(c.Address)

		if c.ExpectedError == nil {
			assert.Equal(t, name, c.ExpectedName)
			assert.Equal(t, domain, c.ExpectedDomain)
		} else {
			assert.Equal(t, errors.Cause(err), c.ExpectedError)
		}
	}
}
