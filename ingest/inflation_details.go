package ingest

type InflationDetail struct{}

func (o *LedgerOperation) InflationDetails() (InflationDetail, error) {
	return InflationDetail{}, nil
}
