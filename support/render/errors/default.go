package errors

import (
	"context"
	"net/http"

	"github.com/stellar/go/support/log"
)

// DefaultContentType is the default content type used with the default error renderer.
var DefaultContentType = "application/json; charset=utf8"

// DefaultE is the default value that is rendered if no value is registered for an error.
var DefaultE E = empty500{}

type empty500 struct{}

func (empty500) StatusCode() int {
	return http.StatusInternalServerError
}

func (empty500) Error() string {
	return http.StatusText(http.StatusInternalServerError)
}

// DefaultLogger is the default logger used with the default error renderer.
var DefaultLogger = log.DefaultLogger

// Default is the error renderer used by the package functions providing a
// global state registry and rendering of errors for an application. For a
// non-global state registry instantiate a new ErrorRenderer with New.
var Default = New(DefaultContentType, DefaultE, nil, DefaultLogger)

// RegisterError records an error -> P mapping, allowing the app to register
// specific errors that may occur in other packages to be rendered as a specific
// P instance.
//
// For example, you might want to render any sql.ErrNoRows errors as a
// problem.NotFound, and you would do so by calling:
//
// problem.RegisterError(sql.ErrNoRows, problem.NotFound) in you application
// initialization sequence
func RegisterError(err error, e E) {
	Default.RegisterError(err, e)
}

// IsKnownError maps an error to a list of known errors
func IsKnownError(err error) error {
	return Default.IsKnownError(err)
}

// UnRegisterErrors removes all registered errors
func UnRegisterErrors() {
	Default.UnRegisterErrors()
}

// RegisterReportFunc registers the report function that you want to use to
// report errors. Once reportFn is initialzied, it will be used to report
// unexpected errors.
func RegisterReportFunc(fn ReportFunc) {
	Default.RegisterReportFunc(fn)
}

// Render writes a http response to `w`, compliant with the "Problem
// Details for HTTP APIs" RFC:
// https://tools.ietf.org/html/draft-ietf-appsawg-http-problem-00
func Render(ctx context.Context, w http.ResponseWriter, err error) {
	Default.Render(ctx, w, err)
}
