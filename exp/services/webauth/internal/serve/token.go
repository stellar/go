package serve

import (
	"net/http"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpdecode"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/txnbuild"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

type tokenHandler struct {
	Logger                      *supportlog.Entry
	HorizonClient               horizonclient.ClientInterface
	NetworkPassphrase           string
	SigningAddress              *keypair.FromAddress
	JWK                         jose.JSONWebKey
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

	jwsOptions := &jose.SignerOptions{}
	jwsOptions.WithType("JWT")
	jws, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.SignatureAlgorithm(h.JWK.Algorithm), Key: h.JWK.Key}, jwsOptions)
	if err != nil {
		h.Logger.Ctx(ctx).WithStack(err).Error(err)
		serverError.Render(w)
		return
	}

	now := time.Now().UTC()
	claims := jwt.Claims{
		Issuer:   h.JWTIssuer,
		Subject:  clientAccountID,
		IssuedAt: jwt.NewNumericDate(now),
		Expiry:   jwt.NewNumericDate(now.Add(h.JWTExpiresIn)),
	}
	tokenStr, err := jwt.Signed(jws).Claims(claims).CompactSerialize()
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
