package scraper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTOMLCache(t *testing.T) {
	c := TOMLCache{}

	toml, ok := c.Get("")
	assert.False(t, ok)
	assert.Zero(t, toml)

	c.Set("url", TOMLIssuer{SigningKey: "signing key"})
	toml, ok = c.Get("")
	assert.False(t, ok)
	assert.Zero(t, toml)
	toml, ok = c.Get("url")
	assert.True(t, ok)
	assert.Equal(t, TOMLIssuer{SigningKey: "signing key"}, toml)
	toml, ok = c.Get("otherurl")
	assert.False(t, ok)
	assert.Zero(t, toml)

	c.Set("otherurl", TOMLIssuer{SigningKey: "other signing key"})
	toml, ok = c.Get("")
	assert.False(t, ok)
	assert.Zero(t, toml)
	toml, ok = c.Get("url")
	assert.False(t, ok)
	assert.Zero(t, toml)
	toml, ok = c.Get("otherurl")
	assert.True(t, ok)
	assert.Equal(t, TOMLIssuer{SigningKey: "other signing key"}, toml)

	c.Set("url", TOMLIssuer{SigningKey: "changed signing key"})
	toml, ok = c.Get("")
	assert.False(t, ok)
	assert.Zero(t, toml)
	toml, ok = c.Get("url")
	assert.True(t, ok)
	assert.Equal(t, TOMLIssuer{SigningKey: "changed signing key"}, toml)
	toml, ok = c.Get("otherurl")
	assert.False(t, ok)
	assert.Zero(t, toml)
}
