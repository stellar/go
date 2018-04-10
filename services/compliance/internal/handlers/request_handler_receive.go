package handlers

import (
	log "github.com/sirupsen/logrus"
	"net/http"

	"github.com/stellar/go/services/internal/bridge-compliance-shared/http/helpers"
	callback "github.com/stellar/go/services/internal/bridge-compliance-shared/protocols/compliance"
	"github.com/zenazn/goji/web"
)

// HandlerReceive implements /receive endpoint
func (rh *RequestHandler) HandlerReceive(c web.C, w http.ResponseWriter, r *http.Request) {
	request := &callback.ReceiveRequest{}
	err := helpers.FromRequest(r, request)
	if err != nil {
		log.Error(err.Error())
		helpers.Write(w, helpers.InvalidParameterError)
		return
	}

	err = request.Validate()
	if err != nil {
		errorResponse := err.(*helpers.ErrorResponse)
		// TODO
		// log.WithFields(errorResponse.LogData).Error(errorResponse.Error())
		helpers.Write(w, errorResponse)
		return
	}

	authorizedTransaction, err := rh.Database.GetAuthorizedTransactionByMemo(request.Memo)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error getting authorizedTransaction")
		helpers.Write(w, helpers.InternalServerError)
		return
	}

	if authorizedTransaction == nil {
		log.WithFields(log.Fields{"memo": request.Memo}).Warn("authorizedTransaction not found")
		helpers.Write(w, callback.TransactionNotFoundError)
		return
	}

	response := callback.ReceiveResponse{Data: authorizedTransaction.Data}
	helpers.Write(w, &response)
}
