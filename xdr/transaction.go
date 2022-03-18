package xdr

// TimeBounds extracts the timebounds (if any) from the transaction's
// Preconditions.
func (tx *Transaction) TimeBounds() *TimeBounds {
	switch tx.Cond.Type {
	case PreconditionTypePrecondNone:
		return nil
	case PreconditionTypePrecondTime:
		return tx.Cond.TimeBounds
	case PreconditionTypePrecondV2:
		return tx.Cond.V2.TimeBounds
	default:
		panic("unsupported transaction type: " + tx.Cond.Type.String())
	}
}
