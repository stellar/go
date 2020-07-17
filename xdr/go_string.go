package xdr

import (
	"fmt"
	"strconv"
	"strings"
)

// GoString prints Uint32 as decimal instead of hexadecimal numbers.
func (u Uint32) GoString() string {
	return strconv.FormatInt(int64(u), 10)
}

func (e TransactionEnvelope) GoString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("xdr.TransactionEnvelope{Type: xdr.%s,", envelopeTypeMap[int32(e.Type)]))
	switch e.Type {
	case EnvelopeTypeEnvelopeTypeTxV0:
		sb.WriteString(fmt.Sprintf("V0: &%#v,", *e.V0))
	case EnvelopeTypeEnvelopeTypeTx:
		sb.WriteString(fmt.Sprintf("V1: &%#v,", *e.V1))
	case EnvelopeTypeEnvelopeTypeTxFeeBump:
		sb.WriteString(fmt.Sprintf("FeeBump: &%#v,", *e.FeeBump))
	default:
		panic("Unknown type")
	}
	sb.WriteString("}")
	return sb.String()
}

func (a Asset) GoString() string {
	if a.Type == AssetTypeAssetTypeNative {
		return "xdr.MustNewNativeAsset()"
	}

	var typ, code, issuer string
	a.MustExtract(&typ, &code, &issuer)
	return fmt.Sprintf("xdr.MustNewCreditAsset(%#v, %#v)", code, issuer)
}

func (m Memo) GoString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("xdr.Memo{Type: xdr.%s", memoTypeMap[int32(m.Type)]))
	switch m.Type {
	case MemoTypeMemoNone:
	case MemoTypeMemoText:
		sb.WriteString(fmt.Sprintf(",Text: &%#v,", *m.Text))
	case MemoTypeMemoId:
		sb.WriteString(fmt.Sprintf(",Id: &%#v,", *m.Id))
	case MemoTypeMemoHash:
		sb.WriteString(fmt.Sprintf(",Hash: &%#v,", *m.Hash))
	case MemoTypeMemoReturn:
		sb.WriteString(fmt.Sprintf(",RetHash: &%#v,", *m.RetHash))
	default:
		panic("Unknown type")
	}
	sb.WriteString("}")
	return sb.String()
}

func (m MuxedAccount) GoString() string {
	switch m.Type {
	case CryptoKeyTypeKeyTypeEd25519:
		accountID := m.ToAccountId()
		return fmt.Sprintf("xdr.MustAddress(%#v)", accountID.Address())
	case CryptoKeyTypeKeyTypeMuxedEd25519:
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("xdr.MuxedAccount{Type: %d,", m.Type))
		sb.WriteString(fmt.Sprintf("Med25519: &%#v,", *m.Med25519))
		sb.WriteString("}")
		return sb.String()
	default:
		panic("Unknown type")
	}

}

func (o Operation) GoString() string {
	var sb strings.Builder
	sb.WriteString("xdr.OperationBody{")
	if o.SourceAccount != nil {
		sb.WriteString(fmt.Sprintf("SourceAccount: &%#v,", *o.SourceAccount))
	}
	sb.WriteString(fmt.Sprintf("Body: &%#v,", o.Body))
	sb.WriteString("}")
	return sb.String()
}

func (o OperationBody) GoString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("xdr.OperationBody{Type: xdr.%s,", operationTypeMap[int32(o.Type)]))
	switch {
	case o.CreateAccountOp != nil:
		sb.WriteString(fmt.Sprintf("CreateAccountOp: &%#v,", *o.CreateAccountOp))
	case o.PaymentOp != nil:
		sb.WriteString(fmt.Sprintf("PaymentOp: &%#v,", *o.PaymentOp))
	case o.PathPaymentStrictReceiveOp != nil:
		sb.WriteString(fmt.Sprintf("PathPaymentStrictReceiveOp: &%#v,", *o.PathPaymentStrictReceiveOp))
	case o.ManageSellOfferOp != nil:
		sb.WriteString(fmt.Sprintf("ManageSellOfferOp: &%#v,", *o.ManageSellOfferOp))
	case o.CreatePassiveSellOfferOp != nil:
		sb.WriteString(fmt.Sprintf("CreatePassiveSellOfferOp: &%#v,", *o.CreatePassiveSellOfferOp))
	case o.SetOptionsOp != nil:
		sb.WriteString(fmt.Sprintf("SetOptionsOp: &%#v,", *o.SetOptionsOp))
	case o.ChangeTrustOp != nil:
		sb.WriteString(fmt.Sprintf("ChangeTrustOp: &%#v,", *o.ChangeTrustOp))
	case o.AllowTrustOp != nil:
		sb.WriteString(fmt.Sprintf("AllowTrustOp: &%#v,", *o.AllowTrustOp))
	case o.Destination != nil:
		sb.WriteString(fmt.Sprintf("Destination: &%#v,", *o.Destination))
	case o.ManageDataOp != nil:
		sb.WriteString(fmt.Sprintf("ManageDataOp: &%#v,", *o.ManageDataOp))
	case o.BumpSequenceOp != nil:
		sb.WriteString(fmt.Sprintf("BumpSequenceOp: &%#v,", *o.BumpSequenceOp))
	case o.ManageBuyOfferOp != nil:
		sb.WriteString(fmt.Sprintf("ManageBuyOfferOp: &%#v,", *o.ManageBuyOfferOp))
	case o.PathPaymentStrictSendOp != nil:
		sb.WriteString(fmt.Sprintf("PathPaymentStrictSendOp: &%#v,", *o.PathPaymentStrictSendOp))
	default:
		panic("Unknown type")
	}
	sb.WriteString("}")
	return sb.String()
}

func (t *TimeBounds) GoString() string {
	if t == nil {
		return "nil"
	}
	return fmt.Sprintf("&xdr.TimeBounds{MinTime: xdr.TimePoint(%d), MaxTime: xdr.TimePoint(%d)}", t.MinTime, t.MaxTime)
}
