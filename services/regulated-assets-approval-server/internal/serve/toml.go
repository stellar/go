package serve

import (
	"fmt"
	"net/http"

	"github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/httperror"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

type stellarTOMLHandler struct {
	assetCode         string
	approvalServer    string
	issuerAddress     string
	networkPassphrase string
	kycThreshold      int64
}

func (h stellarTOMLHandler) validate() error {
	if h.networkPassphrase == "" {
		return errors.New("network passphrase cannot be empty")
	}

	if h.assetCode == "" {
		return errors.New("asset code cannot be empty")
	}

	if h.issuerAddress == "" {
		return errors.New("asset issuer address cannot be empty")
	}

	if !strkey.IsValidEd25519PublicKey(h.issuerAddress) {
		return errors.New("asset issuer address is not a valid public key")
	}

	if h.approvalServer == "" {
		return errors.New("approval server cannot be empty")
	}

	if h.kycThreshold <= 0 {
		return errors.New("kyc threshold cannot be less than or equal to zero")
	}

	return nil
}

func (h stellarTOMLHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := h.validate()
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "validating tomlHandler"))
		httperror.InternalServerError.Render(rw)
		return
	}
	approvalCriteria := fmt.Sprintf("Currently %s is not approving any %s transactions.", h.approvalServer, h.assetCode)

	fmt.Fprintf(rw, "NETWORK_PASSPHRASE=%q\n", h.networkPassphrase)
	fmt.Fprintf(rw, "[[CURRENCIES]]\n")
	fmt.Fprintf(rw, "code=%q\n", h.assetCode)
	fmt.Fprintf(rw, "issuer=%q\n", h.issuerAddress)
	fmt.Fprintf(rw, "regulated=true\n")
	fmt.Fprintf(rw, "approval_server=%q\n", h.approvalServer)
	fmt.Fprintf(rw, "approval_criteria=%q\n", approvalCriteria)
}
