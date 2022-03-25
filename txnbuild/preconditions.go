package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// Preconditions is a container for all transaction preconditions.
type Preconditions struct {
	// Transaction is only valid during a certain time range.
	Timebounds TimeBounds
	// Transaction is valid for ledger numbers n such that minLedger <= n <
	// maxLedger (if maxLedger == 0, then only minLedger is checked)
	Ledgerbounds *LedgerBounds
	// If nil, the transaction is only valid when sourceAccount's sequence
	// number "N" is seqNum - 1. Otherwise, valid when N satisfies minSeqNum <=
	// N < tx.seqNum.
	MinSequenceNumber *int64
	// Transaction is valid if the current ledger time is at least
	// minSequenceNumberAge greater than the source account's seqTime.
	MinSequenceNumberAge xdr.Duration
	// Transaction is valid if the current ledger number is at least
	// minSequenceNumberLedgerGap greater than the source account's seqLedger.
	MinSequenceNumberLedgerGap uint32
	// Transaction is valid if there is a signature corresponding to every
	// Signer in this array, even if the signature is not otherwise required by
	// the source account or operations.
	ExtraSigners []xdr.SignerKey
}

// Validate ensures that all enabled preconditions are valid.
func (cond *Preconditions) Validate() error {
	var err error

	if err = cond.Timebounds.Validate(); err != nil {
		return err
	}

	if err = cond.Ledgerbounds.Validate(); err != nil {
		return err
	}

	if len(cond.ExtraSigners) > 2 {
		return errors.New("only 2 extra signers allowed")
	}

	return nil
}

// BuildXDR will create a precondition structure that varies depending on
// whether or not there are additional preconditions besides timebounds (which
// are required).
func (cond *Preconditions) BuildXDR() xdr.Preconditions {
	xdrCond := xdr.Preconditions{}
	xdrTimeBounds := xdr.TimeBounds{
		MinTime: xdr.TimePoint(cond.Timebounds.MinTime),
		MaxTime: xdr.TimePoint(cond.Timebounds.MaxTime),
	}

	// Only build PRECOND_V2 structure if we need to
	if cond.hasV2Conditions() {
		xdrPrecond := xdr.PreconditionsV2{
			TimeBounds:      &xdrTimeBounds,
			MinSeqAge:       cond.MinSequenceNumberAge,
			MinSeqLedgerGap: xdr.Uint32(cond.MinSequenceNumberLedgerGap),
			ExtraSigners:    cond.ExtraSigners, // should we copy?
		}

		// micro-optimization: if the ledgerbounds will always succeed, omit them
		if cond.Ledgerbounds != nil && !(cond.Ledgerbounds.MinLedger == 0 &&
			cond.Ledgerbounds.MaxLedger == 0) {
			xdrPrecond.LedgerBounds = &xdr.LedgerBounds{
				MinLedger: xdr.Uint32(cond.Ledgerbounds.MinLedger),
				MaxLedger: xdr.Uint32(cond.Ledgerbounds.MaxLedger),
			}
		}

		if cond.MinSequenceNumber != nil {
			seqNum := xdr.SequenceNumber(*cond.MinSequenceNumber)
			xdrPrecond.MinSeqNum = &seqNum
		}

		xdrCond.Type = xdr.PreconditionTypePrecondV2
		xdrCond.V2 = &xdrPrecond
	} else {
		xdrCond.Type = xdr.PreconditionTypePrecondTime
		xdrCond.TimeBounds = &xdrTimeBounds
	}

	return xdrCond
}

// FromXDR fills in the precondition structure from an xdr.Precondition.
func (cond *Preconditions) FromXDR(precondXdr xdr.Preconditions) {
	*cond = Preconditions{} // reset existing values

	switch precondXdr.Type {
	case xdr.PreconditionTypePrecondTime:
		cond.Timebounds = NewTimebounds(
			int64(precondXdr.MustTimeBounds().MinTime),
			int64(precondXdr.MustTimeBounds().MaxTime),
		)

	case xdr.PreconditionTypePrecondV2:
		inner := precondXdr.MustV2()

		if inner.TimeBounds != nil {
			cond.Timebounds = NewTimebounds(
				int64(inner.TimeBounds.MinTime),
				int64(inner.TimeBounds.MaxTime),
			)
		}

		if inner.LedgerBounds != nil {
			cond.Ledgerbounds = &LedgerBounds{
				MinLedger: uint32(inner.LedgerBounds.MinLedger),
				MaxLedger: uint32(inner.LedgerBounds.MaxLedger),
			}
		}

		if inner.MinSeqNum != nil {
			minSeqNum := int64(*inner.MinSeqNum)
			cond.MinSequenceNumber = &minSeqNum
		}

		cond.MinSequenceNumberAge = inner.MinSeqAge
		cond.MinSequenceNumberLedgerGap = uint32(inner.MinSeqLedgerGap)
		cond.ExtraSigners = append(cond.ExtraSigners, inner.ExtraSigners...)

	case xdr.PreconditionTypePrecondNone:
	default: // panic?
	}
}

// hasV2Conditions determines whether or not this has conditions on top of
// the (required) timebound precondition.
func (cond *Preconditions) hasV2Conditions() bool {
	return (cond.Ledgerbounds != nil ||
		cond.MinSequenceNumber != nil ||
		cond.MinSequenceNumberAge > xdr.Duration(0) ||
		cond.MinSequenceNumberLedgerGap > 0 ||
		len(cond.ExtraSigners) > 0)
}
