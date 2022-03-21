package scraper

// TOMLCache caches a max of one toml at a time. Old records are discarded when
// new records are added.
type TOMLCache struct {
	set        bool
	tomlURL    string
	tomlIssuer TOMLIssuer
}

// Get returns the toml for the given URL if it is cached. Ok is true only if
// the toml returned was found in the cache.
func (c TOMLCache) Get(tomlURL string) (tomlIssuer TOMLIssuer, ok bool) {
	if c.set && c.tomlURL == tomlURL {
		tomlIssuer = c.tomlIssuer
		ok = true
	}
	return
}

// Set adds the given toml to the cache using the toml URL as its key for lookup
// later.
func (c *TOMLCache) Set(tomlURL string, tomlIssuer TOMLIssuer) {
	c.set = true
	c.tomlURL = tomlURL
	c.tomlIssuer = tomlIssuer
}
