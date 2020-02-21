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
	Address         *keypair.FromAddress         `path:"address"`
	Type            string                       `json:"type" form:"type"`
	OwnerIdentities accountPostRequestIdentities `json:"owner_identities" form:"owner_identities"`
	OtherIdentities accountPostRequestIdentities `json:"other_identities" form:"other_identities"`
}

type accountPostRequestIdentities struct {
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
		Type:    req.Type, // TODO: Validate input type.
		OwnerIdentities: account.Identities{
			Address:     req.OwnerIdentities.Address,
			PhoneNumber: req.OwnerIdentities.PhoneNumber,
			Email:       req.OwnerIdentities.Email,
		},
		OtherIdentities: account.Identities{
			Address:     req.OtherIdentities.Address,
			PhoneNumber: req.OtherIdentities.PhoneNumber,
			Email:       req.OtherIdentities.Email,
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
		Address:  req.Address.Address(),
		Type:     req.Type,
		Identity: "account",
		Signer:   h.SigningAddress.Address(),
	}
	httpjson.Render(w, resp, httpjson.JSON)
}
