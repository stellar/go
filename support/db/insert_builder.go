package db

import (
	"database/sql"
	"reflect"

	"github.com/pkg/errors"
	"github.com/y0ssar1an/q"
)

// Exec executes the query represented by the builder, inserting each val
// provided to the builder into the database.
func (ib *InsertBuilder) Exec() (sql.Result, error) {
	if len(ib.rows) == 0 {
		return nil, &NoRowsError{}
	}

	template := ib.rows[0]
	cols := columnsForStruct(template)
	sql := ib.sql.Columns(cols...)

	for _, row := range ib.rows {
		rrow := reflect.ValueOf(row)
		rvals := mapper.FieldsByName(rrow, cols)
		vals := make([]interface{}, len(cols))
		for i, rval := range rvals {
			vals[i] = rval.Interface()
		}
		q.Q(vals)
		sql = sql.Values(vals...)
	}

	// TODO: support return inserted id

	r, err := ib.Table.Session.Exec(sql)
	if err != nil {
		return nil, errors.Wrap(err, "insert failed")
	}

	return r, nil
}

// Rows appends more rows onto the insert statement
func (ib *InsertBuilder) Rows(rows ...interface{}) *InsertBuilder {
	ib.rows = append(ib.rows, rows...)
	return ib
}
