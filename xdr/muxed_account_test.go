package xdr_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/stellar/go/xdr"
)

var _ = Describe("xdr.MuxedAccount#Get/SetAddress()", func() {
	It("returns an empty string when muxed account is nil", func() {
		addy := (*MuxedAccount)(nil).Address()
		Expect(addy).To(Equal(""))
	})

	It("returns a strkey string when muxed account is valid", func() {
		var unmuxed MuxedAccount
		err := unmuxed.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(unmuxed.Type).To(Equal(CryptoKeyTypeKeyTypeEd25519))
		Expect(*unmuxed.Ed25519).To(Equal(Uint256{63, 12, 52, 191, 147, 173, 13, 153, 113, 208, 76, 204, 144, 247, 5, 81, 28, 131, 138, 173, 151, 52, 164, 162, 251, 13, 122, 3, 252, 127, 232, 154}))
		muxedy := unmuxed.Address()
		Expect(muxedy).To(Equal("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ"))

		var muxed MuxedAccount
		err = muxed.SetAddress("MCAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITKNOG")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(muxed.Type).To(Equal(CryptoKeyTypeKeyTypeMuxedEd25519))
		Expect(muxed.Med25519.Id).To(Equal(Uint64(9223372036854775808)))
		Expect(muxed.Med25519.Ed25519).To(Equal(*unmuxed.Ed25519))
		muxedy = muxed.Address()
		Expect(muxedy).To(Equal("MCAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITKNOG"))

		err = muxed.SetAddress("MAAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITLVL6")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(muxed.Type).To(Equal(CryptoKeyTypeKeyTypeMuxedEd25519))
		Expect(muxed.Med25519.Id).To(Equal(Uint64(0)))
		Expect(muxed.Med25519.Ed25519).To(Equal(*unmuxed.Ed25519))
		muxedy = muxed.Address()
		Expect(muxedy).To(Equal("MAAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITLVL6"))
	})

	It("returns an error when the strkey is invalid", func() {
		var muxed MuxedAccount

		// Test cases from SEP23

		err := muxed.SetAddress("GAAAAAAAACGC6")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("MAAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITLVL7")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZA")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJUACUSI")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("G47QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVP2I")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("MCAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITKNOGA")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("MAAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITIADJPA")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("M4AAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITIU2K")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("MAAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITLVL6===")
		Expect(err).Should(HaveOccurred())

		err = muxed.SetAddress("MAAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITLVL4")
		Expect(err).Should(HaveOccurred())
	})
})

var _ = Describe("xdr.MuxedAccountd#Equals()", func() {
	It("returns true when the account ids have equivalent values", func() {
		var l, r MuxedAccount
		err := l.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ")
		Expect(err).ShouldNot(HaveOccurred())
		err = r.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ")
		Expect(err).ShouldNot(HaveOccurred())

		Expect(l.Equals(r)).To(BeTrue())
		err = l.SetAddress("MCAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITKNOG")
		Expect(err).ShouldNot(HaveOccurred())
		err = r.SetAddress("MCAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITKNOG")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(l.Equals(r)).To(BeTrue())

		err = l.SetAddress("MAAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITLVL6")
		Expect(err).ShouldNot(HaveOccurred())
		err = r.SetAddress("MAAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITLVL6")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(l.Equals(r)).To(BeTrue())

	})

	It("returns false when the account ids have different values", func() {
		var l, r MuxedAccount
		err := l.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ")
		Expect(err).ShouldNot(HaveOccurred())
		err = r.SetAddress("GCR22L3WS7TP72S4Z27YTO6JIQYDJK2KLS2TQNHK6Y7XYPA3AGT3X4FH")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(l.Equals(r)).To(BeFalse())

		// Same key but different memo ids
		err = l.SetAddress("MCAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITKNOG")
		Expect(err).ShouldNot(HaveOccurred())
		err = r.SetAddress("MAAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITLVL6")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(l.Equals(r)).To(BeFalse())

		err = l.SetAddress("MCAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITKNOG")
		Expect(err).ShouldNot(HaveOccurred())
		err = r.SetAddress("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ")
		Expect(err).ShouldNot(HaveOccurred())
		Expect(l.Equals(r)).To(BeFalse())
	})
})

var _ = Describe("xdr.AddressToMuxedAccount()", func() {
	It("works", func() {
		address := "GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ"
		muxedAccount, err := AddressToMuxedAccount(address)

		Expect(muxedAccount.Address()).To(Equal("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ"))
		Expect(err).ShouldNot(HaveOccurred())

		_, err = AddressToAccountId("GCR22L3")

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

		err = muxed.SetAddress("MCAAAAAAAAAAAAB7BQ2L7E5NBWMXDUCMZSIPOBKRDSBYVLMXGSSKF6YNPIB7Y77ITKNOG")
		Expect(err).ShouldNot(HaveOccurred())
		aid = muxed.ToAccountId()
		Expect(aid.Address()).To(Equal("GA7QYNF7SOWQ3GLR2BGMZEHXAVIRZA4KVWLTJJFC7MGXUA74P7UJVSGZ"))
	})
})
