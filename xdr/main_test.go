package xdr

import (
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// ExampleUnmarshal shows the lowest-level process to decode a base64
// envelope encoded in base64.
func ExampleUnmarshal() {
	data := "AAAAAgAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wAAAAoAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAADuaygAAAAAAAAAAAVb8BfcAAABACmeyD4/+Oj7llOmTrcjKLHLTQJF0TV/VggCOUZ30ZPgMsQy6A2T//Zdzb7MULVo/Y7kDrqAZRS51rvIp7YMUAA=="

	rawr := strings.NewReader(data)
	b64r := base64.NewDecoder(base64.StdEncoding, rawr)

	var tx TransactionEnvelope
	bytesRead, err := Unmarshal(b64r, &tx)

	fmt.Printf("read %d bytes\n", bytesRead)

	if err != nil {
		log.Fatal(err)
	}

	operations := tx.Operations()
	fmt.Printf("This tx has %d operations\n", len(operations))
	// Output: read 196 bytes
	// This tx has 1 operations
}

var _ = Describe("xdr.SafeUnmarshal", func() {
	var (
		result int32
		data   []byte
		err    error
	)

	JustBeforeEach(func() {
		err = SafeUnmarshal(data, &result)
	})

	Context("input data is a single xdr value", func() {
		BeforeEach(func() {
			data = []byte{0x00, 0x00, 0x00, 0x01}
		})

		It("succeeds", func() {
			Expect(err).To(BeNil())
		})

		It("decodes the data correctly", func() {
			Expect(result).To(Equal(int32(1)))
		})
	})

	Context("when the input data contains more than one encoded struct", func() {
		BeforeEach(func() {
			data = []byte{
				0x00, 0x00, 0x00, 0x01,
				0x00, 0x00, 0x00, 0x01,
			}
		})
		It("errors", func() {
			Expect(err).ToNot(BeNil())
		})
	})
})

var _ = Describe("xdr.SafeUnmarshalBase64", func() {
	var (
		result int32
		data   string
		err    error
	)

	JustBeforeEach(func() {
		err = SafeUnmarshalBase64(data, &result)
	})

	Context("input data is a single xdr value", func() {
		BeforeEach(func() {
			data = "AAAAAQ=="
		})

		It("succeeds", func() {
			Expect(err).To(BeNil())
		})

		It("decodes the data correctly", func() {
			Expect(result).To(Equal(int32(1)))
		})
	})

	Context("when the input data contains more than one encoded struct", func() {
		BeforeEach(func() {
			data = "AAAAAQAAAAI="
		})
		It("errors", func() {
			Expect(err).ToNot(BeNil())
		})
	})
})
