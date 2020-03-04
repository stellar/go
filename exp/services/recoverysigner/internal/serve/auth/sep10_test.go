package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSEP10_addsAddressToClaimIfJWTValid(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ctx := context.Context(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})
	middleware := SEP10Middleware(&k.PublicKey)
	handler := middleware(next)

	r := httptest.NewRequest("GET", "/", nil)
	jwtClaims := jwt.MapClaims{
		"sub": "GDKABHI4LTLG7UCE6O7Y4D6REHJVS4DLXTVVXTE3BPRRLXPASHSOKG2D",
	}
	jwtToken, err := jwt.NewWithClaims(jwt.SigningMethodES256, jwtClaims).SignedString(k)
	require.NoError(t, err)
	r.Header.Set("Authorization", "Bearer "+jwtToken)
	handler.ServeHTTP(nil, r)

	assert.NotNil(t, ctx)
	claims, ok := FromContext(ctx)
	assert.Equal(t, true, ok)

	wantClaims := Auth{
		Address: "GDKABHI4LTLG7UCE6O7Y4D6REHJVS4DLXTVVXTE3BPRRLXPASHSOKG2D",
	}
	assert.Equal(t, wantClaims, claims)
}

func TestSEP10_doesNotAddAddressToClaimIfJWTNotPresent(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ctx := context.Context(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})
	middleware := SEP10Middleware(&k.PublicKey)
	handler := middleware(next)

	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(nil, r)

	assert.NotNil(t, ctx)
	claims, ok := FromContext(ctx)
	assert.Equal(t, false, ok)

	wantClaims := Auth{}
	assert.Equal(t, wantClaims, claims)
}

func TestSEP10_doesNotAddAddressToClaimIfJWTNoSignature(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ctx := context.Context(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})
	middleware := SEP10Middleware(&k.PublicKey)
	handler := middleware(next)

	r := httptest.NewRequest("GET", "/", nil)
	jwtClaims := jwt.MapClaims{
		"sub": "GDKABHI4LTLG7UCE6O7Y4D6REHJVS4DLXTVVXTE3BPRRLXPASHSOKG2D",
	}
	jwtToken, err := jwt.NewWithClaims(jwt.SigningMethodES256, jwtClaims).SigningString()
	require.NoError(t, err)
	r.Header.Set("Authorization", "Bearer "+jwtToken)
	handler.ServeHTTP(nil, r)

	assert.NotNil(t, ctx)
	claims, ok := FromContext(ctx)
	assert.Equal(t, false, ok)

	wantClaims := Auth{}
	assert.Equal(t, wantClaims, claims)
}

func TestSEP10_doesNotAddAddressToClaimIfJWTWrongAlg(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ctx := context.Context(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})
	middleware := SEP10Middleware(&k.PublicKey)
	handler := middleware(next)

	r := httptest.NewRequest("GET", "/", nil)
	jwtClaims := jwt.MapClaims{
		"sub": "GDKABHI4LTLG7UCE6O7Y4D6REHJVS4DLXTVVXTE3BPRRLXPASHSOKG2D",
	}
	jwtToken, err := jwt.NewWithClaims(jwt.SigningMethodNone, jwtClaims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)
	r.Header.Set("Authorization", "Bearer "+jwtToken)
	handler.ServeHTTP(nil, r)

	assert.NotNil(t, ctx)
	claims, ok := FromContext(ctx)
	assert.Equal(t, false, ok)

	wantClaims := Auth{}
	assert.Equal(t, wantClaims, claims)
}

func TestSEP10_doesNotAddAddressToClaimIfJWTInvalidSignature(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	k2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ctx := context.Context(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})
	middleware := SEP10Middleware(&k.PublicKey)
	handler := middleware(next)

	r := httptest.NewRequest("GET", "/", nil)
	jwtClaims := jwt.MapClaims{
		"sub": "GDKABHI4LTLG7UCE6O7Y4D6REHJVS4DLXTVVXTE3BPRRLXPASHSOKG2D",
	}
	jwtToken, err := jwt.NewWithClaims(jwt.SigningMethodES256, jwtClaims).SignedString(k2)
	require.NoError(t, err)
	r.Header.Set("Authorization", "Bearer "+jwtToken)
	handler.ServeHTTP(nil, r)

	assert.NotNil(t, ctx)
	claims, ok := FromContext(ctx)
	assert.Equal(t, false, ok)

	wantClaims := Auth{}
	assert.Equal(t, wantClaims, claims)
}

func TestSEP10_doesNotAddAddressToClaimIfJWTExpired(t *testing.T) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	ctx := context.Context(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})
	middleware := SEP10Middleware(&k.PublicKey)
	handler := middleware(next)

	r := httptest.NewRequest("GET", "/", nil)
	jwtClaims := jwt.MapClaims{
		"sub": "GDKABHI4LTLG7UCE6O7Y4D6REHJVS4DLXTVVXTE3BPRRLXPASHSOKG2D",
		"exp": 1,
	}
	jwtToken, err := jwt.NewWithClaims(jwt.SigningMethodES256, jwtClaims).SignedString(k)
	require.NoError(t, err)
	r.Header.Set("Authorization", "Bearer "+jwtToken)
	handler.ServeHTTP(nil, r)

	assert.NotNil(t, ctx)
	claims, ok := FromContext(ctx)
	assert.Equal(t, false, ok)

	wantClaims := Auth{}
	assert.Equal(t, wantClaims, claims)
}
