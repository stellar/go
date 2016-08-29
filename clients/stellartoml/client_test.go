package stellartoml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientURL(t *testing.T) {
	//HACK:  we're testing an internal method rather than setting up a http client
	//mock.

	c := &Client{UseHTTP: false}
	assert.Equal(t, "https://www.stellar.org/.well-known/stellar.toml", c.url("stellar.org"))

	c = &Client{UseHTTP: true}
	assert.Equal(t, "http://www.stellar.org/.well-known/stellar.toml", c.url("stellar.org"))
}
