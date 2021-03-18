package serve

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpdecode"
	"github.com/stellar/go/support/log"
	"github.com/stellar/go/support/render/httpjson"
)

type txApproveHandler struct{}

type txApproveRequest struct {
	Transaction string `json:"tx" form:"tx"`
}

func (h txApproveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	in := txApproveRequest{}
	err := httpdecode.Decode(r, &in)
	if err != nil {
		log.Ctx(ctx).Error(errors.Wrap(err, "decoding input parameters"))
		httpErr := NewHTTPError(http.StatusBadRequest, "Invalid input parameters")
		httpErr.Render(w)
		return
	}
	rejected, err := h.isRejected(ctx, in)
	if err != nil {
		httpErr, ok := err.(*httpError)
		if !ok {
			httpErr = serverError
		}
		httpErr.Render(w)
		return
	}
	if rejected {
		httpjson.Render(w, json.RawMessage(`{
			"status": "rejected",
			"error": "The destination account is blocked."
		  }`), httpjson.JSON)
	}
	httpjson.Render(w, httpjson.DefaultResponse, httpjson.JSON)
}

func (h txApproveHandler) isRejected(ctx context.Context, in txApproveRequest) (bool, error) {
	log.Ctx(ctx).Info(in.Transaction)
	return false, nil
}
