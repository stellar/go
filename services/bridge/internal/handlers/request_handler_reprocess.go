package handlers

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/protocols"
	"github.com/stellar/go/protocols/bridge"
)

// Authorize implements /reprocess endpoint
func (rh *RequestHandler) Reprocess(w http.ResponseWriter, r *http.Request) {
	request := &bridge.ReprocessRequest{}
	err := protocols.FromRequest(r, request)
	if err != nil {
		log.Error(err.Error())
		protocols.Write(w, protocols.InvalidParameterError)
		return
	}

	err = request.Validate()
	if err != nil {
		errorResponse := err.(*protocols.ErrorResponse)
		// TODO
		// log.WithFields(errorResponse.LogData).Error(errorResponse.Error())
		protocols.Write(w, errorResponse)
		return
	}

	operation, err := rh.Horizon.LoadOperation(request.OperationID)
	if err != nil {
		protocols.Write(w, &bridge.ReprocessResponse{Status: "error", Message: err.Error()})
		return
	}

	err = rh.PaymentListener.ReprocessPayment(operation, request.Force)

	if err != nil {
		protocols.Write(w, &bridge.ReprocessResponse{Status: "error", Message: err.Error()})
		return
	}

	protocols.Write(w, &bridge.ReprocessResponse{Status: "ok"})
}
