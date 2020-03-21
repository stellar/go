package account

import (
	"testing"
	"time"

	"github.com/stellar/go/exp/services/recoverysigner/internal/db/dbtest"
	"github.com/stellar/go/support/clock"
	"github.com/stellar/go/support/clock/clocktest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {
	db := dbtest.Open(t)
	session := db.Open()

	store := DBStore{
		DB: session,
	}

	// Store account 1
	a1Address := "GCLLT3VG4F6EZAHZEBKWBWV5JGVPCVIKUCGTY3QEOAIZU5IJGMWCT2TT"
	a1 := Account{
		Address: a1Address,
		Identities: []Identity{
			{
				Role: "sender",
				AuthMethods: []AuthMethod{
					{Type: AuthMethodTypeAddress, Value: "GD4NGMOTV4QOXWA6PGPIGVWZYMRCJAKLQJKZIP55C5DGB3GBHHET3YC6"},
					{Type: AuthMethodTypePhoneNumber, Value: "+10000000000"},
					{Type: AuthMethodTypeEmail, Value: "user1@example.com"},
				},
			},
			{
				Role: "receiver",
				AuthMethods: []AuthMethod{
					{Type: AuthMethodTypeAddress, Value: "GBJCOYGKIJYX3VUEOZ6GVMFP522UO4OEBI5KB5HHWZAZ2DEJTHS6VOHP"},
					{Type: AuthMethodTypePhoneNumber, Value: "+20000000000"},
					{Type: AuthMethodTypeEmail, Value: "user2@example.com"},
				},
			},
		},
	}
	err := store.Add(a1)
	require.NoError(t, err)

	// Store account 2
	a2Address := "GDJ6ZE3SR6XBKF2ZDGNMWXF7TKZEEQZDSBVRLZXJ2HVOFIYMYQ7IAMMU"
	a2 := Account{
		Address: a2Address,
		Identities: []Identity{
			{
				Role: "owner",
				AuthMethods: []AuthMethod{
					{Type: AuthMethodTypeAddress, Value: "GAA5TI5BXVNJTA6UEDF7UTMA5FXHR2TFRCJ2G7QT6FJCJ7WD5ITIKQNE"},
					{Type: AuthMethodTypePhoneNumber, Value: "+30000000000"},
					{Type: AuthMethodTypeEmail, Value: "user3@example.com"},
				},
			},
		},
	}
	err = store.Add(a2)
	require.NoError(t, err)

	// Use a fixed time for deletions
	deletedAt := time.Now().UTC()
	deletedAtMicroseconds := deletedAt.Round(time.Microsecond)
	clock := clock.Clock{
		Source: clocktest.FixedSource(deletedAt),
	}
	store = DBStore{
		DB:    session,
		Clock: &clock,
	}

	// Get account 1 to check it exists
	a1Got, err := store.Get(a1Address)
	require.NoError(t, err)
	assert.Equal(t, a1, a1Got)

	// Get account 2 to check it exists
	a2Got, err := store.Get(a2Address)
	require.NoError(t, err)
	assert.Equal(t, a2, a2Got)

	// Delete account 1
	err = store.Delete(a1Address)
	require.NoError(t, err)

	// Get account 1 to check it no longer exists
	_, err = store.Get(a1Address)
	assert.Equal(t, ErrNotFound, err)

	// Get account 2 to check it was not deleted
	a2Got, err = store.Get(a2Address)
	require.NoError(t, err)
	assert.Equal(t, a2, a2Got)

	// Check that the deleted_at is set for account 1 and not account 2
	{
		type row struct {
			Address   string     `db:"address"`
			DeletedAt *time.Time `db:"deleted_at"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT address, deleted_at FROM accounts`)
		require.NoError(t, err)
		wantRows := []row{
			{Address: a1Address, DeletedAt: &deletedAtMicroseconds},
			{Address: a2Address, DeletedAt: nil},
		}
		assert.ElementsMatch(t, wantRows, rows)
	}

	// Check that the deleted_at is set for account 1 identities and not account 2
	{
		type row struct {
			Role      string     `db:"role"`
			DeletedAt *time.Time `db:"deleted_at"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT role, deleted_at FROM identities`)
		require.NoError(t, err)
		wantRows := []row{
			// Identities for account 1
			{Role: "sender", DeletedAt: &deletedAtMicroseconds},
			{Role: "receiver", DeletedAt: &deletedAtMicroseconds},
			// Identities for account 2
			{Role: "owner", DeletedAt: nil},
		}
		assert.ElementsMatch(t, wantRows, rows)
	}

	// Check that the deleted_at is set for account 1 auth methods and not account 2
	{
		type row struct {
			Value     string     `db:"value"`
			DeletedAt *time.Time `db:"deleted_at"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT value, deleted_at FROM auth_methods`)
		require.NoError(t, err)
		wantRows := []row{
			// Auth methods for account 1
			{Value: "GD4NGMOTV4QOXWA6PGPIGVWZYMRCJAKLQJKZIP55C5DGB3GBHHET3YC6", DeletedAt: &deletedAtMicroseconds},
			{Value: "+10000000000", DeletedAt: &deletedAtMicroseconds},
			{Value: "user1@example.com", DeletedAt: &deletedAtMicroseconds},
			{Value: "GBJCOYGKIJYX3VUEOZ6GVMFP522UO4OEBI5KB5HHWZAZ2DEJTHS6VOHP", DeletedAt: &deletedAtMicroseconds},
			{Value: "+20000000000", DeletedAt: &deletedAtMicroseconds},
			{Value: "user2@example.com", DeletedAt: &deletedAtMicroseconds},
			// Auth methods for account 2
			{Value: "GAA5TI5BXVNJTA6UEDF7UTMA5FXHR2TFRCJ2G7QT6FJCJ7WD5ITIKQNE", DeletedAt: nil},
			{Value: "+30000000000", DeletedAt: nil},
			{Value: "user3@example.com", DeletedAt: nil},
		}
		assert.ElementsMatch(t, wantRows, rows)
	}

	// Store account 3 (same address as account 1)
	a3Address := a1Address
	a3 := Account{
		Address: a3Address,
		Identities: []Identity{
			{
				Role: "sender",
				AuthMethods: []AuthMethod{
					{Type: AuthMethodTypeAddress, Value: "GD4NGMOTV4QOXWA6PGPIGVWZYMRCJAKLQJKZIP55C5DGB3GBHHET3YC6"},
					{Type: AuthMethodTypePhoneNumber, Value: "+10000000000"},
					{Type: AuthMethodTypeEmail, Value: "user1@example.com"},
				},
			},
			{
				Role: "receiver",
				AuthMethods: []AuthMethod{
					{Type: AuthMethodTypeAddress, Value: "GBJCOYGKIJYX3VUEOZ6GVMFP522UO4OEBI5KB5HHWZAZ2DEJTHS6VOHP"},
					{Type: AuthMethodTypePhoneNumber, Value: "+20000000000"},
					{Type: AuthMethodTypeEmail, Value: "user2@example.com"},
				},
			},
		},
	}
	err = store.Add(a3)
	require.NoError(t, err)

	// Get account 3 to check it exists
	a3Got, err := store.Get(a3Address)
	require.NoError(t, err)
	assert.Equal(t, a3, a3Got)

	// Get account 2 to check it exists
	a2Got, err = store.Get(a2Address)
	require.NoError(t, err)
	assert.Equal(t, a2, a2Got)
}

func TestDelete_notFound(t *testing.T) {
	db := dbtest.Open(t)
	session := db.Open()

	store := DBStore{
		DB: session,
	}

	address := "GCLLT3VG4F6EZAHZEBKWBWV5JGVPCVIKUCGTY3QEOAIZU5IJGMWCT2TT"

	err := store.Delete(address)
	assert.Equal(t, ErrNotFound, err)
}
