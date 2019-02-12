package horizon

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/poliha/go/support/errors"
)

func (c *CallBuilder) Call() (interface{}, error) {

	endpoint, err := c.buildUrl()
	if err != nil {
		return nil, err
	}

	resp, err := c.HTTP.Get(endpoint)
	if err != nil {
		return nil, err
	}

	err = decodeResponse(resp, &c.HorizonResponse)
	if err != nil {
		return nil, err
	}

	return c.HorizonResponse, nil

}

func (c *CallBuilder) fixURL() {
	c.URL = strings.TrimRight(c.URL, "/")
}

func (c *CallBuilder) addEndpoint(endpoint string) {
	c.endpoint = endpoint
}

func (c *CallBuilder) addParam(key string, value string) {
	c.params[key] = value
}

func (c *CallBuilder) Cursor(cursor string) {
	c.addParam("cursor", cursor)
}

func (c *CallBuilder) buildUrl() (endpoint string, err error) {
	endpoint = ""
	query := url.Values{}

	for key, val := range c.params {
		query.Add(key, val)
	}
	c.fixURLOnce.Do(c.fixURL)
	if endpoint == "" {
		endpoint = fmt.Sprintf(
			"%s%s?%s",
			c.URL,
			c.endpoint,
			query.Encode(),
		)
	}

	_, err = url.Parse(endpoint)
	if err != nil {
		err = errors.Wrap(err, "failed to parse endpoint")
	}

	return endpoint, err

}
