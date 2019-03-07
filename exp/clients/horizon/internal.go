package horizonclient

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/stellar/go/support/errors"
)

func decodeResponse(resp *http.Response, object interface{}) (err error) {
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	if !(resp.StatusCode >= 200 && resp.StatusCode < 300) {
		horizonError := &Error{
			Response: resp,
		}
		decodeError := decoder.Decode(&horizonError.Problem)
		if decodeError != nil {
			return errors.Wrap(decodeError, "error decoding horizon.Problem")
		}
		return horizonError
	}

	err = decoder.Decode(&object)
	if err != nil {
		return
	}
	return
}

// deprecated. To do: remove from new client package
func loadMemo(p *Payment) error {
	res, err := http.Get(p.Links.Transaction.Href)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return json.NewDecoder(res.Body).Decode(&p.Memo)
}

func countParams(params ...interface{}) int {
	counter := 0
	for _, param := range params {
		if param != "" {
			counter++
		}
	}
	return counter
}

func addQueryParams(params ...interface{}) string {
	query := url.Values{}

	for _, param := range params {
		switch param := param.(type) {
		case Cursor:
			if param != "" {
				query.Add("cursor", string(param))
			}
		case Order:
			if param != "" {
				query.Add("order", string(param))
			}
		case Limit:
			if param != 0 {
				query.Add("limit", strconv.Itoa(int(param)))
			}
		case AssetCode:
			if param != "" {
				query.Add("asset_code", string(param))
			}
		case AssetIssuer:
			if param != "" {
				query.Add("asset_issuer", string(param))
			}
		default:
		}
	}

	return query.Encode()
}
