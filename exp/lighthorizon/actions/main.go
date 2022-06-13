package actions

import (
	"encoding/json"
	"fmt"
	"github.com/stellar/go/exp/lighthorizon/index"
	"github.com/stellar/go/support/render/hal"
	"io"
	"net/http"
)

func sendPageResponse(w http.ResponseWriter, page hal.Page) {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(page)
	if err != nil {
		fmt.Fprintf(w, "Error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func sendErrorResponse(w http.ResponseWriter, errorCode int) {
	w.WriteHeader(errorCode)
}

func indexedCursorFromHash(hash [32]byte, indexStore index.Store) (int64, bool, error) {
	// Skip the cursor ahead to the next active checkpoint for this account
	cursor, err := indexStore.TransactionTOID(hash)
	if err == io.EOF {
		// never active. No results.
		return 0, true, nil
	} else if err != nil {
		return 0, false, err
	}
	return cursor, false, nil
}
