package actions

import (
	supportProblem "github.com/stellar/go/support/render/problem"
	"net/http"
)

func ApiDocs() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Scheme = "http"
		r.URL.Host = "localhost:8080"

		if r.Method != "GET" {
			sendErrorResponse(r.Context(), w, supportProblem.BadRequest)
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
