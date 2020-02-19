package auth

import (
	"crypto/ecdsa"
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpauthz"
)

// SEP10Middleware provides middleware for handling an authentication SEP-10 JWT.
func SEP10Middleware(k *ecdsa.PublicKey) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if address, ok := sep10ClaimsFromRequest(r, k); ok {
				ctx := r.Context()
				claims, _ := FromContext(ctx)
				claims.Address = address
				ctx = NewContext(ctx, claims)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

type sep10JWTClaims struct {
	jwt.StandardClaims
}

func (c sep10JWTClaims) Valid() error {
	// TODO: Verify that iat and exp are present.
	// TODO: Verify that sub is a G... strkey.
	// TODO: Verify that iss is as expected.
	return c.StandardClaims.Valid()
}

func sep10ClaimsFromRequest(r *http.Request, k *ecdsa.PublicKey) (address string, ok bool) {
	authHeader := r.Header.Get("Authorization")
	tokenEncoded := httpauthz.ParseBearerToken(authHeader)
	if tokenEncoded == "" {
		return "", false
	}
	tokenClaims := sep10JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenEncoded, &tokenClaims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return k, nil
	})
	if err != nil {
		return "", false
	}
	if !token.Valid {
		return "", false
	}
	address = tokenClaims.Subject
	_, err = keypair.ParseAddress(address)
	if err != nil {
		return "", false
	}
	return address, true
}
