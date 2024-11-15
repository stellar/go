package apiclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

func (c *APIClient) getRequest(endpoint string, queryParams url.Values) (interface{}, error) {
	client := c.HTTP
	if client == nil {
		client = &http.Client{}
	}

	fullURL := c.url(endpoint, queryParams)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "http GET request creation failed")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "http GET request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return result, nil
}

func (c *APIClient) url(endpoint string, qstr url.Values) string {
	return fmt.Sprintf("%s/%s?%s", c.BaseURL, endpoint, qstr.Encode())
}
