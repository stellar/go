package actions

import (
	"net/http"
	"net/url"

	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/services/horizon/internal/ledger"
	"github.com/stellar/go/services/horizon/internal/resourceadapter"
)

type CoreSettings struct {
	CurrentProtocolVersion       int32
	CoreSupportedProtocolVersion int32
	CoreVersion                  string
}

type CoreSettingsGetter interface {
	GetCoreSettings() CoreSettings
}

type GetRootHandler struct {
	LedgerState *ledger.State
	CoreSettingsGetter
	NetworkPassphrase string
	FriendbotURL      *url.URL
	HorizonVersion    string
}

func (handler GetRootHandler) GetResource(w HeaderWriter, r *http.Request) (interface{}, error) {
	var res horizon.Root
	templates := map[string]string{
		"accounts":           AccountsQuery{}.URITemplate(),
		"claimableBalances":  ClaimableBalancesQuery{}.URITemplate(),
		"offers":             OffersQuery{}.URITemplate(),
		"strictReceivePaths": StrictReceivePathsQuery{}.URITemplate(),
		"strictSendPaths":    FindFixedPathsQuery{}.URITemplate(),
	}
	coreSettings := handler.GetCoreSettings()
	resourceadapter.PopulateRoot(
		r.Context(),
		&res,
		handler.LedgerState.CurrentStatus(),
		handler.HorizonVersion,
		coreSettings.CoreVersion,
		handler.NetworkPassphrase,
		coreSettings.CurrentProtocolVersion,
		coreSettings.CoreSupportedProtocolVersion,
		handler.FriendbotURL,
		templates,
	)
	return res, nil
}
