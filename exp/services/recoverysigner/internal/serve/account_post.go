package serve

import (
	"net/http"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve/auth"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpdecode"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

type accountPostHandler struct {
	Logger         *supportlog.Entry
	SigningAddress *keypair.FromAddress
	AccountStore   account.Store
	HorizonClient  horizonclient.ClientInterface
}

type accountPostRequest struct {
	Address    *keypair.FromAddress         `path:"address"`
	Identities accountPostRequestIdentities `json:"identities" form:"identities"`
}

type accountPostRequestIdentities struct {
	Owner accountPostRequestIdentity `json:"owner" form:"owner"`
	Other accountPostRequestIdentity `json:"other" form:"other"`
}

type accountPostRequestIdentity struct {
	Address     string `json:"account" form:"account"`
	PhoneNumber string `json:"phone_number" form:"phone_number"`
	Email       string `json:"email" form:"email"`
}

func (h accountPostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	claims, _ := auth.FromContext(ctx)
	if claims.Address == "" {
		unauthorized.Render(w)
		return
	}

	req := accountPostRequest{}
	err := httpdecode.Decode(r, &req)
	if err != nil || req.Address == nil {
		badRequest.Render(w)
		return
	}

	if req.Address.Address() != claims.Address {
		unauthorized.Render(w)
		return
	}

	acc := account.Account{
		Address: req.Address.Address(),
		OwnerIdentities: account.Identities{
			Address:     req.Identities.Owner.Address,
			PhoneNumber: req.Identities.Owner.PhoneNumber,
			Email:       req.Identities.Owner.Email,
		},
		OtherIdentities: account.Identities{
			Address:     req.Identities.Other.Address,
			PhoneNumber: req.Identities.Other.PhoneNumber,
			Email:       req.Identities.Other.Email,
		},
	}
	err = h.AccountStore.Add(acc)
	if err == account.ErrAlreadyExists {
		conflict.Render(w)
		return
	} else if err != nil {
		h.Logger.Error(err)
		serverError.Render(w)
		return
	}

	resp := accountResponse{
		Address: req.Address.Address(),
		Identities: accountResponseIdentities{
			Owner: accountResponseIdentity{
				Present: acc.OwnerIdentities.Present(),
			},
			Other: accountResponseIdentity{
				Present: acc.OtherIdentities.Present(),
			},
		},
		Identity: "account",
		Signer:   h.SigningAddress.Address(),
	}
	httpjson.Render(w, resp, httpjson.JSON)
}
