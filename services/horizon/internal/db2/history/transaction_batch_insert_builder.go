package history

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/services/horizon/internal/db2/sqx"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/services/horizon/internal/utf8"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/xdr"
)

// TransactionBatchInsertBuilder is used to insert transactions into the
// history_transactions table
type TransactionBatchInsertBuilder interface {
	Add(transaction io.LedgerTransaction, sequence uint32) error
	Exec() error
}

// transactionBatchInsertBuilder is a simple wrapper around db.BatchInsertBuilder
type transactionBatchInsertBuilder struct {
	builder db.BatchInsertBuilder
}

// NewTransactionBatchInsertBuilder constructs a new TransactionBatchInsertBuilder instance
func (q *Q) NewTransactionBatchInsertBuilder(maxBatchSize int) TransactionBatchInsertBuilder {
	return &transactionBatchInsertBuilder{
		builder: db.BatchInsertBuilder{
			Table:        q.GetTable("history_transactions"),
			MaxBatchSize: maxBatchSize,
		},
	}
}

// Add adds a new transaction to the batch
func (i *transactionBatchInsertBuilder) Add(transaction io.LedgerTransaction, sequence uint32) error {
	row, err := transactionToRow(transaction, sequence)
	if err != nil {
		return err
	}

	return i.builder.RowStruct(row)
}

func (i *transactionBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}

func formatTimeBounds(transaction io.LedgerTransaction) interface{} {
	timeBounds := transaction.Envelope.TimeBounds()
	if timeBounds == nil {
		return nil
	}

	if timeBounds.MaxTime == 0 {
		return sq.Expr("int8range(?,?)", timeBounds.MinTime, nil)
	}

	maxTime := timeBounds.MaxTime
	if maxTime > math.MaxInt64 {
		maxTime = math.MaxInt64
	}

	return sq.Expr("int8range(?,?)", timeBounds.MinTime, maxTime)
}

func signatures(xdrSignatures []xdr.DecoratedSignature) []string {
	signatures := make([]string, len(xdrSignatures))
	for i, sig := range xdrSignatures {
		signatures[i] = base64.StdEncoding.EncodeToString(sig.Signature)
	}
	return signatures
}

func memoType(transaction io.LedgerTransaction) string {
	switch transaction.Envelope.Memo().Type {
	case xdr.MemoTypeMemoNone:
		return "none"
	case xdr.MemoTypeMemoText:
		return "text"
	case xdr.MemoTypeMemoId:
		return "id"
	case xdr.MemoTypeMemoHash:
		return "hash"
	case xdr.MemoTypeMemoReturn:
		return "return"
	default:
		panic(fmt.Errorf("invalid memo type: %v", transaction.Envelope.Memo().Type))
	}
}

func memo(transaction io.LedgerTransaction) null.String {
	var (
		value string
		valid bool
	)
	memo := transaction.Envelope.Memo()
	switch memo.Type {
	case xdr.MemoTypeMemoNone:
		value, valid = "", false
	case xdr.MemoTypeMemoText:
		scrubbed := utf8.Scrub(memo.MustText())
		notnull := strings.Join(strings.Split(scrubbed, "\x00"), "")
		value, valid = notnull, true
	case xdr.MemoTypeMemoId:
		value, valid = fmt.Sprintf("%d", memo.MustId()), true
	case xdr.MemoTypeMemoHash:
		hash := memo.MustHash()
		value, valid =
			base64.StdEncoding.EncodeToString(hash[:]),
			true
	case xdr.MemoTypeMemoReturn:
		hash := memo.MustRetHash()
		value, valid =
			base64.StdEncoding.EncodeToString(hash[:]),
			true
	default:
		panic(fmt.Errorf("invalid memo type: %v", memo.Type))
	}

	return null.NewString(value, valid)
}

type transactionRow struct {
	ID                   int64       `db:"id"`
	TransactionHash      string      `db:"transaction_hash"`
	LedgerSequence       int32       `db:"ledger_sequence"`
	ApplicationOrder     int32       `db:"application_order"`
	Account              string      `db:"account"`
	AccountSequence      string      `db:"account_sequence"`
	MaxFee               int64       `db:"max_fee"`
	FeeCharged           int64       `db:"fee_charged"`
	OperationCount       int32       `db:"operation_count"`
	TxEnvelope           string      `db:"tx_envelope"`
	TxResult             string      `db:"tx_result"`
	TxMeta               string      `db:"tx_meta"`
	TxFeeMeta            string      `db:"tx_fee_meta"`
	Signatures           interface{} `db:"signatures"`
	MemoType             string      `db:"memo_type"`
	Memo                 null.String `db:"memo"`
	TimeBounds           interface{} `db:"time_bounds"`
	CreatedAt            time.Time   `db:"created_at"`
	UpdatedAt            time.Time   `db:"updated_at"`
	Successful           bool        `db:"successful"`
	FeeAccount           null.String `db:"fee_account"`
	InnerTransactionHash null.String `db:"inner_transaction_hash"`
	NewMaxFee            null.Int    `db:"new_max_fee"`
	InnerSignatures      interface{} `db:"inner_signatures"`
}

func transactionToRow(transaction io.LedgerTransaction, sequence uint32) (transactionRow, error) {
	envelopeBase64, err := xdr.MarshalBase64(transaction.Envelope)
	if err != nil {
		return transactionRow{}, err
	}
	resultBase64, err := xdr.MarshalBase64(transaction.Result.Result)
	if err != nil {
		return transactionRow{}, err
	}
	metaBase64, err := xdr.MarshalBase64(transaction.Meta)
	if err != nil {
		return transactionRow{}, err
	}
	feeMetaBase64, err := xdr.MarshalBase64(transaction.FeeChanges)
	if err != nil {
		return transactionRow{}, err
	}

	sourceAccount := transaction.Envelope.SourceAccount().ToAccountId()
	t := transactionRow{
		ID:               toid.New(int32(sequence), int32(transaction.Index), 0).ToInt64(),
		TransactionHash:  hex.EncodeToString(transaction.Result.TransactionHash[:]),
		LedgerSequence:   int32(sequence),
		ApplicationOrder: int32(transaction.Index),
		Account:          sourceAccount.Address(),
		AccountSequence:  strconv.FormatInt(transaction.Envelope.SeqNum(), 10),
		MaxFee:           int64(transaction.Envelope.Fee()),
		FeeCharged:       int64(transaction.Result.Result.FeeCharged),
		OperationCount:   int32(len(transaction.Envelope.Operations())),
		TxEnvelope:       envelopeBase64,
		TxResult:         resultBase64,
		TxMeta:           metaBase64,
		TxFeeMeta:        feeMetaBase64,
		TimeBounds:       formatTimeBounds(transaction),
		MemoType:         memoType(transaction),
		Memo:             memo(transaction),
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
		Successful:       transaction.Result.Successful(),
	}
	if transaction.Envelope.IsFeeBump() {
		innerHash := transaction.Result.InnerHash()
		t.InnerTransactionHash = null.StringFrom(hex.EncodeToString(innerHash[:]))
		feeAccount := transaction.Envelope.FeeBumpAccount().ToAccountId()
		t.FeeAccount = null.StringFrom(feeAccount.Address())
		t.NewMaxFee = null.IntFrom(transaction.Envelope.FeeBumpFee())
		t.InnerSignatures = sqx.StringArray(signatures(transaction.Envelope.Signatures()))
		t.Signatures = sqx.StringArray(signatures(transaction.Envelope.FeeBumpSignatures()))
	} else {
		t.InnerTransactionHash = null.StringFromPtr(nil)
		t.FeeAccount = null.StringFromPtr(nil)
		t.NewMaxFee = null.IntFromPtr(nil)
		t.InnerSignatures = nil
		t.Signatures = sqx.StringArray(signatures(transaction.Envelope.Signatures()))
	}

	return t, nil
}
