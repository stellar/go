package gxdr

import (
	"encoding/base64"
	"strings"

	goxdr "github.com/xdrpp/goxdr/xdr"
)

const DefaultMaxDepth = 500

type depthLimiter struct {
	depth    int
	maxDepth int
	decoder  *goxdr.XdrIn
}

func (*depthLimiter) Sprintf(f string, args ...interface{}) string {
	return ""
}

func (d *depthLimiter) Marshal(field string, i goxdr.XdrType) {
	switch t := goxdr.XdrBaseType(i).(type) {
	case goxdr.XdrAggregate:
		if d.depth > d.maxDepth {
			goxdr.XdrPanic("max depth of %d exceeded", d.maxDepth)
		}
		d.depth++
		t.XdrRecurse(d, field)
		d.depth--
	default:
		d.decoder.Marshal(field, t)
	}
}

// ValidateTransactionEnvelope validates the given transaction envelope
// to make sure that it does not contain malicious arrays or nested
// structures which are too deep
func ValidateTransactionEnvelope(b64Envelope string, maxDepth int) error {
	return validate(b64Envelope, &TransactionEnvelope{}, maxDepth)
}

// ValidateLedgerKey validates the given ledger key
// to make sure that it does not contain malicious arrays or nested
// structures which are too deep
func ValidateLedgerKey(b64Key string, maxDepth int) error {
	return validate(b64Key, &LedgerKey{}, maxDepth)
}

func validate(b64 string, val goxdr.XdrType, maxDepth int) (err error) {
	d := &depthLimiter{
		depth:    0,
		maxDepth: maxDepth,
		decoder: &goxdr.XdrIn{
			In: base64.NewDecoder(base64.StdEncoding, strings.NewReader(b64)),
		},
	}

	defer func() {
		switch i := recover().(type) {
		case nil:
		case goxdr.XdrError:
			err = i
		default:
			panic(i)
		}
	}()
	val.XdrMarshal(d, "")
	return nil
}
