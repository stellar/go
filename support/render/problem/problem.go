// Package problem provides utility functions for rendering errors as RFC7807
// compatible responses.
//
// RFC7807: https://tools.ietf.org/html/rfc7807
//
// The P type is used to define application problems.
// The Render function is used to serialize problems in a HTTP response.
package problem

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

var (
	// ServerError is a well-known problem type. Use it as a shortcut.
	ServerError = P{
		Type:   "server_error",
		Title:  "Internal Server Error",
		Status: http.StatusInternalServerError,
		Detail: "An error occurred while processing this request.  This is usually due " +
			"to a bug within the server software.  Trying this request again may " +
			"succeed if the bug is transient, otherwise please report this issue " +
			"to the issue tracker at: https://github.com/stellar/go/issues." +
			" Please include this response in your issue.",
	}

	// NotFound is a well-known problem type.  Use it as a shortcut in your actions
	NotFound = P{
		Type:   "not_found",
		Title:  "Resource Missing",
		Status: http.StatusNotFound,
		Detail: "The resource at the url requested was not found.  This usually " +
			"occurs for one of two reasons:  The url requested is not valid, or no " +
			"data in our database could be found with the parameters provided.",
	}

	// BadRequest is a well-known problem type.  Use it as a shortcut
	// in your actions.
	BadRequest = P{
		Type:   "bad_request",
		Title:  "Bad Request",
		Status: http.StatusBadRequest,
		Detail: "The request you sent was invalid in some way.",
	}
)

// P is a struct that represents an error response to be rendered to a connected
// client.
type P struct {
	Type   string                 `json:"type"`
	Title  string                 `json:"title"`
	Status int                    `json:"status"`
	Detail string                 `json:"detail,omitempty"`
	Extras map[string]interface{} `json:"extras,omitempty"`
}

func (p P) Error() string {
	return fmt.Sprintf("problem: %s", p.Type)
}

// Problem is an instance of the functionality served by the problem package.
type Problem struct {
	serviceHost     string
	log             *log.Entry
	errToProblemMap map[error]P
	reportFn        ReportFunc
}

// New returns a new instance of Problem.
func New(serviceHost string, log *log.Entry) *Problem {
	return &Problem{
		serviceHost:     serviceHost,
		log:             log,
		errToProblemMap: map[error]P{},
	}
}

// ServiceHost returns the service host the Problem instance is configured with.
func (ps *Problem) ServiceHost() string {
	return ps.serviceHost
}

// RegisterError records an error -> P mapping, allowing the app to register
// specific errors that may occur in other packages to be rendered as a specific
// P instance.
//
// For example, you might want to render any sql.ErrNoRows errors as a
// problem.NotFound, and you would do so by calling:
//
// problem.RegisterError(sql.ErrNoRows, problem.NotFound) in you application
// initialization sequence
func (ps *Problem) RegisterError(err error, p P) {
	ps.errToProblemMap[err] = p
}

// IsKnownError maps an error to a list of known errors
func (ps *Problem) IsKnownError(err error) error {
	origErr := errors.Cause(err)

	switch origErr.(type) {
	case error:
		if err, ok := ps.errToProblemMap[origErr]; ok {
			return err
		}
		return nil
	default:
		return nil
	}
}

// UnRegisterErrors removes all registered errors
func (ps *Problem) UnRegisterErrors() {
	ps.errToProblemMap = map[error]P{}
}

// RegisterHost registers the service host url. It is used to prepend the host
// url to the error type. If you don't wish to prepend anything to the error
// type, register host as an empty string.
func (ps *Problem) RegisterHost(host string) {
	ps.serviceHost = host
}

// ReportFunc is a function type used to report unexpected errors.
type ReportFunc func(context.Context, error)

// RegisterReportFunc registers the report function that you want to use to
// report errors. Once reportFn is initialzied, it will be used to report
// unexpected errors.
func (ps *Problem) RegisterReportFunc(fn ReportFunc) {
	ps.reportFn = fn
}

// Render writes a http response to `w`, compliant with the "Problem
// Details for HTTP APIs" RFC: https://www.rfc-editor.org/rfc/rfc7807.txt
func (ps *Problem) Render(ctx context.Context, w http.ResponseWriter, err error) {
	origErr := errors.Cause(err)

	var problem P
	switch p := origErr.(type) {
	case P:
		problem = p
	case *P:
		problem = *p
	case error:
		var ok bool
		problem, ok = ps.errToProblemMap[origErr]

		// If this error is not a registered error
		// log it and replace it with a 500 error
		if !ok {
			ps.log.Ctx(ctx).WithStack(err).Error(err)
			if ps.reportFn != nil {
				ps.reportFn(ctx, err)
			}
			problem = ServerError
		}
	}

	ps.renderProblem(ctx, w, problem)
}

func (ps *Problem) renderProblem(ctx context.Context, w http.ResponseWriter, p P) {
	if ps.serviceHost != "" && !strings.HasPrefix(p.Type, ps.serviceHost) {
		p.Type = ps.serviceHost + p.Type
	}

	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")

	js, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		err = errors.Wrap(err, "failed to encode problem")
		ps.log.Ctx(ctx).WithStack(err).Error(err)
		http.Error(w, "error rendering problem", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(p.Status)
	w.Write(js)
}

// MakeInvalidFieldProblem is a helper function to make a BadRequest with extras
func MakeInvalidFieldProblem(name string, reason error) *P {
	br := BadRequest
	AddInvalidField(&br, name, reason)
	return &br
}

// AddInvalidField adds invalid field extras to a given P
func AddInvalidField(p *P, name string, reason error) {
	p.Extras = map[string]interface{}{
		"invalid_field": name,
		"reason":        reason.Error(),
	}
}
