package context

import (
	"context"

	"github.com/stellar/go/support/db"
)

type CtxKey string

var AppContextKey = CtxKey("app")
var RequestContextKey = CtxKey("request")
var ClientContextKey = CtxKey("client")
var horizonSessionContextKey = CtxKey("horizon-session")

// HorizonSessionForContext returns a horizon session for the given context
func HorizonSessionForContext(ctx context.Context) *db.Session {
	session, _ := ctx.Value(horizonSessionContextKey).(*db.Session)

	return session
}

// AddHorizonSessionToContext extends the given context with a horizon session
func AddHorizonSessionToContext(ctx context.Context, session *db.Session) context.Context {
	return context.WithValue(ctx, horizonSessionContextKey, session)
}
