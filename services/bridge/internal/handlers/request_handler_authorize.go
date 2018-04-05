package handlers

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	b "github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/protocols"
	"github.com/stellar/go/protocols/bridge"
)

// Authorize implements /authorize endpoint
func (rh *RequestHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	request := &bridge.AuthorizeRequest{}
	err := protocols.FromRequest(r, request)
	if err != nil {
		log.Error(err.Error())
		protocols.Write(w, protocols.InvalidParameterError)
		return
	}

	err = request.Validate( /*TODO rh.Config.Assets,*/ rh.Config.Accounts.IssuingAccountID)
	if err != nil {
		// TODO
		errorResponse := err.(*protocols.ErrorResponse)
		// log.WithFields(errorResponse.LogData).Error(errorResponse.Error())
		protocols.Write(w, errorResponse)
		return
	}

	operationMutator := b.AllowTrust(
		b.Trustor{request.AccountID},
		b.Authorize{true},
		b.AllowTrustAsset{request.AssetCode},
	)

	submitResponse, err := rh.TransactionSubmitter.SubmitTransaction(
		nil,
		rh.Config.Accounts.AuthorizingSeed,
		operationMutator,
		nil,
	)

	jsonEncoder := json.NewEncoder(w)

	if err != nil {
		herr, isHorizonError := err.(*horizon.Error)
		if !isHorizonError {
			log.WithFields(log.Fields{"err": err}).Error("Error submitting transaction")
			protocols.Write(w, protocols.InternalServerError)
			return
		}

		w.WriteHeader(herr.Problem.Status)
		err := jsonEncoder.Encode(herr.Problem)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("Error encoding response")
			protocols.Write(w, protocols.InternalServerError)
			return
		}

		return
	}

	err = jsonEncoder.Encode(submitResponse)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error encoding response")
		protocols.Write(w, protocols.InternalServerError)
		return
	}
}
