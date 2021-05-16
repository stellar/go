package kycstatus

type postRequest struct {
	CallbackID   string
	EmailAddress string `json:"email_address"`
}

type postResponse struct {
	Result  string `json:"result"`
	Message string `json:"message"`
}

func NewApprovedKYCStatusPostResponse() *postResponse {
	return &postResponse{
		Result:  "no_further_action_required",
		Message: "Your KYC has been approved!",
	}
}

func NewRejectedKYCStatusPostResponse() *postResponse {
	return &postResponse{
		Result:  "no_further_action_required",
		Message: "Your KYC has been rejected!",
	}
}
