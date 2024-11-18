package apiclient

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
)

const (
	maxRetries     = 5
	initialBackoff = 1 * time.Second
)

func isRetryableStatusCode(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode == http.StatusServiceUnavailable
}

func (c *APIClient) GetURL(endpoint string, qstr url.Values) string {
	return fmt.Sprintf("%s/%s?%s", c.BaseURL, endpoint, qstr.Encode())
}

func (c *APIClient) CallAPI(reqParams RequestParams) (interface{}, error) {
	if reqParams.QueryParams == nil {
		reqParams.QueryParams = url.Values{}
	}

	if reqParams.Headers == nil {
		reqParams.Headers = map[string]interface{}{}
	}

	url := c.GetURL(reqParams.Endpoint, reqParams.QueryParams)
	reqBody, err := CreateRequestBody(reqParams.RequestType, url)
	if err != nil {
		return nil, errors.Wrap(err, "http request creation failed")
	}

	SetAuthHeaders(reqBody, c.authType, c.authHeaders)
	SetHeaders(reqBody, reqParams.Headers)
	client := c.HTTP
	if client == nil {
		client = &http.Client{}
	}

	var result interface{}
	retries := 0

	for retries <= maxRetries {
		resp, err := client.Do(reqBody)
		if err != nil {
			return nil, errors.Wrap(err, "http request failed")
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read response body: %w", err)
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
			}

			return result, nil
		} else if isRetryableStatusCode(resp.StatusCode) {
			retries++
			backoffDuration := initialBackoff * time.Duration(1<<retries)
			if retries <= maxRetries {
				fmt.Printf("Received retryable status %d. Retrying in %v...\n", resp.StatusCode, backoffDuration)
				time.Sleep(backoffDuration)
			} else {
				return nil, fmt.Errorf("Maximum retries reached after receiving status %d", resp.StatusCode)
			}
		} else {
			return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
		}
	}

	return nil, fmt.Errorf("API request failed after %d retries", retries)
}
