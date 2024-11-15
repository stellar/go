package apiclient

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

func (c *APIClient) createRequestBody(endpoint string, queryParams url.Values) (*http.Request, error) {
	fullURL := c.url(endpoint, queryParams)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "http GET request creation failed")
	}
	return req, nil
}

func (c *APIClient) callAPI(req *http.Request) (interface{}, error) {
	client := c.HTTP
	if client == nil {
		client = &http.Client{}
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

func setHeaders(req *http.Request, args map[string]interface{}) {
	for key, value := range args {
		strValue, ok := value.(string)
		if !ok {
			fmt.Printf("Skipping non-string value for header %s\n", key)
			continue
		}

		req.Header.Set(key, strValue)
	}
}

func setAuthHeaders(req *http.Request, authType string, args map[string]interface{}) error {
	switch authType {
	case "basic":
		username, ok := args["username"].(string)
		if !ok {
			return fmt.Errorf("missing or invalid username")
		}
		password, ok := args["password"].(string)
		if !ok {
			return fmt.Errorf("missing or invalid password")
		}

		authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))
		setHeaders(req, map[string]interface{}{
			"Authorization": authHeader,
		})

	case "api_key":
		apiKey, ok := args["api_key"].(string)
		if !ok {
			return fmt.Errorf("missing or invalid API key")
		}
		setHeaders(req, map[string]interface{}{
			"Authorization": apiKey,
		})

	default:
		return fmt.Errorf("unsupported auth type: %s", authType)
	}
	return nil
}

func (c *APIClient) url(endpoint string, qstr url.Values) string {
	return fmt.Sprintf("%s/%s?%s", c.BaseURL, endpoint, qstr.Encode())
}
