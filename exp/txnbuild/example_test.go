package txnbuild

import (
	"fmt"

	horizonclient "github.com/stellar/go/exp/txnbuild/examplehorizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
)

func ExampleInflation() {
	kp, _ := keypair.Parse("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	client := horizonclient.DefaultTestNetClient
	ar := horizonclient.AccountRequest{AccountID: kp.Address()}
	sourceAccount, err := client.AccountDetail(ar)
	check(err)

	inflation := Inflation{}

	tx := Transaction{
		SourceAccount: &sourceAccount,
		Operations:    []Operation{&inflation},
		Timebounds:    NewInfiniteTimeout(), // Use a real timeout in production!
		Network:       network.TestNetworkPassphrase,
	}

	received, err := tx.BuildSignEncode(kp.(*keypair.Full))
	check(err)
	fmt.Println(received)

	// Output: AAAAAODcbeFyXKxmUWK1L6znNbKKIkPkHRJNbLktcKPqLnLFAAAAZAAMoj8AAAAEAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAJAAAAAAAAAAHqLnLFAAAAQP3NHWXvzKIHB3+jjhHITdc/tBPntWYj3SoTjpON+dxjKqU5ohFamSHeqi5ONXkhE9Uajr5sVZXjQfUcTTzsWAA=
}
