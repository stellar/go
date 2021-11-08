package benchmarks

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stellar/go/gxdr"
	"github.com/stellar/go/xdr"
	goxdr "github.com/xdrpp/goxdr/xdr"
)

const input64 = "AAAAAgAAAACfHrX0tYB0gpXuJYTN9os06cdF62KAaqY9jid+777eyQAAC7gCM9czAAi/DQAAAAEAAAAAAAAAAAAAAABhga2dAAAAAAAAAAMAAAAAAAAADAAAAAAAAAABTU9CSQAAAAA8cTArnmXa4wEQJxDHOw5SwBaDVjBfAP5lRMNZkRtlZAAAAAAG42RBAAf7lQCYloAAAAAAMgbg0AAAAAAAAAADAAAAAU1PQkkAAAAAPHEwK55l2uMBECcQxzsOUsAWg1YwXwD+ZUTDWZEbZWQAAAAAAAAADkpyV7kAARBNABMS0AAAAAAyBuDRAAAAAAAAAAMAAAABTU9CSQAAAAA8cTArnmXa4wEQJxDHOw5SwBaDVjBfAP5lRMNZkRtlZAAAAAAAAAAclOSvewAIl5kAmJaAAAAAADIG4NIAAAAAAAAAAe++3skAAABAs2jt6+cyeyFvXVFphBcwt18GXnj7Jwa+hWQRyaBmPOSR2415GBi8XY3lC4m4aX9S322HvHjrxgQiar7KjgnQDw=="

var input = func() []byte {
	decoded, err := base64.StdEncoding.DecodeString(input64)
	if err != nil {
		panic(err)
	}
	return decoded
}()

var xdrInput = func() xdr.TransactionEnvelope {
	var te xdr.TransactionEnvelope
	if err := te.UnmarshalBinary(input); err != nil {
		panic(err)
	}
	return te
}()

var gxdrInput = func() gxdr.TransactionEnvelope {
	var te gxdr.TransactionEnvelope
	// note goxdr will panic if there's a marshaling error.
	te.XdrMarshal(&goxdr.XdrIn{In: bytes.NewReader(input)}, "")
	return te
}()

func BenchmarkXDRUnmarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = xdrInput.UnmarshalBinary(input)
	}
}

func BenchmarkGXDRUnmarshal(b *testing.B) {
	var te gxdr.TransactionEnvelope
	r := bytes.NewReader(input)
	for i := 0; i < b.N; i++ {
		r.Reset(input)
		te.XdrMarshal(&goxdr.XdrIn{In: r}, "")
	}
}

func BenchmarkXDRMarshal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = xdrInput.MarshalBinary()
	}
}

func BenchmarkXDRMarshalWithEncoder(b *testing.B) {
	e := xdr.NewEncoder()
	for i := 0; i < b.N; i++ {
		_, _ = e.UnsafeMarshalBinary(xdrInput)
	}
}

func BenchmarkGXDRMarshal(b *testing.B) {
	var output bytes.Buffer
	// Benchmark.
	for i := 0; i < b.N; i++ {
		output.Reset()
		gxdrInput.XdrMarshal(&goxdr.XdrOut{Out: &output}, "")
	}
}

func BenchmarkXDRMarshalHex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = xdr.MarshalHex(xdrInput)
	}
}

func BenchmarkXDRMarshalHexWithEncoder(b *testing.B) {
	e := xdr.NewEncoder()
	for i := 0; i < b.N; i++ {
		_, _ = e.MarshalHex(xdrInput)
	}
}

func BenchmarkXDRUnsafeMarshalHexWithEncoder(b *testing.B) {
	e := xdr.NewEncoder()
	for i := 0; i < b.N; i++ {
		_, _ = e.UnsafeMarshalHex(xdrInput)
	}
}

func BenchmarkXDRMarshalBase64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = xdr.MarshalBase64(xdrInput)
	}
}

func BenchmarkXDRMarshalBase64WithEncoder(b *testing.B) {
	e := xdr.NewEncoder()
	for i := 0; i < b.N; i++ {
		_, _ = e.MarshalBase64(xdrInput)
	}
}

func BenchmarkXDRUnsafeMarshalBase64WithEncoder(b *testing.B) {
	e := xdr.NewEncoder()
	for i := 0; i < b.N; i++ {
		_, _ = e.UnsafeMarshalBase64(xdrInput)
	}
}
