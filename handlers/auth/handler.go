package auth

import (
//
)

func (rh *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authreq := &compliance.AuthRequest{}
	authreq.FromRequest(r)

	// Validate request
	err := authreq.Validate()
	// ...

	// Unmarshal AuthData
	var authData compliance.AuthData
	err = json.Unmarshal([]byte(authreq.Data), &authData)
	// ...

	// Fetch sender stellar.toml and check if SigningKey is present
	senderStellarToml, err := rh.StellarTomlResolver.GetStellarTomlByAddress(authData.Sender)
	// ...
	if senderStellarToml.SigningKey == "" {
		// ...
		return
	}

	// Verify signature
	signatureBytes, err := base64.StdEncoding.DecodeString(authreq.Signature)
	// ...
	err = rh.SignatureSignerVerifier.Verify(senderStellarToml.SigningKey, []byte(authreq.Data), signatureBytes)
	// ...

	// Check if tx is valid
	b64r := base64.NewDecoder(base64.StdEncoding, strings.NewReader(authData.Tx))
	var tx xdr.Transaction
	_, err = xdr.Unmarshal(b64r, &tx)
	// ...

	if tx.Memo.Hash == nil {
		// ...
	}

	// Validate memo preimage hash
	memoPreimageHashBytes := sha256.Sum256([]byte(authData.Memo))
	memoBytes := [32]byte(*tx.Memo.Hash)

	if memoPreimageHashBytes != memoBytes {
		// ...
		return
	}

	// Unmarshal memo preimage
	var memoPreimage memo.Memo
	err = json.Unmarshal([]byte(authData.Memo), &memoPreimage)
	if err != nil {
		// ...
		return
	}

	// Create response
	response := compliance.AuthResponse{}

	// Sanctions check
	/////////////////////////////////////////////////////////////////////////
	err := rh.Strategy.SanctionsCheck(authData, &response)
	if err != nil {
		// Handle error
	}
	/////////////////////////////////////////////////////////////////////////

	// User info
	/////////////////////////////////////////////////////////////////////////
	err := rh.Strategy.GetUserData(authData, &response)
	if err != nil {
		// Handle error
	}
	/////////////////////////////////////////////////////////////////////////

	if response.TxStatus == compliance.AuthStatusOk && response.InfoStatus == compliance.AuthStatusOk {
		err = rh.Strategy.PersistTransaction(authData)
		if err != nil {
			// ...
			return
		}
	}

	server.Write(w, &response)
}
