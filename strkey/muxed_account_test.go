package strkey

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMuxedAccount_ID(t *testing.T) {
	muxed := MuxedAccount{}
	assert.Equal(t, uint64(0), muxed.ID())

	muxed = MuxedAccount{id: uint64(9223372036854775808)}
	assert.Equal(t, uint64(9223372036854775808), muxed.ID())
}

func TestMuxedAccount_SetID(t *testing.T) {
	muxed := MuxedAccount{}
	muxed.SetID(123)
	assert.Equal(t, uint64(123), muxed.ID())

	muxed.SetID(456)
	assert.Equal(t, uint64(456), muxed.ID())
}

func TestMuxedAccount_Address(t *testing.T) {
	muxed := MuxedAccount{}
	publicKey, err := muxed.Address()
	assert.EqualError(t, err, "muxed account has no ed25519 key")
	assert.Empty(t, publicKey)

	muxed = MuxedAccount{ed25519: [32]byte{63, 12, 52, 191, 147, 173, 13, 153, 113, 208, 76, 204, 144, 247, 5, 81, 28, 131, 138, 173, 151, 52, 164, 162, 251, 13, 122, 3, 252, 127, 232, 154}}
	publicKey, err = muxed.Address()
	assert.NoError(t, err)
	assert.Equal(t, "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ", publicKey)
}

func TestMuxedAccount_SetAddress(t *testing.T) {
	muxed := MuxedAccount{}
	err := muxed.SetAddress("")
	assert.EqualError(t, err, "invalid ed25519 public key")

	err = muxed.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZ")
	assert.EqualError(t, err, "invalid ed25519 public key")

	err = muxed.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ")
	assert.NoError(t, err)
	publicKey, err := muxed.Address()
	assert.NoError(t, err)
	assert.Equal(t, "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ", publicKey)
	wantMuxed := MuxedAccount{ed25519: [32]byte{63, 12, 52, 191, 147, 173, 13, 153, 113, 208, 76, 204, 144, 247, 5, 81, 28, 131, 138, 173, 151, 52, 164, 162, 251, 13, 122, 3, 252, 127, 232, 154}}
	assert.Equal(t, wantMuxed, muxed)

	muxed.SetID(123)
	wantMuxed = MuxedAccount{
		ed25519: [32]byte{63, 12, 52, 191, 147, 173, 13, 153, 113, 208, 76, 204, 144, 247, 5, 81, 28, 131, 138, 173, 151, 52, 164, 162, 251, 13, 122, 3, 252, 127, 232, 154},
		id:      123,
	}
	assert.Equal(t, wantMuxed, muxed)
}

func TestMuxedAccount_MuxedAddress(t *testing.T) {
	muxed := MuxedAccount{}
	publicKey, err := muxed.MuxedAddress()
	assert.EqualError(t, err, "muxed account has no ed25519 key")
	assert.Empty(t, publicKey)

	muxed = MuxedAccount{
		id:      uint64(9223372036854775808),
		ed25519: [32]byte{63, 12, 52, 191, 147, 173, 13, 153, 113, 208, 76, 204, 144, 247, 5, 81, 28, 131, 138, 173, 151, 52, 164, 162, 251, 13, 122, 3, 252, 127, 232, 154},
	}
	publicKey, err = muxed.MuxedAddress()
	assert.NoError(t, err)
	assert.Equal(t, "MA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVAAAAAAAAAAAAAJLK", publicKey)
}

func TestParseMuxedAccount(t *testing.T) {
	muxed, err := ParseMuxedAccount("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ")
	assert.EqualError(t, err, "invalid muxed account")
	assert.Nil(t, muxed)

	muxed, err = ParseMuxedAccount("MA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVAAAAAAAAAAAAAJLK")
	assert.NoError(t, err)
	assert.Equal(t, uint64(9223372036854775808), muxed.ID())
	publicKey, err := muxed.Address()
	assert.NoError(t, err)
	assert.Equal(t, "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ", publicKey)
}
