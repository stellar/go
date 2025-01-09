package transaction

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/stellar/go/exp/xdrill/ledger"
	"github.com/stellar/go/exp/xdrill/utils"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/toid"
	"github.com/stellar/go/xdr"
)

func Hash(t ingest.LedgerTransaction) string {
	return utils.HashToHexString(t.Result.TransactionHash)
}

func Index(t ingest.LedgerTransaction) uint32 {
	return uint32(t.Index)
}

func ID(t ingest.LedgerTransaction, l xdr.LedgerCloseMeta) int64 {
	return toid.New(int32(ledger.Sequence(l)), int32(t.Index), 0).ToInt64()
}

func Account(t ingest.LedgerTransaction) (string, error) {
	return utils.GetAccountAddressFromMuxedAccount(t.Envelope.SourceAccount())
}

func AccountSequence(t ingest.LedgerTransaction) int64 {
	return t.Envelope.SeqNum()
}

func MaxFee(t ingest.LedgerTransaction) uint32 {
	return t.Envelope.Fee()
}

func FeeCharged(t ingest.LedgerTransaction, l xdr.LedgerCloseMeta) *int64 {
	// Any Soroban Fee Bump transactions before P21 will need the below logic to calculate the correct feeCharged
	// Protocol 20 contained a bug where the feeCharged was incorrectly calculated but was fixed for
	// Protocol 21 with https://github.com/stellar/stellar-core/issues/4188
	var result int64
	_, ok := getSorobanData(t)
	if ok {
		if ledger.LedgerVersion(l) < 21 && t.Envelope.Type == xdr.EnvelopeTypeEnvelopeTypeTxFeeBump {
			resourceFeeRefund := SorobanResourceFeeRefund(t)
			inclusionFeeCharged := SorobanInclusionFeeCharged(t)
			result = int64(t.Result.Result.FeeCharged) - *resourceFeeRefund + *inclusionFeeCharged
			return &result
		}
	}

	result = int64(t.Result.Result.FeeCharged)

	return &result
}

func OperationCount(t ingest.LedgerTransaction) uint32 {
	return uint32(len(t.Envelope.Operations()))
}

func Memo(t ingest.LedgerTransaction) string {
	memoObject := t.Envelope.Memo()
	memoContents := ""
	switch xdr.MemoType(memoObject.Type) {
	case xdr.MemoTypeMemoText:
		memoContents = memoObject.MustText()
	case xdr.MemoTypeMemoId:
		memoContents = strconv.FormatUint(uint64(memoObject.MustId()), 10)
	case xdr.MemoTypeMemoHash:
		hash := memoObject.MustHash()
		memoContents = base64.StdEncoding.EncodeToString(hash[:])
	case xdr.MemoTypeMemoReturn:
		hash := memoObject.MustRetHash()
		memoContents = base64.StdEncoding.EncodeToString(hash[:])
	}

	return memoContents
}

func MemoType(t ingest.LedgerTransaction) string {
	memoObject := t.Envelope.Memo()
	return memoObject.Type.String()
}

func TimeBounds(t ingest.LedgerTransaction) (*string, error) {
	timeBounds := t.Envelope.TimeBounds()
	if timeBounds == nil {
		return nil, nil
	}

	if timeBounds.MaxTime < timeBounds.MinTime && timeBounds.MaxTime != 0 {
		return nil, fmt.Errorf("the max time is earlier than the min time")
	}

	var result string
	if timeBounds.MaxTime == 0 {
		result = fmt.Sprintf("[%d,)", timeBounds.MinTime)
		return &result, nil
	}

	result = fmt.Sprintf("[%d,%d)", timeBounds.MinTime, timeBounds.MaxTime)

	return &result, nil
}

func LedgerBounds(t ingest.LedgerTransaction) *string {
	ledgerBounds := t.Envelope.LedgerBounds()
	if ledgerBounds == nil {
		return nil
	}

	result := fmt.Sprintf("[%d,%d)", int64(ledgerBounds.MinLedger), int64(ledgerBounds.MaxLedger))

	return &result
}

func MinSequence(t ingest.LedgerTransaction) *int64 {
	return t.Envelope.MinSeqNum()
}

func MinSequenceAge(t ingest.LedgerTransaction) *int64 {
	minSequenceAge := t.Envelope.MinSeqAge()
	if minSequenceAge == nil {
		return nil
	}

	minSequenceAgeInt64 := int64(*minSequenceAge)
	return &minSequenceAgeInt64
}

func MinSequenceLedgerGap(t ingest.LedgerTransaction) *int64 {
	minSequenceLedgerGap := t.Envelope.MinSeqLedgerGap()
	result := int64(*minSequenceLedgerGap)
	return &result
}

func getSorobanData(t ingest.LedgerTransaction) (sorobanData xdr.SorobanTransactionData, ok bool) {
	switch t.Envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		return t.Envelope.V1.Tx.Ext.GetSorobanData()
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		return t.Envelope.FeeBump.Tx.InnerTx.V1.Tx.Ext.GetSorobanData()
	}

	return
}

func SorobanResourceFee(t ingest.LedgerTransaction) *int64 {
	sorobanData, ok := getSorobanData(t)
	if !ok {
		return nil
	}

	result := int64(sorobanData.ResourceFee)
	return &result
}

func SorobanResourcesInstructions(t ingest.LedgerTransaction) *uint32 {
	sorobanData, ok := getSorobanData(t)
	if !ok {
		return nil
	}

	result := uint32(sorobanData.Resources.Instructions)
	return &result
}

func SorobanResourcesReadBytes(t ingest.LedgerTransaction) *uint32 {
	sorobanData, ok := getSorobanData(t)
	if !ok {
		return nil
	}

	result := uint32(sorobanData.Resources.ReadBytes)
	return &result
}

func SorobanResourcesWriteBytes(t ingest.LedgerTransaction) *uint32 {
	sorobanData, ok := getSorobanData(t)
	if !ok {
		return nil
	}

	result := uint32(sorobanData.Resources.WriteBytes)
	return &result
}

func InclusionFeeBid(t ingest.LedgerTransaction) *int64 {
	resourceFee := SorobanResourceFee(t)
	if resourceFee == nil {
		return nil
	}

	result := int64(t.Envelope.Fee()) - *resourceFee
	return &result
}

func getFeeAccountAddress(t ingest.LedgerTransaction) (feeAccountAddress string) {
	switch t.Envelope.Type {
	case xdr.EnvelopeTypeEnvelopeTypeTx:
		sourceAccount := t.Envelope.SourceAccount()
		feeAccountAddress = sourceAccount.Address()
	case xdr.EnvelopeTypeEnvelopeTypeTxFeeBump:
		feeBumpAccount := t.Envelope.FeeBumpAccount()
		feeAccountAddress = feeBumpAccount.Address()
	}

	return
}

func SorobanInclusionFeeCharged(t ingest.LedgerTransaction) *int64 {
	resourceFee := SorobanResourceFee(t)
	if resourceFee == nil {
		return nil
	}

	accountBalanceStart, accountBalanceEnd := utils.GetAccountBalanceFromLedgerEntryChanges(t.FeeChanges, getFeeAccountAddress(t))
	initialFeeCharged := accountBalanceStart - accountBalanceEnd
	result := initialFeeCharged - *resourceFee

	return &result
}

func SorobanResourceFeeRefund(t ingest.LedgerTransaction) *int64 {
	meta, ok := t.UnsafeMeta.GetV3()
	if !ok {
		return nil
	}

	accountBalanceStart, accountBalanceEnd := utils.GetAccountBalanceFromLedgerEntryChanges(meta.TxChangesAfter, getFeeAccountAddress(t))
	result := accountBalanceEnd - accountBalanceStart

	return &result
}

func SorobanTotalNonRefundableResourceFeeCharged(t ingest.LedgerTransaction) *int64 {
	meta, ok := t.UnsafeMeta.GetV3()
	if !ok {
		return nil
	}

	switch meta.SorobanMeta.Ext.V {
	case 1:
		result := int64(meta.SorobanMeta.Ext.V1.TotalNonRefundableResourceFeeCharged)
		return &result
	}

	return nil
}

func SorobanTotalRefundableResourceFeeCharged(t ingest.LedgerTransaction) *int64 {
	meta, ok := t.UnsafeMeta.GetV3()
	if !ok {
		return nil
	}

	switch meta.SorobanMeta.Ext.V {
	case 1:
		result := int64(meta.SorobanMeta.Ext.V1.TotalRefundableResourceFeeCharged)
		return &result
	}

	return nil
}

func SorobanRentFeeCharged(t ingest.LedgerTransaction) *int64 {
	meta, ok := t.UnsafeMeta.GetV3()
	if !ok {
		return nil
	}

	switch meta.SorobanMeta.Ext.V {
	case 1:
		result := int64(meta.SorobanMeta.Ext.V1.RentFeeCharged)
		return &result
	}

	return nil
}

func ResultCode(t ingest.LedgerTransaction) string {
	return t.Result.Result.Result.Code.String()
}

func Signers(t ingest.LedgerTransaction) (signers []string) {
	if t.Envelope.IsFeeBump() {
		signers, _ = utils.GetTxSigners(t.Envelope.FeeBump.Signatures)
		return
	}

	signers, _ = utils.GetTxSigners(t.Envelope.Signatures())

	return
}

func AccountMuxed(t ingest.LedgerTransaction) *string {
	sourceAccount := t.Envelope.SourceAccount()
	if sourceAccount.Type != xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		return nil
	}

	result := sourceAccount.Address()

	return &result
}

func FeeAccount(t ingest.LedgerTransaction) *string {
	if !t.Envelope.IsFeeBump() {
		return nil
	}

	feeBumpAccount := t.Envelope.FeeBumpAccount()
	feeAccount := feeBumpAccount.ToAccountId()
	result := feeAccount.Address()

	return &result
}

func FeeAccountMuxed(t ingest.LedgerTransaction) *string {
	if !t.Envelope.IsFeeBump() {
		return nil
	}

	feeBumpAccount := t.Envelope.FeeBumpAccount()
	if feeBumpAccount.Type != xdr.CryptoKeyTypeKeyTypeMuxedEd25519 {
		return nil
	}

	result := feeBumpAccount.Address()

	return &result
}

func InnerTransactionHash(t ingest.LedgerTransaction) *string {
	if !t.Envelope.IsFeeBump() {
		return nil
	}

	innerHash := t.Result.InnerHash()
	result := hex.EncodeToString(innerHash[:])

	return &result
}

func NewMaxFee(t ingest.LedgerTransaction) *uint32 {
	if !t.Envelope.IsFeeBump() {
		return nil
	}

	newMaxFee := uint32(t.Envelope.FeeBumpFee())
	return &newMaxFee
}

func Successful(t ingest.LedgerTransaction) bool {
	return t.Result.Successful()
}
