package txnbuild_test

import (
	"fmt"
	"time"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
)

var clientAccount, _ = keypair.ParseFull("SANVNCABRBVISCV7KH4SZVBKPJWWTT4424OVWUHUHPH2MVSF6RC7HPGN")
var clientSigner1, _ = keypair.ParseFull("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
var clientSigner2, _ = keypair.ParseFull("SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW")
var horizonClient = func() horizonclient.ClientInterface {
	client := &horizonclient.MockClient{}
	client.
		On("AccountDetail", horizonclient.AccountRequest{AccountID: clientAccount.Address()}).
		Return(
			horizon.Account{
				Thresholds: horizon.AccountThresholds{LowThreshold: 1, MedThreshold: 10, HighThreshold: 100},
				Signers: []horizon.Signer{
					{Key: clientSigner1.Address(), Weight: 40},
					{Key: clientSigner2.Address(), Weight: 60},
				},
			},
			nil,
		)
	return client
}()

func ExampleVerifyChallengeTxSigners_verifyAllClientSigners() {
	serverAccount, _ := keypair.ParseFull("SCDXPYDGKV5HOAGVZN3FQSS5FKUPP5BAVBWH4FXKTAWAC24AE4757JSI")
	clientAccount, _ := keypair.ParseFull("SANVNCABRBVISCV7KH4SZVBKPJWWTT4424OVWUHUHPH2MVSF6RC7HPGN")
	clientSigner1, _ := keypair.ParseFull("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	clientSigner2, _ := keypair.ParseFull("SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW")

	// Server builds challenge transaction
	var challengeTx string
	{
		tx, err := txnbuild.BuildChallengeTx(serverAccount.Seed(), clientAccount.Address(), "test", network.TestNetworkPassphrase, time.Minute)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		challengeTx = tx
	}

	// Client reads and signs challenge transaction
	var signedChallengeTx string
	{
		tx, txClientAccountID, err := txnbuild.ReadChallengeTx(challengeTx, serverAccount.Address(), network.TestNetworkPassphrase)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		if txClientAccountID != clientAccount.Address() {
			fmt.Println("Error: challenge tx is not for expected client account")
			return
		}
		err = tx.Sign(clientSigner1, clientSigner2)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		signedChallengeTx, err = tx.Base64()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	// Server verifies signed challenge transaction
	{
		_, txClientAccountID, err := txnbuild.ReadChallengeTx(challengeTx, serverAccount.Address(), network.TestNetworkPassphrase)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// Server gets list of account's signers
		horizonClientAccount, err := horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: txClientAccountID})
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		clientSigners := []string{}
		fmt.Println("Client Signers:")
		for _, horizonSigner := range horizonClientAccount.Signers {
			fmt.Println(horizonSigner.Key)
			clientSigners = append(clientSigners, horizonSigner.Key)
		}

		// Server finds which client signers are found on transaction
		signersFound, err := txnbuild.VerifyChallengeTxSigners(signedChallengeTx, serverAccount.Address(), network.TestNetworkPassphrase, clientSigners...)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// Server checks that all client signers were found on transaction
		if len(signersFound) != len(clientSigners) {
			fmt.Println("Error: not all client signers signed the challenge tx")
			return
		}

		fmt.Println("Client Signers Verified:")
		for _, signerFound := range signersFound {
			fmt.Println(signerFound)
		}
	}

	// Output:
	// Client Signers:
	// GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3
	// GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP
	// Client Signers Verified:
	// GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3
	// GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP
}

func ExampleVerifyChallengeTxSigners_verifyAnyClientSigners() {
	serverAccount, _ := keypair.ParseFull("SCDXPYDGKV5HOAGVZN3FQSS5FKUPP5BAVBWH4FXKTAWAC24AE4757JSI")
	clientAccount, _ := keypair.ParseFull("SANVNCABRBVISCV7KH4SZVBKPJWWTT4424OVWUHUHPH2MVSF6RC7HPGN")
	//clientSigner1, _ := keypair.ParseFull("SBPQUZ6G4FZNWFHKUWC5BEYWF6R52E3SEP7R3GWYSM2XTKGF5LNTWW4R")
	clientSigner2, _ := keypair.ParseFull("SBMSVD4KKELKGZXHBUQTIROWUAPQASDX7KEJITARP4VMZ6KLUHOGPTYW")

	// Server builds challenge transaction
	var challengeTx string
	{
		tx, err := txnbuild.BuildChallengeTx(serverAccount.Seed(), clientAccount.Address(), "test", network.TestNetworkPassphrase, time.Minute)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		challengeTx = tx
	}

	// Client reads and signs challenge transaction
	var signedChallengeTx string
	{
		tx, txClientAccountID, err := txnbuild.ReadChallengeTx(challengeTx, serverAccount.Address(), network.TestNetworkPassphrase)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		if txClientAccountID != clientAccount.Address() {
			fmt.Println("Error: challenge tx is not for expected client account")
			return
		}
		// Signs with only one of the account's signers
		err = tx.Sign(clientSigner2)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		signedChallengeTx, err = tx.Base64()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	// Server verifies signed challenge transaction
	{
		_, txClientAccountID, err := txnbuild.ReadChallengeTx(challengeTx, serverAccount.Address(), network.TestNetworkPassphrase)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// Server gets list of account's signers
		horizonClientAccount, err := horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: txClientAccountID})
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		clientSigners := []string{}
		fmt.Println("Client Signers:")
		for _, horizonSigner := range horizonClientAccount.Signers {
			fmt.Println(horizonSigner.Key)
			clientSigners = append(clientSigners, horizonSigner.Key)
		}

		// Server finds which client signers are found on transaction
		signersFound, err := txnbuild.VerifyChallengeTxSigners(signedChallengeTx, serverAccount.Address(), network.TestNetworkPassphrase, clientSigners...)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Println("Client Signers Verified:")
		for _, signerFound := range signersFound {
			fmt.Println(signerFound)
		}
	}

	// Output:
	// Client Signers:
	// GDQNY3PBOJOKYZSRMK2S7LHHGWZIUISD4QORETLMXEWXBI7KFZZMKTL3
	// GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP
	// Client Signers Verified:
	// GAS4V4O2B7DW5T7IQRPEEVCRXMDZESKISR7DVIGKZQYYV3OSQ5SH5LVP
}

func ExampleVerifyChallengeTxSigners_verifyClientMasterKeySigned() {
	serverAccount, _ := keypair.ParseFull("SCDXPYDGKV5HOAGVZN3FQSS5FKUPP5BAVBWH4FXKTAWAC24AE4757JSI")
	clientAccount, _ := keypair.ParseFull("SANVNCABRBVISCV7KH4SZVBKPJWWTT4424OVWUHUHPH2MVSF6RC7HPGN")

	// Server builds challenge transaction
	var challengeTx string
	{
		tx, err := txnbuild.BuildChallengeTx(serverAccount.Seed(), clientAccount.Address(), "test", network.TestNetworkPassphrase, time.Minute)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		challengeTx = tx
	}

	// Client reads and signs challenge transaction
	var signedChallengeTx string
	{
		tx, txClientAccountID, err := txnbuild.ReadChallengeTx(challengeTx, serverAccount.Address(), network.TestNetworkPassphrase)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		if txClientAccountID != clientAccount.Address() {
			fmt.Println("Error: challenge tx is not for expected client account")
			return
		}
		// Signs with account master key
		err = tx.Sign(clientAccount)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		signedChallengeTx, err = tx.Base64()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
	}

	// Server verifies signed challenge transaction
	{
		_, txClientAccountID, err := txnbuild.ReadChallengeTx(challengeTx, serverAccount.Address(), network.TestNetworkPassphrase)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		// Server finds which client signers are found on transaction
		signersFound, err := txnbuild.VerifyChallengeTxSigners(signedChallengeTx, serverAccount.Address(), network.TestNetworkPassphrase, txClientAccountID)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		fmt.Println("Client Master Key Verified:")
		for _, signerFound := range signersFound {
			fmt.Println(signerFound)
		}
	}

	// Output:
	// Client Master Key Verified:
	// GBOHBZB3Q3RMKXO3OSLJRDJNUSAPOXDVBDOMA52VHIGCIQEVZSXQ44CW
}
