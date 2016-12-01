package compliance

import (
	"encoding/base64"
	"encoding/json"

	"github.com/stellar/go/clients/stellartoml"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
)

// Validate checks if fields are valid:
//
//   * `Data` and `Signature` not empty
//   * `Data` is JSON and pass AuthData.Validate method
//
// This method only performs data validation. You should also call
// VerifySignature to confirm that signature is valid.
func (r *AuthRequest) Validate() error {
	if r.DataJSON == "" || r.Signature == "" {
		return errors.New("`data` and `signature` fields are required")
	}

	authData := AuthData{}
	err := json.Unmarshal([]byte(r.DataJSON), &authData)
	if err != nil {
		return errors.Wrap(err, "Data is not valid JSON")
	}

	err = authData.Validate()
	if err != nil {
		return errors.Wrap(err, "Invalid Data")
	}

	return nil
}

// VerifySignature verifies if signature is valid. It makes a network connection
// to sender server in order to obtain stellar.toml file and signing key.
func (r *AuthRequest) VerifySignature(sender string) error {
	senderStellarToml, err := stellartoml.GetStellarTomlByAddress(sender)
	if err != nil {
		return errors.Wrap(err, "Cannot get stellar.toml of sender")
	}

	if senderStellarToml.SigningKey == "" {
		return errors.New("No SIGNING_KEY in stellar.toml of sender")
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(r.Signature)
	if err != nil {
		return errors.Wrap(err, "Signature is not base64 encoded")
	}

	kp, err := keypair.Parse(senderStellarToml.SigningKey)
	if err != nil {
		return errors.Wrap(err, "SigningKey is invalid")
	}

	err = kp.Verify([]byte(r.DataJSON), signatureBytes)
	if err != nil {
		return errors.Wrap(err, "Signature is invalid")
	}

	return nil
}

// Data returns AuthData from the request. This method
// assumes that Data is valid JSON (checked in Validate).
func (r *AuthRequest) Data() (data *AuthData) {
	json.Unmarshal([]byte(r.DataJSON), &data)
	return
}
