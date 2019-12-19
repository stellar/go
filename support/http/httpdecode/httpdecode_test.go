package httpdecode

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeValidJSON(t *testing.T) {
	body := `{"Foo":"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))

	bodyDecoded := struct {
		Foo string `json:"foo"`
	}{}
	err := DecodeJSON(r, &bodyDecoded)
	require.NoError(t, err)

	assert.Equal(t, "bar", bodyDecoded.Foo)
}

func TestDecodeInvalidJSON(t *testing.T) {
	body := `{"Foo:"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))

	bodyDecoded := struct {
		Foo string `json:"foo"`
	}{}
	err := DecodeJSON(r, &bodyDecoded)
	require.EqualError(t, err, "invalid character 'b' after object key")
}
