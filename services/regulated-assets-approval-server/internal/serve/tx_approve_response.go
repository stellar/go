package serve

import (
	"net/http"

	"github.com/stellar/go/support/render/httpjson"
)

type txApprovalResponse struct {
	Status     sep8Status `json:"status"`
	Error      string     `json:"error,omitempty"`
	StatusCode int        `json:"-"`
}

func (t *txApprovalResponse) Render(w http.ResponseWriter) {
	httpjson.RenderStatus(w, t.StatusCode, t, httpjson.JSON)
}

func NewRejectedTxApprovalResponse(errMessage string) *txApprovalResponse {
	return &txApprovalResponse{
		Status:     sep8StatusRejected,
		Error:      errMessage,
		StatusCode: http.StatusBadRequest,
	}
}

type sep8Status string

const (
	sep8StatusRejected sep8Status = "rejected"
)
