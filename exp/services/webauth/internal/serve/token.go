package serve

import (
	"crypto/ecdsa"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpdecode"
	supportlog "github.com/stellar/go/support/log"

	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/txnbuild"
)

type tokenHandler struct {
	Logger            *supportlog.Entry
	HorizonClient     horizonclient.ClientInterface
	NetworkPassphrase string
	SigningAddress    *keypair.FromAddress
	JWTPrivateKey     *ecdsa.PrivateKey
	JWTExpiresIn      time.Duration
}

type tokenRequest struct {
	Transaction string `json:"transaction" form:"transaction"`
}

type tokenResponse struct {
	Token string `json:"token"`
}

func (h tokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := tokenRequest{}

	err := httpdecode.Decode(r, &req)
	if err != nil {
		badRequest.Render(w)
		return
	}

	_, clientAccountID, err := txnbuild.ReadChallengeTx(req.Transaction, h.SigningAddress.Address(), h.NetworkPassphrase)
	if err != nil {
		badRequest.Render(w)
		return
	}

	clientAccount, err := h.HorizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: clientAccountID})
	if err != nil {
		serverError.Render(w)
		return
	}

	clientSigners := make([]string, 0, len(clientAccount.Signers))
	for _, signer := range clientAccount.Signers {
		clientSigners = append(clientSigners, signer.Key)
	}

	clientSignersFound, err := txnbuild.VerifyChallengeTxSigners(req.Transaction, h.SigningAddress.Address(), h.NetworkPassphrase, clientSigners...)
	if err != nil {
		unauthorized.Render(w)
		return
	}

	weightVerified := int32(0)
	for _, signerFound := range clientSignersFound {
		for _, signer := range clientAccount.Signers {
			if signer.Key == signerFound {
				weightVerified += signer.Weight
			}
		}
	}

	if weightVerified < int32(clientAccount.Thresholds.HighThreshold) {
		unauthorized.Render(w)
		return
	}

	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": h.SigningAddress.Address(),
		"sub": clientAccountID,
		"iat": now.Unix(),
		"exp": now.Add(h.JWTExpiresIn).Unix(),
	})
	tokenStr, err := token.SignedString(h.JWTPrivateKey)
	if err != nil {
		h.Logger.Ctx(ctx).WithStack(err).Error(err)
		serverError.Render(w)
		return
	}

	res := tokenResponse{
		Token: tokenStr,
	}
	httpjson.Render(w, res, httpjson.JSON)
}
