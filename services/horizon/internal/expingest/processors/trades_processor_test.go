package processors

import (
	"fmt"
	"testing"
	"time"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TradeProcessorTestSuiteLedger struct {
	suite.Suite
	processor              *TradeProcessor
	mockQ                  *history.MockQTrades
	mockBatchInsertBuilder *history.MockTradeBatchInsertBuilder

	unmuxedSourceAccount       xdr.AccountId
	unmuxedOpSourceAccount     xdr.AccountId
	sourceAccount              xdr.MuxedAccount
	opSourceAccount            xdr.MuxedAccount
	strictReceiveTrade         xdr.ClaimOfferAtom
	strictSendTrade            xdr.ClaimOfferAtom
	buyOfferTrade              xdr.ClaimOfferAtom
	sellOfferTrade             xdr.ClaimOfferAtom
	passiveSellOfferTrade      xdr.ClaimOfferAtom
	otherPassiveSellOfferTrade xdr.ClaimOfferAtom
	allTrades                  []xdr.ClaimOfferAtom
	sellPrices                 []xdr.Price

	assets []xdr.Asset

	unmuxedAccountToID map[string]int64
	assetToID          map[string]history.Asset

	txs []io.LedgerTransaction
}

func TestTradeProcessorTestSuiteLedger(t *testing.T) {
	suite.Run(t, new(TradeProcessorTestSuiteLedger))
}

func (s *TradeProcessorTestSuiteLedger) SetupTest() {
	s.mockQ = &history.MockQTrades{}
	s.mockBatchInsertBuilder = &history.MockTradeBatchInsertBuilder{}

	s.unmuxedSourceAccount = xdr.MustAddress("GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY")
	s.sourceAccount = xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      0xdeadbeef,
			Ed25519: *s.unmuxedSourceAccount.Ed25519,
		},
	}
	s.unmuxedOpSourceAccount = xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML")
	s.opSourceAccount = xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      0xcafebabe,
			Ed25519: *s.unmuxedOpSourceAccount.Ed25519,
		},
	}
	s.strictReceiveTrade = xdr.ClaimOfferAtom{
		SellerId:     xdr.MustAddress("GA2YS6YBWIBUMUJCNYROC5TXYTTUA4TCZF7A4MJ2O4TTGT3LFNWIOMY4"),
		OfferId:      11,
		AssetSold:    xdr.MustNewNativeAsset(),
		AmountSold:   111,
		AmountBought: 211,
		AssetBought:  xdr.MustNewCreditAsset("HUF", s.unmuxedSourceAccount.Address()),
	}
	s.strictSendTrade = xdr.ClaimOfferAtom{
		SellerId:     xdr.MustAddress("GALOBQKDZUSAEUDE7F4OYUIQTUZBL62G6TRCXU2ED6SA7TL72MBUQSYJ"),
		OfferId:      12,
		AssetSold:    xdr.MustNewCreditAsset("USD", s.unmuxedSourceAccount.Address()),
		AmountSold:   112,
		AmountBought: 212,
		AssetBought:  xdr.MustNewCreditAsset("RUB", s.unmuxedSourceAccount.Address()),
	}
	s.buyOfferTrade = xdr.ClaimOfferAtom{
		SellerId:     xdr.MustAddress("GCWRLPH5X5A3GABFDLDILZ4RLY6O76AYOIIR5H2PAI6TNZZZNLZWBXSH"),
		OfferId:      13,
		AssetSold:    xdr.MustNewCreditAsset("EUR", s.unmuxedSourceAccount.Address()),
		AmountSold:   113,
		AmountBought: 213,
		AssetBought:  xdr.MustNewCreditAsset("NOK", s.unmuxedSourceAccount.Address()),
	}
	s.sellOfferTrade = xdr.ClaimOfferAtom{
		SellerId:     xdr.MustAddress("GAVOLNFXVVUJOELN4T3YVSH2FFA3VSP2XN4NJRYF2ZWVCHS77C5KXLHZ"),
		OfferId:      14,
		AssetSold:    xdr.MustNewCreditAsset("PLN", s.unmuxedSourceAccount.Address()),
		AmountSold:   114,
		AmountBought: 214,
		AssetBought:  xdr.MustNewCreditAsset("UAH", s.unmuxedSourceAccount.Address()),
	}
	s.passiveSellOfferTrade = xdr.ClaimOfferAtom{
		SellerId:     xdr.MustAddress("GDQWI6FKB72DPOJE4CGYCFQZKRPQQIOYXRMZ5KEVGXMG6UUTGJMBCASH"),
		OfferId:      15,
		AssetSold:    xdr.MustNewCreditAsset("SEK", s.unmuxedSourceAccount.Address()),
		AmountSold:   115,
		AmountBought: 215,
		AssetBought:  xdr.MustNewCreditAsset("GBP", s.unmuxedSourceAccount.Address()),
	}
	s.otherPassiveSellOfferTrade = xdr.ClaimOfferAtom{
		SellerId:     xdr.MustAddress("GCPZFOJON3PSSYUBNT7MCGEDSGP47UTSJSB4XGCVEWEJO4XQ6U4XN3N2"),
		OfferId:      16,
		AssetSold:    xdr.MustNewCreditAsset("CHF", s.unmuxedSourceAccount.Address()),
		AmountSold:   116,
		AmountBought: 216,
		AssetBought:  xdr.MustNewCreditAsset("JPY", s.unmuxedSourceAccount.Address()),
	}

	s.unmuxedAccountToID = map[string]int64{
		s.unmuxedSourceAccount.Address():   1000,
		s.unmuxedOpSourceAccount.Address(): 1001,
	}
	s.assetToID = map[string]history.Asset{}
	s.allTrades = []xdr.ClaimOfferAtom{
		s.strictReceiveTrade,
		s.strictSendTrade,
		s.buyOfferTrade,
		s.sellOfferTrade,
		s.passiveSellOfferTrade,
		s.otherPassiveSellOfferTrade,
	}

	s.assets = []xdr.Asset{}
	s.sellPrices = []xdr.Price{}
	for i, trade := range s.allTrades {
		s.unmuxedAccountToID[trade.SellerId.Address()] = int64(1002 + i)
		s.assetToID[trade.AssetSold.String()] = history.Asset{ID: int64(10000 + i)}
		s.assetToID[trade.AssetBought.String()] = history.Asset{ID: int64(100 + i)}
		s.assets = append(s.assets, trade.AssetSold, trade.AssetBought)
		n := xdr.Int32(i + 1)
		s.sellPrices = append(s.sellPrices, xdr.Price{N: n, D: 100})
	}

	s.processor = NewTradeProcessor(
		s.mockQ,
		xdr.LedgerHeaderHistoryEntry{
			Header: xdr.LedgerHeader{
				LedgerSeq: 100,
			},
		},
	)
}

func (s *TradeProcessorTestSuiteLedger) TearDownTest() {
	s.mockQ.AssertExpectations(s.T())
	s.mockBatchInsertBuilder.AssertExpectations(s.T())
}

func (s *TradeProcessorTestSuiteLedger) TestIgnoreFailedTransactions() {
	err := s.processor.ProcessTransaction(createTransaction(false, 1))
	s.Assert().NoError(err)

	err = s.processor.Commit()
	s.Assert().NoError(err)
}

func (s *TradeProcessorTestSuiteLedger) mockReadTradeTransactions(
	ledger xdr.LedgerHeaderHistoryEntry,
) []history.InsertTrade {
	closeTime := time.Unix(int64(ledger.Header.ScpValue.CloseTime), 0).UTC()
	inserts := []history.InsertTrade{
		{
			HistoryOperationID: toid.New(int32(ledger.Header.LedgerSeq), 1, 2).ToInt64(),
			Order:              1,
			LedgerCloseTime:    closeTime,
			BuyOfferExists:     false,
			BuyOfferID:         0,
			SellerAccountID:    s.unmuxedAccountToID[s.strictReceiveTrade.SellerId.Address()],
			BuyerAccountID:     s.unmuxedAccountToID[s.unmuxedOpSourceAccount.Address()],
			Trade:              s.strictReceiveTrade,
			SoldAssetID:        s.assetToID[s.strictReceiveTrade.AssetSold.String()].ID,
			BoughtAssetID:      s.assetToID[s.strictReceiveTrade.AssetBought.String()].ID,
			SellPrice:          s.sellPrices[0],
		},
		{
			HistoryOperationID: toid.New(int32(ledger.Header.LedgerSeq), 1, 3).ToInt64(),
			Order:              0,
			LedgerCloseTime:    closeTime,
			BuyOfferExists:     false,
			BuyOfferID:         0,
			SellerAccountID:    s.unmuxedAccountToID[s.strictSendTrade.SellerId.Address()],
			BuyerAccountID:     s.unmuxedAccountToID[s.unmuxedOpSourceAccount.Address()],
			Trade:              s.strictSendTrade,
			SoldAssetID:        s.assetToID[s.strictSendTrade.AssetSold.String()].ID,
			BoughtAssetID:      s.assetToID[s.strictSendTrade.AssetBought.String()].ID,
			SellPrice:          s.sellPrices[1],
		},
		{
			HistoryOperationID: toid.New(int32(ledger.Header.LedgerSeq), 1, 4).ToInt64(),
			Order:              1,
			LedgerCloseTime:    closeTime,
			BuyOfferExists:     true,
			BuyOfferID:         879136,
			SellerAccountID:    s.unmuxedAccountToID[s.buyOfferTrade.SellerId.Address()],
			BuyerAccountID:     s.unmuxedAccountToID[s.unmuxedOpSourceAccount.Address()],
			Trade:              s.buyOfferTrade,
			SoldAssetID:        s.assetToID[s.buyOfferTrade.AssetSold.String()].ID,
			BoughtAssetID:      s.assetToID[s.buyOfferTrade.AssetBought.String()].ID,
			SellPrice:          s.sellPrices[2],
		},
		{
			HistoryOperationID: toid.New(int32(ledger.Header.LedgerSeq), 1, 5).ToInt64(),
			Order:              2,
			LedgerCloseTime:    closeTime,
			BuyOfferExists:     false,
			BuyOfferID:         0,
			SellerAccountID:    s.unmuxedAccountToID[s.sellOfferTrade.SellerId.Address()],
			BuyerAccountID:     s.unmuxedAccountToID[s.unmuxedOpSourceAccount.Address()],
			Trade:              s.sellOfferTrade,
			SoldAssetID:        s.assetToID[s.sellOfferTrade.AssetSold.String()].ID,
			BoughtAssetID:      s.assetToID[s.sellOfferTrade.AssetBought.String()].ID,
			SellPrice:          s.sellPrices[3],
		},
		{
			HistoryOperationID: toid.New(int32(ledger.Header.LedgerSeq), 1, 6).ToInt64(),
			Order:              0,
			LedgerCloseTime:    closeTime,
			BuyOfferExists:     false,
			BuyOfferID:         0,
			SellerAccountID:    s.unmuxedAccountToID[s.passiveSellOfferTrade.SellerId.Address()],
			BuyerAccountID:     s.unmuxedAccountToID[s.unmuxedSourceAccount.Address()],
			Trade:              s.passiveSellOfferTrade,
			SoldAssetID:        s.assetToID[s.passiveSellOfferTrade.AssetSold.String()].ID,
			BoughtAssetID:      s.assetToID[s.passiveSellOfferTrade.AssetBought.String()].ID,
			SellPrice:          s.sellPrices[4],
		},
		{
			HistoryOperationID: toid.New(int32(ledger.Header.LedgerSeq), 1, 7).ToInt64(),
			Order:              0,
			LedgerCloseTime:    closeTime,
			BuyOfferExists:     false,
			BuyOfferID:         0,
			SellerAccountID:    s.unmuxedAccountToID[s.otherPassiveSellOfferTrade.SellerId.Address()],
			BuyerAccountID:     s.unmuxedAccountToID[s.unmuxedOpSourceAccount.Address()],
			Trade:              s.otherPassiveSellOfferTrade,
			SoldAssetID:        s.assetToID[s.otherPassiveSellOfferTrade.AssetSold.String()].ID,
			BoughtAssetID:      s.assetToID[s.otherPassiveSellOfferTrade.AssetBought.String()].ID,
			SellPrice:          s.sellPrices[5],
		},
	}

	emptyTrade := xdr.ClaimOfferAtom{
		SellerId:     s.sourceAccount.ToAccountId(),
		OfferId:      123,
		AssetSold:    xdr.MustNewNativeAsset(),
		AmountSold:   0,
		AssetBought:  xdr.MustNewCreditAsset("EUR", s.unmuxedSourceAccount.Address()),
		AmountBought: 0,
	}

	operationResults := []xdr.OperationResult{
		{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeBumpSequence,
				BumpSeqResult: &xdr.BumpSequenceResult{
					Code: xdr.BumpSequenceResultCodeBumpSequenceSuccess,
				},
			},
		},
		{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypePathPaymentStrictReceive,
				PathPaymentStrictReceiveResult: &xdr.PathPaymentStrictReceiveResult{
					Code: xdr.PathPaymentStrictReceiveResultCodePathPaymentStrictReceiveSuccess,
					Success: &xdr.PathPaymentStrictReceiveResultSuccess{
						Offers: []xdr.ClaimOfferAtom{
							emptyTrade,
							s.strictReceiveTrade,
						},
					},
				},
			},
		},
		{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypePathPaymentStrictSend,
				PathPaymentStrictSendResult: &xdr.PathPaymentStrictSendResult{
					Code: xdr.PathPaymentStrictSendResultCodePathPaymentStrictSendSuccess,
					Success: &xdr.PathPaymentStrictSendResultSuccess{
						Offers: []xdr.ClaimOfferAtom{
							s.strictSendTrade,
							emptyTrade,
						},
					},
				},
			},
		},
		{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageBuyOffer,
				ManageBuyOfferResult: &xdr.ManageBuyOfferResult{
					Code: xdr.ManageBuyOfferResultCodeManageBuyOfferSuccess,
					Success: &xdr.ManageOfferSuccessResult{
						OffersClaimed: []xdr.ClaimOfferAtom{
							emptyTrade,
							s.buyOfferTrade,
							emptyTrade,
						},
						Offer: xdr.ManageOfferSuccessResultOffer{
							Offer: &xdr.OfferEntry{
								OfferId: 879136,
							},
						},
					},
				},
			},
		},
		{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageSellOffer,
				ManageSellOfferResult: &xdr.ManageSellOfferResult{
					Code: xdr.ManageSellOfferResultCodeManageSellOfferSuccess,
					Success: &xdr.ManageOfferSuccessResult{
						OffersClaimed: []xdr.ClaimOfferAtom{
							emptyTrade,
							emptyTrade,
							s.sellOfferTrade,
						},
						Offer: xdr.ManageOfferSuccessResultOffer{
							Effect: xdr.ManageOfferEffectManageOfferDeleted,
						},
					},
				},
			},
		},
		{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeManageSellOffer,
				ManageSellOfferResult: &xdr.ManageSellOfferResult{
					Code: xdr.ManageSellOfferResultCodeManageSellOfferSuccess,
					Success: &xdr.ManageOfferSuccessResult{
						OffersClaimed: []xdr.ClaimOfferAtom{
							s.passiveSellOfferTrade,
							emptyTrade,
							emptyTrade,
						},
						Offer: xdr.ManageOfferSuccessResultOffer{
							Effect: xdr.ManageOfferEffectManageOfferDeleted,
						},
					},
				},
			},
		},
		{
			Tr: &xdr.OperationResultTr{
				Type: xdr.OperationTypeCreatePassiveSellOffer,
				CreatePassiveSellOfferResult: &xdr.ManageSellOfferResult{
					Code: xdr.ManageSellOfferResultCodeManageSellOfferSuccess,
					Success: &xdr.ManageOfferSuccessResult{
						OffersClaimed: []xdr.ClaimOfferAtom{
							s.otherPassiveSellOfferTrade,
						},
						Offer: xdr.ManageOfferSuccessResultOffer{
							Effect: xdr.ManageOfferEffectManageOfferDeleted,
						},
					},
				},
			},
		},
	}

	operations := []xdr.Operation{
		{
			Body: xdr.OperationBody{
				Type:           xdr.OperationTypeBumpSequence,
				BumpSequenceOp: &xdr.BumpSequenceOp{BumpTo: 30000},
			},
		},
		{
			Body: xdr.OperationBody{
				Type:                       xdr.OperationTypePathPaymentStrictReceive,
				PathPaymentStrictReceiveOp: &xdr.PathPaymentStrictReceiveOp{},
			},
			SourceAccount: &s.opSourceAccount,
		},
		{
			Body: xdr.OperationBody{
				Type:                    xdr.OperationTypePathPaymentStrictSend,
				PathPaymentStrictSendOp: &xdr.PathPaymentStrictSendOp{},
			},
			SourceAccount: &s.opSourceAccount,
		},
		{
			Body: xdr.OperationBody{
				Type:             xdr.OperationTypeManageBuyOffer,
				ManageBuyOfferOp: &xdr.ManageBuyOfferOp{},
			},
			SourceAccount: &s.opSourceAccount,
		},
		{
			Body: xdr.OperationBody{
				Type:              xdr.OperationTypeManageSellOffer,
				ManageSellOfferOp: &xdr.ManageSellOfferOp{},
			},
			SourceAccount: &s.opSourceAccount,
		},
		{
			Body: xdr.OperationBody{
				Type:                     xdr.OperationTypeCreatePassiveSellOffer,
				CreatePassiveSellOfferOp: &xdr.CreatePassiveSellOfferOp{},
			},
		},
		{
			Body: xdr.OperationBody{
				Type:                     xdr.OperationTypeCreatePassiveSellOffer,
				CreatePassiveSellOfferOp: &xdr.CreatePassiveSellOfferOp{},
			},
			SourceAccount: &s.opSourceAccount,
		},
	}

	tx := io.LedgerTransaction{
		Result: xdr.TransactionResultPair{
			Result: xdr.TransactionResult{
				Result: xdr.TransactionResultResult{
					Code:    xdr.TransactionResultCodeTxSuccess,
					Results: &operationResults,
				},
			},
		},
		Envelope: xdr.TransactionEnvelope{
			Type: xdr.EnvelopeTypeEnvelopeTypeTx,
			V1: &xdr.TransactionV1Envelope{
				Tx: xdr.Transaction{
					SourceAccount: s.sourceAccount,
					Operations:    operations,
				},
			},
		},
		Index:      1,
		FeeChanges: []xdr.LedgerEntryChange{},
		Meta: xdr.TransactionMeta{
			V: 2,
			V2: &xdr.TransactionMetaV2{
				Operations: []xdr.OperationMeta{
					{
						Changes: xdr.LedgerEntryChanges{},
					},
				},
			},
		},
	}

	for i, trade := range s.allTrades {
		tx.Meta.V2.Operations = append(tx.Meta.V2.Operations, xdr.OperationMeta{
			Changes: xdr.LedgerEntryChanges{
				xdr.LedgerEntryChange{
					Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
					State: &xdr.LedgerEntry{
						Data: xdr.LedgerEntryData{
							Type: xdr.LedgerEntryTypeOffer,
							Offer: &xdr.OfferEntry{
								Price:    s.sellPrices[i],
								SellerId: trade.SellerId,
								OfferId:  trade.OfferId,
							},
						},
					},
				},
				xdr.LedgerEntryChange{
					Type: xdr.LedgerEntryChangeTypeLedgerEntryRemoved,
					Removed: &xdr.LedgerKey{
						Type: xdr.LedgerEntryTypeOffer,
						Offer: &xdr.LedgerKeyOffer{
							SellerId: trade.SellerId,
							OfferId:  trade.OfferId,
						},
					},
				},
			},
		})
	}

	s.txs = []io.LedgerTransaction{
		tx,
	}

	s.mockQ.On("NewTradeBatchInsertBuilder", maxBatchSize).
		Return(s.mockBatchInsertBuilder).Once()

	return inserts
}

func (s *TradeProcessorTestSuiteLedger) TestIngestTradesSucceeds() {
	inserts := s.mockReadTradeTransactions(s.processor.ledger)

	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				mapKeysToList(s.unmuxedAccountToID),
				arg,
			)
		}).Return(s.unmuxedAccountToID, nil).Once()

	s.mockQ.On("CreateAssets", mock.AnythingOfType("[]xdr.Asset"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]xdr.Asset)
			s.Assert().ElementsMatch(
				s.assets,
				arg,
			)
		}).Return(s.assetToID, nil).Once()

	for _, insert := range inserts {
		s.mockBatchInsertBuilder.On("Add", []history.InsertTrade{
			insert,
		}).Return(nil).Once()
	}

	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}

	err := s.processor.Commit()
	s.Assert().NoError(err)
}

func (s *TradeProcessorTestSuiteLedger) TestCreateAccountsError() {
	s.mockReadTradeTransactions(s.processor.ledger)

	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				mapKeysToList(s.unmuxedAccountToID),
				arg,
			)
		}).Return(map[string]int64{}, fmt.Errorf("create accounts error")).Once()

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}

	err := s.processor.Commit()

	s.Assert().EqualError(err, "Error creating account ids: create accounts error")
}

func (s *TradeProcessorTestSuiteLedger) TestCreateAssetsError() {
	s.mockReadTradeTransactions(s.processor.ledger)

	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				mapKeysToList(s.unmuxedAccountToID),
				arg,
			)
		}).Return(s.unmuxedAccountToID, nil).Once()

	s.mockQ.On("CreateAssets", mock.AnythingOfType("[]xdr.Asset"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]xdr.Asset)
			s.Assert().ElementsMatch(
				s.assets,
				arg,
			)
		}).Return(s.assetToID, fmt.Errorf("create assets error")).Once()

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}

	err := s.processor.Commit()
	s.Assert().EqualError(err, "Error creating asset ids: create assets error")
}

func (s *TradeProcessorTestSuiteLedger) TestBatchAddError() {
	s.mockReadTradeTransactions(s.processor.ledger)

	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				mapKeysToList(s.unmuxedAccountToID),
				arg,
			)
		}).Return(s.unmuxedAccountToID, nil).Once()

	s.mockQ.On("CreateAssets", mock.AnythingOfType("[]xdr.Asset"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]xdr.Asset)
			s.Assert().ElementsMatch(
				s.assets,
				arg,
			)
		}).Return(s.assetToID, nil).Once()

	s.mockBatchInsertBuilder.On("Add", mock.AnythingOfType("[]history.InsertTrade")).
		Return(fmt.Errorf("batch add error")).Once()

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}

	err := s.processor.Commit()
	s.Assert().EqualError(err, "Error adding trade to batch: batch add error")
}

func (s *TradeProcessorTestSuiteLedger) TestBatchExecError() {
	insert := s.mockReadTradeTransactions(s.processor.ledger)

	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				mapKeysToList(s.unmuxedAccountToID),
				arg,
			)
		}).Return(s.unmuxedAccountToID, nil).Once()

	s.mockQ.On("CreateAssets", mock.AnythingOfType("[]xdr.Asset"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]xdr.Asset)
			s.Assert().ElementsMatch(
				s.assets,
				arg,
			)
		}).Return(s.assetToID, nil).Once()

	s.mockBatchInsertBuilder.On("Add", mock.AnythingOfType("[]history.InsertTrade")).
		Return(nil).Times(len(insert))
	s.mockBatchInsertBuilder.On("Exec").Return(fmt.Errorf("exec error")).Once()
	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}

	err := s.processor.Commit()
	s.Assert().EqualError(err, "Error flushing operation batch: exec error")
}

func (s *TradeProcessorTestSuiteLedger) TestIgnoreCheckIfSmallLedger() {
	insert := s.mockReadTradeTransactions(s.processor.ledger)

	s.mockQ.On("CreateAccounts", mock.AnythingOfType("[]string"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]string)
			s.Assert().ElementsMatch(
				mapKeysToList(s.unmuxedAccountToID),
				arg,
			)
		}).Return(s.unmuxedAccountToID, nil).Once()

	s.mockQ.On("CreateAssets", mock.AnythingOfType("[]xdr.Asset"), maxBatchSize).
		Run(func(args mock.Arguments) {
			arg := args.Get(0).([]xdr.Asset)
			s.Assert().ElementsMatch(
				s.assets,
				arg,
			)
		}).Return(s.assetToID, nil).Once()

	s.mockBatchInsertBuilder.On("Add", mock.AnythingOfType("[]history.InsertTrade")).
		Return(nil).Times(len(insert))
	s.mockBatchInsertBuilder.On("Exec").Return(nil).Once()

	for _, tx := range s.txs {
		err := s.processor.ProcessTransaction(tx)
		s.Assert().NoError(err)
	}

	err := s.processor.Commit()
	s.Assert().NoError(err)
}

func TestTradeProcessor_ProcessTransaction_MuxedAccount(t *testing.T) {
	unmuxed := xdr.MustAddress("GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2")
	muxed := xdr.MuxedAccount{
		Type: xdr.CryptoKeyTypeKeyTypeMuxedEd25519,
		Med25519: &xdr.MuxedAccountMed25519{
			Id:      0xdeadbeefdeadbeef,
			Ed25519: *unmuxed.Ed25519,
		},
	}
	tx := createTransaction(true, 1)
	tx.Index = 1
	tx.Envelope.Operations()[0].Body = xdr.OperationBody{
		Type: xdr.OperationTypePayment,
		PaymentOp: &xdr.PaymentOp{
			Destination: muxed,
			Asset:       xdr.Asset{Type: xdr.AssetTypeAssetTypeNative},
			Amount:      100,
		},
	}
}
