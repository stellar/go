package horizonclient

import (
	"testing"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
	"github.com/stretchr/testify/assert"
)

func TestIsNotFoundError(t *testing.T) {
	testCases := []struct {
		desc string
		err  error
		is   bool
	}{
		{
			desc: "nil error",
			err:  nil,
			is:   false,
		},
		{
			desc: "another Go type of error",
			err:  errors.New("error"),
			is:   false,
		},
		{
			desc: "not found problem (pointer)",
			err: &Error{
				Problem: problem.P{
					Type:   "https://stellar.org/horizon-errors/not_found",
					Title:  "Resource Missing",
					Status: 404,
				},
			},
			is: true,
		},
		{
			desc: "not found problem (not a pointer)",
			err: Error{
				Problem: problem.P{
					Type:   "https://stellar.org/horizon-errors/not_found",
					Title:  "Resource Missing",
					Status: 404,
				},
			},
			is: true,
		},
		{
			desc: "some other problem (pointer)",
			err: &Error{
				Problem: problem.P{
					Type:   "https://stellar.org/horizon-errors/server_error",
					Title:  "Server Error",
					Status: 500,
				},
			},
			is: false,
		},
		{
			desc: "some other problem (not a pointer)",
			err: Error{
				Problem: problem.P{
					Type:   "https://stellar.org/horizon-errors/server_error",
					Title:  "Server Error",
					Status: 500,
				},
			},
			is: false,
		},
		{
			desc: "a nil *horizonclient.Error",
			err:  (*Error)(nil),
			is:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			is := IsNotFoundError(tc.err)
			assert.Equal(t, tc.is, is)
		})
	}
}
