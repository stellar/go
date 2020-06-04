package problem

import "context"

type contextKey string

const requestErrorKey = contextKey("request_error")

// NewContext returns a new http request context which can be used to obtain the error which
// terminates a request
func NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestErrorKey, new(error))
}

// GetRequestError returns nil if the request corresponding to the context responded successfully.
// Otherwise, the function returns the error which terminated the request.
func GetRequestError(ctx context.Context) error {
	if v := ctx.Value(requestErrorKey); v != nil {
		return *v.(*error)
	}
	return nil
}

func setRequestError(ctx context.Context, err error) {
	if v := ctx.Value(requestErrorKey); v != nil {
		*v.(*error) = err
	}
}
