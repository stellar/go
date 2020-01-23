package healthjson

import (
	"net/http"

	"github.com/stellar/go/support/render/httpjson"
)

type PassHandler struct{}

func (h PassHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response := Response{
		Status: StatusPass,
	}
	httpjson.Render(w, response, httpjson.HEALTHJSON)
}
