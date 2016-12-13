package compliance

// Strategy defines strategy for handling auth requests.
// The SanctionsCheck and GetUserData functions will be called in the
// order above. Both methods can set `Pending` field so make sure
// the final value is the max of two.
type Strategy interface {
	// SanctionsCheck performs AML sanctions check of the sender.
	SanctionsCheck(data AuthData, response *AuthResponse) error
	// GetUserData check if user data is required and if so decides
	// whether to allow access to customer data or not.
	GetUserData(data AuthData, response *AuthResponse) error
	// PersistTransaction save authorized transaction to persistent storage so
	// memo preimage can be fetched when a transaction is sent.
	// PersistTransaction(data AuthData) error
}

// CallbackStrategy sends requests to given callbacks to decide
// whether to allow incoming transaction.
// If SanctionsCheckURL is empty it will allow every transaction.
// If GetUserDataURL is empty it will deny access to user data for each request.
type CallbackStrategy struct {
	// SanctionsCheckURL callback should respond with one of the following
	// status codes:
	//   * `200 OK` when sender/receiver is allowed and the payment should be processed,
	//   * `202 Accepted` when your callback needs some time for processing,
	//   * `403 Forbidden` when sender/receiver is denied.
	// Any other status code will be considered an error.
	//
	// When `202 Accepted` is returned the response body should contain JSON object
	// with a pending field which represents the estimated number of seconds needed
	// for processing. For example, the following response means to try the payment
	// again in an hour:
	//
	//   {"pending": 3600}
	SanctionsCheckURL string
	// GetUserDataURL callback should respond with one of the following
	// status codes:
	//   * `200 OK` when you allow to share recipient data.
	//   * `202 Accepted` when your callback needs some time for processing,
	//   * `403 Forbidden` when you deny to share recipient data.
	// Any other status code will be considered an error.
	//
	// When `200 OK` is returned the response body should contain JSON object with
	// your customer data. The customer information that is exchanged between FIs is
	// flexible but the typical fields are:
	//
	//   * Full Name
	//   * Date of birth
	//   * Physical address
	//
	// When `202 Accepted` is returned the response body should contain JSON object
	// with a pending field which represents the estimated number of seconds needed
	// for processing. For example, the following response means to try the payment
	// again in an hour:
	//
	//   {"pending": 3600}
	GetUserDataURL string
}

// AuthHandler ...
type AuthHandler struct {
	Strategy Strategy
	// PersistTransaction save authorized transaction to persistent storage so
	// memo preimage can be fetched when a transaction is sent.
	PersistTransaction func(data AuthData) error
}

// AuthStatus represents auth status returned by Auth Server
type AuthStatus string

const (
	// AuthStatusOk is returned when authentication was successful
	AuthStatusOk AuthStatus = "ok"
	// AuthStatusPending is returned when authentication is pending
	AuthStatusPending AuthStatus = "pending"
	// AuthStatusDenied is returned when authentication was denied
	AuthStatusDenied AuthStatus = "denied"
)

// AuthRequest represents auth request sent to compliance server
type AuthRequest struct {
	// Marshalled AuthData JSON object (because of the attached signature, json can be marshalled to infinite number of valid JSON strings)
	DataJSON string `name:"data" required:""`
	// Signature of sending FI
	Signature string `name:"sig" required:""`
}

// AuthData represents how AuthRequest.Data field looks like.
type AuthData struct {
	// The stellar address of the customer that is initiating the send.
	Sender string `json:"sender"`
	// If the caller needs the recipient's AML info in order to send the payment.
	NeedInfo bool `json:"need_info"`
	// The transaction that the sender would like to send in XDR format. This transaction is unsigned.
	Tx string `json:"tx"`
	// The full text of the memo the hash of this memo is included in the transaction.
	MemoJSON string `json:"memo"`
}

// AuthResponse represents response sent by auth server
type AuthResponse struct {
	// If this FI is willing to share AML information or not. {ok, denied, pending}
	InfoStatus AuthStatus `json:"info_status"`
	// If this FI is willing to accept this transaction. {ok, denied, pending}
	TxStatus AuthStatus `json:"tx_status"`
	// (only present if info_status is ok) JSON of the recipient's AML information. in the Stellar memo convention
	DestInfo string `json:"dest_info,omitempty"`
	// (only present if info_status or tx_status is pending) Estimated number of seconds till the sender can check back for a change in status. The sender should just resubmit this request after the given number of seconds.
	Pending int `json:"pending,omitempty"`
}

// Memo represents memo in Stellar memo convention
type Memo struct {
	Transaction `json:"transaction"`
	Operations  []Operation `json:"operations"`
}

// Transaction represents transaction field in Stellar memo
type Transaction struct {
	SenderInfo string `json:"sender_info"`
	Route      string `json:"route"`
	Extra      string `json:"extra"`
	Note       string `json:"note"`
}

// Operation represents a single operation object in Stellar memo
type Operation struct {
	// Overriddes Transaction field for this operation
	SenderInfo string `json:"sender_info"`
	// Overriddes Transaction field for this operation
	Route string `json:"route"`
	// Overriddes Transaction field for this operation
	Note string `json:"note"`
}
