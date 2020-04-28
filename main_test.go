package stellargo

import (
	"encoding/base64"
	"fmt"
	"github.com/stellar/go/network"
	"log"
	"strings"

	b "github.com/stellar/go/build"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/xdr"
)

// ExampleDecodeTransaction shows the lowest-level process to decode a base64
// envelope encoded in base64.
func ExampleDecodeTransaction() {
	data := "AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAACgAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAO5rKAAAAAAAAAAABVvwF9wAAAEAKZ7IPj/46PuWU6ZOtyMosctNAkXRNX9WCAI5RnfRk+AyxDLoDZP/9l3NvsxQtWj9juQOuoBlFLnWu8intgxQA"

	rawr := strings.NewReader(data)
	b64r := base64.NewDecoder(base64.StdEncoding, rawr)

	var tx xdr.TransactionEnvelope
	bytesRead, err := xdr.Unmarshal(b64r, &tx)

	fmt.Printf("read %d bytes\n", bytesRead)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("This tx has %d operations\n", len(tx.V0.Tx.Operations))
	// Output: read 192 bytes
	// This tx has 1 operations
}

// ExampleBuildTransaction creates and signs a simple transaction using the
// build package. The build package is designed to make it easier and more
// intuitive to configure and sign a transaction.
func ExampleBuildTransaction() {
	source := "SA26PHIKZM6CXDGR472SSGUQQRYXM6S437ZNHZGRM6QA4FOPLLLFRGDX"
	tx, err := b.Transaction(
		b.SourceAccount{source},
		b.Sequence{1},
		b.Payment(
			b.Destination{"SBQHO2IMYKXAYJFCWGXC7YKLJD2EGDPSK3IUDHVJ6OOTTKLSCK6Z6POM"},
			b.NativeAmount{"50.0"},
		),
	)
	if err != nil {
		panic(err)
	}

	txe, err := tx.Sign(source)
	if err != nil {
		panic(err)
	}

	txeB64, err := txe.Base64()
	if err != nil {
		panic(err)
	}

	fmt.Printf("tx base64: %s", txeB64)
}

// ExampleLowLevelTransaction creates and signs a simple transaction, and then
// encodes it into a hex string capable of being submitted to stellar-core.
//
// It uses the low-level xdr facilities to create the transaction.
func ExampleLowLevelTransaction() {
	skp := keypair.MustParse("SA26PHIKZM6CXDGR472SSGUQQRYXM6S437ZNHZGRM6QA4FOPLLLFRGDX")
	dkp := keypair.MustParse("SBQHO2IMYKXAYJFCWGXC7YKLJD2EGDPSK3IUDHVJ6OOTTKLSCK6Z6POM")

	asset, err := xdr.NewAsset(xdr.AssetTypeAssetTypeNative, nil)
	if err != nil {
		panic(err)
	}

	var destination xdr.MuxedAccount
	err = destination.SetAddress(dkp.Address())
	if err != nil {
		panic(err)
	}

	op := xdr.PaymentOp{
		Destination: destination,
		Asset:       asset,
		Amount:      50 * 10000000,
	}

	memo, err := xdr.NewMemo(xdr.MemoTypeMemoNone, nil)

	var source xdr.MuxedAccount
	err = source.SetAddress(skp.Address())
	if err != nil {
		panic(err)
	}

	body, err := xdr.NewOperationBody(xdr.OperationTypePayment, op)
	if err != nil {
		panic(err)
	}

	tx := xdr.TransactionV0{
		SourceAccountEd25519: source.MustEd25519(),
		Fee:                  10,
		SeqNum:               xdr.SequenceNumber(1),
		Memo:                 memo,
		Operations: []xdr.Operation{
			{Body: body},
		},
	}

	txHash, err := network.HashTransactionV0(tx, network.TestNetworkPassphrase)
	if err != nil {
		panic(err)
	}

	signature, err := skp.Sign(txHash[:])
	if err != nil {
		panic(err)
	}

	ds := xdr.DecoratedSignature{
		Hint:      skp.Hint(),
		Signature: xdr.Signature(signature[:]),
	}

	txe := xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTxV0,
		V0: &xdr.TransactionV0Envelope{
			Tx:         tx,
			Signatures: []xdr.DecoratedSignature{ds},
		},
	}

	txeB64, err := xdr.MarshalBase64(txe)
	if err != nil {
		panic(err)
	}

	var inverse xdr.TransactionEnvelope
	err = xdr.SafeUnmarshalBase64(txeB64, &inverse)
	if err != nil {
		panic(err)
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAAU08yUQ8sHqhY8j9mXWwERfHC/3cKFSe/spAr0rGtO2AAAACgAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAA+fnTe7/v4whpBUx96oj92jfZPz7S00l3O2xeyeqWIA0AAAAAAAAAAB3NZQAAAAAAAAAAASsa07YAAABAAAdMJ5pjUjX4BUZc/DkC28QOJWe7SBF4Y2hDPHfQ7WMayugxbPDQ88JI4qpynqSsIMjzHHaKtH3TPqdvyADEDQ==
}
