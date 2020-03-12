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
	Identities []accountPostRequestIdentity `json:"identities" form:"identities"`
}

type accountPostRequestIdentity struct {
	Role        string                                 `json:"role" form:"role"`
	AuthMethods []accountPostRequestIdentityAuthMethod `json:"auth_methods" form:"auth_methods"`
}

type accountPostRequestIdentityAuthMethod struct {
	Type  string `json:"type" form:"type"`
	Value string `json:"value" form:"value"`
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
		Address:    req.Address.Address(),
		Identities: []account.Identity{},
	}
	for _, i := range req.Identities {
		accIdentity := account.Identity{
			Role: i.Role,
		}
		for _, m := range i.AuthMethods {
			t, tErr := account.AuthMethodTypeFromString(m.Type)
			if tErr != nil {
				badRequest.Render(w)
				return
			}

			accIdentity.AuthMethods = append(accIdentity.AuthMethods, account.AuthMethod{
				Type:  t,
				Value: m.Value,
			})
		}
		acc.Identities = append(acc.Identities, accIdentity)
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
		Address: acc.Address,
		Signer:  h.SigningAddress.Address(),
	}
	for _, i := range acc.Identities {
		respIdentity := accountResponseIdentity{
			Role: i.Role,
		}
		resp.Identities = append(resp.Identities, respIdentity)
	}
	httpjson.Render(w, resp, httpjson.JSON)
}
