package requestid

import (
	"context"
	"testing"

	"github.com/go-chi/chi/middleware"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestRequestContext(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	ctx := Context(context.Background(), "2")
	tt.Assert.Equal(ctx.Value(&key), "2")

	ctx2 := Context(ctx, "3")
	tt.Assert.Equal(ctx2.Value(&key), "3")
	tt.Assert.Equal(ctx.Value(&key), "2")
}

func TestRequestContextFromCHI(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "foobar")
	ctx2 := ContextFromChi(ctx)
	tt.Assert.Equal(FromContext(ctx2), "foobar")
}

func TestRequestFromContext(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()

	ctx := Context(context.Background(), "2")
	ctx2 := Context(ctx, "3")
	tt.Assert.Equal(FromContext(context.Background()), "")
	tt.Assert.Equal(FromContext(ctx), "2")
	tt.Assert.Equal(FromContext(ctx2), "3")
}
