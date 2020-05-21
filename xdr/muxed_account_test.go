package xdr_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/stellar/go/xdr"
)

var _ = Describe("xdr.MuxedAccount#Get/SetAddress()", func() {

	It("returns an error when the strkey is invalid", func() {
		var muxed MuxedAccount

		// Test cases from SEP23

		err := muxed.SetAddress("GAAAAAAAACGC6")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZA")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUACUSI")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("G47QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVP2I")
		Expect(err).Should(HaveOccurred())

	})
})

var _ = Describe("xdr.MuxedAccount.ToAccountId()", func() {
	It("works", func() {
		var muxed MuxedAccount
		err := muxed.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ")
		Expect(err).ShouldNot(HaveOccurred())
		aid := muxed.ToAccountId()
		Expect(aid.Address()).To(Equal("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ"))

		muxed = MuxedAccount{
			Type: CryptoKeyTypeKeyTypeMuxedEd25519,
			Med25519: &MuxedAccountMed25519{
				Id:      0xcafebabe,
				Ed25519: *muxed.Ed25519,
			},
		}

		aid = muxed.ToAccountId()
		Expect(aid.Address()).To(Equal("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ"))
	})
})
