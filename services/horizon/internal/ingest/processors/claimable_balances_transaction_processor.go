package processors

import (
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type claimableBalance struct {
	internalID     int64 // Bigint auto-generated by postgres
	transactionSet map[int64]struct{}
	operationSet   map[int64]struct{}
}

func (b *claimableBalance) addTransactionID(id int64) {
	if b.transactionSet == nil {
		b.transactionSet = map[int64]struct{}{}
	}
	b.transactionSet[id] = struct{}{}
}

func (b *claimableBalance) addOperationID(id int64) {
	if b.operationSet == nil {
		b.operationSet = map[int64]struct{}{}
	}
	b.operationSet[id] = struct{}{}
}

type ClaimableBalancesTransactionProcessor struct {
	sequence            uint32
	claimableBalanceSet map[xdr.ClaimableBalanceId]claimableBalance
	qClaimableBalances  history.QHistoryClaimableBalances
}

func NewClaimableBalancesTransactionProcessor(Q history.QHistoryClaimableBalances, sequence uint32) *ClaimableBalancesTransactionProcessor {
	return &ClaimableBalancesTransactionProcessor{
		qClaimableBalances:  Q,
		sequence:            sequence,
		claimableBalanceSet: map[xdr.ClaimableBalanceId]claimableBalance{},
	}
}

func (p *ClaimableBalancesTransactionProcessor) ProcessTransaction(transaction ingest.LedgerTransaction) error {
	err := p.addTransactionClaimableBalances(p.claimableBalanceSet, p.sequence, transaction)
	if err != nil {
		return err
	}

	err = p.addOperationClaimableBalances(p.claimableBalanceSet, p.sequence, transaction)
	if err != nil {
		return err
	}

	return nil
}

func (p *ClaimableBalancesTransactionProcessor) addTransactionClaimableBalances(cbSet map[xdr.ClaimableBalanceId]claimableBalance, sequence uint32, transaction ingest.LedgerTransaction) error {
	transactionID := toid.New(int32(sequence), int32(transaction.Index), 0).ToInt64()
	transactionClaimableBalances, err := claimableBalancesForTransaction(
		sequence,
		transaction,
	)
	if err != nil {
		return errors.Wrap(err, "Could not determine claimable balances for transaction")
	}

	for _, cb := range transactionClaimableBalances {
		entry := cbSet[cb]
		entry.addTransactionID(transactionID)
		cbSet[cb] = entry
	}

	return nil
}

func claimableBalancesForTransaction(
	sequence uint32,
	transaction ingest.LedgerTransaction,
) ([]xdr.ClaimableBalanceId, error) {
	cbs := []xdr.ClaimableBalanceId{}
	c, err := claimableBalancesForMeta(transaction.Meta)
	if err != nil {
		return nil, err
	}
	cbs = append(cbs, c...)

	for opi, op := range transaction.Envelope.Operations() {
		operation := transactionOperationWrapper{
			index:          uint32(opi),
			transaction:    transaction,
			operation:      op,
			ledgerSequence: sequence,
		}

		c, err = operation.ClaimableBalances()
		if err != nil {
			return cbs, errors.Wrapf(err, "reading operation %v claimable balances", operation.ID())
		}
		cbs = append(cbs, c...)
	}

	return dedupeClaimableBalances(cbs)
}

func claimableBalancesForMeta(meta xdr.TransactionMeta) ([]xdr.ClaimableBalanceId, error) {
	var balances []xdr.ClaimableBalanceId
	if meta.Operations == nil {
		return balances, nil
	}

	for _, op := range *meta.Operations {
		var cbs []xdr.ClaimableBalanceId
		cbs, err := claimableBalancesForChanges(op.Changes)
		if err != nil {
			return nil, err
		}

		balances = append(balances, cbs...)
	}

	return balances, nil
}

func claimableBalancesForChanges(
	changes xdr.LedgerEntryChanges,
) ([]xdr.ClaimableBalanceId, error) {
	var cbs []xdr.ClaimableBalanceId

	for _, c := range changes {
		var cb *xdr.ClaimableBalanceId

		switch c.Type {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			cb = claimableBalanceForLedgerEntry(c.MustCreated())
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			cb = claimableBalanceForLedgerKey(c.MustRemoved())
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			cb = claimableBalanceForLedgerEntry(c.MustUpdated())
		case xdr.LedgerEntryChangeTypeLedgerEntryState:
			cb = claimableBalanceForLedgerEntry(c.MustState())
		default:
			return nil, errors.Errorf("Unknown change type: %s", c.Type)
		}

		if cb != nil {
			cbs = append(cbs, *cb)
		}
	}

	return cbs, nil
}

func claimableBalanceForLedgerEntry(le xdr.LedgerEntry) *xdr.ClaimableBalanceId {
	if le.Data.Type != xdr.LedgerEntryTypeClaimableBalance {
		return nil
	}
	id := le.Data.MustClaimableBalance().BalanceId
	return &id
}

func claimableBalanceForLedgerKey(lk xdr.LedgerKey) *xdr.ClaimableBalanceId {
	if lk.Type != xdr.LedgerEntryTypeClaimableBalance {
		return nil
	}
	id := lk.MustClaimableBalance().BalanceId
	return &id
}

func (p *ClaimableBalancesTransactionProcessor) addOperationClaimableBalances(cbSet map[xdr.ClaimableBalanceId]claimableBalance, sequence uint32, transaction ingest.LedgerTransaction) error {
	claimableBalances, err := operationsClaimableBalances(transaction, sequence)
	if err != nil {
		return errors.Wrap(err, "could not determine operation claimable balances")
	}

	for operationID, cbs := range claimableBalances {
		for _, cb := range cbs {
			entry := cbSet[cb]
			entry.addOperationID(operationID)
			cbSet[cb] = entry
		}
	}

	return nil
}

func operationsClaimableBalances(transaction ingest.LedgerTransaction, sequence uint32) (map[int64][]xdr.ClaimableBalanceId, error) {
	cbs := map[int64][]xdr.ClaimableBalanceId{}

	for opi, op := range transaction.Envelope.Operations() {
		operation := transactionOperationWrapper{
			index:          uint32(opi),
			transaction:    transaction,
			operation:      op,
			ledgerSequence: sequence,
		}

		cb, err := operation.ClaimableBalances()
		if err != nil {
			return cbs, errors.Wrapf(err, "reading operation %v claimable balances", operation.ID())
		}
		cbs[operation.ID()] = cb
	}

	return cbs, nil
}

func (p *ClaimableBalancesTransactionProcessor) Commit() error {
	if len(p.claimableBalanceSet) > 0 {
		if err := p.loadClaimableBalanceIDs(p.claimableBalanceSet); err != nil {
			return err
		}

		if err := p.insertDBTransactionClaimableBalances(p.claimableBalanceSet); err != nil {
			return err
		}

		if err := p.insertDBOperationsClaimableBalances(p.claimableBalanceSet); err != nil {
			return err
		}
	}

	return nil
}

func (p *ClaimableBalancesTransactionProcessor) loadClaimableBalanceIDs(claimableBalanceSet map[xdr.ClaimableBalanceId]claimableBalance) error {
	ids := make([]xdr.ClaimableBalanceId, 0, len(claimableBalanceSet))
	for id := range claimableBalanceSet {
		ids = append(ids, id)
	}

	toInternalID, err := p.qClaimableBalances.CreateHistoryClaimableBalances(ids, maxBatchSize)
	if err != nil {
		return errors.Wrap(err, "Could not create claimable balance ids")
	}

	for _, id := range ids {
		hexID, err := xdr.MarshalHex(id)
		if err != nil {
			return errors.New("error parsing BalanceID")
		}
		internalID, ok := toInternalID[hexID]
		if !ok {
			// TODO: Figure out the right way to convert the id to a string here. %v will be nonsense.
			return errors.Errorf("no internal id found for claimable balance %v", id)
		}

		cb := claimableBalanceSet[id]
		cb.internalID = internalID
		claimableBalanceSet[id] = cb
	}

	return nil
}

func (p ClaimableBalancesTransactionProcessor) insertDBTransactionClaimableBalances(claimableBalanceSet map[xdr.ClaimableBalanceId]claimableBalance) error {
	batch := p.qClaimableBalances.NewTransactionClaimableBalanceBatchInsertBuilder(maxBatchSize)

	for _, entry := range claimableBalanceSet {
		for transactionID := range entry.transactionSet {
			if err := batch.Add(transactionID, entry.internalID); err != nil {
				return errors.Wrap(err, "could not insert transaction claimable balance in db")
			}
		}
	}

	if err := batch.Exec(); err != nil {
		return errors.Wrap(err, "could not flush transaction claimable balances to db")
	}
	return nil
}

func (p ClaimableBalancesTransactionProcessor) insertDBOperationsClaimableBalances(claimableBalanceSet map[xdr.ClaimableBalanceId]claimableBalance) error {
	batch := p.qClaimableBalances.NewOperationClaimableBalanceBatchInsertBuilder(maxBatchSize)

	for _, entry := range claimableBalanceSet {
		for operationID := range entry.operationSet {
			if err := batch.Add(operationID, entry.internalID); err != nil {
				return errors.Wrap(err, "could not insert operation claimable balance in db")
			}
		}
	}

	if err := batch.Exec(); err != nil {
		return errors.Wrap(err, "could not flush operation claimable balances to db")
	}
	return nil
}
