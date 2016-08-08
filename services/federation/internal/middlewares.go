package federation

import (
	"net/http"
)

func jsonMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
