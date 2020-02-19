package auth

import (
	"context"
)

type contextKey int

const (
	claimsContextKey contextKey = iota
)

// Claims holds a set of authenticated claims.
type Claims struct {
	Address     string
	PhoneNumber string
	Email       string
}

// FromContext returns authenticated claims that are stored in the context.
func FromContext(ctx context.Context) (Claims, bool) {
	if claims, ok := ctx.Value(claimsContextKey).(Claims); ok {
		return claims, true
	}
	return Claims{}, false
}

// NewContext returns a new context that is a copy of the given context with
// the claims set within. Claims can be retrieved from the context using
// FromContext.
func NewContext(ctx context.Context, c Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, c)
}
