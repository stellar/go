package examplehorizonclient

import (
	"github.com/stellar/go/exp/txnbuild"
	hProtocol "github.com/stellar/go/protocols/horizon"
)

type AccountRequest struct {
	AccountID string
}

type Client struct {
}

var DefaultTestNetClient = Client{}

func (client *Client) AccountDetail(req AccountRequest) (hProtocol.Account, error) {
	return hProtocol.Account{
		HistoryAccount: hProtocol.HistoryAccount{
			AccountID: req.AccountID,
		},
		Sequence: "3556091187167235",
	}, nil
}

func (client *Client) FetchTimebounds(seconds int64) (txnbuild.Timebounds, error) {
	// TODO: Make timebounds an interface, then we can mock it here
	return txnbuild.NewInfiniteTimeout(), nil
}
