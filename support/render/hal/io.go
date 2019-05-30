package hal

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/stellar/go/support/errors"
)

// renderToString renders the provided data as a json string
func renderToString(data interface{}, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(data, "", "  ")
	}

	return json.Marshal(data)
}

// Render write data to w, after marshalling to json
func Render(w http.ResponseWriter, data interface{}) {
	js, err := renderToString(data, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Content-Type", "application/hal+json; charset=utf-8")
	w.Write(js)
}

var ErrBadRequest = errors.New("bad request")

// read decodes a json text from r into v.
func read(r io.Reader, v interface{}) error {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	err := dec.Decode(v)
	if err != nil {
		return ErrBadRequest
	}

	return nil
}
