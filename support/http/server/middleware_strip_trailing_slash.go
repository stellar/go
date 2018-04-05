package server

import (
	"net/http"
	"strings"
)

// StripTrailingSlashMiddleware strips trailing slash.
func StripTrailingSlashMiddleware(ignoredPrefixes ...string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path

			// Do not change ignored prefixes
			for _, prefix := range ignoredPrefixes {
				if strings.HasPrefix(path, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			l := len(path)

			// if the path is longer than 1 char (i.e., not '/')
			// and has a trailing slash, remove it.
			if l > 1 && path[l-1] == '/' {
				r.URL.Path = path[0 : l-1]
			}

			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
