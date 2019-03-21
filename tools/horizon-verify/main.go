package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"

	"github.com/spf13/cobra"
	"github.com/stellar/go/exp/clients/horizon"
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

var horizonURL string
var startSequence uint32
var count uint

func init() {
	rootCmd.PersistentFlags().StringVarP(&horizonURL, "url", "u", "", "Horizon server URL")
	rootCmd.PersistentFlags().Uint32VarP(&startSequence, "start", "s", 0, "Sequence number of a start ledger (follows descending order, defaults to the latest ledger)")
	rootCmd.PersistentFlags().UintVarP(&count, "count", "c", 10000, "Number of ledgers to check")
}

var rootCmd = &cobra.Command{
	Use:   "horizon-verify",
	Short: "tool to check horizon data consistency",
	Run: func(cmd *cobra.Command, args []string) {
		if horizonURL == "" {
			cmd.Help()
			return
		}

		client := horizonclient.Client{
			HorizonURL: horizonURL,
			HTTP:       http.DefaultClient,
		}

		ledgerCursor := ""

		if startSequence != 0 {
			startSequence++

			ledger, err := client.LedgerDetail(startSequence)
			if err != nil {
				panic(err)
			}

			ledgerCursor = ledger.PagingToken()
		}

		fmt.Printf("%s: Checking %d ledgers starting from cursor \"%s\"\n\n", horizonURL, count, ledgerCursor)

		for {
			ledgersPage, err := client.Ledgers(horizonclient.LedgerRequest{
				Limit:  200,
				Order:  horizonclient.OrderDesc,
				Cursor: ledgerCursor,
			})

			if err != nil {
				panic(err)
			}

			if len(ledgersPage.Embedded.Records) == 0 {
				fmt.Println("Done")
				return
			}

			for _, ledger := range ledgersPage.Embedded.Records {
				fmt.Printf("Checking ledger: %d (successful=%d failed=%d)\n", ledger.Sequence, ledger.SuccessfulTransactionCount, *ledger.FailedTransactionCount)

				ledgerCursor = ledger.PagingToken()

				body := getBody(horizonURL + fmt.Sprintf("/ledgers/%d/transactions?limit=200&include_failed=true", ledger.Sequence))
				transactionsPage := TransactionsPage{}
				err = json.Unmarshal(body, &transactionsPage)
				if err != nil {
					panic(err)
				}

				var wg sync.WaitGroup

				var successful, failed int32

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

						body := getBody(horizonURL + fmt.Sprintf("/transactions/%s/operations?limit=200", transaction.Hash))
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

				if successful != ledger.SuccessfulTransactionCount || failed != *ledger.FailedTransactionCount {
					panic(fmt.Sprintf("Invalid ledger counters %d", ledger.Sequence))
				}

				count--
				if count == 0 {
					fmt.Println("Done")
					return
				}
			}
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
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
