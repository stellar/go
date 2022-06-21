package actions

import (
	"net/http"
)

func ApiDocs() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Scheme = "http"
		r.URL.Host = "localhost:8080"

		if r.Method != "GET" {
			sendErrorResponse(w, http.StatusMethodNotAllowed, "")
			return
		}

		p, err := staticFiles.ReadFile("static/api_docs.yml")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/openapi+yaml")
		w.Write(p)
	}
}
