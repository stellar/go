package gxdr

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	goxdr "github.com/xdrpp/goxdr/xdr"
)

func buildVec(depth int) SCVal {
	if depth <= 0 {
		symbol := SCSymbol("s")
		return SCVal{
			Type: SCV_SYMBOL,
			_u:   &symbol,
		}
	}
	vec := &SCVec{
		buildVec(depth - 1),
	}
	return SCVal{Type: SCV_VEC, _u: &vec}
}

func buildMaliciousVec(t *testing.T) string {
	vals := &SCVec{}
	for i := 0; i < 0x0D; i++ {
		symbol := SCSymbol("s")
		*vals = append(*vals, SCVal{
			Type: SCV_SYMBOL,
			_u:   &symbol,
		})
	}
	vec := SCVal{Type: SCV_VEC, _u: &vals}
	raw := Dump(&vec)
	// raw[8-11] represents the part of the xdr that holds the
	// length of the vector
	for i, b := range raw {
		if b == 0x0D {
			assert.Equal(t, 11, i)
		}
	}
	// here we override the most significant byte in the vector length
	// so that the vector length in the xdr is 0xFA00000D which
	// is equal to 4194304013
	raw[8] = 0xFA
	return base64.StdEncoding.EncodeToString(raw)
}

func TestValidator(t *testing.T) {
	shallowVec := buildVec(2)
	deepVec := buildVec(100)
	for _, testCase := range []struct {
		name          string
		input         string
		maxDepth      int
		val           goxdr.XdrType
		expectedError string
	}{
		{
			"invalid base 64 input",
			"{}<>~!@$#",
			500,
			&LedgerEntry{},
			"illegal base64 data at input byte 0",
		},
		{
			"valid depth",
			base64.StdEncoding.EncodeToString(Dump(&shallowVec)),
			500,
			&SCVal{},
			"",
		},
		{
			"invalid depth",
			base64.StdEncoding.EncodeToString(Dump(&deepVec)),
			50,
			&SCVal{},
			"max depth of 50 exceeded",
		},
		{
			"malicious length",
			buildMaliciousVec(t),
			500,
			&SCVal{},
			"EOF",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			err := validate(testCase.input, testCase.val, testCase.maxDepth)
			if testCase.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, testCase.input, base64.StdEncoding.EncodeToString(Dump(testCase.val)))
			} else {
				assert.EqualError(t, err, testCase.expectedError)
			}
		})
	}
}
