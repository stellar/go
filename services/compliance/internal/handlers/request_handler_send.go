package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/stellar/go/address"
	b "github.com/stellar/go/build"
	"github.com/stellar/go/protocols/compliance"
	"github.com/stellar/go/protocols/federation"
	shared "github.com/stellar/go/services/internal/bridge-compliance-shared"
	"github.com/stellar/go/services/internal/bridge-compliance-shared/http/helpers"
	callback "github.com/stellar/go/services/internal/bridge-compliance-shared/protocols/compliance"
	"github.com/stellar/go/xdr"
	"github.com/zenazn/goji/web"
)

// HandlerSend implements /send endpoint
func (rh *RequestHandler) HandlerSend(c web.C, w http.ResponseWriter, r *http.Request) {
	request := &callback.SendRequest{}
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

	var domain string
	var destinationObject *federation.NameResponse

	if request.ForwardDestination == nil {
		destinationObject, err = rh.FederationResolver.LookupByAddress(request.Destination)
		if err != nil {
			log.WithFields(log.Fields{
				"destination": request.Destination,
				"err":         err,
			}).Print("Cannot resolve address")
			helpers.Write(w, callback.CannotResolveDestination)
			return
		}

		_, domain, err = address.Split(request.Destination)
		if err != nil {
			log.WithFields(log.Fields{
				"destination": request.Destination,
				"err":         err,
			}).Print("Cannot resolve address")
			helpers.Write(w, callback.CannotResolveDestination)
			return
		}
	} else {
		destinationObject, err = rh.FederationResolver.ForwardRequest(request.ForwardDestination.Domain, request.ForwardDestination.Fields)
		if err != nil {
			log.WithFields(log.Fields{
				"destination": request.Destination,
				"err":         err,
			}).Print("Cannot resolve address")
			helpers.Write(w, callback.CannotResolveDestination)
			return
		}

		domain = request.ForwardDestination.Domain
	}

	stellarToml, err := rh.StellarTomlResolver.GetStellarToml(domain)
	if err != nil {
		log.WithFields(log.Fields{
			"destination": request.Destination,
			"err":         err,
		}).Print("Cannot resolve address")
		helpers.Write(w, callback.CannotResolveDestination)
		return
	}

	if stellarToml.AuthServer == "" {
		log.Print("No AUTH_SERVER in stellar.toml")
		helpers.Write(w, callback.AuthServerNotDefined)
		return
	}

	var payWithMutator *b.PayWithPath

	if request.SendMax != "" {
		// Path payment
		var sendAsset b.Asset
		if request.SendAssetCode != "" && request.SendAssetIssuer != "" {
			sendAsset = b.CreditAsset(request.SendAssetCode, request.SendAssetIssuer)
		} else if request.SendAssetCode == "" && request.SendAssetIssuer == "" {
			sendAsset = b.NativeAsset()
		} else {
			log.Print("Missing send asset param.")
			helpers.Write(w, helpers.NewMissingParameter("send asset"))
			return
		}

		payWith := b.PayWith(sendAsset, request.SendMax)

		for _, asset := range request.Path {
			if asset.Code == "" && asset.Issuer == "" {
				payWith = payWith.Through(b.NativeAsset())
			} else {
				payWith = payWith.Through(b.CreditAsset(asset.Code, asset.Issuer))
			}
		}

		payWithMutator = &payWith
	}

	mutators := []interface{}{
		b.Destination{destinationObject.AccountID},
		b.CreditAmount{
			request.AssetCode,
			request.AssetIssuer,
			request.Amount,
		},
	}

	if payWithMutator != nil {
		mutators = append(mutators, *payWithMutator)
	}

	operationMutator := b.Payment(mutators...)
	if operationMutator.Err != nil {
		log.WithFields(log.Fields{
			"err": operationMutator.Err,
		}).Error("Error creating operation")
		helpers.Write(w, helpers.InternalServerError)
		return
	}

	// Fetch Sender Info
	senderInfo := make(map[string]string)

	if rh.Config.Callbacks.FetchInfo != "" {
		fetchInfoRequest := callback.FetchInfoRequest{Address: request.Sender}
		resp, err := rh.Client.PostForm(
			rh.Config.Callbacks.FetchInfo,
			helpers.ToValues(fetchInfoRequest),
		)
		if err != nil {
			log.WithFields(log.Fields{
				"fetch_info": rh.Config.Callbacks.FetchInfo,
				"err":        err,
			}).Error("Error sending request to fetch_info server")
			helpers.Write(w, helpers.InternalServerError)
			return
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"fetch_info": rh.Config.Callbacks.FetchInfo,
				"err":        err,
			}).Error("Error reading fetch_info server response")
			helpers.Write(w, helpers.InternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.WithFields(log.Fields{
				"fetch_info": rh.Config.Callbacks.FetchInfo,
				"status":     resp.StatusCode,
				"body":       string(body),
			}).Error("Error response from fetch_info server")
			helpers.Write(w, helpers.InternalServerError)
			return
		}

		err = json.Unmarshal(body, &senderInfo)
		if err != nil {
			log.WithFields(log.Fields{
				"fetch_info": rh.Config.Callbacks.FetchInfo,
				"err":        err,
			}).Error("Error unmarshalling sender_info server response")
			helpers.Write(w, helpers.InternalServerError)
			return
		}
	}

	attachment := &compliance.Attachment{
		Nonce: rh.NonceGenerator.Generate(),
		Transaction: compliance.Transaction{
			SenderInfo: senderInfo,
			Route:      compliance.Route(destinationObject.Memo.Value),
			Extra:      request.ExtraMemo,
		},
	}

	attachmentJSON, err := attachment.Marshal()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error marshalling attachment")
		helpers.Write(w, helpers.InternalServerError)
		return
	}
	attachmentHashBytes, err := attachment.Hash()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error hashing attachment")
		helpers.Write(w, helpers.InternalServerError)
		return
	}
	memoMutator := &b.MemoHash{xdr.Hash(attachmentHashBytes)}

	transaction, err := shared.BuildTransaction(
		request.Source,
		rh.Config.NetworkPassphrase,
		operationMutator,
		memoMutator,
	)

	txBase64, err := xdr.MarshalBase64(transaction)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Error mashaling transaction")
		helpers.Write(w, helpers.InternalServerError)
		return
	}

	authData := compliance.AuthData{
		Sender:         request.Sender,
		NeedInfo:       rh.Config.NeedsAuth,
		Tx:             txBase64,
		AttachmentJSON: string(attachmentJSON),
	}

	data, err := authData.Marshal()
	if err != nil {
		log.Error("Error mashaling authData")
		helpers.Write(w, helpers.InternalServerError)
		return
	}
	sig, err := rh.SignatureSignerVerifier.Sign(rh.Config.Keys.SigningSeed, data)
	if err != nil {
		log.Error("Error signing authData")
		helpers.Write(w, helpers.InternalServerError)
		return
	}

	authRequest := compliance.AuthRequest{
		DataJSON:  string(data),
		Signature: sig,
	}
	resp, err := rh.Client.PostForm(
		stellarToml.AuthServer,
		authRequest.ToURLValues(),
	)
	if err != nil {
		log.WithFields(log.Fields{
			"auth_server": stellarToml.AuthServer,
			"err":         err,
		}).Error("Error sending request to auth server")
		helpers.Write(w, helpers.InternalServerError)
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("Error reading auth server response")
		helpers.Write(w, helpers.InternalServerError)
		return
	}

	if resp.StatusCode != 200 {
		log.WithFields(log.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("Error response from auth server")
		helpers.Write(w, helpers.InternalServerError)
		return
	}

	var authResponse compliance.AuthResponse
	err = json.Unmarshal(body, &authResponse)
	if err != nil {
		log.WithFields(log.Fields{
			"status": resp.StatusCode,
			"body":   string(body),
		}).Error("Error unmarshalling auth response")
		helpers.Write(w, helpers.InternalServerError)
		return
	}

	response := callback.SendResponse{
		AuthResponse:   authResponse,
		TransactionXdr: txBase64,
	}
	helpers.Write(w, &response)
}
