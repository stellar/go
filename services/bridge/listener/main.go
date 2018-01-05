package listener

import (
	"net/http"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/services/bridge/database"
	"github.com/stellar/go/support/log"
)

type PaymentStatus string

const (
	PaymentStatusProcessing          PaymentStatus = "processing"
	PaymentStatusNotReceiver         PaymentStatus = "not_receiver"
	PaymentStatusNotPaymentOperation PaymentStatus = "not_payment"
	PaymentStatusSuccess             PaymentStatus = "success"
	PaymentStatusError               PaymentStatus = "error"
)

// PaymentListener is listening for a new payments received by ReceivingAccount
type PaymentListener struct {
	ReceivingAccount string
	CallbackURL      string

	HTTPClient *http.Client            `inject:""`
	Horizon    horizon.ClientInterface `inject:""`
	Database   database.Database       `inject:""`

	log *log.Entry
}
