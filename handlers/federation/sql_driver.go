package federation

import "github.com/stellar/go/support/db"
import "github.com/stellar/go/support/errors"

// LookupRecord implements `Driver` by performing `drv.LookupRecordQuery`
// against `drv.DB` using the provided parameters
func (drv *SQLDriver) LookupRecord(name, domain string) (*Record, error) {
	drv.initDB()
	var result Record

	err := drv.db.GetRaw(&result, drv.LookupRecordQuery, name, domain)

	if drv.db.NoRows(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "db get")
	}

	return &result, nil
}

// LookupReverseRecord implements `ReverseDriver` by performing
// `drv.LookupReverseRecordQuery` against `drv.DB` using the provided parameter
func (drv *SQLDriver) LookupReverseRecord(
	accountid string,
) (*ReverseRecord, error) {
	drv.initDB()
	var result ReverseRecord

	err := drv.db.GetRaw(&result, drv.LookupReverseRecordQuery, accountid)

	if drv.db.NoRows(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "db get")
	}

	return &result, nil
}

var _ Driver = &SQLDriver{}
var _ ReverseDriver = &SQLDriver{}

func (drv *SQLDriver) initDB() {
	drv.init.Do(func() {
		drv.db = db.Wrap(drv.DB, drv.Dialect)
	})
}
