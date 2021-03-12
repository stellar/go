package serve

import (
	"net/http"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/render/httpjson"
)

var serverError = errorResponse{
	Status: http.StatusInternalServerError,
	Error:  "An error occurred while processing this request.",
}
var notFound = errorResponse{
	Status: http.StatusNotFound,
	Error:  "The resource at the url requested was not found.",
}
var methodNotAllowed = errorResponse{
	Status: http.StatusMethodNotAllowed,
	Error:  "The method is not allowed for resource at the url requested.",
}
var badRequest = errorResponse{
	Status: http.StatusBadRequest,
	Error:  "The request was invalid in some way.",
}

func makeBadRequestError(msg string) errorResponse {
	return errorResponse{
		Status: http.StatusBadRequest,
		Error:  msg,
	}
}

type errorResponse struct {
	Status int    `json:"-"`
	Error  string `json:"error"`
}

func (e errorResponse) Render(w http.ResponseWriter) {
	httpjson.RenderStatus(w, e.Status, e, httpjson.JSON)
}

type errorHandler struct {
	Error errorResponse
}

func (h errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Error.Render(w)
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
