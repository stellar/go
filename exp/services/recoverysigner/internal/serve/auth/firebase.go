package auth

import (
	"context"
	"net/http"

	firebase "firebase.google.com/go"
	firebaseauth "firebase.google.com/go/auth"
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

func NewFirebaseApp(firebaseProjectID string) (*firebase.App, error) {
	credentialsJSON := `{"type":"service_account","project_id":"` + firebaseProjectID + `"}`
	firebaseCredentials := option.WithCredentialsJSON([]byte(credentialsJSON))
	return firebase.NewApp(context.Background(), nil, firebaseCredentials)
}

type FirebaseTokenVerifierLive struct {
	App *firebase.App
}

func (v FirebaseTokenVerifierLive) Verify(r *http.Request) (*firebaseauth.Token, bool) {
	ctx := r.Context()
	client, err := v.App.Auth(ctx)
	if err != nil {
		return nil, false
	}
	authHeader := r.Header.Get("Authorization")
	tokenEncoded := httpauthz.ParseBearerToken(authHeader)
	if tokenEncoded == "" {
		return nil, false
	}
	token, err := client.VerifyIDToken(ctx, tokenEncoded)
	if err != nil {
		return nil, false
	}
	return token, true
}
