package context

type CtxKey string

var RequestContextKey = CtxKey("request")
var SessionContextKey = CtxKey("session")
