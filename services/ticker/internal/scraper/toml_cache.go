package scraper

type TOMLCache struct {
	tomlCache map[string]TOMLIssuer
}

func (c TOMLCache) Get(tomlURL string) (tomlIssuer TOMLIssuer, ok bool) {
	tomlIssuer, ok = c.tomlCache[tomlURL]
	return
}

func (c *TOMLCache) Set(tomlURL string, tomlIssuer TOMLIssuer) {
	if c.tomlCache == nil {
		c.tomlCache = map[string]TOMLIssuer{}
	}
	c.tomlCache[tomlURL] = tomlIssuer
}
