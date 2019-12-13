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
	"fmt"
	"net/http"
	"strings"

	"github.com/stellar/go/support/log"
	rendererrors "github.com/stellar/go/support/render/errors"
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

func (p P) StatusCode() int {
	return p.Status
}

func (p P) Error() string {
	return fmt.Sprintf("problem: %s", p.Type)
}

func (p P) E() rendererrors.E {
	return p
}

// Problem is an instance of the functionality served by the problem package.
type Problem struct {
	*rendererrors.Errors
	serviceHost string
}

// New returns a new instance of Problem.
func New(serviceHost string, log *log.Entry) *Problem {
	ps := &Problem{
		serviceHost: serviceHost,
	}
	ps.Errors = rendererrors.New(
		"application/problem+json; charset=utf-8",
		ServerError,
		ps.beforeRender,
		log,
	)
	return ps
}

// ServiceHost returns the service host the Problem instance is configured with.
func (ps *Problem) ServiceHost() string {
	return ps.serviceHost
}

// RegisterHost registers the service host url. It is used to prepend the host
// url to the error type. If you don't wish to prepend anything to the error
// type, register host as an empty string.
func (ps *Problem) RegisterHost(host string) {
	ps.serviceHost = host
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
	ps.Errors.RegisterError(err, p)
}

// ReportFunc is a function type used to report unexpected errors.
type ReportFunc func(context.Context, error)

// RegisterReportFunc registers the report function that you want to use to
// report errors. Once reportFn is initialzied, it will be used to report
// unexpected errors.
func (ps *Problem) RegisterReportFunc(fn ReportFunc) {
	ps.Errors.RegisterReportFunc(rendererrors.ReportFunc(fn))
}

func (ps *Problem) beforeRender(e rendererrors.E) rendererrors.E {
	var p P
	switch v := e.(type) {
	case P:
		p = v
	case *P:
		p = *v
	default:
		return e
	}
	if ps.serviceHost != "" && !strings.HasPrefix(p.Type, ps.serviceHost) {
		p.Type = ps.serviceHost + p.Type
	}
	return p
}

// MakeInvalidFieldProblem is a helper function to make a BadRequest with extras
func MakeInvalidFieldProblem(name string, reason error) *P {
	br := BadRequest
	br.Extras = map[string]interface{}{
		"invalid_field": name,
		"reason":        reason.Error(),
	}
	return &br
}
