package server

import (
	"github.com/r3labs/sse"
	"github.com/stellar/go/services/bifrost/config"
	"github.com/stellar/go/services/bifrost/database"
	"github.com/stellar/go/services/bifrost/ethereum"
	"github.com/stellar/go/services/bifrost/queue"
	"github.com/stellar/go/services/bifrost/stellar"
	"github.com/stellar/go/support/log"
)

type Server struct {
	Config                     *config.Config               `inject:""`
	Database                   database.Database            `inject:""`
	EthereumListener           *ethereum.Listener           `inject:""`
	EthereumAddressGenerator   *ethereum.AddressGenerator   `inject:""`
	StellarAccountConfigurator *stellar.AccountConfigurator `inject:""`
	TransactionsQueue          queue.Queue                  `inject:""`

	eventsServer *sse.Server
	log          *log.Entry
}

type GenerateEthereumAddressResponse struct {
	EthereumAddress string `json:"ethereum-address"`
}

// AddressEvent is an event sent to address SSE stream.
type AddressEvent string

const (
	TransactionReceivedAddressEvent AddressEvent = "transaction_received"
	AccountCreatedAddressEvent      AddressEvent = "account_created"
	AccountCreditedAddressEvent     AddressEvent = "account_credited"
)
