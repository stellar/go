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
	IssuerPublicKey   string
	SignerSecretKey   string
	NeedsAuthorize    bool
	TokenAssetCode    string
	TokenPriceBTC     string
	TokenPriceETH     string
	StartingBalance   string
	OnAccountCreated  func(destination string)
	OnExchanged       func(destination string)

	signerPublicKey      string
	signerSequence       uint64
	signerSequenceMutex  sync.Mutex
	processingCount      int
	processingCountMutex sync.Mutex
	log                  *log.Entry
}
