package auth

import (
	"crypto/ecdsa"
	"net/http"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/http/httpauthz"
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
	token, err := jwt.ParseString(
		tokenEncoded,
		jwt.WithVerify(jwa.ES256, k),
	)
	if err != nil {
		return "", false
	}
	err = token.Verify()
	if err != nil {
		return "", false
	}
	address = token.Subject()
	_, err = keypair.ParseAddress(address)
	if err != nil {
		return "", false
	}
	return address, true
}
