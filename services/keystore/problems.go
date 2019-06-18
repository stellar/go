package keystore

import (
	"errors"
	"net/http"

	"github.com/stellar/go/support/render/problem"
)

var errRequiredField = errors.New("field value cannot be empty")

var (
	probInvalidRequest = problem.P{
		Type:   "invalid_request_body",
		Title:  "Invalid Request Body",
		Status: 400,
		Detail: "Your request body is invalid.",
	}

	probMethodNotAllowed = problem.P{
		Type:   "method_not_allowed",
		Title:  "Method Not Allowed",
		Status: http.StatusMethodNotAllowed,
		Detail: "This endpoint does not support the request method you used. " +
			"The server supports HTTP GET/PUT/DELETE for the /keys endpoint.",
	}

	probInvalidKeysBlob = problem.P{
		Type:   "invalid_keys_blob",
		Title:  "Invalid Keys Blob",
		Status: 400,
		Detail: "The keysBlob in your request body is not a valid base64 string. " +
			"Please encode the keysBlob in your request body as a base64 string " +
			"properly and try again.",
	}

	probNotAuthorized = problem.P{
		Type:   "not_authorized",
		Title:  "Not Authorized",
		Status: 401,
		Detail: "Your request is not authorized.",
	}
)
