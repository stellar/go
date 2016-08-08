package federation

import (
	sdb "github.com/stellar/go/internal/db"
)

// App is probably redunant with RequestHandler. ;)
type App struct {
	config Config
	db     *sdb.Repo
}

// Config represents the configuration of a federation server
type Config struct {
	Port     int
	Domain   string
	Database struct {
		Type string
		URL  string
	}
	Queries struct {
		Federation        string
		ReverseFederation string `mapstructure:"reverse-federation"`
	}
	TLS struct {
		CertificateFile string `mapstructure:"certificate-file"`
		PrivateKeyFile  string `mapstructure:"private-key-file"`
	}
}

// Error represents the JSON response sent to a client when the request
// triggered an error.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// FederationRecord represents the result from the database when performing a
// federation request.
type FederationRecord struct {
	AccountID string `db:"id"`
	MemoType  string `db:"memo_type"`
	Memo      string `db:"memo"`
}

// RequestHandler is the implementation of http.Handler that provides the
// federation protocol.
type RequestHandler struct {
	config *Config
	db     *sdb.Repo
}

// Response represents the successful JSON response that will be delivered to a
// client.
type Response struct {
	StellarAddress string `json:"stellar_address"`
	AccountID      string `json:"account_id"`
	MemoType       string `json:"memo_type,omitempty"`
	Memo           string `json:"memo,omitempty"`
}

// ReverseFederation represents the result from the database when performing a
// reverse federation request.
type ReverseFederationRecord struct {
	Name string `db:"name"`
}
