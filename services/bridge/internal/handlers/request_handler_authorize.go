package handlers

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
	b "github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/services/bridge/internal/protocols"
	"github.com/stellar/go/services/bridge/internal/protocols/bridge"
	"github.com/stellar/go/services/bridge/internal/server"
)

// Authorize implements /authorize endpoint
func (rh *RequestHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	request := &bridge.AuthorizeRequest{}
	err := request.FromRequest(r)
	if err != nil {
		log.Error(err.Error())
		server.Write(w, protocols.InvalidParameterError)
		return
	}

	err = request.Validate(rh.Config.Assets, rh.Config.Accounts.IssuingAccountID)
	if err != nil {
		errorResponse := err.(*protocols.ErrorResponse)
		log.WithFields(errorResponse.LogData).Error(errorResponse.Error())
		server.Write(w, errorResponse)
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
			server.Write(w, protocols.InternalServerError)
			return
		}

		w.WriteHeader(herr.Problem.Status)
		err := jsonEncoder.Encode(herr.Problem)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("Error encoding response")
			server.Write(w, protocols.InternalServerError)
			return
		}

		return
	}

	err = jsonEncoder.Encode(submitResponse)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error encoding response")
		server.Write(w, protocols.InternalServerError)
		return
	}
}
