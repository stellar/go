package xdr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLedgerEntrySponsorship(t *testing.T) {
	entry := LedgerEntry{}
	desc := entry.SponsoringID()
	assert.Nil(t, desc)

	sponsor := MustAddress("GCO26ZSBD63TKYX45H2C7D2WOFWOUSG5BMTNC3BG4QMXM3PAYI6WHKVZ")
	desc = SponsorshipDescriptor(&sponsor)

	entry = LedgerEntry{
		Ext: LedgerEntryExt{
			V: 1,
			V1: &LedgerEntryExtensionV1{
				SponsoringId: desc,
			},
		},
	}
	actualDesc := entry.SponsoringID()
	assert.Equal(t, desc, actualDesc)
}
