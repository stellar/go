package httpdecode

import (
	"encoding/json"
	"net/http"
)

// DecodeJSON decodes JSON request from r into v.
func DecodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	return dec.Decode(v)
}
