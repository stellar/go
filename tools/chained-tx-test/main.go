package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"golang.org/x/sync/errgroup"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	horizonURL := "https://horizon-testnet.stellar.org"
	flag.StringVar(&horizonURL, "horizon", horizonURL, "horizon's http endpoint")

	coreURL := ""
	flag.StringVar(&coreURL, "core", coreURL, "stellar-core's http :11626 endpoint if set submission occurs via stellar-core directly")

	count := 10
	flag.IntVar(&count, "count", count, "the number of transactions to submit")

	flag.Parse()

	log.Printf("horizon url: %s", horizonURL)
	client := &horizonclient.Client{HorizonURL: horizonURL}
	networkDetails, err := client.Root()
	if err != nil {
		panic(err)
	}
	log.Printf("network passphrase: %s", networkDetails.NetworkPassphrase)

	kp := keypair.MustRandom()
	log.Printf("account: %v", kp.Address())

	log.Printf("funding...")
	fundingTx, err := client.Fund(kp.Address())
	if horizonclient.IsNotFoundError(err) {
		root := keypair.Root(networkDetails.NetworkPassphrase)
		rootAccountDetail, err := client.AccountDetail(horizonclient.AccountRequest{AccountID: root.Address()})
		if err != nil {
			panic(err)
		}
		rootSequence, _ := rootAccountDetail.GetSequenceNumber()
		tx, _, err := buildTx(networkDetails.NetworkPassphrase, root, rootSequence+1, kp.FromAddress(), "10000")
		if err != nil {
			panic(err)
		}
		fundingTx, err = client.SubmitTransactionXDR(tx)
		if err != nil {
			panic(err)
		}
		time.Sleep(5 * time.Second)
	} else if err != nil {
		panic(err)
	}

	seqNum := int64(fundingTx.Ledger) << 32
	log.Printf("funded and has seq num: %d", seqNum)

	log.Printf("creating %d accounts...", count)
	group := errgroup.Group{}
	for i := 0; i < count; i++ {
		i := i
		group.Go(func() error {
			acc := keypair.MustRandom()
			txSeqNum := seqNum + 1 + int64(i)
			tx, txHash, err := buildTx(networkDetails.NetworkPassphrase, kp, txSeqNum, acc.FromAddress(), "20")
			if err != nil {
				return fmt.Errorf("building tx %d: %w", i, err)
			}
			log.Printf("tx %d seq=%d h=%s pending", i, txSeqNum, txHash)
			startTime := time.Now()
			ledger := int32(0)
			if coreURL == "" {
				ledger, err = submitToHorizon(client, tx)
			} else {
				ledger, err = submitToCore(coreURL, client, tx, txHash)
			}
			if err != nil {
				return fmt.Errorf("submitting tx %d: %w", i, err)
			}
			duration := time.Since(startTime)
			log.Printf("tx %d seq=%d h=%s ✅ ledger=%d dur=%v", i, txSeqNum, txHash, ledger, duration)
			return nil
		})
		time.Sleep(100 * time.Millisecond)
	}
	err = group.Wait()
	if err != nil {
		panic(err)
	}
	log.Printf("done")
}

func submitToHorizon(client *horizonclient.Client, tx string) (ledger int32, err error) {
	txResp, err := client.SubmitTransactionXDR(tx)
	if err != nil {
		return 0, err
	}
	return txResp.Ledger, nil
}

func submitToCore(coreURL string, client *horizonclient.Client, tx, txHash string) (ledger int32, err error) {
	q := url.Values{}
	q.Set("blob", tx)
	u := coreURL + "/tx?" + q.Encode()
	resp, err := http.Get(u)
	if err != nil {
		return 0, err
	}
	r := struct {
		Status string
	}{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return 0, err
	}
	if r.Status != "PENDING" {
		return 0, fmt.Errorf("tx status not pending: %v", r.Status)
	}
	for i := 0; i < 20; i++ {
		tx, err := client.TransactionDetail(txHash)
		if horizonclient.IsNotFoundError(err) {
			time.Sleep(500 * time.Millisecond)
		} else if err != nil {
			return 0, err
		} else {
			return tx.Ledger, nil
		}
	}
	return 0, fmt.Errorf("timed out waiting for response")
}

func buildTx(networkPassphrase string, creator *keypair.Full, seqNum int64, account *keypair.FromAddress, amount string) (txXdr, txHash string, err error) {
	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount: &txnbuild.SimpleAccount{AccountID: creator.Address(), Sequence: seqNum},
		BaseFee:       txnbuild.MinBaseFee,
		Timebounds:    txnbuild.NewInfiniteTimeout(),
		Operations: []txnbuild.Operation{
			&txnbuild.CreateAccount{Destination: account.Address(), Amount: amount},
		},
	})
	if err != nil {
		return "", "", fmt.Errorf("building tx: %w", err)
	}
	txHash, err = tx.HashHex(networkPassphrase)
	if err != nil {
		return "", "", fmt.Errorf("hashing tx: %w", err)
	}
	tx, err = tx.Sign(networkPassphrase, creator)
	if err != nil {
		return "", txHash, fmt.Errorf("signing tx: %w", err)
	}
	xdr, err := tx.Base64()
	if err != nil {
		return "", txHash, fmt.Errorf("base64ing tx: %w", err)
	}
	return xdr, txHash, nil
}
