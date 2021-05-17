package kycstatus

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/httperror"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

type kycRecord struct {
	StellarAddress string     `json:"stellar_address"`
	CallbackID     string     `json:"callback_id"`
	EmailAddress   string     `json:"email_address,omitempty"`
	CreatedAt      *time.Time `json:"created_at"`
	KYCSubmittedAt *time.Time `json:"kyc_submitted_at,omitempty"`
	ApprovedAt     *time.Time `json:"approved_at,omitempty"`
	RejectedAt     *time.Time `json:"rejected_at,omitempty"`
}

type GetDetailHandler struct {
	DB *sqlx.DB
}

func (h GetDetailHandler) validate() error {
	if h.DB == nil {
		return errors.New("database cannot be nil")
	}
	return nil
}

type getDetailRequest struct {
	StellarAddressOrCallbackID string
}

func (h GetDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	in := getDetailRequest{
		StellarAddressOrCallbackID: chi.URLParam(r, "stellar_address_or_callback_id"),
	}

	resp, err := h.handle(ctx, in)
	if err != nil {
		httpErr, ok := err.(*httperror.Error)
		if !ok {
			httpErr = httperror.InternalServer
		}
		httpErr.Render(w)
		return
	}

	httpjson.Render(w, resp, httpjson.JSON)
}

func (h GetDetailHandler) handle(ctx context.Context, in getDetailRequest) (*kycRecord, error) {
	err := h.validate()
	if err != nil {
		err = errors.Wrap(err, "validating KYCStatusGetDetailHandler")
		log.Ctx(ctx).Error(err)
		return nil, err
	}
	if in.StellarAddressOrCallbackID == "" {
		return nil, httperror.NewHTTPError(http.StatusBadRequest, "Missing stellar address.")
	}
	var (
		stellarAddress, callbackID             string
		emailAddress                           sql.NullString
		createdAt                              time.Time
		kycSubmittedAt, approvedAt, rejectedAt sql.NullTime
	)
	const q = `
		SELECT stellar_address, email_address, created_at, kyc_submitted_at, approved_at, rejected_at, callback_id
		FROM accounts_kyc_status
		WHERE stellar_address = $1 OR callback_id = $1
	`
	err = h.DB.QueryRowContext(ctx, q, in.StellarAddressOrCallbackID).Scan(&stellarAddress, &emailAddress, &createdAt, &kycSubmittedAt, &approvedAt, &rejectedAt, &callbackID)
	if err == sql.ErrNoRows {
		return nil, httperror.NewHTTPError(http.StatusNotFound, "Not found.")
	}
	if err != nil {
		return nil, errors.Wrap(err, "querying the database")
	}
	return &kycRecord{
		StellarAddress: stellarAddress,
		CallbackID:     callbackID,
		EmailAddress:   emailAddress.String,
		CreatedAt:      &createdAt,
		KYCSubmittedAt: timePointerIfValid(kycSubmittedAt),
		ApprovedAt:     timePointerIfValid(approvedAt),
		RejectedAt:     timePointerIfValid(rejectedAt),
	}, nil
}

// timePointerIfValid returns a pointer to the date from the provided
// `sql.NullTime` if it's valid or `nil` if it's not.
func timePointerIfValid(nt sql.NullTime) *time.Time {
	if nt.Valid {
		return &nt.Time
	}
	return nil
}
