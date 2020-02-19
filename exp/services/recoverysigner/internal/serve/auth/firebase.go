package auth

import (
	"context"
	"net/http"

	firebase "firebase.google.com/go"
	firebaseauth "firebase.google.com/go/auth"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/http/httpauthz"
	"google.golang.org/api/option"
)

type FirebaseTokenVerifier interface {
	Verify(r *http.Request) (*firebaseauth.Token, bool)
}

type FirebaseTokenVerifierFunc func(r *http.Request) (*firebaseauth.Token, bool)

func (v FirebaseTokenVerifierFunc) Verify(r *http.Request) (*firebaseauth.Token, bool) {
	return v(r)
}

func FirebaseMiddleware(v FirebaseTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token, ok := v.Verify(r); ok {
				ctx := r.Context()
				auth, _ := FromContext(ctx)
				auth.PhoneNumber, _ = token.Claims["phone_number"].(string)
				auth.Email, _ = token.Claims["email"].(string)
				ctx = NewContext(ctx, auth)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func NewFirebaseAuthClient(firebaseProjectID string) (*firebaseauth.Client, error) {
	credentialsJSON := `{"type":"service_account","project_id":"` + firebaseProjectID + `"}`
	firebaseCredentials := option.WithCredentialsJSON([]byte(credentialsJSON))
	firebaseApp, err := firebase.NewApp(context.Background(), nil, firebaseCredentials)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating firebase app")
	}
	firebaseAuthClient, err := firebaseApp.Auth(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "instantiating firebase auth client")
	}
	return firebaseAuthClient, nil
}

type FirebaseTokenVerifierLive struct {
	AuthClient *firebaseauth.Client
}

func (v FirebaseTokenVerifierLive) Verify(r *http.Request) (*firebaseauth.Token, bool) {
	ctx := r.Context()
	authHeader := r.Header.Get("Authorization")
	tokenEncoded := httpauthz.ParseBearerToken(authHeader)
	if tokenEncoded == "" {
		return nil, false
	}
	token, err := v.AuthClient.VerifyIDToken(ctx, tokenEncoded)
	if err != nil {
		return nil, false
	}
	return token, true
}
