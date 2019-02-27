package txnbuild

import (
	"log"
	"strconv"

	// "github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// SeqNumFromAccount gets the sequence number from an account,
// and returns it as a 64-bit integer.
func SeqNumFromAccount(account horizon.Account) (xdr.SequenceNumber, error) {
	seqNum, err := strconv.ParseUint(account.Sequence, 10, 64)

	if err != nil {
		return 0, errors.Wrap(err, "Failed to parse account sequence number")
	}

	return xdr.SequenceNumber(seqNum), nil
}

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
