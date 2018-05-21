package requestid

import (
	"context"
	"testing"

	"github.com/go-chi/chi/middleware"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRequestId(t *testing.T) {
	Convey("requestid.Context", t, func() {
		ctx := Context(context.Background(), "2")
		So(ctx.Value(&key), ShouldEqual, "2")

		ctx2 := Context(ctx, "3")

		So(ctx.Value(&key), ShouldEqual, "2")
		So(ctx2.Value(&key), ShouldEqual, "3")
	})

	Convey("requestid.ContextFromCHI", t, func() {
		ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "foobar")

		ctx2 := ContextFromChi(ctx)
		So(FromContext(ctx2), ShouldEqual, "foobar")
	})

	Convey("requestid.FromContext", t, func() {
		ctx := Context(context.Background(), "2")
		ctx2 := Context(ctx, "3")

		So(FromContext(context.Background()), ShouldEqual, "")
		So(FromContext(ctx), ShouldEqual, "2")
		So(FromContext(ctx2), ShouldEqual, "3")
	})
}
