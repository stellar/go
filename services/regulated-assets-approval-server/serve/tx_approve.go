package serve

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

type txApproveHandler struct{}

type txApproveRequest struct {
	Transaction string `json:"tx" form:"tx"`
}

func (h txApproveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	in := txApproveRequest{}
	err := httpdecode.Decode(r, &in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding input parameters"))
		httpErr := NewHTTPError(http.StatusBadRequest, "Invalid input parameters")
		httpErr.Render(w)
		return
	}
	rejected, err := h.isRejected(ctx, in)
	if err != nil {
		httpErr, ok := err.(*httpError)
		if !ok {
			httpErr = serverError
		}
		httpErr.Render(w)
		return
	}
	if rejected {
		httpjson.Render(w, json.RawMessage(`{
			"status": "rejected",
			"error": "The destination account is blocked."
		  }`), httpjson.JSON)
	}
	httpjson.Render(w, httpjson.DefaultResponse, httpjson.JSON)
}

func (h txApproveHandler) isRejected(ctx context.Context, in txApproveRequest) (bool, error) {
	log.Ctx(ctx).Info(in.Transaction)
	/*
		err := h.validate()
		if err != nil {
			err = errors.Wrap(err, "validating txApproveHandler")
			log.Ctx(ctx).Error(err)
			return err
		}

			if in.Address == "" {
				return NewHTTPError(http.StatusBadRequest, `Missing query paramater "addr".`)
			}

			if !strkey.IsValidEd25519PublicKey(in.Address) {
				return NewHTTPError(http.StatusBadRequest, `"addr" is not a valid Stellar address.`)
			}

			account, err := h.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: in.Address})
			if err != nil {
				log.Ctx(ctx).Error(errors.Wrapf(err, "getting detail for account %s", in.Address))
				return NewHTTPError(http.StatusBadRequest, `Please make sure the provided account address already exists in the network.`)
			}

			kp, err := keypair.ParseFull(h.accountIssuerSecret)
			if err != nil {
				err = errors.Wrap(err, "parsing secret")
				log.Ctx(ctx).Error(err)
				return err
			}

			asset := txnbuild.CreditAsset{
				Code:   h.assetCode,
				Issuer: kp.Address(),
			}

			var accountHasTrustline bool
			for _, b := range account.Balances {
				if b.Asset.Code == asset.Code && b.Asset.Issuer == asset.Issuer {
					accountHasTrustline = true
					break
				}
			}
			if !accountHasTrustline {
				return NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Account with address %s doesn't have a trustline for %s:%s", in.Address, asset.Code, asset.Issuer))
			}

			issuerAcc, err := h.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: kp.Address()})
			if err != nil {
				log.Ctx(ctx).Error(errors.Wrapf(err, "getting detail for issuer account %s", kp.Address()))
				return serverError
			}

			tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
				SourceAccount:        &issuerAcc,
				IncrementSequenceNum: true,
				Operations: []txnbuild.Operation{
					&txnbuild.AllowTrust{
						Trustor:   in.Address,
						Type:      asset,
						Authorize: true,
					},
					&txnbuild.Payment{
						Destination: in.Address,
						Amount:      "10000",
						Asset:       asset,
					},
					&txnbuild.AllowTrust{
						Trustor:   in.Address,
						Type:      asset,
						Authorize: false,
					},
				},
				BaseFee:    300,
				Timebounds: txnbuild.NewTimeout(300),
			})
			if err != nil {
				err = errors.Wrap(err, "building transaction")
				log.Ctx(ctx).Error(err)
				return err
			}

			tx, err = tx.Sign(h.networkPassphrase, kp)
			if err != nil {
				err = errors.Wrap(err, "signing transaction")
				log.Ctx(ctx).Error(err)
				return err
			}

			_, err = h.horizonClient.SubmitTransaction(tx)
			if err != nil {
				err = parseHorizonError(err)
				log.Ctx(ctx).Error(err)
				return err
			}
	*/
	return false, nil
}
