package serve

import (
	"fmt"
	"net/http"

	"github.com/stellar/go/keypair"
)

func stellarTOMLHandler(cfg Options) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		issuerKP, err := keypair.Parse(cfg.IssuerAccountSecret)
		if err != nil {
			return
		}
		approvalCriteria := fmt.Sprintf("Currently %q is not approving any %q transactions", r.Host, cfg.AssetCode)
		approvalServer := fmt.Sprintf("%s/tx_approve", r.Host)
		fmt.Fprintf(rw, "NETWORK_PASSPHRASE=%q\n", cfg.NetworkPassphrase)
		fmt.Fprintf(rw, "[[CURRENCIES]]\n")
		fmt.Fprintf(rw, "code=%q\n", cfg.AssetCode)
		fmt.Fprintf(rw, "issuer=%q\n", issuerKP.Address())
		fmt.Fprintf(rw, "regulated=true\n")
		fmt.Fprintf(rw, "approval_server=%q\n", approvalServer)
		fmt.Fprintf(rw, "approval_criteria=%q\n", approvalCriteria)
	})
}
