package utf8

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
)

func TestScrub(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	tt.Assert.Equal("scott", Scrub("scott"))
	tt.Assert.Equal("scött", Scrub("scött"))
	tt.Assert.Equal("�(", Scrub(string([]byte{0xC3, 0x28})))
}
