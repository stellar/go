//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package txsub

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

type SystemTestSuite struct {
	suite.Suite
	ctx           context.Context
	submitter     *MockSubmitter
	db            *mockDBQ
	system        *System
	noResults     Result
	successTx     Result
	successXDR    xdr.TransactionEnvelope
	badSeq        SubmissionResult
	unmuxedSource xdr.AccountId
}

func (suite *SystemTestSuite) SetupTest() {
	suite.ctx = test.Context()
	suite.submitter = &MockSubmitter{}
	suite.db = &mockDBQ{}

	suite.system = &System{
		Submitter: suite.submitter,
		DB: func(ctx context.Context) HorizonDB {
			return suite.db
		},
	}

	suite.unmuxedSource = xdr.MustAddress("GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H")
	source := xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      0xcafebabe,
			Ed25519: *suite.unmuxedSource.Ed25519,
		},
	}

	tx := xdr.TransactionEnvelope{
		Type: xdr.EnvelopeTypeEnvelopeTypeTx,
		V1: &xdr.TransactionV1Envelope{
			Tx: xdr.Transaction{
				SourceAccount: source,
				Fee:           100,
				SeqNum:        1,
				Operations: []xdr.Operation{
					{
						Body: xdr.OperationBody{
							CreateAccountOp: &xdr.CreateAccountOp{
								Destination:     xdr.MustAddress("GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"),
								StartingBalance: 1000000000,
							},
						},
					},
				},
			},
			Signatures: []xdr.DecoratedSignature{
				{
					Hint:      xdr.SignatureHint{86, 252, 5, 247},
					Signature: xdr.Signature{131, 206, 171, 228, 64, 20, 40, 52, 2, 98, 124, 244, 87, 14, 130, 225, 190, 220, 156, 79, 121, 69, 60, 36, 57, 214, 9, 29, 176, 81, 218, 4, 213, 176, 211, 148, 191, 86, 21, 180, 94, 9, 43, 208, 32, 79, 19, 131, 90, 21, 93, 138, 153, 203, 55, 103, 2, 230, 137, 190, 19, 70, 179, 11},
				},
			},
		},
	}

	result := xdr.TransactionResult{
		FeeCharged: 123,
		Result: xdr.TransactionResultResult{
			Code:    xdr.TransactionResultCodeTxSuccess,
			Results: &[]xdr.OperationResult{},
		},
	}
	resultXDR, err := xdr.MarshalBase64(result)
	suite.Assert().NoError(err)

	suite.noResults = Result{Err: ErrNoResults}
	envelopeBase64, _ := xdr.MarshalBase64(tx)
	suite.successTx = Result{
		Transaction: history.Transaction{
			TransactionWithoutLedger: history.TransactionWithoutLedger{
				TransactionHash: "2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d",
				LedgerSequence:  2,
				TxEnvelope:      envelopeBase64,
				TxResult:        resultXDR,
			},
		},
	}
	assert.NoError(suite.T(), xdr.SafeUnmarshalBase64(suite.successTx.Transaction.TxEnvelope, &suite.successXDR))

	suite.badSeq = SubmissionResult{
		Err: ErrBadSequence,
	}

	suite.db.On("GetLatestHistoryLedger").Return(uint32(1000), nil).Maybe()
}

func (suite *SystemTestSuite) TearDownTest() {
	t := suite.T()
	suite.db.AssertExpectations(t)
}

// Returns the result provided by the ResultProvider.
func (suite *SystemTestSuite) TestSubmit_Basic() {
	suite.db.On("BeginTx", mock.AnythingOfType("*context.valueCtx"), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}).Return(nil).Once()
	suite.db.On("Rollback").Return(nil).Once()
	suite.db.On("PreFilteredTransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Run(func(args mock.Arguments) {
			ptr := args.Get(1).(*history.Transaction)
			*ptr = suite.successTx.Transaction
		}).
		Return(nil).Once()

	r := <-suite.system.Submit(
		suite.ctx,
		suite.successTx.Transaction.TxEnvelope,
		suite.successXDR,
		suite.successTx.Transaction.TransactionHash,
	)

	assert.Nil(suite.T(), r.Err)
	assert.Equal(suite.T(), suite.successTx, r)
	assert.False(suite.T(), suite.submitter.WasSubmittedTo)
}

func (suite *SystemTestSuite) TestTimeoutDuringSequnceLoop() {
	var cancel context.CancelFunc
	suite.ctx, cancel = context.WithTimeout(suite.ctx, time.Duration(0))
	defer cancel()

	suite.submitter.R = suite.badSeq
	suite.db.On("BeginTx", mock.AnythingOfType("*context.timerCtx"), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}).Return(nil).Once()
	suite.db.On("Rollback").Return(nil).Once()
	suite.db.On("PreFilteredTransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Once()
	suite.db.On("TransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Once()
	suite.db.On("NoRows", sql.ErrNoRows).Return(true).Twice()
	suite.db.On("GetSequenceNumbers", suite.ctx, []string{suite.unmuxedSource.Address()}).
		Return(map[string]uint64{suite.unmuxedSource.Address(): 0}, nil)

	r := <-suite.system.Submit(
		suite.ctx,
		suite.successTx.Transaction.TxEnvelope,
		suite.successXDR,
		suite.successTx.Transaction.TransactionHash,
	)

	assert.NotNil(suite.T(), r.Err)
	assert.Equal(suite.T(), ErrTimeout, r.Err)
}

func (suite *SystemTestSuite) TestClientDisconnectedDuringSequnceLoop() {
	var cancel context.CancelFunc
	suite.ctx, cancel = context.WithCancel(suite.ctx)

	suite.submitter.R = suite.badSeq
	suite.db.On("BeginTx", mock.AnythingOfType("*context.cancelCtx"), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}).Return(nil).Once()
	suite.db.On("Rollback").Return(nil).Once()
	suite.db.On("PreFilteredTransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Once()
	suite.db.On("TransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Once()
	suite.db.On("NoRows", sql.ErrNoRows).Return(true).Twice()
	suite.db.On("GetSequenceNumbers", suite.ctx, []string{suite.unmuxedSource.Address()}).
		Return(map[string]uint64{suite.unmuxedSource.Address(): 0}, nil).
		Run(func(args mock.Arguments) {
			// simulate client disconnecting while looping on sequnce number check
			cancel()
			suite.ctx.Deadline()
		}).
		Once()
	suite.db.On("GetSequenceNumbers", suite.ctx, []string{suite.unmuxedSource.Address()}).
		Return(map[string]uint64{suite.unmuxedSource.Address(): 0}, nil)

	r := <-suite.system.Submit(
		suite.ctx,
		suite.successTx.Transaction.TxEnvelope,
		suite.successXDR,
		suite.successTx.Transaction.TransactionHash,
	)

	assert.NotNil(suite.T(), r.Err)
	assert.Equal(suite.T(), ErrCanceled, r.Err)
}

func getMetricValue(metric prometheus.Metric) *dto.Metric {
	value := &dto.Metric{}
	err := metric.Write(value)
	if err != nil {
		panic(err)
	}
	return value
}

// Returns the error from submission if no result is found by hash and the suite.submitter returns an error.
func (suite *SystemTestSuite) TestSubmit_NotFoundError() {
	suite.db.On("BeginTx", mock.AnythingOfType("*context.valueCtx"), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}).Return(nil).Once()
	suite.db.On("Rollback").Return(nil).Once()
	suite.db.On("PreFilteredTransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Once()
	suite.db.On("TransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Once()
	suite.db.On("NoRows", sql.ErrNoRows).Return(true).Twice()
	suite.db.On("GetSequenceNumbers", suite.ctx, []string{suite.unmuxedSource.Address()}).
		Return(map[string]uint64{suite.unmuxedSource.Address(): 0}, nil).
		Once()

	suite.submitter.R.Err = errors.New("busted for some reason")
	r := <-suite.system.Submit(
		suite.ctx,
		suite.successTx.Transaction.TxEnvelope,
		suite.successXDR,
		suite.successTx.Transaction.TransactionHash,
	)

	assert.NotNil(suite.T(), r.Err)
	assert.True(suite.T(), suite.submitter.WasSubmittedTo)
	assert.Equal(suite.T(), float64(0), getMetricValue(suite.system.Metrics.SuccessfulSubmissionsCounter).GetCounter().GetValue())
	assert.Equal(suite.T(), float64(1), getMetricValue(suite.system.Metrics.FailedSubmissionsCounter).GetCounter().GetValue())
	assert.Equal(suite.T(), uint64(1), getMetricValue(suite.system.Metrics.SubmissionDuration).GetSummary().GetSampleCount())
}

// If the error is bad_seq and the result at the transaction's sequence number is for the same hash, return result.
func (suite *SystemTestSuite) TestSubmit_BadSeq() {
	suite.submitter.R = suite.badSeq
	suite.db.On("BeginTx", mock.AnythingOfType("*context.valueCtx"), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}).Return(nil).Once()
	suite.db.On("Rollback").Return(nil).Once()
	suite.db.On("NoRows", sql.ErrNoRows).Return(true).Once()
	suite.db.On("GetSequenceNumbers", suite.ctx, []string{suite.unmuxedSource.Address()}).
		Return(map[string]uint64{suite.unmuxedSource.Address(): 0}, nil).
		Once()
	suite.db.On("GetSequenceNumbers", suite.ctx, []string{suite.unmuxedSource.Address()}).
		Return(map[string]uint64{suite.unmuxedSource.Address(): 1}, nil).
		Once()
	suite.db.On("PreFilteredTransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Twice()
	suite.db.On("NoRows", sql.ErrNoRows).Return(true).Once()
	suite.db.On("TransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Once()
	suite.db.On("NoRows", sql.ErrNoRows).Return(true).Once()
	suite.db.On("TransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Run(func(args mock.Arguments) {
			ptr := args.Get(1).(*history.Transaction)
			*ptr = suite.successTx.Transaction
		}).
		Return(nil).Once()

	r := <-suite.system.Submit(
		suite.ctx,
		suite.successTx.Transaction.TxEnvelope,
		suite.successXDR,
		suite.successTx.Transaction.TransactionHash,
	)

	assert.Nil(suite.T(), r.Err)
	assert.Equal(suite.T(), suite.successTx, r)
	assert.True(suite.T(), suite.submitter.WasSubmittedTo)
}

// If error is bad_seq and no result is found, return error.
func (suite *SystemTestSuite) TestSubmit_BadSeqNotFound() {
	suite.submitter.R = suite.badSeq
	suite.db.On("BeginTx", mock.AnythingOfType("*context.valueCtx"), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}).Return(nil).Once()
	suite.db.On("Rollback").Return(nil).Once()
	suite.db.On("PreFilteredTransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Twice()
	suite.db.On("NoRows", sql.ErrNoRows).Return(true).Twice()
	suite.db.On("TransactionByHash", suite.ctx, mock.Anything, suite.successTx.Transaction.TransactionHash).
		Return(sql.ErrNoRows).Twice()
	suite.db.On("NoRows", sql.ErrNoRows).Return(true).Twice()
	suite.db.On("GetSequenceNumbers", suite.ctx, []string{suite.unmuxedSource.Address()}).
		Return(map[string]uint64{suite.unmuxedSource.Address(): 0}, nil).
		Times(3)
	suite.db.On("GetSequenceNumbers", suite.ctx, []string{suite.unmuxedSource.Address()}).
		Return(map[string]uint64{suite.unmuxedSource.Address(): 1}, nil).
		Once()

	// set poll interval to 1ms, so we don't need to wait 3 seconds for the test to complete
	suite.system.Init()
	suite.system.accountSeqPollInterval = time.Millisecond

	r := <-suite.system.Submit(
		suite.ctx,
		suite.successTx.Transaction.TxEnvelope,
		suite.successXDR,
		suite.successTx.Transaction.TransactionHash,
	)

	assert.NotNil(suite.T(), r.Err)
	assert.True(suite.T(), suite.submitter.WasSubmittedTo)
}

func TestSystemTestSuite(t *testing.T) {
	suite.Run(t, new(SystemTestSuite))
}
