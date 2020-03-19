package build

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBuild(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Package: github.com/stellar/go/build")
}

// ExampleTransactionBuilder creates and signs a simple transaction, and then
// encodes it into a base64 string capable of being submitted to stellar-core.
//
// It uses the transaction builder system
func ExampleTransactionBuilder() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		Payment(
			Destination{"GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA"},
			NativeAmount{"50"},
		),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAABAAAAAC0kaS7Qi79nm6GZRIhw0hkeh2/s2S/dn20nTaTm3hNBAAAAAAAAAAAdzWUAAAAAAAAAAAEbQzHwAAAAQNqCHkMaCZed0TEVhXiwfOuc3JMt/X9oH9j5lLmwriWHHBHj0Uq0wvEEKzDoBsA/GuqHvHYJbb3s3Q7JplwT8gc=
}

// ExampleCreateAccount creates a transaction to fund a new stallar account with a balance. It then
// encodes the transaction into a base64 string capable of being submitted to stellar-core. It uses
// the transaction builder system.
func ExampleCreateAccount() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		CreateAccount(
			Destination{"GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA"},
			NativeAmount{"50"},
		),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAC0kaS7Qi79nm6GZRIhw0hkeh2/s2S/dn20nTaTm3hNBAAAAAB3NZQAAAAAAAAAAARtDMfAAAABA4h0BLqyyE1/O85pBIvEa3H+cBqdSay50g6EyYjfsjYyWVRd1nASYKBsiCdzSaPrTZk6gC1ayc4oomRnaLpMXDQ==
}

// ExampleBumpSequence creates a transaction to bump sequence of a given account. It then
// encodes the transaction into a base64 string capable of being submitted to stellar-core. It uses
// the transaction builder system.
func ExampleBumpSequence() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		BumpSequence(
			BumpTo(5),
		),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAALAAAAAAAAAAUAAAAAAAAAARtDMfAAAABAGZDIRf+LYp/3ogMPso9d4kPcQKYNMGnUsiDHZNg+90b8L4McKnwBheqpi9XfKqq4wpWe1YtDGAQz8TkHIcDEAA==
}

// ExamplePayment creates and signs a native-asset Payment, encodes it into a base64 string capable of
// being submitted to stellar-core. It uses the transaction builder system.
func ExamplePayment() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		Payment(
			Destination{"GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA"},
			NativeAmount{"50"},
		),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAABAAAAAC0kaS7Qi79nm6GZRIhw0hkeh2/s2S/dn20nTaTm3hNBAAAAAAAAAAAdzWUAAAAAAAAAAAEbQzHwAAAAQNqCHkMaCZed0TEVhXiwfOuc3JMt/X9oH9j5lLmwriWHHBHj0Uq0wvEEKzDoBsA/GuqHvHYJbb3s3Q7JplwT8gc=
}

// ExamplePathPayment creates and signs a simple transaction with PathPayment operation, and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExamplePathPayment() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		Payment(
			Destination{"GBDT3K42LOPSHNAEHEJ6AVPADIJ4MAR64QEKKW2LQPBSKLYD22KUEH4P"},
			CreditAmount{"USD", "GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA", "50"},
			PayWith(CreditAsset("EUR", "GCPZJ3MJQ3GUGJSBL6R3MLYZS6FKVHG67BPAINMXL3NWNXR5S6XG657P"), "100").
				Through(Asset{Native: true}).
				Through(CreditAsset("BTC", "GAHJZHVKFLATAATJH46C7OK2ZOVRD47GZBGQ7P6OCVF6RJDCEG5JMQBQ")),
		),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAACAAAAAUVVUgAAAAAAn5TtiYbNQyZBX6O2LxmXiqqc3vheBDWXXttm3j2Xrm8AAAAAO5rKAAAAAABHPauaW58jtAQ5E+BV4BoTxgI+5AilW0uDwyUvA9aVQgAAAAFVU0QAAAAAAC0kaS7Qi79nm6GZRIhw0hkeh2/s2S/dn20nTaTm3hNBAAAAAB3NZQAAAAACAAAAAAAAAAFCVEMAAAAAAA6cnqoqwTACaT88L7lay6sR8+bITQ+/zhVL6KRiIbqWAAAAAAAAAAEbQzHwAAAAQOcbiCbvyhikEbg63ZMzbEeB40+sF+OuEh2f8r2MFW/wJAMsn3jCwzYYCmwcUDFnN1KOX07LdVB5XQvo2DXgIQk=
}

// ExampleSetOptions creates and signs a simple transaction with SetOptions operation, and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExampleSetOptions() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		SetOptions(
			InflationDest("GCT7S5BA6ZC7SV7GGEMEYJTWOBYTBOA7SC4JEYP7IAEDG7HQNIWKRJ4G"),
			SetAuthRequired(),
			SetAuthRevocable(),
			SetAuthImmutable(),
			ClearAuthRequired(),
			ClearAuthRevocable(),
			ClearAuthImmutable(),
			MasterWeight(1),
			SetThresholds(2, 3, 4),
			HomeDomain("stellar.org"),
			AddSigner("GC6DDGPXVWXD5V6XOWJ7VUTDYI7VKPV2RAJWBVBHR47OPV5NASUNHTJW", 5),
		),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAAFAAAAAQAAAACn+XQg9kX5V+YxGEwmdnBxMLgfkLiSYf9ACDN88GosqAAAAAEAAAAHAAAAAQAAAAcAAAABAAAAAQAAAAEAAAACAAAAAQAAAAMAAAABAAAABAAAAAEAAAALc3RlbGxhci5vcmcAAAAAAQAAAAC8MZn3ra4+19d1k/rSY8I/VT66iBNg1CePPufXrQSo0wAAAAUAAAAAAAAAARtDMfAAAABAmRde6Tn2XMnO2CVzMsfvjzMnSLy+oNsmXDw2PIePrZpc/t67BOrrl9EesX4DZ7jJ7byuGDJmjr6eA72mTDDaAA==
}

// ExampleSetOptions_manyOperations creates and signs a simple transaction with many SetOptions operations, and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExampleSetOptions_manyOperations() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		InflationDest("GCT7S5BA6ZC7SV7GGEMEYJTWOBYTBOA7SC4JEYP7IAEDG7HQNIWKRJ4G"),
		SetAuthRequired(),
		SetAuthRevocable(),
		SetAuthImmutable(),
		ClearAuthRequired(),
		ClearAuthRevocable(),
		ClearAuthImmutable(),
		MasterWeight(1),
		SetThresholds(2, 3, 4),
		HomeDomain("stellar.org"),
		RemoveSigner("GC6DDGPXVWXD5V6XOWJ7VUTDYI7VKPV2RAJWBVBHR47OPV5NASUNHTJW"),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAABEwAAAAAAAAAAQAAAAAAAAAAAAAACwAAAAAAAAAFAAAAAQAAAACn+XQg9kX5V+YxGEwmdnBxMLgfkLiSYf9ACDN88GosqAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAUAAAAAAAAAAAAAAAEAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAUAAAAAAAAAAAAAAAEAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAUAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAUAAAAAAAAAAQAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAUAAAAAAAAAAQAAAAIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAUAAAAAAAAAAQAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAUAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAABAAAAAwAAAAEAAAAEAAAAAAAAAAAAAAAAAAAABQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAC3N0ZWxsYXIub3JnAAAAAAAAAAAAAAAABQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAC8MZn3ra4+19d1k/rSY8I/VT66iBNg1CePPufXrQSo0wAAAAAAAAAAAAAAARtDMfAAAABADl7C2xaN3vH6asnmRBxj/aPyK0A0EcqH/gwNRHlJL1BmzYcue8V3KTIWaUbcF04VgWMeM2uKvk0a56GULXs4DQ==
}

// ExampleChangeTrust creates and signs a simple transaction with ChangeTrust operation, and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExampleChangeTrust() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		Trust("USD", "GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA", Limit("100.25")),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAAGAAAAAVVTRAAAAAAALSRpLtCLv2eboZlEiHDSGR6Hb+zZL92fbSdNpObeE0EAAAAAO8DvoAAAAAAAAAABG0Mx8AAAAEDiMtfV9/GN43Bc74Q7Jl7uog8682+IsH0tjQKvRhgBYkSVBqBV/ObyrZTXF80KaReF/KCDDTxG++H7mRPzfhgB
}

// ExampleChangeTrust_maxLimit creates and signs a simple transaction with ChangeTrust operation (maximum limit), and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExampleChangeTrust_maxLimit() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		Trust("USD", "GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA"),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAAGAAAAAVVTRAAAAAAALSRpLtCLv2eboZlEiHDSGR6Hb+zZL92fbSdNpObeE0F//////////wAAAAAAAAABG0Mx8AAAAECUAukd0ajWsOazpmsaqQBNJQ+aJzP545/SNkVZYbUtkhotme8ZuuVEmFVl0yRVakwn08MLvkFxXDrqoCY1bRwJ
}

// ExampleRemoveTrust creates and signs a simple transaction with ChangeTrust operation (remove trust), and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExampleRemoveTrust() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	operationSource := "GCVJCNUHSGKOTBBSXZJ7JJZNOSE2YDNGRLIDPMQDUEQWJQSE6QZSDPNU"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		RemoveTrust(
			"USD",
			"GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA",
			SourceAccount{operationSource},
		),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAEAAAAAqpE2h5GU6YQyvlP0py10iawNporQN7IDoSFkwkT0MyEAAAAGAAAAAVVTRAAAAAAALSRpLtCLv2eboZlEiHDSGR6Hb+zZL92fbSdNpObeE0EAAAAAAAAAAAAAAAAAAAABG0Mx8AAAAEA+RXhgRMCcnmrivlin3MWAXisOthLQqrwtKfWftXGN9nykPuSc/bhNS2g7B1uTbq7UrSnFE4huY9qjQ39UKWIM
}

// ExampleManageOffer creates and signs a simple transaction with ManageOffer operations, and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExampleManageOffer() {
	rate := Rate{
		Selling: NativeAsset(),
		Buying:  CreditAsset("USD", "GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA"),
		Price:   Price("125.12"),
	}

	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		CreateOffer(rate, "20"),
		UpdateOffer(rate, "40", OfferID(2)),
		DeleteOffer(rate, OfferID(1)),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAASwAAAAAAAAAAQAAAAAAAAAAAAAAAwAAAAAAAAADAAAAAAAAAAFVU0QAAAAAAC0kaS7Qi79nm6GZRIhw0hkeh2/s2S/dn20nTaTm3hNBAAAAAAvrwgAAAAw4AAAAGQAAAAAAAAAAAAAAAAAAAAMAAAAAAAAAAVVTRAAAAAAALSRpLtCLv2eboZlEiHDSGR6Hb+zZL92fbSdNpObeE0EAAAAAF9eEAAAADDgAAAAZAAAAAAAAAAIAAAAAAAAAAwAAAAAAAAABVVNEAAAAAAAtJGku0Iu/Z5uhmUSIcNIZHodv7Nkv3Z9tJ02k5t4TQQAAAAAAAAAAAAAMOAAAABkAAAAAAAAAAQAAAAAAAAABG0Mx8AAAAEAX6LJPrfKqVCxz+KaTV9sVT4ofJZp2xxWXXrsUj+qYKQgl9OUu82eD7I1UGNpzpy67upgU3Na/EqPr8R7NxVwH
}

// ExampleCreatePassiveOffer creates and signs a simple transaction with CreatePassiveOffer operation, and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExampleCreatePassiveOffer() {
	rate := Rate{
		Selling: NativeAsset(),
		Buying:  CreditAsset("USD", "GAWSI2JO2CF36Z43UGMUJCDQ2IMR5B3P5TMS7XM7NUTU3JHG3YJUDQXA"),
		Price:   Price("125.12"),
	}

	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		CreatePassiveOffer(rate, "20"),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAAEAAAAAAAAAAFVU0QAAAAAAC0kaS7Qi79nm6GZRIhw0hkeh2/s2S/dn20nTaTm3hNBAAAAAAvrwgAAAAw4AAAAGQAAAAAAAAABG0Mx8AAAAEB7/9cS5/gK3yFKFo591dM1ZGurbpajGMeMyWYRqnZk8PIgl2NGbR0vsvu2BoEd7zNQ6Sjnb1k5oXnvrAlyKaUA
}

// ExampleAccountMerge creates and signs a simple transaction with AccountMerge operation, and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExampleAccountMerge() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		AccountMerge(
			Destination{"GBDT3K42LOPSHNAEHEJ6AVPADIJ4MAR64QEKKW2LQPBSKLYD22KUEH4P"},
		),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAAIAAAAAEc9q5pbnyO0BDkT4FXgGhPGAj7kCKVbS4PDJS8D1pVCAAAAAAAAAAEbQzHwAAAAQId6maz+U/V4NC3c8Di8f3gPwe284/vI/rPXlzbru/y3XIeyOpFuYmM+lX8t9hVZyjbaMpE6Ae50GNqNr6NbagM=
}

// ExampleInflation creates and signs a simple transaction with Inflation operation, and then
// encodes it into a base64 string capable of being submitted to stellar-core.
func ExampleInflation() {
	seed := "SDOTALIMPAM2IV65IOZA7KZL7XWZI5BODFXTRVLIHLQZQCKK57PH5F3H"
	tx, err := Transaction(
		SourceAccount{seed},
		Sequence{1},
		TestNetwork,
		Inflation(),
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	txe, err := tx.Sign(seed)
	if err != nil {
		fmt.Println(err)
		return
	}

	txeB64, err := txe.Base64()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("tx base64: %s", txeB64)
	// Output: tx base64: AAAAAgAAAAA2WP51mNIMem3jKX+EvFK9Kh7IwPH2xbQcwcdXG0Mx8AAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAQAAAAAAAAAJAAAAAAAAAAEbQzHwAAAAQM8wxuFeys8p1mNBEf1eBx9FoA75d4ayIsw3Clt6hEGOCG98l7n1b3W87BE/w4+Y7OXYA5HGIIpB8EQqaovyfAU=
}
