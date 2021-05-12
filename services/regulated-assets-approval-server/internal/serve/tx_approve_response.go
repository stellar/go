package serve

import (
	"net/http"

	"github.com/stellar/go/support/render/httpjson"
)

type txApprovalResponse struct {
	Error      string     `json:"error,omitempty"`
	Message    string     `json:"message,omitempty"`
	Status     sep8Status `json:"status"`
	StatusCode int        `json:"-"`
	Tx         string     `json:"tx,omitempty"`
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

func NewRevisedTxApprovalResponse(tx string) *txApprovalResponse {
	return &txApprovalResponse{
		Status:     sep8StatusRevised,
		Tx:         tx,
		StatusCode: http.StatusOK,
		Message:    "Authorization and deauthorization operations were added.",
	}
}

type sep8Status string

const (
	sep8StatusRejected sep8Status = "rejected"
	sep8StatusRevised  sep8Status = "revised"
)
