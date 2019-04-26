package examplehorizonclient

import (
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
