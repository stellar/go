package serve

import (
	"fmt"
	"net/http"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/txnbuild"
)

type friendbotHandler struct {
	accountIssuerSecret string
	assetCode           string
	horizonClient       horizonclient.ClientInterface
	horizonURL          string
	networkPassphrase   string
}

func (h friendbotHandler) validate() error {
	// TODO: validation
	return nil
}

type friendbotRequest struct {
	Address string `query:"addr"`
}

// TODO: tests

func (h friendbotHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	in := friendbotRequest{}
	err := httpdecode.Decode(r, &in)
	if err != nil || in.Address == "" {
		makeBadRequestError(`Missing query paramater "addr".`).Render(w)
		return
	}

	if !strkey.IsValidEd25519PublicKey(in.Address) {
		makeBadRequestError(`"addr" is not a valid Stellar address.`).Render(w)
		return
	}

	account, err := h.horizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: in.Address})
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrapf(err, "getting detail for account %s", in.Address))
		makeBadRequestError(`Please make sure the provided account address already exists in the network.`).Render(w)
		return
	}

	kp, err := keypair.ParseFull(h.accountIssuerSecret)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "parsing secret"))
		serverError.Render(w)
		return
	}
	asset := txnbuild.CreditAsset{
		Code:   h.assetCode,
		Issuer: kp.Address(),
	}

	var hasRegulatedAssetTrustline bool
	for _, b := range account.Balances {
		if b.Asset.Code == asset.Code && b.Asset.Issuer == asset.Issuer {
			hasRegulatedAssetTrustline = true
			break
		}
	}
	if !hasRegulatedAssetTrustline {
		makeBadRequestError(fmt.Sprintf("Address %s doesn't have a trustline for %s:%s", in.Address, asset.Code, asset.Issuer)).Render(w)
		return
	}

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &txnbuild.SimpleAccount{AccountID: kp.Address()},
		IncrementSequenceNum: true,
		Operations: []txnbuild.Operation{
			&txnbuild.SetTrustLineFlags{
				Trustor: in.Address,
				Asset:   asset,
				SetFlags: []txnbuild.TrustLineFlag{
					txnbuild.TrustLineAuthorized,
				},
			},
			&txnbuild.Payment{
				Destination: in.Address,
				Amount:      "10000",
				Asset:       asset,
			},
			&txnbuild.SetTrustLineFlags{
				Trustor:  in.Address,
				Asset:    asset,
				SetFlags: []txnbuild.TrustLineFlag{},
			},
		},
		BaseFee:    300,
		Timebounds: txnbuild.NewTimeout(300),
	})
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "building transaction"))
		serverError.Render(w)
		return
	}

	tx, err = tx.Sign(h.networkPassphrase, kp)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "signing transaction"))
		serverError.Render(w)
		return
	}

	_, err = h.horizonClient.SubmitTransaction(tx)
	if err != nil {
		err = parseHorizonError(err)
		log.Ctx(ctx).Error(err)
		serverError.Render(w)
		return
	}

	httpjson.Render(w, nil, httpjson.JSON)

	// h.Render(w, *result)
	// httpjson.RenderStatus(w, e.Status, e, httpjson.JSON)
	// httpjson.Render(w, resp, httpjson.JSON)
}
