// Package problem provides utility functions for rendering errors as RFC7807
// compatible responses.
//
// RFC7807: https://tools.ietf.org/html/rfc7807
//
// The P type is used to define application problems.
// The Render function is used to serialize problems in a HTTP response.
package errors

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

// Renderable is a value that can be rendered as a HTTP JSON response.
type E interface {
	StatusCode() int
	Error() string
}

// Errors is a registry of errors to E responses.
type Errors struct {
	log          *log.Entry
	contentType  string
	errToEMap    map[error]E
	defaultE     E
	beforeRender func(e E) E
	reportFn     ReportFunc
}

// New returns a new instance of Errors.
func New(contentType string, defaultE E, beforeRender func(e E) E, log *log.Entry) *Errors {
	return &Errors{
		log:          log,
		contentType:  contentType,
		errToEMap:    map[error]E{},
		defaultE:     defaultE,
		beforeRender: beforeRender,
	}
}

// RegisterError records an error -> E mapping, allowing the app to register
// specific errors that may occur in other packages to be rendered as a specific
// value.
//
// For example, you might want to render any sql.ErrNoRows errors as a
// specific response, and you would do so by calling:
//
//   errors.RegisterError(sql.ErrNoRows, ...)
func (er *Errors) RegisterError(err error, e E) {
	er.errToEMap[err] = e
}

// IsKnownError maps an error to a list of known errors
func (er *Errors) IsKnownError(err error) error {
	origErr := errors.Cause(err)

	switch origErr.(type) {
	case error:
		if err, ok := er.errToEMap[origErr]; ok {
			return err
		}
		return nil
	default:
		return nil
	}
}

// UnRegisterErrors removes all registered errors
func (er *Errors) UnRegisterErrors() {
	er.errToEMap = map[error]E{}
}

// ReportFunc is a function type used to report unexpected errors.
type ReportFunc func(context.Context, error)

// RegisterReportFunc registers the report function that you want to use to
// report errors. Once reportFn is initialzied, it will be used to report
// unexpected errors.
func (er *Errors) RegisterReportFunc(fn ReportFunc) {
	er.reportFn = fn
}

// Render writes a http response to `w` for the err.
func (er *Errors) Render(ctx context.Context, w http.ResponseWriter, err error) {
	origErr := errors.Cause(err)

	var e E
	switch v := origErr.(type) {
	case E:
		e = v
	case error:
		var ok bool
		e, ok = er.errToEMap[origErr]

		// If this error is not a registered error
		// log it and replace it with the default
		if !ok {
			er.log.Ctx(ctx).WithStack(err).Error(err)
			if er.reportFn != nil {
				er.reportFn(ctx, err)
			}
			e = er.defaultE
		}
	}

	if er.beforeRender != nil {
		e = er.beforeRender(e)
	}

	er.render(ctx, w, e)
}

func (er *Errors) render(ctx context.Context, w http.ResponseWriter, e E) {
	w.Header().Set("Content-Type", er.contentType)

	js, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		err = errors.Wrap(err, "failed to encode renderable")
		er.log.Ctx(ctx).WithStack(err).Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(e.StatusCode())
	w.Write(js)
}
