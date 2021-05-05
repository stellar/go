package serve

import (
	"fmt"
	"net/http"
)

type stellarTOMLHandler struct {
	assetCode         string
	approvalServer    string
	issuerAddress     string
	networkPassphrase string
	kycThreshold      float64
}

func (h stellarTOMLHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(rw, "NETWORK_PASSPHRASE=%q\n", h.networkPassphrase)
	fmt.Fprintf(rw, "[[CURRENCIES]]\n")
	fmt.Fprintf(rw, "code=%q\n", h.assetCode)
	fmt.Fprintf(rw, "issuer=%q\n", h.issuerAddress)
	fmt.Fprintf(rw, "regulated=true\n")
	fmt.Fprintf(rw, "approval_server=%q\n", h.approvalServer)
	fmt.Fprintf(rw, `approval_criteria="The approval server currently only accepts payments and the transaction must have exactly one operation and it must be of type payment. If an account is registered in the internal list of unauthorized addresses, all payments to/from that account will be rejected. Also, if the payment amount exceeds %.2f %s it will need KYC approval."`, h.kycThreshold, h.assetCode)
}
