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
	Logger                      *supportlog.Entry
	HorizonClient               horizonclient.ClientInterface
	NetworkPassphrase           string
	SigningAddress              *keypair.FromAddress
	JWTPrivateKey               *ecdsa.PrivateKey
	JWTIssuer                   string
	JWTExpiresIn                time.Duration
	AllowAccountsThatDoNotExist bool
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

	var clientAccountExists bool
	clientAccount, err := h.HorizonClient.AccountDetail(horizonclient.AccountRequest{AccountID: clientAccountID})
	switch {
	case err == nil:
		clientAccountExists = true
	case horizonclient.IsNotFoundError(err):
		clientAccountExists = false
	default:
		serverError.Render(w)
		return
	}

	if clientAccountExists {
		requiredThreshold := txnbuild.Threshold(clientAccount.Thresholds.HighThreshold)
		clientSignerSummary := clientAccount.SignerSummary()
		_, err = txnbuild.VerifyChallengeTxThreshold(req.Transaction, h.SigningAddress.Address(), h.NetworkPassphrase, requiredThreshold, clientSignerSummary)
		if err != nil {
			unauthorized.Render(w)
			return
		}
	} else {
		if !h.AllowAccountsThatDoNotExist {
			unauthorized.Render(w)
			return
		}
		_, err = txnbuild.VerifyChallengeTxSigners(req.Transaction, h.SigningAddress.Address(), h.NetworkPassphrase, clientAccountID)
		if err != nil {
			unauthorized.Render(w)
			return
		}
	}

	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"iss": h.JWTIssuer,
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
