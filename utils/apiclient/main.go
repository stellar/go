package apiclient

import (
	"net/http"
	"net/url"
)

type HTTP interface {
	Do(req *http.Request) (resp *http.Response, err error)
	Get(url string) (resp *http.Response, err error)
	PostForm(url string, data url.Values) (resp *http.Response, err error)
}

type APIClient struct {
	BaseURL string
	HTTP    HTTP
}
