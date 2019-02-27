package txnbuild

import (
	"log"

	"github.com/stellar/go/protocols/horizon"
)

// PrintTransactionSuccess prints the fields of a Horizon response.
func PrintTransactionSuccess(resp horizon.TransactionSuccess) {
	log.Println("***TransactionSuccess dump***")
	log.Println("    Links:", resp.Links)
	log.Println("    Hash:", resp.Hash)
	log.Println("    Ledger:", resp.Ledger)
	log.Println("    Env:", resp.Env)
	log.Println("    Result:", resp.Result)
	log.Println("    Meta:", resp.Meta)
	log.Println("")
}
