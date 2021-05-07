package serve

import (
	"net/http"

	"github.com/stellar/go/support/render/httpjson"
)

type txApprovalResponse struct {
	Status       sep8Status `json:"status"`
	Tx           string     `json:"tx,omitempty"`
	Message      string     `json:"message,omitempty"`
	Error        string     `json:"error,omitempty"`
	StatusCode   int        `json:"-"`
	ActionURL    string     `json:"action_url,omitempty"`
	ActionMethod string     `json:"action_method,omitempty"`
	ActionFields []string   `json:"action_fields,omitempty"`
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

func NewActionRequiredTxApprovalResponse(message, actionURL string, actionFields []string) *txApprovalResponse {
	return &txApprovalResponse{
		Status:       sep8StatusActionRequired,
		Message:      message,
		ActionMethod: "POST",
		StatusCode:   http.StatusOK,
		ActionURL:    actionURL,
		ActionFields: actionFields,
	}
}

func NewRevisedTxApprovalResponse(tx string) *txApprovalResponse {
	return &txApprovalResponse{
		Status:     sep8StatusRevised,
		Tx:         tx,
		StatusCode: http.StatusOK,
	}
}

type sep8Status string

const (
	sep8StatusActionRequired sep8Status = "action_required"
	sep8StatusPending        sep8Status = "pending"
	sep8StatusRejected       sep8Status = "rejected"
	sep8StatusRevised        sep8Status = "revised"
	sep8StatusSuccess        sep8Status = "success"
)

func (k sep8Status) String() string {
	extensions := [...]string{"action_required", "pending", "rejected", "revised", "success"}

	sep8StatusStr := string(k)
	for _, v := range extensions {
		if v == sep8StatusStr {
			return sep8StatusStr
		}
	}

	return ""
}
