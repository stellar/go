package keystore

import (
	"github.com/stellar/go/support/render/problem"
)

var (
	probInvalidRequest = problem.P{
		Type:   "invalid_request_body",
		Title:  "Invalid Request Body",
		Status: 400,
		Detail: "Your request body is invalid.",
	}

	probInvalidKeysBlob = problem.P{
		Type:   "invalid_keys_blob",
		Title:  "Invalid Keys Blob",
		Status: 400,
		Detail: "The keys-blob in your request body is not a valid base64 string. " +
			"Please encode the keys-blob in your request body as a base64 string " +
			"properly and try again.",
	}

	probNotAuthorized = problem.P{
		Type:   "not_authorized",
		Title:  "Not Authorized",
		Status: 401,
		Detail: "Your request is not authorized.",
	}

	probDuplicateKeys = problem.P{
		Type:   "duplicate_keys",
		Title:  "Duplicate Keys",
		Status: 400,
		Detail: "You have previously stored a keys-blob with the same encrypter. " +
			"Please use the update endpoint if you wish to modify the keys-blob.",
	}
)
