package horizonclient

import (
	"encoding/json"

	hProtocol "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func (herr Error) Error() string {
	return `Horizon error: "` + herr.Problem.Title + `". Check horizon.Error.Problem for more information.`
}

// Envelope extracts the transaction envelope that triggered this error from the
// extra fields.
func (herr *Error) Envelope() (*xdr.TransactionEnvelope, error) {
	raw, ok := herr.Problem.Extras["envelope_xdr"]
	if !ok {
		return nil, ErrEnvelopeNotPopulated
	}

	var b64 string
	var result xdr.TransactionEnvelope
	b64, ok = raw.(string)
	if !ok {
		return nil, errors.New("type assertion failed")
	}

	err := xdr.SafeUnmarshalBase64(b64, &result)
	if err != nil {
		return nil, errors.Wrap(err, "xdr decode failed")
	}

	return &result, nil
}

// ResultString extracts the transaction result as a string.
func (herr *Error) ResultString() (string, error) {
	raw, ok := herr.Problem.Extras["result_xdr"]
	if !ok {
		return "", ErrResultNotPopulated
	}

	b64, ok := raw.(string)
	if !ok {
		return "", errors.New("type assertion failed")
	}

	return b64, nil
}

// ResultCodes extracts a result code summary from the error, if possible.
func (herr *Error) ResultCodes() (*hProtocol.TransactionResultCodes, error) {

	raw, ok := herr.Problem.Extras["result_codes"]
	if !ok {
		return nil, ErrResultCodesNotPopulated
	}

	// converts map to []byte
	dataString, err := json.Marshal(raw)
	if err != nil {
		return nil, errors.Wrap(err, "Marshaling failed")
	}

	var result hProtocol.TransactionResultCodes
	if err = json.Unmarshal(dataString, &result); err != nil {
		return nil, errors.Wrap(err, "Unmarshaling failed")
	}

	return &result, nil
}
