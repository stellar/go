package txnbuild

import (
	"encoding/json"
	"log"
	"strconv"

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

// PrintHorizonError decodes and prints the contents of horizon.Error.Problem.
// Decoded XDR can be pasted into the Stellar Laboratory XDR viewer
// (https://www.stellar.org/laboratory) for further analysis.
func PrintHorizonError(hError *horizon.Error) error {
	problem := hError.Problem
	log.Println("Error type:", problem.Type)
	log.Println("Error title:", problem.Title)
	log.Println("Error status:", problem.Status)
	log.Println("Error detail:", problem.Detail)
	log.Println("Error instance:", problem.Instance)

	var decodedResultCodes map[string]interface{}
	var decodedResult, decodedEnvelope string
	var err error

	err = json.Unmarshal(problem.Extras["result_codes"], &decodedResultCodes)
	if err != nil {
		return errors.Wrap(err, "Couldn't unmarshal result_codes")
	}
	log.Println("Error extras result codes:", decodedResultCodes)

	err = json.Unmarshal(problem.Extras["result_xdr"], &decodedResult)
	if err != nil {
		return errors.Wrap(err, "Couldn't unmarshal result_xdr")
	}
	log.Println("Error extras result (TransactionResult) XDR:", decodedResult)

	err = json.Unmarshal(problem.Extras["envelope_xdr"], &decodedEnvelope)
	if err != nil {
		return errors.Wrap(err, "Couldn't unmarshal envelope_xdr")
	}
	log.Println("Error extras envelope (TransactionEnvelope) XDR:", decodedEnvelope)

	return nil
}
