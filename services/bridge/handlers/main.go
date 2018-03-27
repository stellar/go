package handlers

import (
	"github.com/stellar/go/clients/federation"
	"github.com/stellar/go/services/bridge/config"
	"github.com/stellar/go/services/bridge/db"
	"github.com/stellar/go/services/bridge/external"
	"github.com/stellar/go/services/bridge/horizon"
	"github.com/stellar/go/services/bridge/listener"
	"github.com/stellar/go/services/bridge/net"
	"github.com/stellar/go/services/bridge/submitter"
)

// RequestHandler implements bridge server request handlers
type RequestHandler struct {
	Config               *config.Config                          `inject:""`
	Client               net.HTTPClientInterface                 `inject:""`
	Horizon              horizon.HorizonInterface                `inject:""`
	Driver               db.Driver                               `inject:""`
	Repository           db.RepositoryInterface                  `inject:""`
	StellarTomlResolver  external.StellarTomlClientInterface     `inject:""`
	FederationResolver   federation.ClientInterface              `inject:""`
	TransactionSubmitter submitter.TransactionSubmitterInterface `inject:""`
	PaymentListener      *listener.PaymentListener               `inject:""`
}

func (rh *RequestHandler) isAssetAllowed(code string, issuer string) bool {
	for _, asset := range rh.Config.Assets {
		if asset.Code == code && asset.Issuer == issuer {
			return true
		}
	}
	return false
}
