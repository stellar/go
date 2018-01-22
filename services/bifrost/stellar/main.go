package stellar

import (
	"sync"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/support/log"
)

type SubmissionType string

const (
	SubmissionTypeCreateAccount SubmissionType = "submission_create_account"
	SubmissionTypeSendTokens    SubmissionType = "submission_send_tokens"
)

type SubmissionArchive interface {
	Find(txID, assetCode string, st SubmissionType) (string, error)
	Store(txID, assetCode string, st SubmissionType, xdr string) error
	Delete(txID, assetCode string, st SubmissionType) error
}

// AccountConfigurator is responsible for configuring new Stellar accounts that
// participate in ICO.
type AccountConfigurator struct {
	Horizon           horizon.ClientInterface `inject:""`
	NetworkPassphrase string
	IssuerPublicKey   string
	SignerSecretKey   string
	NeedsAuthorize    bool
	TokenAssetCode    string
	OnAccountCreated  func(destination string)
	OnAccountCredited func(destination string, assetCode string, amount string)
	SubmissionArchive SubmissionArchive `inject:""`

	signerPublicKey      string
	sequence             uint64
	sequenceMutex        sync.Mutex
	processingCount      int
	processingCountMutex sync.Mutex
	log                  *log.Entry
}
