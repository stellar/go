package digitalbitstoml

import (
	"strings"
	"testing"

	"net/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xdbfoundation/go/support/http/httptest"
)

func TestClientURL(t *testing.T) {
	//HACK:  we're testing an internal method rather than setting up a http client
	//mock.

	c := &Client{UseHTTP: false}
	assert.Equal(t, "https://livenet.digitalbits.io/.well-known/digitalbits.toml", c.url("digitalbits.org"))

	c = &Client{UseHTTP: true}
	assert.Equal(t, "http://livenet.digitalbits.io/.well-known/digitalbits.toml", c.url("digitalbits.org"))
}

func TestClient(t *testing.T) {
	h := httptest.NewClient()
	c := &Client{HTTP: h}

	// happy path
	h.
		On("GET", "https://livenet.digitalbits.io/.well-known/digitalbits.toml").
		ReturnString(http.StatusOK,
			`FEDERATION_SERVER="https://localhost/federation"`,
		)
	stoml, err := c.GetDigitalBitsToml("digitalbits.org")
	require.NoError(t, err)
	assert.Equal(t, "https://localhost/federation", stoml.FederationServer)

	// digitalbits.toml exceeds limit
	h.
		On("GET", "https://toobig.org/.well-known/digitalbits.toml").
		ReturnString(http.StatusOK,
			`FEDERATION_SERVER="https://localhost/federation`+strings.Repeat("0", DigitalBitsTomlMaxSize)+`"`,
		)
	stoml, err = c.GetDigitalBitsToml("toobig.org")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "digitalbits.toml response exceeds")
	}

	// not found
	h.
		On("GET", "https://missing.org/.well-known/digitalbits.toml").
		ReturnNotFound()
	stoml, err = c.GetDigitalBitsToml("missing.org")
	assert.EqualError(t, err, "http request failed with non-200 status code")

	// invalid toml
	h.
		On("GET", "https://json.org/.well-known/digitalbits.toml").
		ReturnJSON(http.StatusOK, map[string]string{"hello": "world"})
	stoml, err = c.GetDigitalBitsToml("json.org")

	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "toml decode failed")
	}
}
