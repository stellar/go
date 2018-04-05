package server

import (
	"net/http"
	"strings"
)

// HeadersMiddleware sends headers
func HeadersMiddleware(headers http.Header, ignoredPrefixes ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Do not change ignored prefixes
			for _, prefix := range ignoredPrefixes {
				if strings.HasPrefix(r.URL.Path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			headers.Write(w)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
