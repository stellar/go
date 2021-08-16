package ingest

import (
	"io"
	"regexp"
	"testing"

	"github.com/stellar/go/gxdr"
	"github.com/stellar/go/historyarchive"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/randxdr"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

func addOffers(tt *test.T, q *history.Q, mockChangeReader *ingest.MockChangeReader) {
	pp := processors.NewOffersProcessor(q, 10)
	gen := randxdr.NewGenerator()

	var changes []xdr.LedgerEntryChange
	for i := 0; i < 1000; i++ {
		change := xdr.LedgerEntryChange{}
		shape := &gxdr.LedgerEntryChange{}
		gen.Next(
			shape,
			[]randxdr.Preset{
				{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
				// the offers postgres table is configured with some database constraints which validate the following
				// fields:
				{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
				{randxdr.FieldEquals("created.data.offer.amount"), randxdr.SetPositiveNum64},
				{randxdr.FieldEquals("created.data.offer.price.n"), randxdr.SetPositiveNum32},
				{randxdr.FieldEquals("created.data.offer.price.d"), randxdr.SetPositiveNum32},
			},
		)
		tt.Assert.NoError(gxdr.Convert(shape, &change))
		changes = append(changes, change)
	}

	for _, change := range ingest.GetChangesFromLedgerEntryChanges(changes) {
		if change.Type == xdr.LedgerEntryTypeOffer && change.Post != nil && change.Pre == nil {
			mockChangeReader.On("Read").Return(change, nil).Once()
		}
		tt.Assert.NoError(pp.ProcessChange(tt.Ctx, change))
	}

	tt.Assert.NoError(pp.Commit(tt.Ctx))
}

func addLiquidityPools(tt *test.T, q *history.Q, mockChangeReader *ingest.MockChangeReader) {
	pp := processors.NewLiquidityPoolsProcessor(q)
	gen := randxdr.NewGenerator()

	var changes []xdr.LedgerEntryChange
	for i := 0; i < 1000; i++ {
		change := xdr.LedgerEntryChange{}
		shape := &gxdr.LedgerEntryChange{}
		gen.Next(
			shape,
			[]randxdr.Preset{
				{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
				// liquidity pools cannot be sponsored
				{randxdr.FieldEquals("created.ext.v1.sponsoringID"), randxdr.SetPtrToNil},
				{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.params.fee"), randxdr.SetPositiveNum32},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.reserveA"), randxdr.SetPositiveNum64},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.reserveB"), randxdr.SetPositiveNum64},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.totalPoolShares"), randxdr.SetPositiveNum64},
				{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.poolSharesTrustLineCount"), randxdr.SetPositiveNum64},
			},
		)
		tt.Assert.NoError(gxdr.Convert(shape, &change))
		changes = append(changes, change)
	}

	for _, change := range ingest.GetChangesFromLedgerEntryChanges(changes) {
		if change.Type == xdr.LedgerEntryTypeLiquidityPool && change.Post != nil && change.Pre == nil {
			mockChangeReader.On("Read").Return(change, nil).Once()
		}
		tt.Assert.NoError(pp.ProcessChange(tt.Ctx, change))
	}

	tt.Assert.NoError(pp.Commit(tt.Ctx))
}

func addTrustLines(tt *test.T, q *history.Q, mockChangeReader *ingest.MockChangeReader) {
	pp := processors.NewTrustLinesProcessor(q)
	assetStats := processors.NewAssetStatsProcessor(q, true)
	gen := randxdr.NewGenerator()

	var changes []xdr.LedgerEntryChange
	for i := 0; i < 1000; i++ {
		change := xdr.LedgerEntryChange{}
		shape := &gxdr.LedgerEntryChange{}
		gen.Next(
			shape,
			[]randxdr.Preset{
				{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
				{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
				{randxdr.FieldEquals("created.data.trustLine.flags"), randxdr.SetPositiveNum32},
				{randxdr.FieldEquals("created.data.trustLine.asset.alphaNum4.assetCode"), randxdr.SetAssetCode()},
				{randxdr.FieldEquals("created.data.trustLine.asset.alphaNum12.assetCode"), randxdr.SetAssetCode()},
				{randxdr.FieldEquals("created.data.trustLine.balance"), randxdr.SetPositiveNum64},
				{randxdr.FieldEquals("created.data.trustLine.limit"), randxdr.SetPositiveNum64},
				{randxdr.FieldEquals("created.data.trustLine.ext.v1.liabilities.selling"), randxdr.SetPositiveNum64},
				{randxdr.FieldEquals("created.data.trustLine.ext.v1.liabilities.buying"), randxdr.SetPositiveNum64},
			},
		)
		tt.Assert.NoError(gxdr.Convert(shape, &change))
		changes = append(changes, change)
	}

	for _, change := range ingest.GetChangesFromLedgerEntryChanges(changes) {
		if change.Type == xdr.LedgerEntryTypeTrustline && change.Post != nil && change.Pre == nil {
			mockChangeReader.On("Read").Return(change, nil).Once()
		}
		tt.Assert.NoError(pp.ProcessChange(tt.Ctx, change))
		// asst stats can handle both trustlines or claimable balances
		// we only care about trustlines in this function so skip claimable balances
		if change.Type != xdr.LedgerEntryTypeClaimableBalance {
			tt.Assert.NoError(assetStats.ProcessChange(tt.Ctx, change))
		}
	}

	tt.Assert.NoError(pp.Commit(tt.Ctx))
	tt.Assert.NoError(assetStats.Commit(tt.Ctx))
}

func addClaimableBalances(tt *test.T, q *history.Q, mockChangeReader *ingest.MockChangeReader) {
	pp := processors.NewClaimableBalancesChangeProcessor(q)
	assetStats := processors.NewAssetStatsProcessor(q, true)
	gen := randxdr.NewGenerator()

	var changes []xdr.LedgerEntryChange
	for i := 0; i < 1000; i++ {
		change := xdr.LedgerEntryChange{}
		shape := &gxdr.LedgerEntryChange{}
		gen.Next(
			shape,
			[]randxdr.Preset{
				{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
				{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
				{randxdr.FieldEquals("created.data.claimableBalance.ext.v1.flags"), randxdr.SetPositiveNum32},
				{
					randxdr.And(
						randxdr.IsPtr,
						randxdr.FieldMatches(regexp.MustCompile("created\\.data\\.claimableBalance\\.claimants.*notPredicate")),
					),
					randxdr.SetPtrToPresent,
				},
				{randxdr.FieldEquals("created.data.claimableBalance.amount"), randxdr.SetPositiveNum64},
				{randxdr.FieldEquals("created.data.claimableBalance.asset.alphaNum4.assetCode"), randxdr.SetAssetCode()},
				{randxdr.FieldEquals("created.data.claimableBalance.asset.alphaNum12.assetCode"), randxdr.SetAssetCode()},
			},
		)
		tt.Assert.NoError(gxdr.Convert(shape, &change))
		changes = append(changes, change)
	}

	for _, change := range ingest.GetChangesFromLedgerEntryChanges(changes) {
		if change.Type == xdr.LedgerEntryTypeClaimableBalance && change.Post != nil && change.Pre == nil {
			mockChangeReader.On("Read").Return(change, nil).Once()
		}
		tt.Assert.NoError(pp.ProcessChange(tt.Ctx, change))
		// asst stats can handle both trustlines or claimable balances
		// we only care about claimable balances in this function so skip trustlines
		if change.Type != xdr.LedgerEntryTypeTrustline {
			tt.Assert.NoError(assetStats.ProcessChange(tt.Ctx, change))
		}
	}

	tt.Assert.NoError(pp.Commit(tt.Ctx))
	tt.Assert.NoError(assetStats.Commit(tt.Ctx))
}

func TestStateVerifier(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{&db.Session{DB: tt.HorizonDB}}

	q.UpdateLastLedgerIngest(tt.Ctx, 63)

	mockChangeReader := &ingest.MockChangeReader{}
	addLiquidityPools(tt, q, mockChangeReader)
	addOffers(tt, q, mockChangeReader)
	addTrustLines(tt, q, mockChangeReader)
	addClaimableBalances(tt, q, mockChangeReader)

	mockChangeReader.On("Read").Return(ingest.Change{}, io.EOF).Once()
	mockChangeReader.On("Read").Return(ingest.Change{}, io.EOF).Once()
	mockChangeReader.On("Close").Return(nil).Once()

	mockHistoryAdapter := &mockHistoryArchiveAdapter{}
	mockHistoryAdapter.On("GetState", tt.Ctx, uint32(63)).Return(mockChangeReader, nil).Once()

	sys := &system{
		ctx:               tt.Ctx,
		historyQ:          q,
		historyAdapter:    mockHistoryAdapter,
		checkpointManager: historyarchive.NewCheckpointManager(64),
	}
	sys.initMetrics()

	tt.Assert.NoError(sys.verifyState(false))
	mockChangeReader.AssertExpectations(t)
	mockHistoryAdapter.AssertExpectations(t)
}
