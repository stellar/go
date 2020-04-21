package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpauthz"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// SEP10Middleware provides middleware for handling an authentication SEP-10 JWT.
func SEP10Middleware(issuer string, k jose.JSONWebKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if address, ok := sep10ClaimsFromRequest(r, issuer, k); ok {
				ctx := r.Context()
				auth, _ := FromContext(ctx)
				auth.Address = address
				ctx = NewContext(ctx, auth)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

type sep10JWTClaims struct {
	jwt.Claims
}

func (c sep10JWTClaims) Validate(issuer string) error {
	if c.Claims.IssuedAt == nil {
		return errors.New("validation failed, no issued at (iat) in token")
	}
	if c.Claims.Expiry == nil {
		return errors.New("validation failed, no expiry (exp) in token")
	}
	expectedClaims := jwt.Expected{
		Issuer: issuer,
		Time:   time.Now(),
	}
	return c.Claims.Validate(expectedClaims)
}

func sep10ClaimsFromRequest(r *http.Request, issuer string, k jose.JSONWebKey) (address string, ok bool) {
	authHeader := r.Header.Get("Authorization")
	tokenEncoded := httpauthz.ParseBearerToken(authHeader)
	if tokenEncoded == "" {
		return "", false
	}
	token, err := jwt.ParseSigned(tokenEncoded)
	if err != nil {
		return "", false
	}
	tokenClaims := sep10JWTClaims{}
	err = token.Claims(k, &tokenClaims)
	if err != nil {
		return "", false
	}
	err = tokenClaims.Validate(issuer)
	if err != nil {
		return "", false
	}
	address = tokenClaims.Subject
	_, err = keypair.ParseAddress(address)
	if err != nil {
		return "", false
	}
	return address, true
}
