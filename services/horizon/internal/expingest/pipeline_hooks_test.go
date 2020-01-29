package expingest

// import (
// 	"context"
// 	"testing"

// 	"github.com/jmoiron/sqlx"
// 	"github.com/stellar/go/exp/orderbook"
// 	"github.com/stellar/go/services/horizon/internal/db2"
// 	"github.com/stellar/go/services/horizon/internal/db2/history"
// 	"github.com/stellar/go/services/horizon/internal/test"
// 	"github.com/stellar/go/support/errors"
// 	"github.com/stellar/go/xdr"
// 	"github.com/stretchr/testify/suite"
// )

// type PreProcessingHookTestSuite struct {
// 	suite.Suite
// 	historyQ             *mockDBQ
// 	system               *System
// 	ctx                  context.Context
// 	ledgerSeqFromContext uint32
// }

// func TestPreProcessingHookTestSuite(t *testing.T) {
// 	suite.Run(t, new(PreProcessingHookTestSuite))
// }

// func (s *PreProcessingHookTestSuite) SetupTest() {
// 	s.system = &System{
// 		state: state{systemState: resumeState},
// 	}
// 	s.historyQ = &mockDBQ{}
// 	s.ledgerSeqFromContext = uint32(5)

// 	// s.ctx = context.WithValue(
// 	// 	context.Background(),
// 	// 	pipeline.LedgerSequenceContextKey,
// 	// 	s.ledgerSeqFromContext,
// 	// )
// }

// func (s *PreProcessingHookTestSuite) TearDownTest() {
// 	s.historyQ.AssertExpectations(s.T())
// }

// func (s *PreProcessingHookTestSuite) TestStateHookSucceedsWithPreExistingTx() {
// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()

// }

// func (s *PreProcessingHookTestSuite) TestStateHookSucceedsWithoutPreExistingTx() {
// 	var nilTx *sqlx.Tx
// 	s.historyQ.On("GetTx").Return(nilTx).Once()
// 	s.historyQ.On("Begin").Return(nil).Once()
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), nil).Once()

// }

// func (s *PreProcessingHookTestSuite) TestStateHookRollsbackOnGetLastLedgerExpIngestError() {
// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("transient error")).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// }

// func (s *PreProcessingHookTestSuite) TestStateHookRollsbackOnBeginError() {
// 	var nilTx *sqlx.Tx
// 	s.historyQ.On("GetTx").Return(nilTx).Once()
// 	s.historyQ.On("Begin").Return(errors.New("transient error")).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// }

// func (s *PreProcessingHookTestSuite) TestLedgerHookSucceedsWithPreExistingTx() {
// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(1), nil).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// }

// func (s *PreProcessingHookTestSuite) TestLedgerHookSucceedsWithoutPreExistingTx() {
// 	var nilTx *sqlx.Tx
// 	s.historyQ.On("GetTx").Return(nilTx).Once()
// 	s.historyQ.On("Begin").Return(nil).Once()
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(1), nil).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// }

// func (s *PreProcessingHookTestSuite) TestLedgerHookSucceedsAsMaster() {
// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(s.ledgerSeqFromContext-1, nil).Once()

// }

// func (s *PreProcessingHookTestSuite) TestLedgerHookSucceedsAsMasterHistoryCatchup() {
// 	s.system.state = state{systemState: ingestHistoryRangeState}

// }

// func (s *PreProcessingHookTestSuite) TestLedgerHookRollsbackOnGetLastLedgerExpIngestError() {
// 	s.historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	s.historyQ.On("GetLastLedgerExpIngest").Return(uint32(0), errors.New("transient error")).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// }

// func (s *PreProcessingHookTestSuite) TestLedgerHookRollsbackOnBeginError() {
// 	var nilTx *sqlx.Tx
// 	s.historyQ.On("GetTx").Return(nilTx).Once()
// 	s.historyQ.On("Begin").Return(errors.New("transient error")).Once()
// 	s.historyQ.On("Rollback").Return(nil).Once()

// }

// func TestPostProcessingHook(t *testing.T) {
// 	tt := test.Start(t).Scenario("base")
// 	defer tt.Finish()

// 	account := "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2"
// 	signer := "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"
// 	weight := int32(123)
// 	accountSigner := history.AccountSigner{
// 		Account: account,
// 		Signer:  signer,
// 		Weight:  weight,
// 	}

// 	session := tt.HorizonSession()
// 	defer session.Rollback()
// 	historyQ := &history.Q{session}
// 	for _, testCase := range []struct {
// 		name           string
// 		err            error
// 		lastLedger     uint32
// 		pipelineLedger uint32
// 		inTx           bool
// 		expectedError  string
// 	}{
// 		{
// 			"succeeds when last ledger in db is 0",
// 			nil,
// 			0,
// 			3,
// 			true,
// 			"",
// 		},
// 		{
// 			"succeeds when local latest sequence is equal to global sequence",
// 			nil,
// 			2,
// 			3,
// 			true,
// 			"",
// 		},
// 		{
// 			"succeeds when we're not in a tx",
// 			nil,
// 			1,
// 			3,
// 			false,
// 			"",
// 		},
// 		{
// 			"fails because of passed in error",
// 			errors.New("test case error"),
// 			2,
// 			3,
// 			false,
// 			"test case error",
// 		},
// 		{
// 			"fails because local latest sequence is not equal to global sequence",
// 			nil,
// 			1,
// 			3,
// 			true,
// 			"local latest sequence is not equal to global sequence",
// 		},
// 	} {
// 		t.Run(testCase.name, func(_ *testing.T) {
// 			tt.Assert.Nil(historyQ.UpdateLastLedgerExpIngest(testCase.lastLedger))
// 			tt.Assert.Nil(historyQ.UpdateExpIngestVersion(0))
// 			_, err := historyQ.RemoveAccountSigner(account, signer)
// 			tt.Assert.NoError(err)

// 			graph := orderbook.NewOrderBookGraph()
// 			// queue an offer on the orderbook so we can check if the post
// 			// processing hook applied it
// 			graph.AddOffer(eurOffer)

// 			if testCase.inTx {
// 				tt.Assert.Nil(session.Begin())
// 				// queue an insert on the transaction so we can check if the post
// 				// processing hook committed it to the db
// 				_, err = historyQ.CreateAccountSigner(account, signer, weight)
// 				tt.Assert.NoError(err)
// 			}

// 			if testCase.expectedError == "" {
// 				tt.Assert.NoError(err)
// 				tt.Assert.Equal(graph.Offers(), []xdr.OfferEntry{eurOffer})
// 			} else {
// 				tt.Assert.Contains(err.Error(), testCase.expectedError)
// 				tt.Assert.Equal(graph.Offers(), []xdr.OfferEntry{})
// 			}
// 			tt.Assert.Nil(session.GetTx())

// 			if testCase.inTx && testCase.expectedError == "" {
// 				// check that the ingest version and the ingest sequence was updated
// 				version, err := historyQ.GetExpIngestVersion()
// 				tt.Assert.NoError(err)
// 				tt.Assert.Equal(version, CurrentVersion)
// 				seq, err := historyQ.GetLastLedgerExpIngestNonBlocking()
// 				tt.Assert.NoError(err)
// 				tt.Assert.Equal(seq, testCase.pipelineLedger)

// 				// check that the transaction was committed
// 				accounts, err := historyQ.AccountsForSigner(signer, db2.PageQuery{Order: "asc", Limit: 10})
// 				tt.Assert.NoError(err)
// 				tt.Assert.Len(accounts, 1)
// 				tt.Assert.Equal(accountSigner, accounts[0])
// 			} else {
// 				// check that the transaction was rolled back and nothing was committed
// 				version, err := historyQ.GetExpIngestVersion()
// 				tt.Assert.NoError(err)
// 				tt.Assert.Equal(version, 0)
// 				seq, err := historyQ.GetLastLedgerExpIngestNonBlocking()
// 				tt.Assert.NoError(err)
// 				tt.Assert.Equal(seq, testCase.lastLedger)

// 				accounts, err := historyQ.AccountsForSigner(signer, db2.PageQuery{Order: "asc", Limit: 10})
// 				tt.Assert.NoError(err)
// 				tt.Assert.Len(accounts, 0)
// 			}
// 		})
// 	}
// }

// func TestPostProcessingHookHistoryCatchup(t *testing.T) {
// 	// system := &System{
// 	// 	state: state{systemState: ingestHistoryRangeState},
// 	// }
// 	historyQ := &mockDBQ{}

// 	historyQ.On("GetTx").Return(&sqlx.Tx{}).Once()
// 	// Doesn't update version and last ledger in ingestHistoryRangeState
// 	historyQ.On("Commit").Return(nil).Once()
// 	historyQ.On("GetExpStateInvalid").Return(false, nil).Once()
// 	historyQ.On("Rollback").Return(nil).Once() // defer

// }
