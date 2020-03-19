package txnbuild

import (
	"fmt"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	horizonclient "github.com/stellar/go/txnbuild/examplehorizonclient"
)

func ExampleInflation() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := Inflation{}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACQAAAAAAAAAB6i5yxQAAAED9zR1l78yiBwd/o44RyE3XP7QT57VmI90qE46TjfncYyqlOaIRWpkh3qouTjV5IRPVGo6+bFWV40H1HE087FgA
}

func ExampleCreateAccount() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := CreateAccount{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAACE4N7avBtJL576CIWTzGCbGPvSlVfMQAOjcYbSsSF2VAAAAAAF9eEAAAAAAAAAAAHqLnLFAAAAQKsrlxt6Ri/WuDGcK1+Tk1hdYHdPeK7KMIds10mcwzw6BpQFZYxP8o6O6ejJFGO06TAGt2PolwuWnpeiVQ9Kcg0=
}

func ExamplePayment() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := Payment{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
		Asset:       NativeAsset{},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAACE4N7avBtJL576CIWTzGCbGPvSlVfMQAOjcYbSsSF2VAAAAAAAAAAABfXhAAAAAAAAAAAB6i5yxQAAAEB2/C066OEFac3Bszk6FtvKd+NKOeCl+f8caHQATPos8HkJW1Sm/WyEkVDrvrDX4udMHl3gHhlS/qE0EuWEeJYC
}

func ExamplePayment_setBaseFee() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op1 := Payment{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "10",
		Asset:       NativeAsset{},
	}

	op2 := Payment{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Amount:      "100",
		Asset:       NativeAsset{},
	}

	// get fees from network
	feeStats, err := client.FeeStats()
	check(err)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op1, &op2},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
		BaseFee:       uint32(feeStats.MaxFee.P50),
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAABLAADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAIAAAAAAAAAAQAAAACE4N7avBtJL576CIWTzGCbGPvSlVfMQAOjcYbSsSF2VAAAAAAAAAAABfXhAAAAAAAAAAABAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAAAAAA7msoAAAAAAAAAAAHqLnLFAAAAQMmOXP+k93ENYtu7evNTu2h63UkNrQnF6ci49Oh1XufQ3rhzS4Dd1+6AXqgWa4FbcvlTVRjxCurkflI4Rov2xgQ=
}

func ExampleBumpSequence() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := BumpSequence{
		BumpTo: 9606132444168300,
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACwAiILoAAABsAAAAAAAAAAHqLnLFAAAAQEIvyOHdPn82ckKXISGF6sR4YU5ox735ivKrC/wS4615j1AA42vbXSLqShJA5/7/DX56UUv+Lt7vlcu9M7jsRw4=
}

func ExampleAccountMerge() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := AccountMerge{
		Destination: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACAAAAACE4N7avBtJL576CIWTzGCbGPvSlVfMQAOjcYbSsSF2VAAAAAAAAAAB6i5yxQAAAEAvOx3WHzmaTsf4rK+yRDsvXn9xh+dU6CkpAum+FCXQ5LZQqhxQg9HErbSfxeTFMdknEpMKXgJRFUfAetl+jf4O
}

func ExampleManageData() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := ManageData{
		Name:  "Fruit preference",
		Value: []byte("Apple"),
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACgAAABBGcnVpdCBwcmVmZXJlbmNlAAAAAQAAAAVBcHBsZQAAAAAAAAAAAAAB6i5yxQAAAEDtRCyQRKKgQ8iLEu7kicHtSzoplfxPtPTMhdRv/sq8UoIBVTxIw+S13Jv+jzs3tyLDLiGCVNXreUNlbfX+980K
}

func ExampleManageData_removeDataEntry() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := ManageData{
		Name: "Fruit preference",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAACgAAABBGcnVpdCBwcmVmZXJlbmNlAAAAAAAAAAAAAAAB6i5yxQAAAEDFpI1vphzG8Dny4aVDA7tyOlP579d9kWO0U/vmq6pWTrNocd6+xTiU753W50ksEscA6f1WNwUsQf+DCwmZfqIA
}

func ExampleSetOptions() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := SetOptions{
		InflationDestination: NewInflationDestination("GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z"),
		ClearFlags:           []AccountFlag{AuthRevocable},
		SetFlags:             []AccountFlag{AuthRequired, AuthImmutable},
		MasterWeight:         NewThreshold(10),
		LowThreshold:         NewThreshold(1),
		MediumThreshold:      NewThreshold(2),
		HighThreshold:        NewThreshold(2),
		HomeDomain:           NewHomeDomain("LovelyLumensLookLuminous.com"),
		Signer:               &Signer{Address: "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z", Weight: Threshold(4)},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAABQAAAAEAAAAAhODe2rwbSS+e+giFk8xgmxj70pVXzEADo3GG0rEhdlQAAAABAAAAAgAAAAEAAAAFAAAAAQAAAAoAAAABAAAAAQAAAAEAAAACAAAAAQAAAAIAAAABAAAAHExvdmVseUx1bWVuc0xvb2tMdW1pbm91cy5jb20AAAABAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAABAAAAAAAAAAB6i5yxQAAAEBxncRuLogeNQ8sG9TojUMB6QmKDWYmhF00Wz43UX90pAQnSNcJAQxur0RA7Fn6LjJLObqyjcdIc4P2DC02u08G
}

func ExampleChangeTrust() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := ChangeTrust{
		Line:  CreditAsset{"ABCD", "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z"},
		Limit: "10",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAABgAAAAFBQkNEAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAX14QAAAAAAAAAAAeoucsUAAABAqqUuIlFMrlElYnGSLHlaI/A41oGA3rdtc1EHhza9bXk35ZwlEvmsBUOZTasZfgBzwd+CczekWKBCEqBCHzaSBw==
}

func ExampleChangeTrust_removeTrustline() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := RemoveTrustlineOp(CreditAsset{"ABCD", "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z"})

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAABgAAAAFBQkNEAAAAAITg3tq8G0kvnvoIhZPMYJsY+9KVV8xAA6NxhtKxIXZUAAAAAAAAAAAAAAAAAAAAAeoucsUAAABAKLmUWcLjxeY+vG8jEXMNprU6EupxbMRiXGYzuKBptnVlbFUtTBqhYa/ibyCZTEVCinT8bWQKDvZI0m6VLKVHAg==
}

func ExampleAllowTrust() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := AllowTrust{
		Trustor:   "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z",
		Type:      CreditAsset{"ABCD", "GCCOBXW2XQNUSL467IEILE6MMCNRR66SSVL4YQADUNYYNUVREF3FIV2Z"},
		Authorize: true,
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAABwAAAACE4N7avBtJL576CIWTzGCbGPvSlVfMQAOjcYbSsSF2VAAAAAFBQkNEAAAAAQAAAAAAAAAB6i5yxQAAAEAY3MnWiMcL18SxRITSuI5tZSXmEo0Q38UZg0jiJGU2U6kSnsCNTTJiGACGQlIrPfAMYt9koarrX11w7HLBosQN
}

func ExampleManageSellOffer() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	selling := NativeAsset{}
	buying := CreditAsset{"ABCD", "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP"}
	sellAmount := "100"
	price := "0.01"
	op, err := CreateOfferOp(selling, buying, sellAmount, price)
	check(err)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAwAAAAAAAAABQUJDRAAAAAAlyvHaD8duz+iEXkJUUbsHkklIlH46oMrMMYrt0odkfgAAAAA7msoAAAAAAQAAAGQAAAAAAAAAAAAAAAAAAAAB6i5yxQAAAEBtfrN+VUE7iCwBk0+rmg0/Ua4DItMWEy6naGWxoDBi4ksCIJSZPzkv79Q65rIaFyIcC/zuyJcnIcv73AP+HQEK
}

func ExampleManageSellOffer_deleteOffer() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	offerID := int64(2921622)
	op, err := DeleteOfferOp(offerID)
	check(err)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAwAAAAAAAAABRkFLRQAAAABBB4BkxJWGYvNgJBoiXUo2tjgWlNmhHMMKdwGN7RSdsQAAAAAAAAAAAAAAAQAAAAEAAAAAACyUlgAAAAAAAAAB6i5yxQAAAEBnE+oILauqt6m8fj7DIBNW/XBmKJ34SLvHdxP04vb26aI8q9i/2p9/pJMnWPeOoIw0f6jreR306qPJFhjMtl4G
}

func ExampleManageSellOffer_updateOffer() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	selling := NativeAsset{}
	buying := CreditAsset{"ABCD", "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP"}
	sellAmount := "50"
	price := "0.02"
	offerID := int64(2497628)
	op, err := UpdateOfferOp(selling, buying, sellAmount, price, offerID)
	check(err)

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAwAAAAAAAAABQUJDRAAAAAAlyvHaD8duz+iEXkJUUbsHkklIlH46oMrMMYrt0odkfgAAAAAdzWUAAAAAAQAAADIAAAAAACYcXAAAAAAAAAAB6i5yxQAAAECmO+4yukAuLRtR4IRWPVtoyZ2LJeaipPuec+/M1JGDoTFPULDl3kgugPwV3mr0jvMNArBdR8S3NUw31gtT5TcO
}

func ExampleCreatePassiveSellOffer() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	op := CreatePassiveSellOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{"ABCD", "GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP"},
		Amount:  "10",
		Price:   "1.0",
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAABAAAAAAAAAABQUJDRAAAAAAlyvHaD8duz+iEXkJUUbsHkklIlH46oMrMMYrt0odkfgAAAAAF9eEAAAAAAQAAAAEAAAAAAAAAAeoucsUAAABAE4XbLdDVz1MwC9Bs84nkqK8hyHheVbYznNSiAP0hiP8auvcKAMnYz3HJvzM8H0q/K5MPvgBaehHZ/tQtaPSGBg==
}

func ExamplePathPayment() {
	kp, _ := keypair.Parse("SBZVMB74Z76QZ3ZOY7UTDFYKMEGKW5XFJEB6PFKBF4UYSSWHG4EDH7PY")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	abcdAsset := CreditAsset{"ABCD", "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"}
	op := PathPayment{
		SendAsset:   NativeAsset{},
		SendMax:     "10",
		Destination: kp.Address(),
		DestAsset:   NativeAsset{},
		DestAmount:  "1",
		Path:        []Asset{abcdAsset},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAAB+Ecs01jX14asC1KAsPdWlpGbYCM2PEgFZCD3NLhVZmAAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAgAAAAAAAAAABfXhAAAAAAB+Ecs01jX14asC1KAsPdWlpGbYCM2PEgFZCD3NLhVZmAAAAAAAAAAAAJiWgAAAAAEAAAABQUJDRAAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAAAAAAABLhVZmAAAAEDhhPsNm7yKfCUCDyBV1pOZDu+3DVDpT2cJSLQOVevP6pmU2yVqvMKnWbYxC5GbTXEEF+MfBE6EoW5+Z4rRt0QO
}

func ExamplePathPaymentStrictReceive() {
	kp, _ := keypair.Parse("SBZVMB74Z76QZ3ZOY7UTDFYKMEGKW5XFJEB6PFKBF4UYSSWHG4EDH7PY")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	abcdAsset := CreditAsset{"ABCD", "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"}
	op := PathPaymentStrictReceive{
		SendAsset:   NativeAsset{},
		SendMax:     "10",
		Destination: kp.Address(),
		DestAsset:   NativeAsset{},
		DestAmount:  "1",
		Path:        []Asset{abcdAsset},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAAB+Ecs01jX14asC1KAsPdWlpGbYCM2PEgFZCD3NLhVZmAAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAgAAAAAAAAAABfXhAAAAAAB+Ecs01jX14asC1KAsPdWlpGbYCM2PEgFZCD3NLhVZmAAAAAAAAAAAAJiWgAAAAAEAAAABQUJDRAAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAAAAAAABLhVZmAAAAEDhhPsNm7yKfCUCDyBV1pOZDu+3DVDpT2cJSLQOVevP6pmU2yVqvMKnWbYxC5GbTXEEF+MfBE6EoW5+Z4rRt0QO
}

func ExamplePathPaymentStrictSend() {
	kp, _ := keypair.Parse("SBZVMB74Z76QZ3ZOY7UTDFYKMEGKW5XFJEB6PFKBF4UYSSWHG4EDH7PY")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	abcdAsset := CreditAsset{"ABCD", "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"}
	op := PathPaymentStrictSend{
		SendAsset:   NativeAsset{},
		SendAmount:  "1",
		Destination: kp.Address(),
		DestAsset:   NativeAsset{},
		DestMin:     "10",
		Path:        []Asset{abcdAsset},
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&op},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAAB+Ecs01jX14asC1KAsPdWlpGbYCM2PEgFZCD3NLhVZmAAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAADQAAAAAAAAAAAJiWgAAAAAB+Ecs01jX14asC1KAsPdWlpGbYCM2PEgFZCD3NLhVZmAAAAAAAAAAABfXhAAAAAAEAAAABQUJDRAAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAAAAAAABLhVZmAAAAEDV6CmR4ATvtm2qBzHE9UqqS95ZnIIHgpuU7hTZO38DHhf+oeZQ02DGvst4vYMMAIPGkMAsLlfAN/AFinz74DAD
}

func ExampleManageBuyOffer() {
	kp, _ := keypair.Parse("SBZVMB74Z76QZ3ZOY7UTDFYKMEGKW5XFJEB6PFKBF4UYSSWHG4EDH7PY")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	buyOffer := ManageBuyOffer{
		Selling: NativeAsset{},
		Buying:  CreditAsset{"ABCD", "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"},
		Amount:  "100",
		Price:   "0.01",
		OfferID: 0,
	}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&buyOffer},
		Timebounds:    NewInfiniteTimeout(),
		Network:       network.TestNetworkPassphrase,
	}

	txe, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(txe)

	// Output: AAAAAgAAAAB+Ecs01jX14asC1KAsPdWlpGbYCM2PEgFZCD3NLhVZmAAAAGQADKI/AAAABAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAADAAAAAAAAAABQUJDRAAAAADg3G3hclysZlFitS+s5zWyiiJD5B0STWy5LXCj6i5yxQAAAAA7msoAAAAAAQAAAGQAAAAAAAAAAAAAAAAAAAABLhVZmAAAAED4fIdU68w6XIMwf1RPFdF9qRRlfPycrmK8dCOW0XwSbiya9JfMi9YrD9cGY7zHV+3zYpLcEi7lLo++PZ1gOsAK

}

func ExampleBuildChallengeTx() {
	// Generate random nonce
	serverSignerSeed := "SBZVMB74Z76QZ3ZOY7UTDFYKMEGKW5XFJEB6PFKBF4UYSSWHG4EDH7PY"
	clientAccountID := "GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3"
	anchorName := "SDF"
	timebound := time.Duration(5 * time.Minute)

	tx, err := BuildChallengeTx(serverSignerSeed, clientAccountID, anchorName, network.TestNetworkPassphrase, timebound)
	_, err = checkChallengeTx(tx, anchorName)

	check(err)
}
