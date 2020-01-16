package httpdecode

import (
	"encoding/json"
	"mime"
	"net/http"

	"github.com/gorilla/schema"
	"github.com/stellar/go/support/errors"
)

// DecodeJSON decodes JSON request from r into v.
func DecodeJSON(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.UseNumber()
	return dec.Decode(v)
}

// DecodeForm decodes form URL encoded requests from r into v.
//
// The type of the value given can use `form` tags on fields in the same way as
// the `json` tag to name fields.
//
// An error will be returned if the request is not a POST, PUT, or PATCH
// request.
//
// An error will be returned if the request has a media type in the
// Content-Type not equal to application/x-www-form-urlencoded, or if the
// Content-Type header cannot be parsed.
func DecodeForm(r *http.Request, v interface{}) error {
	if r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" {
		return errors.Errorf("method POST, PUT, or PATCH required for form decoding: request has method %q", r.Method)
	}

	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return errors.Wrap(err, "content type application/x-www-form-urlencoded required for form decoding")
	}
	if mediaType != "application/x-www-form-urlencoded" {
		return errors.Errorf("content type application/x-www-form-urlencoded required for form decoding: received content type %q", mediaType)
	}

	err = r.ParseForm()
	if err != nil {
		return err
	}

	dec := schema.NewDecoder()
	dec.SetAliasTag("form")
	dec.IgnoreUnknownKeys(true)
	return dec.Decode(v, r.PostForm)
}

// Decode decodes form URL encoded requests and JSON requests from r into v.
//
// The requests Content-Type header informs if the request should be decoded
// using a form URL encoded decoder or using a JSON decoder.
//
// A Content-Type of application/x-www-form-urlencoded will result in form
// decoding. Any other content type will result in JSON decoding because it is
// common to make JSON requests without a Content-Type where-as correctly
// formatted form URL encoded requests are more often accompanied by the
// appropriate Content-Type.
//
// An error is returned if the Content-Type cannot be parsed by a mime
// media-type parser.
//
// See DecodeForm and DecodeJSON for details about the types of errors that may
// occur.
func Decode(r *http.Request, v interface{}) error {
	contentType := r.Header.Get("Content-Type")
	if contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			return errors.Wrap(err, "content type could not be parsed")
		}
		if mediaType == "application/x-www-form-urlencoded" {
			return DecodeForm(r, v)
		}
	}
	return DecodeJSON(r, v)
}
