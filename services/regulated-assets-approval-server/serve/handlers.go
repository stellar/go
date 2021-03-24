package serve

import (
	"fmt"
	"net/http"

	"github.com/stellar/go/keypair"
)

func stellarTOMLHandler(opts Options) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		issuerKP, err := keypair.Parse(opts.IssuerAccountSecret)
		if err != nil {
			return
		}
		approvalCriteria := fmt.Sprintf("Currently %q is not approving any %q transactions", r.Host, opts.AssetCode)
		approvalServer := fmt.Sprintf("%s/tx_approve", r.Host)
		fmt.Fprintf(rw, "NETWORK_PASSPHRASE=%q\n", opts.NetworkPassphrase)
		fmt.Fprintf(rw, "[[CURRENCIES]]\n")
		fmt.Fprintf(rw, "code=%q\n", opts.AssetCode)
		fmt.Fprintf(rw, "issuer=%q\n", issuerKP.Address())
		fmt.Fprintf(rw, "regulated=true\n")
		fmt.Fprintf(rw, "approval_server=%q\n", approvalServer)
		fmt.Fprintf(rw, "approval_criteria=%q\n", approvalCriteria)
	})
}
