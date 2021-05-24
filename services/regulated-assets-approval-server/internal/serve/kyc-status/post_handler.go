package kycstatus

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/httperror"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

type kycPostRequest struct {
	CallbackID   string `path:"callback_id"`
	EmailAddress string `json:"email_address"`
}

type kycPostResponse struct {
	Result     string `json:"result"`
	StatusCode int    `json:"-"`
}

func (k *kycPostResponse) Render(w http.ResponseWriter) {
	httpjson.RenderStatus(w, k.StatusCode, k, httpjson.JSON)
}

func NewKYCStatusPostResponse() *kycPostResponse {
	return &kycPostResponse{
		Result:     "no_further_action_required",
		StatusCode: http.StatusOK,
	}
}

type PostHandler struct {
	DB *sqlx.DB
}

func (h PostHandler) validate() error {
	if h.DB == nil {
		return errors.New("database cannot be nil")
	}
	return nil
}

func (h PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	err := h.validate()
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "validating kyc-status PostHandler"))
		httperror.InternalServer.Render(w)
		return
	}

	in := kycPostRequest{}
	err = httpdecode.Decode(r, &in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding kyc-status POST Request"))
		httperror.BadRequest.Render(w)
		return
	}

	kycResponse, err := h.handle(ctx, in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "validating the input POST request for kyc-status"))
		httpErr, ok := err.(*httperror.Error)
		if !ok {
			httpErr = httperror.InternalServer
		}
		httpErr.Render(w)
		return
	}

	kycResponse.Render(w)
}

func (h PostHandler) handle(ctx context.Context, in kycPostRequest) (resp *kycPostResponse, err error) {
	defer func() {
		log.Ctx(ctx).Debug("==== will log responses ====")
		log.Ctx(ctx).Debugf("req: %+v", in)
		log.Ctx(ctx).Debugf("resp: %+v", resp)
		log.Ctx(ctx).Debugf("err: %+v", err)
		log.Ctx(ctx).Debug("====  did log responses ====")
	}()

	// Check if kycPostRequest values are present or not malformed.
	if in.CallbackID == "" {
		return nil, httperror.NewHTTPError(http.StatusBadRequest, "Missing callbackID.")
	}
	if in.EmailAddress == "" {
		return nil, httperror.NewHTTPError(http.StatusBadRequest, "Missing email_address.")
	}
	if !RxEmail.MatchString(in.EmailAddress) {
		return nil, httperror.NewHTTPError(http.StatusBadRequest, "The provided email_address is invalid.")
	}

	var exists bool
	query, args := in.buildUpdateKYCQuery()
	err = h.DB.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err != nil {
		return nil, errors.Wrap(err, "querying the database")
	}
	if !exists {
		return nil, httperror.NewHTTPError(http.StatusNotFound, "Not found.")
	}

	return NewKYCStatusPostResponse(), nil
}

// isKYCRuleRespected validates if KYC data is approved or rejected.
// Current rule(s) emails starting "x" are rejected and other emails are automatically approved.
func (in kycPostRequest) isKYCRuleRespected() bool {
	return !strings.HasPrefix(strings.ToLower(in.EmailAddress), "x")
}

// buildUpdateKYCQuery builds a query that will approve or reject stellar account from accounts_kyc_status table.
// Afterwards the query should return an exists boolean if present.
func (in kycPostRequest) buildUpdateKYCQuery() (string, []interface{}) {
	var (
		query strings.Builder
		args  []interface{}
	)
	query.WriteString("WITH updated_row AS (")
	query.WriteString("UPDATE accounts_kyc_status ")
	query.WriteString("SET kyc_submitted_at = NOW(), ")

	// Append email address for query built.
	args = append(args, in.EmailAddress)
	query.WriteString(fmt.Sprintf("email_address = $%d, ", len(args)))

	// Check if KYC info is approved or rejected
	if in.isKYCRuleRespected() {
		query.WriteString("approved_at = NOW(), rejected_at = NULL ")
	} else {
		query.WriteString("rejected_at = NOW(), approved_at = NULL ")
	}

	// Append CallbackID for query built.
	args = append(args, in.CallbackID)
	query.WriteString(fmt.Sprintf("WHERE callback_id = $%d ", len(args)))

	// Build remaining query.
	query.WriteString("RETURNING * ")
	query.WriteString(")")
	query.WriteString(`
		SELECT EXISTS(
			SELECT * FROM updated_row
		)
	`)

	return query.String(), args
}

// RxEmail is a regex used to validate e-mail addresses, according with the reference https://www.alexedwards.net/blog/validation-snippets-for-go#email-validation.
// It's free to use under the [MIT Licence](https://opensource.org/licenses/MIT)
var RxEmail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
