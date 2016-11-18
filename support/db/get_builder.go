package db

import (
	"github.com/stellar/go/support/errors"
)

// Exec executes the query represented by the builder, populating the
// destination with the results returned by running the query against the
// current database session.
func (gb *GetBuilder) Exec() error {
	err := gb.Table.Session.Get(gb.dest, gb.sql)
	if err != nil {
		return errors.Wrap(err, "select failed")
	}

	return nil
}

// Limit is a passthrough call to the squirrel.  See
// https://godoc.org/github.com/Masterminds/squirrel#GetBuilder.Limit
func (gb *GetBuilder) Limit(limit uint64) *GetBuilder {
	gb.sql = gb.sql.Limit(limit)
	return gb
}

// Offset is a passthrough call to the squirrel.  See
// https://godoc.org/github.com/Masterminds/squirrel#GetBuilder.Offset
func (gb *GetBuilder) Offset(offset uint64) *GetBuilder {
	gb.sql = gb.sql.Offset(offset)
	return gb
}

// OrderBy is a passthrough call to the squirrel.  See
// https://godoc.org/github.com/Masterminds/squirrel#GetBuilder.OrderBy
func (gb *GetBuilder) OrderBy(
	orderBys ...string,
) *GetBuilder {
	gb.sql = gb.sql.OrderBy(orderBys...)
	return gb
}

// Where is a passthrough call to the squirrel.  See
// https://godoc.org/github.com/Masterminds/squirrel#GetBuilder.Where
func (gb *GetBuilder) Where(
	pred interface{},
	args ...interface{},
) *GetBuilder {
	gb.sql = gb.sql.Where(pred, args...)
	return gb
}
