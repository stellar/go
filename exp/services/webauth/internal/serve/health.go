package serve

import (
	"net/http"

	"github.com/stellar/go/support/render/httpjson"
)

type healthHandler struct{}

type healthResponse struct {
	Status string `json:"status"`
}

func (h healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := healthResponse{
		Status: "pass",
	}
	httpjson.Render(w, res, httpjson.HEALTHJSON)
}
