package auth

import (
	"crypto/ecdsa"
	"net/http"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpauthz"
	"gopkg.in/square/go-jose.v2/jwt"
)

// SEP10Middleware provides middleware for handling an authentication SEP-10 JWT.
func SEP10Middleware(k *ecdsa.PublicKey) func(http.Handler) http.Handler {
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

func sep10ClaimsFromRequest(r *http.Request, k *ecdsa.PublicKey) (address string, ok bool) {
	authHeader := r.Header.Get("Authorization")
	tokenEncoded := httpauthz.ParseBearerToken(authHeader)
	if tokenEncoded == "" {
		return "", false
	}
	token, err := jwt.ParseSigned(tokenEncoded)
	if err != nil {
		return "", false
	}
	cl := jwt.Claims{}
	err = token.Claims(k, &cl)
	if err != nil {
		return "", false
	}
	err = cl.Validate(jwt.Expected{Time: time.Now()})
	if err != nil {
		return "", false
	}
	address = cl.Subject
	_, err = keypair.ParseAddress(address)
	if err != nil {
		return "", false
	}
	return address, true
}
