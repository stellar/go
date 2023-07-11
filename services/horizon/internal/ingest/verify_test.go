package ingest

import (
	"database/sql"
	"io"
	"math/rand"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/stellar/go/gxdr"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/randxdr"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/processors"
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

func genAccount(tt *test.T, gen randxdr.Generator) xdr.LedgerEntryChange {
	change := xdr.LedgerEntryChange{}
	shape := &gxdr.LedgerEntryChange{}
	numSigners := uint32(rand.Int31n(xdr.MaxSigners))
	gen.Next(
		shape,
		[]randxdr.Preset{
			{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
			{randxdr.FieldEquals("created.data.type"), randxdr.SetU32(gxdr.ACCOUNT.GetU32())},
			{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.account.seqNum"), randxdr.SetPositiveNum64},
			{randxdr.FieldEquals("created.data.account.numSubEntries"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.account.balance"), randxdr.SetPositiveNum64},
			{randxdr.FieldEquals("created.data.account.homeDomain"), randxdr.SetPrintableASCII},
			{randxdr.FieldEquals("created.data.account.flags"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.account.signers"), randxdr.SetVecLen(numSigners)},
			{randxdr.FieldMatches(regexp.MustCompile("created\\.data\\.account\\.signers.*weight")), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.account.ext.v1.liabilities.selling"), randxdr.SetPositiveNum64},
			{randxdr.FieldEquals("created.data.account.ext.v1.liabilities.buying"), randxdr.SetPositiveNum64},
			{randxdr.FieldEquals("created.data.account.ext.v1.ext.v2.numSponsoring"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.account.ext.v1.ext.v2.numSponsored"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.account.ext.v1.ext.v2.signerSponsoringIDs"), randxdr.SetVecLen(numSigners)},
			{randxdr.FieldEquals("created.data.account.ext.v1.ext.v2.ext.v3.seqLedger"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.account.ext.v1.ext.v2.ext.v3.seqTime"), randxdr.SetPositiveNum64},
		},
	)

	tt.Assert.NoError(gxdr.Convert(shape, &change))
	return change
}

func genAccountData(tt *test.T, gen randxdr.Generator) xdr.LedgerEntryChange {
	change := xdr.LedgerEntryChange{}
	shape := &gxdr.LedgerEntryChange{}
	gen.Next(
		shape,
		[]randxdr.Preset{
			{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
			{randxdr.FieldEquals("created.data.type"), randxdr.SetU32(gxdr.DATA.GetU32())},
			{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.data.dataName"), randxdr.SetPrintableASCII},
		},
	)

	tt.Assert.NoError(gxdr.Convert(shape, &change))
	return change
}

func genOffer(tt *test.T, gen randxdr.Generator) xdr.LedgerEntryChange {
	change := xdr.LedgerEntryChange{}
	shape := &gxdr.LedgerEntryChange{}
	gen.Next(
		shape,
		[]randxdr.Preset{
			{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
			{randxdr.FieldEquals("created.data.type"), randxdr.SetU32(gxdr.OFFER.GetU32())},
			{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.offer.amount"), randxdr.SetPositiveNum64},
			{randxdr.FieldEquals("created.data.offer.price.n"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.offer.price.d"), randxdr.SetPositiveNum32},
		},
	)
	tt.Assert.NoError(gxdr.Convert(shape, &change))
	return change
}

func genLiquidityPool(tt *test.T, gen randxdr.Generator) xdr.LedgerEntryChange {
	change := xdr.LedgerEntryChange{}
	shape := &gxdr.LedgerEntryChange{}
	gen.Next(
		shape,
		[]randxdr.Preset{
			{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
			// liquidity pools cannot be sponsored
			{randxdr.FieldEquals("created.ext.v1.sponsoringID"), randxdr.SetPtr(false)},
			{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.type"), randxdr.SetU32(gxdr.LIQUIDITY_POOL.GetU32())},
			{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.params.fee"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.params.assetA.alphaNum4.assetCode"), randxdr.SetAssetCode},
			{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.params.assetA.alphaNum12.assetCode"), randxdr.SetAssetCode},
			{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.params.assetB.alphaNum4.assetCode"), randxdr.SetAssetCode},
			{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.params.assetB.alphaNum12.assetCode"), randxdr.SetAssetCode},
			{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.reserveA"), randxdr.SetPositiveNum64},
			{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.reserveB"), randxdr.SetPositiveNum64},
			{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.totalPoolShares"), randxdr.SetPositiveNum64},
			{randxdr.FieldEquals("created.data.liquidityPool.body.constantProduct.poolSharesTrustLineCount"), randxdr.SetPositiveNum64},
		},
	)
	tt.Assert.NoError(gxdr.Convert(shape, &change))
	return change
}

func genTrustLine(tt *test.T, gen randxdr.Generator, extra ...randxdr.Preset) xdr.LedgerEntryChange {
	change := xdr.LedgerEntryChange{}
	shape := &gxdr.LedgerEntryChange{}
	presets := []randxdr.Preset{
		{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
		{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
		{randxdr.FieldEquals("created.data.type"), randxdr.SetU32(gxdr.TRUSTLINE.GetU32())},
		{randxdr.FieldEquals("created.data.trustLine.flags"), randxdr.SetPositiveNum32},
		{randxdr.FieldEquals("created.data.trustLine.asset.alphaNum4.assetCode"), randxdr.SetAssetCode},
		{randxdr.FieldEquals("created.data.trustLine.asset.alphaNum12.assetCode"), randxdr.SetAssetCode},
		{randxdr.FieldEquals("created.data.trustLine.balance"), randxdr.SetPositiveNum64},
		{randxdr.FieldEquals("created.data.trustLine.limit"), randxdr.SetPositiveNum64},
		{randxdr.FieldEquals("created.data.trustLine.ext.v1.liabilities.selling"), randxdr.SetPositiveNum64},
		{randxdr.FieldEquals("created.data.trustLine.ext.v1.liabilities.buying"), randxdr.SetPositiveNum64},
	}
	presets = append(presets, extra...)
	gen.Next(shape, presets)
	tt.Assert.NoError(gxdr.Convert(shape, &change))
	return change
}

func genClaimableBalance(tt *test.T, gen randxdr.Generator) xdr.LedgerEntryChange {
	change := xdr.LedgerEntryChange{}
	shape := &gxdr.LedgerEntryChange{}
	gen.Next(
		shape,
		[]randxdr.Preset{
			{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
			{randxdr.FieldEquals("created.lastModifiedLedgerSeq"), randxdr.SetPositiveNum32},
			{randxdr.FieldEquals("created.data.type"), randxdr.SetU32(gxdr.CLAIMABLE_BALANCE.GetU32())},
			{randxdr.FieldEquals("created.data.claimableBalance.ext.v1.flags"), randxdr.SetPositiveNum32},
			{
				randxdr.And(
					randxdr.IsPtr,
					randxdr.FieldMatches(regexp.MustCompile("created\\.data\\.claimableBalance\\.claimants.*notPredicate")),
				),
				randxdr.SetPtr(true),
			},
			{randxdr.FieldEquals("created.data.claimableBalance.amount"), randxdr.SetPositiveNum64},
			{randxdr.FieldEquals("created.data.claimableBalance.asset.alphaNum4.assetCode"), randxdr.SetAssetCode},
			{randxdr.FieldEquals("created.data.claimableBalance.asset.alphaNum12.assetCode"), randxdr.SetAssetCode},
		},
	)
	tt.Assert.NoError(gxdr.Convert(shape, &change))
	return change
}

func genContractCode(tt *test.T, gen randxdr.Generator) xdr.LedgerEntryChange {
	change := xdr.LedgerEntryChange{}
	shape := &gxdr.LedgerEntryChange{}
	gen.Next(
		shape,
		[]randxdr.Preset{
			{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
			{randxdr.FieldEquals("created.data.type"), randxdr.SetU32(gxdr.CONTRACT_CODE.GetU32())},
			//{randxdr.FieldEquals("created.data.contractcode.body.bodytype"), randxdr.SetU32(xdr.Body)},
		},
	)
	tt.Assert.NoError(gxdr.Convert(shape, &change))
	return change
}

func genConfigSetting(tt *test.T, gen randxdr.Generator) xdr.LedgerEntryChange {
	change := xdr.LedgerEntryChange{}
	shape := &gxdr.LedgerEntryChange{}
	gen.Next(
		shape,
		[]randxdr.Preset{
			{randxdr.FieldEquals("type"), randxdr.SetU32(gxdr.LEDGER_ENTRY_CREATED.GetU32())},
			{randxdr.FieldEquals("created.data.type"), randxdr.SetU32(gxdr.CONFIG_SETTING.GetU32())},
		},
	)
	tt.Assert.NoError(gxdr.Convert(shape, &change))
	return change
}

func genAssetContractMetadata(tt *test.T, gen randxdr.Generator) []xdr.LedgerEntryChange {
	assetPreset := randxdr.Preset{
		randxdr.FieldEquals("created.data.trustLine.asset.type"),
		randxdr.SetU32(
			gxdr.ASSET_TYPE_CREDIT_ALPHANUM4.GetU32(),
			gxdr.ASSET_TYPE_CREDIT_ALPHANUM12.GetU32(),
		),
	}
	trustline := genTrustLine(tt, gen, assetPreset)
	assetContractMetadata := assetContractMetadataFromTrustline(tt, trustline)

	otherTrustline := genTrustLine(tt, gen, assetPreset)
	otherAssetContractMetadata := assetContractMetadataFromTrustline(tt, otherTrustline)

	return []xdr.LedgerEntryChange{
		assetContractMetadata,
		trustline,
		balanceContractDataFromTrustline(tt, trustline),
		otherAssetContractMetadata,
		balanceContractDataFromTrustline(tt, otherTrustline),
		balanceContractDataFromTrustline(tt, genTrustLine(tt, gen, assetPreset)),
	}
}

func assetContractMetadataFromTrustline(tt *test.T, trustline xdr.LedgerEntryChange) xdr.LedgerEntryChange {
	contractID, err := trustline.Created.Data.MustTrustLine().Asset.ToAsset().ContractID("")
	tt.Assert.NoError(err)
	var assetType xdr.AssetType
	var code, issuer string
	tt.Assert.NoError(
		trustline.Created.Data.MustTrustLine().Asset.Extract(&assetType, &code, &issuer),
	)
	ledgerData, err := processors.AssetToContractData(assetType == xdr.AssetTypeAssetTypeNative, code, issuer, contractID)
	tt.Assert.NoError(err)
	assetContractMetadata := xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
		Created: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: trustline.Created.LastModifiedLedgerSeq,
			Data:                  ledgerData,
		},
	}
	return assetContractMetadata
}

func balanceContractDataFromTrustline(tt *test.T, trustline xdr.LedgerEntryChange) xdr.LedgerEntryChange {
	contractID, err := trustline.Created.Data.MustTrustLine().Asset.ToAsset().ContractID("")
	tt.Assert.NoError(err)
	var assetType xdr.AssetType
	var code, issuer string
	trustlineData := trustline.Created.Data.MustTrustLine()
	tt.Assert.NoError(
		trustlineData.Asset.Extract(&assetType, &code, &issuer),
	)
	assetContractMetadata := xdr.LedgerEntryChange{
		Type: xdr.LedgerEntryChangeTypeLedgerEntryCreated,
		Created: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: trustline.Created.LastModifiedLedgerSeq,
			Data:                  processors.BalanceToContractData(contractID, *trustlineData.AccountId.Ed25519, uint64(trustlineData.Balance)),
		},
	}
	return assetContractMetadata
}

func TestStateVerifierLockBusy(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{&db.Session{DB: tt.HorizonDB}}

	checkpointLedger := uint32(63)
	changeProcessor := buildChangeProcessor(q, &ingest.StatsChangeProcessor{}, ledgerSource, checkpointLedger, "")

	gen := randxdr.NewGenerator()
	var changes []xdr.LedgerEntryChange
	for i := 0; i < 10; i++ {
		changes = append(changes,
			genLiquidityPool(tt, gen),
			genClaimableBalance(tt, gen),
			genOffer(tt, gen),
			genTrustLine(tt, gen),
			genAccount(tt, gen),
			genAccountData(tt, gen),
		)
	}
	for _, change := range ingest.GetChangesFromLedgerEntryChanges(changes) {
		tt.Assert.NoError(changeProcessor.ProcessChange(tt.Ctx, change))
	}
	tt.Assert.NoError(changeProcessor.Commit(tt.Ctx))

	q.UpdateLastLedgerIngest(tt.Ctx, checkpointLedger)

	mockHistoryAdapter := &mockHistoryArchiveAdapter{}
	sys := &system{
		ctx:                          tt.Ctx,
		historyQ:                     q,
		historyAdapter:               mockHistoryAdapter,
		runStateVerificationOnLedger: ledgerEligibleForStateVerification(64, 1),
		config:                       Config{StateVerificationTimeout: time.Hour},
	}
	sys.initMetrics()

	otherQ := &history.Q{q.Clone()}
	tt.Assert.NoError(otherQ.BeginTx(tt.Ctx, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	}))
	ok, err := otherQ.TryStateVerificationLock(tt.Ctx)
	tt.Assert.NoError(err)
	tt.Assert.True(ok)

	tt.Assert.NoError(sys.verifyState(false))
	mockHistoryAdapter.AssertExpectations(t)

	tt.Assert.NoError(otherQ.Rollback())
}

func TestStateVerifier(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &history.Q{&db.Session{DB: tt.HorizonDB}}

	checkpointLedger := uint32(63)
	changeProcessor := buildChangeProcessor(q, &ingest.StatsChangeProcessor{}, ledgerSource, checkpointLedger, "")
	mockChangeReader := &ingest.MockChangeReader{}

	gen := randxdr.NewGenerator()
	var changes []xdr.LedgerEntryChange
	for i := 0; i < 100; i++ {
		changes = append(changes,
			genLiquidityPool(tt, gen),
			genClaimableBalance(tt, gen),
			genOffer(tt, gen),
			genTrustLine(tt, gen),
			genAccount(tt, gen),
			genAccountData(tt, gen),
			genContractCode(tt, gen),
			genConfigSetting(tt, gen),
		)
		changes = append(changes, genAssetContractMetadata(tt, gen)...)
	}

	coverage := map[xdr.LedgerEntryType]int{}
	for _, change := range ingest.GetChangesFromLedgerEntryChanges(changes) {
		mockChangeReader.On("Read").Return(change, nil).Once()
		tt.Assert.NoError(changeProcessor.ProcessChange(tt.Ctx, change))
		coverage[change.Type]++
	}
	tt.Assert.NoError(changeProcessor.Commit(tt.Ctx))
	tt.Assert.Equal(len(xdr.LedgerEntryTypeMap), len(coverage))

	q.UpdateLastLedgerIngest(tt.Ctx, checkpointLedger)

	mockChangeReader.On("Read").Return(ingest.Change{}, io.EOF).Twice()
	mockChangeReader.On("Close").Return(nil).Once()

	mockHistoryAdapter := &mockHistoryArchiveAdapter{}
	mockHistoryAdapter.On("GetState", mock.AnythingOfType("*context.timerCtx"), uint32(checkpointLedger)).Return(mockChangeReader, nil).Once()

	sys := &system{
		ctx:                          tt.Ctx,
		historyQ:                     q,
		historyAdapter:               mockHistoryAdapter,
		runStateVerificationOnLedger: ledgerEligibleForStateVerification(64, 1),
		config:                       Config{StateVerificationTimeout: time.Hour},
	}
	sys.initMetrics()

	tt.Assert.NoError(sys.verifyState(false))
	mockChangeReader.AssertExpectations(t)
	mockHistoryAdapter.AssertExpectations(t)
}
