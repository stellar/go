package federation

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func (rh *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestType := r.URL.Query().Get("type")
	q := r.URL.Query().Get("q")

	switch {
	case requestType == "name" && q != "":
		rh.FederationRequest(q, w)
	case requestType == "id" && q != "":
		rh.ReverseFederationRequest(q, w)
	case requestType == "txid" && q != "":
		rh.writeErrorResponse(w, ErrorResponseString("not_implemented", "txid requests are not supported"), http.StatusNotImplemented)
	default:
		rh.writeErrorResponse(w, ErrorResponseString("invalid_request", "Invalid request"), http.StatusBadRequest)
	}
}

func (rh *RequestHandler) ReverseFederationRequest(accountID string, w http.ResponseWriter) {
	response := Response{
		AccountID: accountID,
	}

	var record ReverseFederationRecord
	err := rh.db.GetRaw(&record, rh.config.Queries.ReverseFederation, accountID)

	switch {
	case rh.db.NoRows(err):
		log.Print("Federation record NOT found")
		rh.writeErrorResponse(w, ErrorResponseString("not_found", "Account not found"), http.StatusNotFound)
	case err != nil:
		log.Print("Server error: ", err)
		rh.writeErrorResponse(w, ErrorResponseString("server_error", "Server error"), http.StatusInternalServerError)
	default:
		response.StellarAddress = record.Name + "*" + rh.config.Domain
		rh.writeResponse(w, response)

	}
}

func (rh *RequestHandler) FederationRequest(stellarAddress string, w http.ResponseWriter) {
	var name, domain string

	if i := strings.Index(stellarAddress, "*"); i >= 0 {
		name = stellarAddress[:i]
		domain = stellarAddress[i+1:]
	}

	if name == "" || domain != rh.config.Domain {
		rh.writeErrorResponse(w, ErrorResponseString("not_found", "Incorrect Domain"), http.StatusNotFound)
		return
	}

	response := Response{
		StellarAddress: stellarAddress,
	}

	var record FederationRecord
	err := rh.db.GetRaw(&record, rh.config.Queries.Federation, name)

	switch {
	case rh.db.NoRows(err):
		log.Print("Federation record NOT found")
		rh.writeErrorResponse(w, ErrorResponseString("not_found", "Account not found"), http.StatusNotFound)
	case err != nil:
		log.Print("Server error: ", err)
		rh.writeErrorResponse(w, ErrorResponseString("server_error", "Server error"), http.StatusInternalServerError)
	default:
		response.AccountID = record.AccountID
		response.MemoType = record.MemoType
		response.Memo = record.Memo
		rh.writeResponse(w, response)

	}
}

func (rh *RequestHandler) writeResponse(w http.ResponseWriter, response Response) {
	log.Print("Federation record found")

	json, err := json.Marshal(response)

	if err != nil {
		rh.writeErrorResponse(w, ErrorResponseString("server_error", "Server error"), http.StatusInternalServerError)
		return
	}

	w.Write(json)
}

func (rh *RequestHandler) writeErrorResponse(w http.ResponseWriter, response string, errorCode int) {
	w.WriteHeader(errorCode)
	w.Write([]byte(response))
}

func ErrorResponseString(code string, message string) string {
	error := Error{Code: code, Message: message}
	json, _ := json.Marshal(error)
	return string(json)
}
