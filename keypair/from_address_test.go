package keypair

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("keypair.FromAddress", func() {
	var subject KP

	JustBeforeEach(func() {
		subject = &FromAddress{address}
	})

	ItBehavesLikeAKP(&subject)

	Describe("Sign()", func() {
		It("fails", func() {
			_, err := subject.Sign(message)
			Expect(err).To(HaveOccurred())
		})

	})
	Describe("SignBase64()", func() {
		It("fails", func() {
			_, err := subject.SignBase64(message)
			Expect(err).To(HaveOccurred())
		})

	})
	Describe("SignDecorated()", func() {
		It("fails", func() {
			_, err := subject.SignDecorated(message)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("MarshalText()", func() {
		type Case struct {
			Input     *FromAddress
			BytesCase types.GomegaMatcher
			ErrCase   types.GomegaMatcher
		}
		DescribeTable("MarshalText()",
			func(c Case) {
				bytes, err := c.Input.MarshalText()
				Expect(bytes).To(c.BytesCase)
				Expect(err).To(c.ErrCase)
			},
			Entry("a valid address", Case{
				Input:     &FromAddress{"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"},
				BytesCase: Equal([]byte("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")),
				ErrCase:   Not(HaveOccurred()),
			}),
			Entry("an empty address", Case{
				Input:     &FromAddress{},
				BytesCase: Equal([]byte("")),
				ErrCase:   Not(HaveOccurred()),
			}),
		)
	})

	Describe("UnmarshalText()", func() {
		type Case struct {
			Address     *FromAddress
			Input       []byte
			AddressCase types.GomegaMatcher
			ErrCase     types.GomegaMatcher
			FuncCase    types.GomegaMatcher
		}
		DescribeTable("UnmarshalText()",
			func(c Case) {
				f := func() {
					err := c.Address.UnmarshalText(c.Input)
					Expect(c.Address).To(c.AddressCase)
					Expect(err).To(c.ErrCase)
				}
				Expect(f).To(c.FuncCase)
			},
			Entry("a valid address into an empty FromAddress", Case{
				Address:     &FromAddress{},
				Input:       []byte("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"),
				AddressCase: Equal(&FromAddress{"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"}),
				ErrCase:     Not(HaveOccurred()),
				FuncCase:    Not(Panic()),
			}),
			Entry("an invalid address into an empty FromAddress", Case{
				Address:     &FromAddress{},
				Input:       []byte("?BRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"),
				AddressCase: Equal(&FromAddress{}),
				ErrCase:     HaveOccurred(),
				FuncCase:    Not(Panic()),
			}),
			Entry("a valid address into a nil FromAddress", Case{
				// This test case is included to indicate nil handling is not
				// supported. Handling this case is unnecessary because the
				// encoding packages in the stdlib protect against unmarshaling
				// into nil objects when calling Unmarshal directly on a nil
				// object and by allocating a new value in other cases.
				Address:     nil,
				Input:       []byte("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"),
				AddressCase: Equal(&FromAddress{"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"}),
				ErrCase:     Not(HaveOccurred()),
				FuncCase:    Panic(),
			}),
		)
	})

})
