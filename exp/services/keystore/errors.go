package keystore

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/problem"
)

var (
	errBadKeysBlob   = errors.New("invalid keys blob")
	errNotAuthorized = errors.New("request is not authorized")
)

var (
	probInvalidInput = problem.P{
		Type:   "invalid_keys_blob",
		Title:  "Invalid Keys Blob",
		Status: 400,
		Detail: "The keys blob in your request body is not a valid base64 string. " +
			"Please encode the keys blob in your request body as a base64 string " +
			"properly and try again.",
	}

	probNotAuthorized = problem.P{
		Type:   "not_authorized",
		Title:  "Not Authorized",
		Status: 401,
		Detail: "Your request is not authorized.",
	}
)
