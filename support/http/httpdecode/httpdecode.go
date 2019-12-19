package httpdecode

import (
	"encoding/json"
	"net/http"
)

// DecodeJSON decodes JSON request from r into v.
func DecodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	err := dec.Decode(v)
	if err != nil {
		return err
	}

	return nil
}
