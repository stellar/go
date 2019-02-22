package txnbuild

import (
	"encoding/json"
	"log"

	"github.com/stellar/go/clients/horizon"
)

// TODO: Consider package heirarchy, rename as needed

func PrintTransactionSuccess(resp horizon.TransactionSuccess) {
	log.Println("TransactionSuccess:")
	log.Println("")
	log.Println("Links:", resp.Links)
	log.Println("Hash:", resp.Hash)
	log.Println("Ledger:", resp.Ledger)
	log.Println("Env:", resp.Env)
	log.Println("Result:", resp.Result)
	log.Println("Meta:", resp.Meta)
	log.Println("")
}

func PrintHorizonError(hError *horizon.Error) {
	problem := hError.Problem
	log.Println("Error type:", problem.Type)
	log.Println("Error title:", problem.Title)
	log.Println("Error status:", problem.Status)
	log.Println("Error detail:", problem.Detail)
	log.Println("Error instance:", problem.Instance)

	var decodedResultCodes map[string]interface{}
	var decodedResult string
	var decodedEnvelope string
	var err error

	err = json.Unmarshal(problem.Extras["result_codes"], &decodedResultCodes)
	CheckError("json unmarshal horizon.Error.Problem.Extras[\"result_codes\"]", err)
	log.Println("Error extras result codes:", decodedResultCodes)

	err = json.Unmarshal(problem.Extras["result_xdr"], &decodedResult)
	CheckError("json unmarshal horizon.Error.Problem.Extras[\"result_xdr\"]", err)
	log.Println("Error extras result (TransactionResult) XDR:", decodedResult)

	err = json.Unmarshal(problem.Extras["envelope_xdr"], &decodedEnvelope)
	CheckError("json unmarshal horizon.Error.Problem.Extras[\"envelope_xdr\"]", err)
	log.Println("Error extras envelope (TransactionEnvelope) XDR:", decodedEnvelope)
}

func CheckError(desc string, err error) {
	if err != nil {
		log.Fatalf("Fatal error (%s): %s", desc, err)
	}
}
