package apiclient

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

func (c *APIClient) getRequest(endpoint string, queryParams url.Values) error {
	fullURL := c.url(endpoint, queryParams)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return errors.Wrap(err, "http GET request creation failed")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "http GET request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c *APIClient) url(endpoint string, qstr url.Values) string {
	return fmt.Sprintf("%s/%s?%s", c.BaseURL, endpoint, qstr.Encode())
}
