package benchmarks

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/stellar/go/gxdr"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
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

func BenchmarkXDRUnmarshal(b *testing.B) {
	b.StopTimer()
	te := xdr.TransactionEnvelope{}

	// Make sure the input is valid.
	err := te.UnmarshalBinary(input)
	require.NoError(b, err)
	b.StartTimer()
	// Benchmark.
	for i := 0; i < b.N; i++ {
		_ = te.UnmarshalBinary(input)
	}
}

func BenchmarkGXDRUnmarshal(b *testing.B) {
	b.StopTimer()
	te := gxdr.TransactionEnvelope{}

	// Make sure the input is valid, note goxdr will panic if there's a
	// marshaling error.
	te.XdrMarshal(&goxdr.XdrIn{In: bytes.NewReader(input)}, "")
	b.StartTimer()

	// Benchmark.
	r := bytes.NewReader(input)
	for i := 0; i < b.N; i++ {
		r.Reset(input)
		te.XdrMarshal(&goxdr.XdrIn{In: r}, "")
	}
}

func BenchmarkXDRMarshal(b *testing.B) {
	b.StopTimer()
	te := xdr.TransactionEnvelope{}

	// Make sure the input is valid.
	err := te.UnmarshalBinary(input)
	require.NoError(b, err)
	output, err := te.MarshalBinary()
	require.NoError(b, err)
	require.Equal(b, input, output)
	b.StartTimer()

	// Benchmark.
	for i := 0; i < b.N; i++ {
		_, _ = te.MarshalBinary()
	}
}

func BenchmarkGXDRMarshal(b *testing.B) {
	b.StopTimer()
	te := gxdr.TransactionEnvelope{}

	// Make sure the input is valid, note goxdr will panic if there's a
	// marshaling error.
	te.XdrMarshal(&goxdr.XdrIn{In: bytes.NewReader(input)}, "")
	output := bytes.Buffer{}
	te.XdrMarshal(&goxdr.XdrOut{Out: &output}, "")

	b.StartTimer()
	// Benchmark.
	for i := 0; i < b.N; i++ {
		output.Reset()
		te.XdrMarshal(&goxdr.XdrOut{Out: &output}, "")
	}
}
