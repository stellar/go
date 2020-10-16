package stellartoml

import (
	"fmt"
	"io"
	"net/http"

	"github.com/stellar/go/support/errors"
	"github.com/BurntSushi/toml"
	StellarAddress "github.com/stellar/go/address"
)

// HTTP represents the http client that a stellertoml resolver uses to make http
// requests.
type HTTP interface {
	Get(url string) (*http.Response, error)
}

// Client represents a client that is capable of resolving a Stellar.toml file
// using the internet.
type Client struct {
	// HTTP is the http client used when resolving a Stellar.toml file
	HTTP HTTP

	// UseHTTP forces the client to resolve against servers using plain HTTP.
	// Useful for debugging.
	UseHTTP bool
}

type ClientInterface interface {
	GetStellarToml(domain string) (*Response, error)
	GetStellarTomlByAddress(address string) (*Response, error)
}

// GetStellarToml returns stellar.toml file for a given domain
func (c *Client) GetStellarToml(domain string) (resp *Response, err error) {
	var hresp *http.Response
	hresp, err = c.HTTP.Get(c.url(domain))
	if err != nil {
		err = errors.Wrap(err, "http request errored")
		return
	}
	defer hresp.Body.Close()

	if !(hresp.StatusCode >= 200 && hresp.StatusCode < 300) {
		err = errors.New("http request failed with non-200 status code")
		return
	}

	limitReader := io.LimitReader(hresp.Body, StellarTomlMaxSize)
	_, err = toml.DecodeReader(limitReader, &resp)

	// There is one corner case not handled here: response is exactly
	// StellarTomlMaxSize long and is incorrect toml. Check discussion:
	// https://github.com/stellar/go/pull/24#discussion_r89909696
	if err != nil && limitReader.(*io.LimitedReader).N == 0 {
		err = errors.Errorf("stellar.toml response exceeds %d bytes limit", StellarTomlMaxSize)
		return
	}

	if err != nil {
		err = errors.Wrap(err, "toml decode failed")
		return
	}

	return
}

// GetStellarTomlByAddress returns stellar.toml file of a domain fetched from a
// given address
func (c *Client) GetStellarTomlByAddress(address string) (*Response, error) {
	_, domain, err := StellarAddress.Split(address)
	if err != nil {
		return nil, errors.Wrap(err, "parse address failed")
	}

	return c.GetStellarToml(domain)
}

// url returns the appropriate url to load for resolving domain's stellar.toml
// file
func (c *Client) url(domain string) string {
	var scheme string

	if c.UseHTTP {
		scheme = "http"
	} else {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s%s", scheme, domain, WellKnownPath)
}
