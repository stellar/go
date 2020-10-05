package history

// Add adds a new offer entry to the batch.
func (i *offersBatchInsertBuilder) Add(offer Offer) error {
	return i.builder.RowStruct(offer)
}

func (i *offersBatchInsertBuilder) Exec() error {
	return i.builder.Exec()
}
