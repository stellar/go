package processors

import (
	"context"
	stdio "io"
	"sort"

	"github.com/stellar/go/exp/ingest/io"
	ingestpipeline "github.com/stellar/go/exp/ingest/pipeline"
	"github.com/stellar/go/exp/support/pipeline"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/ingest/participants"
	"github.com/stellar/go/services/horizon/internal/toid"
	"github.com/stellar/go/support/errors"
)

// ParticipantsProcessor is a processor which ingests various participants
// from different sources (transactions, operations, etc)
type ParticipantsProcessor struct {
	ParticipantsQ history.QParticipants
}

type participant struct {
	accountID      int64
	transactionSet map[int64]struct{}
	operationSet   map[int64]struct{}
}

func (p *participant) addTransactionID(id int64) {
	if p.transactionSet == nil {
		p.transactionSet = map[int64]struct{}{}
	}
	p.transactionSet[id] = struct{}{}
}

func (p *participant) addOperationID(id int64) {
	if p.operationSet == nil {
		p.operationSet = map[int64]struct{}{}
	}
	p.operationSet[id] = struct{}{}
}

func (p *ParticipantsProcessor) loadAccountIDs(participantSet map[string]participant) error {
	addresses := make([]string, 0, len(participantSet))
	for address := range participantSet {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)

	addressToID, err := p.ParticipantsQ.CreateExpAccounts(addresses)
	if err != nil {
		return errors.Wrap(err, "Could not create account ids")
	}

	for _, address := range addresses {
		id, ok := addressToID[address]
		if !ok {
			return errors.Errorf("no id found for account address %s", address)
		}

		participantForAddress := participantSet[address]
		participantForAddress.accountID = id
		participantSet[address] = participantForAddress
	}

	return nil
}

func (p *ParticipantsProcessor) addTransactionParticipants(
	participantSet map[string]participant,
	sequence uint32,
	transaction io.LedgerTransaction,
) error {
	transactionID := toid.New(int32(sequence), int32(transaction.Index), 0).ToInt64()
	transactionParticipants, err := participants.ForTransaction(
		&transaction.Envelope.Tx,
		&transaction.Meta,
		&transaction.FeeChanges,
	)
	if err != nil {
		return errors.Wrap(err, "Could not determine participants for transaction")
	}

	for _, participant := range transactionParticipants {
		address := participant.Address()
		entry := participantSet[address]
		entry.addTransactionID(transactionID)
		participantSet[address] = entry
	}

	return nil
}

func (p *ParticipantsProcessor) addOperationsParticipants(
	participantSet map[string]participant,
	sequence uint32,
	transaction io.LedgerTransaction,
) error {
	participants, err := operationsParticipants(transaction, sequence)
	if err != nil {
		return errors.Wrap(err, "could not determine operation participants")
	}

	for operationID, p := range participants {
		for _, participant := range p {
			address := participant.Address()
			entry := participantSet[address]
			entry.addOperationID(operationID)
			participantSet[address] = entry
		}
	}

	return nil
}

func (p *ParticipantsProcessor) insertDBTransactionParticipants(participantSet map[string]participant) error {
	batch := p.ParticipantsQ.NewTransactionParticipantsBatchInsertBuilder(maxBatchSize)

	for _, entry := range participantSet {
		for transactionID := range entry.transactionSet {
			if err := batch.Add(transactionID, entry.accountID); err != nil {
				return errors.Wrap(err, "Could not insert transaction participant in db")
			}
		}
	}

	if err := batch.Exec(); err != nil {
		return errors.Wrap(err, "Could not flush transaction participants to db")
	}
	return nil
}

func (p *ParticipantsProcessor) insertDBOperationsParticipants(participantSet map[string]participant) error {
	batch := p.ParticipantsQ.NewOperationParticipantBatchInsertBuilder(maxBatchSize)

	for _, entry := range participantSet {
		for operationID := range entry.operationSet {
			if err := batch.Add(operationID, entry.accountID); err != nil {
				return errors.Wrap(err, "could not insert operation participant in db")
			}
		}
	}

	if err := batch.Exec(); err != nil {
		return errors.Wrap(err, "could not flush operation participants to db")
	}
	return nil
}

func (p *ParticipantsProcessor) ProcessLedger(ctx context.Context, store *pipeline.Store, r io.LedgerReader, w io.LedgerWriter) (err error) {
	defer func() {
		// io.LedgerReader.Close() returns error if upgrade changes have not
		// been processed so it's worth checking the error.
		closeErr := r.Close()
		// Do not overwrite the previous error
		if err == nil {
			err = closeErr
		}
	}()
	defer w.Close()
	r.IgnoreUpgradeChanges()

	// Exit early if not ingesting into a DB
	if v := ctx.Value(IngestUpdateDatabase); v == nil {
		return nil
	}

	participantSet := map[string]participant{}
	sequence := r.GetSequence()

	for {
		var transaction io.LedgerTransaction
		transaction, err = r.Read()
		if err != nil {
			if err == stdio.EOF {
				break
			} else {
				return err
			}
		}

		err = p.addTransactionParticipants(participantSet, sequence, transaction)
		if err != nil {
			return err
		}

		err = p.addOperationsParticipants(participantSet, sequence, transaction)
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		default:
			continue
		}
	}

	if len(participantSet) > 0 {
		if err = p.loadAccountIDs(participantSet); err != nil {
			return err
		}

		if err = p.insertDBTransactionParticipants(participantSet); err != nil {
			return err
		}

		if err = p.insertDBOperationsParticipants(participantSet); err != nil {
			return err
		}
	}

	// use an older lookup sequence because the experimental ingestion system and the
	// legacy ingestion system might not be in sync
	if sequence > 10 {
		checkSequence := int32(sequence - 10)
		var valid bool
		valid, err = p.ParticipantsQ.CheckExpParticipants(checkSequence)
		if err != nil {
			log.WithField("sequence", checkSequence).WithError(err).
				Error("Could not compare participants for ledger")
			return nil
		}

		if !valid {
			log.WithField("sequence", checkSequence).
				Error("participants do not match")
		}
	}

	return nil
}

func (p *ParticipantsProcessor) Name() string {
	return "ParticipantsProcessor"
}

func (p *ParticipantsProcessor) Reset() {}

var _ ingestpipeline.LedgerProcessor = &ParticipantsProcessor{}
