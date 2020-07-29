package actions

import (
	"io"
	"net/http"

	"github.com/stellar/go/services/horizon/internal/context"
	"github.com/stellar/go/services/horizon/internal/db2/history"
)

// AccountDataQuery query struct for account data end-point
type AccountDataQuery struct {
	AccountID string `schema:"account_id" valid:"accountID"`
	Key       string `schema:"key" valid:"length(1|64)"`
}

type accountDataResponse struct {
	Value string `json:"value"`
}

func (adr accountDataResponse) Equals(other StreamableObjectResponse) bool {
	other, ok := other.(accountDataResponse)
	if !ok {
		return false
	}
	return adr == other
}

type GetAccountDataHandler struct{}

func (handler GetAccountDataHandler) GetResource(w HeaderWriter, r *http.Request) (StreamableObjectResponse, error) {
	data, err := loadAccountData(r)
	if err != nil {
		return nil, err
	}
	response := accountDataResponse{data.Value.Base64()}
	return response, nil
}

func (handler GetAccountDataHandler) WriteRawResponse(w io.Writer, r *http.Request) error {
	data, err := loadAccountData(r)
	if err != nil {
		return err
	}
	_, err = w.Write(data.Value)
	return err
}

func loadAccountData(r *http.Request) (history.Data, error) {
	qp := AccountDataQuery{}
	err := getParams(&qp, r)
	if err != nil {
		return history.Data{}, err
	}
	historyQ, err := context.HistoryQFromRequest(r)
	if err != nil {
		return history.Data{}, err
	}
	data, err := historyQ.GetAccountDataByName(qp.AccountID, qp.Key)
	if err != nil {
		return history.Data{}, err
	}
	return data, nil
}
