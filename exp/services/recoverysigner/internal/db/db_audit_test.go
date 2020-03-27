package db

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAccountsAudit confirms that the columns for accounts_audit match those
// in the accounts table, except for the header columns.
func TestAccountsAudit(t *testing.T) {
	db := dbtest.Open(t)
	conn, err := Open(db.DSN)
	require.NoError(t, err)

	assertAuditTableCols(t, conn, "accounts", "accounts_audit")
}

// TestIdentitiesAudit confirms that the columns for identities_audit match
// those in the identities table, except for the header columns.
func TestIdentitiesAudit(t *testing.T) {
	db := dbtest.Open(t)
	conn, err := Open(db.DSN)
	require.NoError(t, err)

	assertAuditTableCols(t, conn, "identities", "identities_audit")
}

// TestAuthMethodsAudit confirms that the columns for auth_methods_audit match
// those in the auth_methods table, except for the header columns.
func TestAuthMethodsAudit(t *testing.T) {
	db := dbtest.Open(t)
	conn, err := Open(db.DSN)
	require.NoError(t, err)

	assertAuditTableCols(t, conn, "auth_methods", "auth_methods_audit")
}

// assertAuditTableCols checks that the audit table for the given
// table has the same columns as the given table, as well as the header
// columns, that all the types and columns are as we expect.
func assertAuditTableCols(t *testing.T, db *sqlx.DB, tableName, auditTableName string) {
	cols := tableCols(t, db, tableName)

	wantAuditHeaderCols := []tableCol{
		{Name: "audit_id", DataType: "bigint", UDTName: "int8", IsNullable: "NO"},
		{Name: "audit_at", DataType: "timestamp with time zone", UDTName: "timestamptz", IsNullable: "NO"},
		{Name: "audit_user", DataType: "text", UDTName: "text", IsNullable: "NO"},
		{Name: "audit_op", DataType: "USER-DEFINED", UDTName: "audit_op", IsNullable: "NO"},
	}
	wantAuditCols := append(append([]tableCol{}, wantAuditHeaderCols...), cols...)

	auditCols := tableCols(t, db, auditTableName)
	assert.Equal(t, wantAuditCols, auditCols)
}

type tableCol struct {
	Name       string
	DataType   string
	UDTName    string
	IsNullable string
}

// tableCols returns the column names for the table.
func tableCols(t *testing.T, db *sqlx.DB, tableName string) []tableCol {
	cols := []tableCol{}
	err := db.Select(
		&cols,
		`SELECT
			column_name as Name,
			data_type as DataType,
			udt_name as UDTName,
			is_nullable as IsNullable
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE table_schema = 'public'
		AND table_name = $1
		ORDER BY ordinal_position ASC;`,
		tableName,
	)
	require.NoError(t, err)
	return cols
}
