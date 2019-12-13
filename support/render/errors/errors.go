// Package errors provides utility functions for rendering error responses.
//
// The E interface is used to define error responses. Values that satisfy E
// will be json marshaled.
//
// The Render function is used to serialize problems in a HTTP response.
//
// Errors that do not satisfy the E interface will be mapped to an E or to the
// default E.
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
	error
	// StatusCode returns the HTTP status code that will be used in the HTTP
	// response when this error is rendered.
	StatusCode() int
	// E returns itself, and acts an an indicator a type implements this
	// interface.
	E() E
}

// Errors is a registry of errors to E error responses.
type Errors struct {
	log          *log.Entry
	contentType  string
	errToEMap    map[error]E
	defaultE     E
	beforeRender func(e E) E
	reportFn     ReportFunc
}

// New returns a new instance of Errors.
//
// Error responses are always rendered as JSON, but the specific content type
// can be configured.
//
// The defaultE will be used for rendering an error in any case an error is not
// mapped to an E.
//
// The beforeRender function is called before rendering any E, allowing for
// modification.
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
