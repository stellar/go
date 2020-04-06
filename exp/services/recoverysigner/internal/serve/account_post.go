package serve

import (
	"net/http"

	"github.com/stellar/go/exp/services/recoverysigner/internal/account"
	"github.com/stellar/go/exp/services/recoverysigner/internal/serve/auth"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	supportlog "github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

type accountPostHandler struct {
	Logger         *supportlog.Entry
	SigningAddress *keypair.FromAddress
	AccountStore   account.Store
}

type accountPostRequest struct {
	Address    *keypair.FromAddress         `path:"address"`
	Identities []accountPostRequestIdentity `json:"identities" form:"identities"`
}

func (r accountPostRequest) Validate() error {
	if len(r.Identities) == 0 {
		return errors.Errorf("no identities provided but at least one is required")
	}
	for _, i := range r.Identities {
		err := i.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

type accountPostRequestIdentity struct {
	Role        string                                 `json:"role" form:"role"`
	AuthMethods []accountPostRequestIdentityAuthMethod `json:"auth_methods" form:"auth_methods"`
}

func (i accountPostRequestIdentity) Validate() error {
	if i.Role == "" {
		return errors.Errorf("role is not set but required")
	}
	if len(i.AuthMethods) == 0 {
		return errors.Errorf("auth methods not provided for identity but required")
	}
	for _, am := range i.AuthMethods {
		err := am.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

type accountPostRequestIdentityAuthMethod struct {
	Type  string `json:"type" form:"type"`
	Value string `json:"value" form:"value"`
}

func (am accountPostRequestIdentityAuthMethod) Validate() error {
	if !account.AuthMethodType(am.Type).Valid() {
		return errors.Errorf("auth method type %q unrecognized", am.Type)
	}
	// TODO: Validate auth method values: Stellar address, phone number and email.
	return nil
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

	if req.Validate() != nil {
		badRequest.Render(w)
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
			accIdentity.AuthMethods = append(accIdentity.AuthMethods, account.AuthMethod{
				Type:  account.AuthMethodType(m.Type),
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
