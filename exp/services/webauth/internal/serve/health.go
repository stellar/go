package serve

import (
	"net/http"

	"github.com/stellar/go/support/render/httpjson"
)

type healthHandler struct{}

type healthResponse struct {
	OK bool `json:"ok"`
}

func (h healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	res := healthResponse{
		OK: true,
	}
	httpjson.Render(w, res, httpjson.JSON)
}
