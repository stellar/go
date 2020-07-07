package ledgerbackend

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/suite"
)

func TestHistoryArchiveBackendTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryArchiveBackendTestSuite))
}

type HistoryArchiveBackendTestSuite struct {
	suite.Suite
	mockArchive *historyarchive.MockArchive
	backend     *HistoryArchiveBackend
}

func (s *HistoryArchiveBackendTestSuite) SetupTest() {
	s.mockArchive = &historyarchive.MockArchive{}
	s.backend = NewHistoryArchiveBackendFromArchive(s.mockArchive)
}

func (s *HistoryArchiveBackendTestSuite) TearDownTest() {
	s.mockArchive.AssertExpectations(s.T())
}

func (s *HistoryArchiveBackendTestSuite) TestGetLatestLedgerSequence() {
	latestSequence := uint32(100)
	s.mockArchive.
		On("GetRootHAS").
		Return(historyarchive.HistoryArchiveState{
			CurrentLedger: latestSequence,
		}, nil).Once()

	sequence, err := s.backend.GetLatestLedgerSequence()
	s.Require().NoError(err)
	s.Assert().Equal(latestSequence, sequence)
}

func (s *HistoryArchiveBackendTestSuite) TestGetLedger() {
	// []interface{} to have a single function creating xdr streams
	ledgers := []interface{}{}
	transactions := []interface{}{}
	results := []interface{}{}
	for i := uint32(64); i <= 127; i++ {
		ledgers = append(ledgers, xdr.LedgerHeaderHistoryEntry{
			Header: xdr.LedgerHeader{
				LedgerSeq: xdr.Uint32(i),
			},
		})

		transactions = append(transactions, xdr.TransactionHistoryEntry{
			LedgerSeq: xdr.Uint32(i),
			TxSet: xdr.TransactionSet{
				Txs: []xdr.TransactionEnvelope{createSampleTx(i)},
			},
		})

		opResults := []xdr.OperationResult{}
		results = append(results, xdr.TransactionHistoryResultEntry{
			LedgerSeq: xdr.Uint32(i),
			TxResultSet: xdr.TransactionResultSet{
				Results: []xdr.TransactionResultPair{
					{
						Result: xdr.TransactionResult{
							FeeCharged: xdr.Int64(i),
							Result: xdr.TransactionResultResult{
								Code:    xdr.TransactionResultCodeTxSuccess,
								Results: &opResults,
							},
						},
					},
				},
			},
		})
	}

	s.mockArchive.On("CategoryCheckpointExists", "ledger", uint32(127)).Return(true, nil).Once()
	s.mockArchive.On("CategoryCheckpointExists", "transactions", uint32(127)).Return(true, nil).Once()
	s.mockArchive.On("CategoryCheckpointExists", "results", uint32(127)).Return(true, nil).Once()

	s.mockArchive.On("GetXdrStream", "ledger/00/00/00/ledger-0000007f.xdr.gz").Return(createXdrStream(ledgers), nil).Once()
	s.mockArchive.On("GetXdrStream", "transactions/00/00/00/transactions-0000007f.xdr.gz").Return(createXdrStream(transactions), nil).Once()
	s.mockArchive.On("GetXdrStream", "results/00/00/00/results-0000007f.xdr.gz").Return(createXdrStream(results), nil).Once()

	exists, _, err := s.backend.GetLedger(100)
	s.Require().NoError(err)
	s.Assert().True(exists)
	s.Assert().Equal(uint32(64), s.backend.rangeFrom)
	s.Assert().Equal(uint32(127), s.backend.rangeTo)

	for sequence := uint32(64); sequence <= 127; sequence++ {
		var err2 error
		exists, ledger, err2 := s.backend.GetLedger(sequence)
		s.Require().NoError(err2)
		s.Assert().True(exists)

		s.Assert().Equal(sequence, uint32(ledger.V0.LedgerHeader.Header.LedgerSeq))
		s.Assert().Equal(sequence, uint32(ledger.V0.TxSet.Txs[0].Operations()[0].Body.BumpSequenceOp.BumpTo))
		s.Assert().Equal(sequence, uint32(ledger.V0.TxProcessing[0].Result.Result.FeeCharged))
		s.Assert().Empty(ledger.V0.TxProcessing[0].FeeProcessing)
		s.Assert().Empty(ledger.V0.TxProcessing[0].TxApplyProcessing)
		s.Assert().Empty(ledger.V0.UpgradesProcessing)
		s.Assert().Empty(ledger.V0.ScpInfo)
	}

	err = s.backend.Close()
	s.Require().NoError(err)
	s.Assert().Zero(s.backend.rangeFrom)
	s.Assert().Zero(s.backend.rangeTo)
	s.Assert().Empty(s.backend.cache)
}

func (s *HistoryArchiveBackendTestSuite) TestGetLedgerFirstCheckpoint() {
	// []interface{} to have a single function creating xdr streams
	ledgers := []interface{}{}
	transactions := []interface{}{}
	results := []interface{}{}
	for i := uint32(1); i <= 63; i++ {
		ledgers = append(ledgers, xdr.LedgerHeaderHistoryEntry{
			Header: xdr.LedgerHeader{
				LedgerSeq: xdr.Uint32(i),
			},
		})

		transactions = append(transactions, xdr.TransactionHistoryEntry{
			LedgerSeq: xdr.Uint32(i),
			TxSet: xdr.TransactionSet{
				Txs: []xdr.TransactionEnvelope{createSampleTx(i)},
			},
		})

		opResults := []xdr.OperationResult{}
		results = append(results, xdr.TransactionHistoryResultEntry{
			LedgerSeq: xdr.Uint32(i),
			TxResultSet: xdr.TransactionResultSet{
				Results: []xdr.TransactionResultPair{
					{
						Result: xdr.TransactionResult{
							FeeCharged: xdr.Int64(i),
							Result: xdr.TransactionResultResult{
								Code:    xdr.TransactionResultCodeTxSuccess,
								Results: &opResults,
							},
						},
					},
				},
			},
		})
	}

	s.mockArchive.On("CategoryCheckpointExists", "ledger", uint32(63)).Return(true, nil).Once()
	s.mockArchive.On("CategoryCheckpointExists", "transactions", uint32(63)).Return(true, nil).Once()
	s.mockArchive.On("CategoryCheckpointExists", "results", uint32(63)).Return(true, nil).Once()

	s.mockArchive.On("GetXdrStream", "ledger/00/00/00/ledger-0000003f.xdr.gz").Return(createXdrStream(ledgers), nil).Once()
	s.mockArchive.On("GetXdrStream", "transactions/00/00/00/transactions-0000003f.xdr.gz").Return(createXdrStream(transactions), nil).Once()
	s.mockArchive.On("GetXdrStream", "results/00/00/00/results-0000003f.xdr.gz").Return(createXdrStream(results), nil).Once()

	exists, _, err := s.backend.GetLedger(60)
	s.Require().NoError(err)
	s.Assert().True(exists)
	s.Assert().Equal(uint32(1), s.backend.rangeFrom)
	s.Assert().Equal(uint32(63), s.backend.rangeTo)

	for sequence := uint32(1); sequence <= 63; sequence++ {
		var err2 error
		exists, ledger, err2 := s.backend.GetLedger(sequence)
		s.Require().NoError(err2)
		s.Assert().True(exists)

		s.Assert().Equal(sequence, uint32(ledger.V0.LedgerHeader.Header.LedgerSeq))
		s.Assert().Equal(sequence, uint32(ledger.V0.TxSet.Txs[0].Operations()[0].Body.BumpSequenceOp.BumpTo))
		s.Assert().Equal(sequence, uint32(ledger.V0.TxProcessing[0].Result.Result.FeeCharged))
		s.Assert().Empty(ledger.V0.TxProcessing[0].FeeProcessing)
		s.Assert().Empty(ledger.V0.TxProcessing[0].TxApplyProcessing)
		s.Assert().Empty(ledger.V0.UpgradesProcessing)
		s.Assert().Empty(ledger.V0.ScpInfo)
	}

	err = s.backend.Close()
	s.Require().NoError(err)
	s.Assert().Zero(s.backend.rangeFrom)
	s.Assert().Zero(s.backend.rangeTo)
	s.Assert().Empty(s.backend.cache)
}

func createXdrStream(entries []interface{}) *historyarchive.XdrStream {
	b := &bytes.Buffer{}
	for _, e := range entries {
		err := xdr.MarshalFramed(b, e)
		if err != nil {
			panic(err)
		}
	}

	return xdrStreamFromBuffer(b)
}

func xdrStreamFromBuffer(b *bytes.Buffer) *historyarchive.XdrStream {
	return historyarchive.NewXdrStream(ioutil.NopCloser(b))
}

func createSampleTx(sequence uint32) xdr.TransactionEnvelope {
	kp, err := keypair.Random()
	if err != nil {
		panic(err)
	}

	sourceAccount := txnbuild.NewSimpleAccount(kp.Address(), int64(0))
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount: &sourceAccount,
			Operations: []txnbuild.Operation{
				&txnbuild.BumpSequence{
					BumpTo: int64(sequence),
				},
			},
			BaseFee:    txnbuild.MinBaseFee,
			Timebounds: txnbuild.NewInfiniteTimeout(),
		},
	)
	if err != nil {
		panic(err)
	}

	env, err := tx.TxEnvelope()
	if err != nil {
		panic(err)
	}

	return env
}
