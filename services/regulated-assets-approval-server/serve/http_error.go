package serve

import (
	"net/http"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/httpjson"
)

type httpError struct {
	ErrorMessage string `json:"error"`
	Status       int    `json:"-"`
}

func (h *httpError) Error() string {
	return h.ErrorMessage
}

func NewHTTPError(status int, errorMessage string) *httpError {
	return &httpError{
		ErrorMessage: errorMessage,
		Status:       status,
	}
}

func (e *httpError) Render(w http.ResponseWriter) {
	httpjson.RenderStatus(w, e.Status, e, httpjson.JSON)
}

var serverError = &httpError{
	ErrorMessage: "An error occurred while processing this request.",
	Status:       http.StatusInternalServerError,
}

func parseHorizonError(err error) error {
	if err == nil {
		return nil
	}

	rootErr := errors.Cause(err)
	if hError := horizonclient.GetError(rootErr); hError != nil {
		resultCode, _ := hError.ResultCodes()
		err = errors.Wrapf(err, "error submitting transaction: %+v, %+v\n", hError.Problem, resultCode)
	} else {
		err = errors.Wrap(err, "error submitting transaction")
	}
	return err
}
