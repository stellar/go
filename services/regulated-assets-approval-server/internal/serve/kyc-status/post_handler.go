package kycstatus

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi"
	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/services/regulated-assets-approval-server/internal/serve/httperror"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

type PostHandler struct {
	DB *sqlx.DB
}

func (h PostHandler) validate() error {
	if h.DB == nil {
		return errors.New("database cannot be nil")
	}
	return nil
}

type postRequest struct {
	CallbackID   string
	EmailAddress string `json:"email_address"`
}

type postResponse struct {
	Result  string `json:"result"`
	Message string `json:"message"`
}

func NewApprovedKYCStatusPostResponse() *postResponse {
	return &postResponse{
		Result:  "no_further_action_required",
		Message: "Your KYC has been approved!",
	}
}

func NewRejectedKYCStatusPostResponse() *postResponse {
	return &postResponse{
		Result:  "no_further_action_required",
		Message: "Your KYC has been rejected!",
	}
}

func (h PostHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := h.validate()
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "validating kyc-status PostHandler"))
		httperror.InternalServer.Render(w)
		return
	}
	in := postRequest{
		CallbackID: chi.URLParam(r, "callback_id"),
	}
	err = httpdecode.Decode(r, &in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding kyc-status POST Request"))
		httperror.BadRequest.Render(w)
		return
	}

	resp, err := h.handle(ctx, in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "validating the input POST request for kyc-status"))
		httpErr, ok := err.(*httperror.Error)
		if !ok {
			httpErr = httperror.InternalServer
		}
		httpErr.Render(w)
		return
	}

	httpjson.Render(w, resp, httpjson.JSON)
}

func (h PostHandler) handle(ctx context.Context, in postRequest) (*postResponse, error) {
	err := h.validate()
	if err != nil {
		return nil, errors.Wrap(err, "validating KYCStatusGetDetailHandler")
	}
	if in.CallbackID == "" {
		return nil, httperror.NewHTTPError(http.StatusBadRequest, "Missing callback ID.")
	}
	if in.EmailAddress == "" {
		return nil, httperror.NewHTTPError(http.StatusBadRequest, "Missing email_address.")
	}
	if !RxEmail.MatchString(in.EmailAddress) {
		return nil, httperror.NewHTTPError(http.StatusBadRequest, "The provided email_address is invalid.")
	}
	var exists bool
	query, args := buildUpdateKYCQuery(in)
	err = h.DB.QueryRowContext(ctx, query, args...).Scan(&exists)
	if err != nil {
		return nil, errors.Wrap(err, "querying the database")
	}
	if !exists {
		return nil, httperror.NewHTTPError(http.StatusNotFound, "Not found.")
	}
	if in.isKYCRuleRespected() {
		return NewApprovedKYCStatusPostResponse(), nil
	}
	return NewRejectedKYCStatusPostResponse(), nil
}

// isKYCRuleRespected is an arbitrary rule where emails starting with "x" are
// rejected and other emails are automatically approved.
func (in postRequest) isKYCRuleRespected() bool {
	return !strings.HasPrefix(strings.ToLower(in.EmailAddress), "xx")
}

func buildUpdateKYCQuery(in postRequest) (string, []interface{}) {
	var query strings.Builder
	var args []interface{}

	query.WriteString("WITH updated_row AS (")
	query.WriteString("UPDATE accounts_kyc_status ")
	query.WriteString("SET kyc_submitted_at = NOW(), ")

	args = append(args, in.EmailAddress)
	query.WriteString(fmt.Sprintf("email_address = $%d, ", len(args)))

	if in.isKYCRuleRespected() {
		query.WriteString("approved_at = NOW(), rejected_at = NULL ")
	} else {
		query.WriteString("rejected_at = NOW(), approved_at = NULL ")
	}

	args = append(args, in.CallbackID)
	query.WriteString(fmt.Sprintf("WHERE callback_id = $%d ", len(args)))

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
