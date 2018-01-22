package fixtures

import (
	"fmt"
	"time"

	"github.com/stellar/go/services/bifrost/queue"
)

func Transaction() queue.Transaction {
	return queue.Transaction{
		TransactionID:    fmt.Sprintf("anyTx-%d", time.Now().UnixNano()),
		AssetCode:        "FOO",
		Amount:           "100",
		StellarPublicKey: "myStellarPublicKeyxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}
}
