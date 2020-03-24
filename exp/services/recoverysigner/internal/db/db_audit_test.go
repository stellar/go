package db

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stellar/go/exp/services/recoverysigner/internal/db/dbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var auditHeaderCols = []string{
	"audit_id",
	"audit_at",
	"audit_user",
	"audit_op",
}

// TestAccountsAudit confirms that the columns for accounts_audit match those
// in the accounts table, except for the header columns.
func TestAccountsAudit(t *testing.T) {
	db := dbtest.Open(t)
	conn, err := Open(db.DSN)
	require.NoError(t, err)

	assertAuditColsEqualTableCols(t, conn, "accounts", "accounts_audit")
}

// TestIdentitiesAudit confirms that the columns for identities_audit match
// those in the identities table, except for the header columns.
func TestIdentitiesAudit(t *testing.T) {
	db := dbtest.Open(t)
	conn, err := Open(db.DSN)
	require.NoError(t, err)

	assertAuditColsEqualTableCols(t, conn, "identities", "identities_audit")
}

// TestAuthMethodsAudit confirms that the columns for auth_methods_audit match
// those in the auth_methods table, except for the header columns.
func TestAuthMethodsAudit(t *testing.T) {
	db := dbtest.Open(t)
	conn, err := Open(db.DSN)
	require.NoError(t, err)

	assertAuditColsEqualTableCols(t, conn, "auth_methods", "auth_methods_audit")
}

// assertAuditColsEqualTableCols checks that the audit table for the given
// table has the same columns as the given table.
func assertAuditColsEqualTableCols(t *testing.T, db *sqlx.DB, tableName, auditTableName string) {
	cols := tableCols(t, db, tableName)
	wantAuditCols := append(append([]string{}, auditHeaderCols...), cols...)
	auditCols := tableCols(t, db, auditTableName)
	assert.Equal(t, wantAuditCols, auditCols)
}

func tableCols(t *testing.T, db *sqlx.DB, tableName string) []string {
	cols := []string{}
	err := db.Select(
		&cols,
		`SELECT column_name
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_NAME = $1;`,
		tableName,
	)
	require.NoError(t, err)
	return cols
}
