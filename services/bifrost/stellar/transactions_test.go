package stellar

import (
	"encoding/json"
	"testing"

	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/services/bifrost/common"
	"github.com/stellar/go/support/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeleteArchivedXDROnHorizonErrors(t *testing.T) {
	horizonMock := &horizon.MockClient{}
	archiver := &MockArchiver{}
	ac := &AccountConfigurator{
		log:               common.CreateLogger("test account configurer"),
		SubmissionArchive: archiver,
		Horizon:           horizonMock,
	}
	myError := &horizon.Error{Problem: horizon.Problem{Extras: map[string]json.RawMessage{}}}
	horizonMock.Mock.
		On("LoadAccount", mock.Anything).
		Return(horizon.Account{}, nil). // for sequence
		On("SubmitTransaction", "myXDRContent").
		Return(horizon.TransactionSuccess{}, myError)

	archiver.On("Delete", "myTxID", "myAssetCode", SubmissionTypeSendTokens).
		Return(nil)

	// when
	err := ac.submitArchivedXDR("myTxID", "myAssetCode", SubmissionTypeSendTokens, "myXDRContent")

	// then
	require.Error(t, err)
	assert.Equal(t, myError, errors.Cause(err))
	horizonMock.Mock.AssertExpectations(t)
	archiver.Mock.AssertExpectations(t)

}
