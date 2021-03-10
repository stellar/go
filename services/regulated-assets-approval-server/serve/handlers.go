package serve

import (
	"fmt"
	"net/http"
)

func stellarTOMLHandler(cfg Options) http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(rw, "NETWORK_PASSPHRASE=%q\n", cfg.NetworkPassphrase)
	})
}
