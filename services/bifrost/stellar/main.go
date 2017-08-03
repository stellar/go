package stellar

import (
	"sync"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/log"
)

// AccountConfigurator is responsible for configuring new Stellar accounts that
// participate in ICO.
type AccountConfigurator struct {
	Horizon           horizon.ClientInterface `inject:""`
	NetworkPassphrase string
	IssuerSecretKey   string
	OnAccountCreated  func(string)
	OnAccountCredited func(string, string, string)

	issuerPublicKey string
	sequence        uint64
	sequenceMutex   sync.Mutex
	log             *log.Entry
}
