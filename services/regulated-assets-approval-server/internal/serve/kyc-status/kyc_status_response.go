package kycstatus

import (
	"net/http"

	"github.com/stellar/go/support/render/httpjson"
)

type kycPostRequest struct {
	CallbackID   string
	EmailAddress string `json:"email_address"`
}

type kycPostResponse struct {
	Result     string `json:"result"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
}

func (k *kycPostResponse) Render(w http.ResponseWriter) {
	httpjson.RenderStatus(w, k.StatusCode, k, httpjson.JSON)
}

func NewApprovedKYCStatusPostResponse() *kycPostResponse {
	return &kycPostResponse{
		Result:     "no_further_action_required",
		Message:    "Your KYC has been approved!",
		StatusCode: http.StatusOK,
	}
}

func NewRejectedKYCStatusPostResponse() *kycPostResponse {
	return &kycPostResponse{
		Result:     "no_further_action_required",
		Message:    "Your KYC has been rejected!",
		StatusCode: http.StatusOK,
	}
}
