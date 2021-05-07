package serve

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/httperror"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
	"github.com/stellar/go/txnbuild"
)

type txApproveHandler struct {
	issuerKP          *keypair.Full
	assetCode         string
	baseURL           string
	db                *sqlx.DB
	horizonClient     horizonclient.ClientInterface
	kycThreshold      int64
	networkPassphrase string
}

type txApproveRequest struct {
	Transaction string `json:"tx" form:"tx"`
}

func (h txApproveHandler) validate() error {
	if h.issuerKP == nil {
		return errors.New("issuer keypair cannot be nil")
	}

	if h.assetCode == "" {
		return errors.New("asset code cannot be empty")
	}

	if h.baseURL == "" {
		return errors.New("base url cannot be empty")
	}

	if h.db == nil {
		return errors.New("db cannot be nil")
	}

	if h.horizonClient == nil {
		return errors.New("horizon client cannot be nil")
	}

	if h.kycThreshold <= 0 {
		return errors.New("kyc threshold cannot be less than or equal to zero")
	}

	if h.networkPassphrase == "" {
		return errors.New("network passphrase cannot be empty")
	}

	return nil
}

func (h txApproveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	in := txApproveRequest{}
	err := httpdecode.Decode(r, &in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding input parameters"))
		httpErr := httperror.NewHTTPError(http.StatusBadRequest, "Invalid input parameters")
		httpErr.Render(w)
		return
	}
	rejectedResponse, err := h.isRejected(ctx, in)
	if err != nil {
		httpErr, ok := err.(*httperror.Error)
		if !ok {
			httpErr = httperror.InternalServerError
		}
		httpErr.Render(w)
		return
	}
	httpjson.RenderStatus(w, http.StatusBadRequest, rejectedResponse, httpjson.JSON)
}

func (h txApproveHandler) isRejected(ctx context.Context, in txApproveRequest) (*txApprovalResponse, error) {
	if in.Transaction == "" {
		return NewRejectedTxApprovalResponse("Missing parameter \"tx\"."), nil
	}

	// Decode the request transaction.
	parsed, err := txnbuild.TransactionFromXDR(in.Transaction)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrapf(err, "Parsing transaction %s failed", in.Transaction))
		return NewRejectedTxApprovalResponse("Invalid parameter \"tx\"."), nil
	}

	tx, ok := parsed.Transaction()
	if !ok {
		log.Ctx(ctx).Errorf("Transaction %s is not a simple transaction.", in.Transaction)
		return NewRejectedTxApprovalResponse("Invalid parameter \"tx\"."), nil
	}

	// Check if transaction's source account is the same as the server issuer account.
	if tx.SourceAccount().AccountID == h.issuerKP.Address() {
		log.Ctx(ctx).Errorf("Transaction %s sourceAccount is the same as the server issuer account %s",
			in.Transaction,
			h.issuerKP.Address())
		return NewRejectedTxApprovalResponse("The source account is invalid."), nil
	}

	// Check if transaction's operation(s)' sourceaccount is the same as the server issuer account.
	for _, op := range tx.Operations() {
		if op.GetSourceAccount() == h.issuerKP.Address() {
			return NewRejectedTxApprovalResponse("There is one or more unauthorized operations in the provided transaction."), nil
		}
	}

	return NewRejectedTxApprovalResponse("Not implemented."), nil
}
