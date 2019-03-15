package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/stellar/go/xdr"
)

type Operation struct {
	TransactionSuccessful bool `json:"transaction_successful"`
}

type OperationsPage struct {
	Embedded struct {
		Records []Operation
	} `json:"_embedded"`
}

type Transaction struct {
	Successful     bool   `json:"successful"`
	TxResult       string `json:"result_xdr"`
	OperationCount int    `json:"operation_count"`
	Hash           string `json:"hash"`
}

type TransactionsPage struct {
	Embedded struct {
		Records []Transaction
	} `json:"_embedded"`
}

type Ledger struct {
	Sequence                   int    `json:"sequence"`
	PagingToken                string `json:"paging_token"`
	SuccessfulTransactionCount int    `json:"successful_transaction_count"`
	FailedTransactionCount     int    `json:"failed_transaction_count"`
}

type LedgersPage struct {
	Embedded struct {
		Records []Ledger
	} `json:"_embedded"`
}

func main() {
	ledgerCursor := ""

	HorizonURL := os.Args[1]

	for {
		body := getBody(HorizonURL + fmt.Sprintf("/ledgers?order=desc&limit=200&cursor=%s", ledgerCursor))

		ledgersPage := LedgersPage{}
		err := json.Unmarshal(body, &ledgersPage)
		if err != nil {
			panic(err)
		}

		if len(ledgersPage.Embedded.Records) == 0 {
			fmt.Println("Done")
			return
		}

		for _, ledger := range ledgersPage.Embedded.Records {
			fmt.Printf("Checking ledger: %d (successful=%d failed=%d)\n", ledger.Sequence, ledger.SuccessfulTransactionCount, ledger.FailedTransactionCount)

			ledgerCursor = ledger.PagingToken

			body := getBody(HorizonURL + fmt.Sprintf("/ledgers/%d/transactions?limit=200&include_failed=true", ledger.Sequence))
			transactionsPage := TransactionsPage{}
			err = json.Unmarshal(body, &transactionsPage)
			if err != nil {
				panic(err)
			}

			var wg sync.WaitGroup

			successful := 0
			failed := 0

			for _, transaction := range transactionsPage.Embedded.Records {
				wg.Add(1)

				if transaction.Successful {
					successful++
				} else {
					failed++
				}

				go func(transaction Transaction) {
					defer wg.Done()

					var resultXDR xdr.TransactionResult
					err = xdr.SafeUnmarshalBase64(transaction.TxResult, &resultXDR)
					if err != nil {
						return
					}

					if transaction.Successful && resultXDR.Result.Code != xdr.TransactionResultCodeTxSuccess {
						panic(fmt.Sprintf("Corrupted data! %s %s", transaction.Hash, transaction.TxResult))
						return
					}

					if !transaction.Successful && resultXDR.Result.Code == xdr.TransactionResultCodeTxSuccess {
						panic(fmt.Sprintf("Corrupted data! %s %s", transaction.Hash, transaction.TxResult))
						return
					}

					body := getBody(HorizonURL + fmt.Sprintf("/transactions/%s/operations?limit=200", transaction.Hash))
					operationsPage := OperationsPage{}
					err = json.Unmarshal(body, &operationsPage)
					if err != nil {
						panic(err)
					}

					if len(operationsPage.Embedded.Records) != transaction.OperationCount {
						panic(fmt.Sprintf("Corrupted data! %s operations count %d vs %d (body=%s)", transaction.Hash, len(operationsPage.Embedded.Records), transaction.OperationCount, string(body)))
					}
				}(transaction)
			}

			wg.Wait()

			if successful != ledger.SuccessfulTransactionCount || failed != ledger.FailedTransactionCount {
				panic(fmt.Sprintf("Invalid ledger counters %d", ledger.Sequence))
			}
		}
	}
}

func getBody(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("%d response for %s", resp.StatusCode, url))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	return body
}
