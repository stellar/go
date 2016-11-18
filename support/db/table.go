package db

import sq "github.com/Masterminds/squirrel"

// Insert returns a new query builder configured to insert structs into the
// table
func (tbl *Table) Delete(
	pred interface{},
	args ...interface{},
) *DeleteBuilder {
	return &DeleteBuilder{
		Table: tbl,
		sql:   sq.Delete(tbl.Name).Where(pred, args...),
	}
}

// Get returns a new query builder configured to select into the provided
// `dest`.
func (tbl *Table) Get(
	dest interface{},
	pred interface{},
	args ...interface{},
) *GetBuilder {

	cols := columnsForStruct(dest)
	sql := sq.Select(cols...).From(tbl.Name).Where(pred, args...).Limit(1)

	return &GetBuilder{
		Table: tbl,
		dest:  dest,
		sql:   sql,
	}
}

// Insert returns a new query builder configured to insert structs into the
// table.
func (tbl *Table) Insert(rows ...interface{}) *InsertBuilder {
	return &InsertBuilder{
		Table: tbl,
		sql:   sq.Insert(tbl.Name),
		rows:  rows,
	}
}

// Select returns a new query builder configured to select into the provided
// `dest`.
func (tbl *Table) Select(
	dest interface{},
	pred interface{},
	args ...interface{},
) *SelectBuilder {

	cols := columnsForStruct(dest)
	sql := sq.Select(cols...).From(tbl.Name).Where(pred, args...)

	return &SelectBuilder{
		Table: tbl,
		dest:  dest,
		sql:   sql,
	}
}
