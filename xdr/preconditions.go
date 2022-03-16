package xdr

// UpgradePrecondition converts the old-style optional union into a new
// (post-CAP-21) xdr.Preconditions structure. This lets you avoid mucking around
// with types and transparently convert from nil/TimeBounds to the appropriate
// Preconditions struct.
func UpgradePrecondition(timebounds *TimeBounds) Preconditions {
	cond := Preconditions{Type: PreconditionTypePrecondNone}
	if timebounds != nil {
		cond.Type = PreconditionTypePrecondTime
		cond.TimeBounds = timebounds
	}
	return cond
}

func GetTimebounds(tx *Transaction) *TimeBounds {
	if tx == nil {
		return nil
	}
	switch tx.Cond.Type {
	case PreconditionTypePrecondNone:
		return nil
	case PreconditionTypePrecondTime:
		return tx.Cond.TimeBounds
	case PreconditionTypePrecondV2:
		return tx.Cond.V2.TimeBounds
	}
	return nil
}
