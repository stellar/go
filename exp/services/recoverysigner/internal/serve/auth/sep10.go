package auth

import (
	"net/http"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpauthz"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// SEP10Middleware provides middleware for handling an authentication SEP-10 JWT.
func SEP10Middleware(k jose.JSONWebKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if address, ok := sep10ClaimsFromRequest(r, k); ok {
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

func (c sep10JWTClaims) Validate() error {
	// TODO: Verify that iat and exp are present.
	// TODO: Verify that sub is a G... strkey.
	// TODO: Verify that iss is as expected.
	return c.Claims.Validate(jwt.Expected{Time: time.Now()})
}

func sep10ClaimsFromRequest(r *http.Request, k jose.JSONWebKey) (address string, ok bool) {
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
	err = tokenClaims.Validate()
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
