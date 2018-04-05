package handlers

import (
	"encoding/hex"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/stellar/go/address"
	b "github.com/stellar/go/build"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/protocols"
	"github.com/stellar/go/protocols/bridge"
	"github.com/stellar/go/protocols/compliance"
	callback "github.com/stellar/go/protocols/compliance/server"
	"github.com/stellar/go/protocols/federation"
	"github.com/stellar/go/xdr"
)

// Payment implements /payment endpoint
func (rh *RequestHandler) Payment(w http.ResponseWriter, r *http.Request) {
	request := &bridge.PaymentRequest{}
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

	var paymentID *string

	if request.ID != "" {
		sentTransaction, err := rh.Database.GetSentTransactionByPaymentID(request.ID)
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Error("Error getting sent transaction")
			protocols.Write(w, protocols.InternalServerError)
			return
		}

		if sentTransaction == nil {
			paymentID = &request.ID
		} else {
			log.WithFields(log.Fields{"paymentID": request.ID, "tx": sentTransaction.EnvelopeXdr}).Info("Transaction with given ID already exists, resubmitting...")
			submitResponse, err := rh.Horizon.SubmitTransaction(sentTransaction.EnvelopeXdr)
			rh.handleTransactionSubmitResponse(w, submitResponse, err)
			return
		}
	}

	if request.Source == "" {
		request.Source = rh.Config.Accounts.BaseSeed
	}

	// Will use compliance if compliance server is connected and:
	// * User passed extra memo OR
	// * User explicitly wants to use compliance protocol
	if rh.Config.Compliance != "" &&
		(request.ExtraMemo != "" || (request.ExtraMemo == "" && request.UseCompliance)) {
		rh.complianceProtocolPayment(w, request, paymentID)
	} else {
		rh.standardPayment(w, request, paymentID)
	}
}

func (rh *RequestHandler) complianceProtocolPayment(w http.ResponseWriter, request *bridge.PaymentRequest, paymentID *string) {
	// Compliance server part
	sendRequest := request.ToComplianceSendRequest()

	resp, err := rh.Client.PostForm(
		rh.Config.Compliance+"/send",
		protocols.ToValues(sendRequest),
	)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error sending request to compliance server")
		protocols.Write(w, protocols.InternalServerError)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error reading compliance server response")
		protocols.Write(w, protocols.InternalServerError)
		return
	}

	if resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("Error response from compliance server")
		protocols.Write(w, protocols.InternalServerError)
		return
	}

	var callbackSendResponse callback.SendResponse
	err = json.Unmarshal(body, &callbackSendResponse)
	if err != nil {
		log.Error("Error unmarshalling from compliance server")
		protocols.Write(w, protocols.InternalServerError)
		return
	}

	if callbackSendResponse.AuthResponse.InfoStatus == compliance.AuthStatusPending ||
		callbackSendResponse.AuthResponse.TxStatus == compliance.AuthStatusPending {
		log.WithFields(log.Fields{"response": callbackSendResponse}).Info("Compliance response pending")
		protocols.Write(w, bridge.NewPaymentPendingError(callbackSendResponse.AuthResponse.Pending))
		return
	}

	if callbackSendResponse.AuthResponse.InfoStatus == compliance.AuthStatusDenied ||
		callbackSendResponse.AuthResponse.TxStatus == compliance.AuthStatusDenied {
		log.WithFields(log.Fields{"response": callbackSendResponse}).Info("Compliance response denied")
		protocols.Write(w, bridge.PaymentDenied)
		return
	}

	var tx xdr.Transaction
	err = xdr.SafeUnmarshalBase64(callbackSendResponse.TransactionXdr, &tx)
	if err != nil {
		log.Error("Error unmarshalling transaction returned by compliance server")
		protocols.Write(w, protocols.InternalServerError)
		return
	}

	submitResponse, err := rh.TransactionSubmitter.SignAndSubmitRawTransaction(paymentID, request.Source, &tx)
	rh.handleTransactionSubmitResponse(w, submitResponse, err)
}

func (rh *RequestHandler) standardPayment(w http.ResponseWriter, request *bridge.PaymentRequest, paymentID *string) {
	destinationObject := &federation.NameResponse{}
	var err error

	if request.ForwardDestination == nil {
		_, _, err = address.Split(request.Destination)
		if err != nil {
			destinationObject.AccountID = request.Destination
		} else {
			destinationObject, err = rh.FederationResolver.LookupByAddress(request.Destination)
			if err != nil {
				log.WithFields(log.Fields{"destination": request.Destination, "err": err}).Print("Cannot resolve address")
				protocols.Write(w, bridge.PaymentCannotResolveDestination)
				return
			}
		}
	} else {
		destinationObject, err = rh.FederationResolver.ForwardRequest(request.ForwardDestination.Domain, request.ForwardDestination.Fields)
		if err != nil {
			log.WithFields(log.Fields{"destination": request.Destination, "err": err}).Print("Cannot resolve address")
			protocols.Write(w, bridge.PaymentCannotResolveDestination)
			return
		}
	}

	// TODO
	// if !protocols.IsValidAccountID(destinationObject.AccountID) {
	// 	log.WithFields(log.Fields{"AccountId": destinationObject.AccountID}).Print("Invalid AccountId in destination")
	// 	protocols.Write(w, protocols.NewInvalidParameterError("destination", request.Destination, "Destination public key must start with `G`."))
	// 	return
	// }

	var payWithMutator *b.PayWithPath

	if request.SendMax != "" {
		// Path payment
		var sendAsset b.Asset
		if request.SendAssetCode == "" && request.SendAssetIssuer == "" {
			sendAsset = b.NativeAsset()
		} else {
			sendAsset = b.CreditAsset(request.SendAssetCode, request.SendAssetIssuer)
		}

		payWith := b.PayWith(sendAsset, request.SendMax)

		for _, asset := range request.Path {
			payWith = payWith.Through(asset.ToBaseAsset())
		}

		payWithMutator = &payWith
	}

	var operationBuilder interface{}

	if request.AssetCode != "" && request.AssetIssuer != "" {
		mutators := []interface{}{
			b.Destination{destinationObject.AccountID},
			b.CreditAmount{request.AssetCode, request.AssetIssuer, request.Amount},
		}

		if payWithMutator != nil {
			mutators = append(mutators, *payWithMutator)
		}

		operationBuilder = b.Payment(mutators...)
	} else {
		mutators := []interface{}{
			b.Destination{destinationObject.AccountID},
			b.NativeAmount{request.Amount},
		}

		if payWithMutator != nil {
			mutators = append(mutators, *payWithMutator)
		}

		// Check if destination account exist
		_, err = rh.Horizon.LoadAccount(destinationObject.AccountID)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Error loading account")
			operationBuilder = b.CreateAccount(mutators...)
		} else {
			operationBuilder = b.Payment(mutators...)
		}
	}

	memoType := request.MemoType
	memo := request.Memo

	if destinationObject.MemoType != "" {
		if request.MemoType != "" {
			log.Print("Memo given in request but federation returned memo fields.")
			protocols.Write(w, bridge.PaymentCannotUseMemo)
			return
		}

		memoType = destinationObject.MemoType
		memo = destinationObject.Memo.Value
	}

	var memoMutator interface{}
	switch {
	case memoType == "":
		break
	case memoType == "id":
		id, err := strconv.ParseUint(memo, 10, 64)
		if err != nil {
			log.WithFields(log.Fields{"memo": memo}).Print("Cannot convert memo_id value to uint64")
			protocols.Write(w, protocols.NewInvalidParameterError("memo", request.Memo, "Memo.id must be a number"))
			return
		}
		memoMutator = b.MemoID{id}
	case memoType == "text":
		memoMutator = b.MemoText{memo}
	case memoType == "hash":
		memoBytes, err := hex.DecodeString(memo)
		if err != nil || len(memoBytes) != 32 {
			log.WithFields(log.Fields{"memo": memo}).Print("Cannot decode hash memo value")
			protocols.Write(w, protocols.NewInvalidParameterError("memo", request.Memo, "Memo.hash must be 32 bytes and hex encoded."))
			return
		}
		var b32 [32]byte
		copy(b32[:], memoBytes[0:32])
		hash := xdr.Hash(b32)
		memoMutator = b.MemoHash{hash}
	default:
		log.Print("Not supported memo type: ", memoType)
		protocols.Write(w, protocols.NewInvalidParameterError("memo", request.Memo, "Memo type not supported"))
		return
	}

	submitResponse, err := rh.TransactionSubmitter.SubmitTransaction(paymentID, request.Source, operationBuilder, memoMutator)
	rh.handleTransactionSubmitResponse(w, submitResponse, err)
}

func (rh *RequestHandler) handleTransactionSubmitResponse(w http.ResponseWriter, submitResponse horizon.TransactionSuccess, err error) {
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
