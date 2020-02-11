package auth

import (
	"context"
	"net/http"
	"strings"

	firebase "firebase.google.com/go"
	firebaseauth "firebase.google.com/go/auth"
	"google.golang.org/api/option"
)

func Firebase(app *firebase.App) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token, ok := firebaseTokenFromRequest(r, app); ok {
				ctx := r.Context()
				claims, _ := FromContext(ctx)
				claims.PhoneNumber = token.Claims["phone_number"].(string)
				claims.Email = token.Claims["email"].(string)
				ctx = NewContext(ctx, claims)
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

func firebaseTokenFromRequest(r *http.Request, app *firebase.App) (*firebaseauth.Token, bool) {
	ctx := r.Context()
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, false
	}
	authHeader := r.Header.Get("Authorization")
	tokenEncoded := strings.TrimPrefix(authHeader, "BEARER ")
	token, err := client.VerifyIDToken(ctx, tokenEncoded)
	if err != nil {
		return nil, false
	}
	return token, true
}
