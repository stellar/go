package txnbuild_test

import (
	"fmt"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
)

func ExampleVerifyChallengeTxSigners_verifyAllClientSigners() {
	serverAccount := keypair.MustRandom()
	clientAccount := keypair.MustRandom()
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
		// Server gets list of account's signers
		clientSigners := []string{clientSigner1.Address(), clientSigner2.Address()}
		fmt.Println("Client Signers:")
		for _, clientSigner := range clientSigners {
			fmt.Println(clientSigner)
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
	serverAccount := keypair.MustRandom()
	clientAccount := keypair.MustRandom()
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
		// Server gets list of account's signers
		clientSigners := []string{clientSigner1.Address(), clientSigner2.Address()}
		fmt.Println("Client Signers:")
		for _, clientSigner := range clientSigners {
			fmt.Println(clientSigner)
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
	serverAccount := keypair.MustRandom()
	clientAccount, _ := keypair.ParseFull("SDZQ46SVA4VEE4WGBKZMN76JDMFRNEICSLWPIGNINLEOFOSB7AVMV2PL")

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
		// Server gets list of account's signers
		clientSigners := []string{clientAccount.Address()}
		fmt.Println("Client Signers:")
		for _, clientSigner := range clientSigners {
			fmt.Println(clientSigner)
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
	// GDTEJZDQADCQZ4U6AFPWCOO55DYZCQ332KLYHHGEJIOFFL5HZN5MEGVY
	// Client Signers Verified:
	// GDTEJZDQADCQZ4U6AFPWCOO55DYZCQ332KLYHHGEJIOFFL5HZN5MEGVY
}
