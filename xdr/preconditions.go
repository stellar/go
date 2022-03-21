package xdr

// NewPreconditionsWithTimebounds constructs the simplest possible
// `Preconditions` instance given the (possibly empty) timebounds.
func NewPreconditionsWithTimeBounds(timebounds *TimeBounds) Preconditions {
	cond := Preconditions{Type: PreconditionTypePrecondNone}
	if timebounds != nil {
		cond.Type = PreconditionTypePrecondTime
		cond.TimeBounds = timebounds
	}
	return cond
}

// GetPreconditions normalizes and returns the preconditions from the transaction.
func GetPreconditions(tx *Transaction) PreconditionsV2 {
	if tx == nil {
		return PreconditionsV2{}
	}
	switch tx.Cond.Type {
	case PreconditionTypePrecondNone:
		return PreconditionsV2{}
	case PreconditionTypePrecondTime:
		return PreconditionsV2{
			TimeBounds: tx.Cond.TimeBounds,
		}
	case PreconditionTypePrecondV2:
		if tx.Cond.V2 == nil {
			return PreconditionsV2{}
		}
		return *tx.Cond.V2
	}
	return PreconditionsV2{}
}
