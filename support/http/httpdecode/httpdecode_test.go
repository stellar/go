package httpdecode

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeJSON_valid(t *testing.T) {
	body := `{"foo":"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))

	bodyDecoded := struct {
		Foo string `json:"foo"`
	}{}
	err := DecodeJSON(r, &bodyDecoded)
	require.NoError(t, err)

	assert.Equal(t, "bar", bodyDecoded.Foo)
}

func TestDecodeJSON_invalid(t *testing.T) {
	body := `{"foo:"bar"}`
	r, _ := http.NewRequest("POST", "/", strings.NewReader(body))

	bodyDecoded := struct {
		Foo string `json:"foo"`
	}{}
	err := DecodeJSON(r, &bodyDecoded)
	require.EqualError(t, err, "invalid character 'b' after object key")
}
