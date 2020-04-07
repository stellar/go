package serve

import (
	"net/http"

	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve/auth"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpdecode"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

type accountGetHandler struct {
	Logger         *supportlog.Entry
	SigningAddress *keypair.FromAddress
	AccountStore   account.Store
}

type accountGetRequest struct {
	Address *keypair.FromAddress `path:"address"`
}

func (h accountGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, _ := auth.FromContext(ctx)
	if claims.Address == "" && claims.PhoneNumber == "" && claims.Email == "" {
		unauthorized.Render(w)
		return
	}

	req := accountGetRequest{}
	err := httpdecode.Decode(r, &req)
	if err != nil || req.Address == nil {
		badRequest.Render(w)
		return
	}

	acc, err := h.AccountStore.Get(req.Address.Address())
	if err == account.ErrNotFound {
		notFound.Render(w)
		return
	} else if err != nil {
		h.Logger.Error(err)
		serverError.Render(w)
		return
	}

	resp := accountResponse{
		Address: acc.Address,
		Signer:  h.SigningAddress.Address(),
	}

	// Authorized if authenticated as the account.
	authorized := claims.Address == req.Address.Address()

	// Authorized if authenticated as an identity registered with the account.
	for _, i := range acc.Identities {
		respIdentity := accountResponseIdentity{
			Role: i.Role,
		}
		for _, m := range i.AuthMethods {
			if m.Value != "" && ((m.Type == account.AuthMethodTypeAddress && m.Value == claims.Address) ||
				(m.Type == account.AuthMethodTypePhoneNumber && m.Value == claims.PhoneNumber) ||
				(m.Type == account.AuthMethodTypeEmail && m.Value == claims.Email)) {
				respIdentity.Authenticated = true
				authorized = true
				break
			}
		}

		resp.Identities = append(resp.Identities, respIdentity)
	}
	if !authorized {
		notFound.Render(w)
		return
	}

	httpjson.Render(w, resp, httpjson.JSON)
}
