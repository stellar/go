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

func TestAdd(t *testing.T) {
	db := dbtest.Open(t)
	session := db.Open()

	createdAt := time.Now().UTC()

	clock := clock.Clock{
		Source: clocktest.FixedSource(createdAt),
	}
	store := DBStore{
		DB:    session,
		Clock: &clock,
	}

	a := Account{
		Address: "GCLLT3VG4F6EZAHZEBKWBWV5JGVPCVIKUCGTY3QEOAIZU5IJGMWCT2TT",
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
	err := store.Add(a)
	require.NoError(t, err)

	// Check the account row has been added.
	{
		type row struct {
			ID        int64      `db:"id"`
			CreatedAt time.Time  `db:"created_at"`
			DeletedAt *time.Time `db:"deleted_at"`
			Address   string     `db:"address"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT * FROM accounts`)
		require.NoError(t, err)
		wantRows := []row{
			{
				ID:        1,
				CreatedAt: createdAt.Round(time.Microsecond),
				DeletedAt: nil,
				Address:   "GCLLT3VG4F6EZAHZEBKWBWV5JGVPCVIKUCGTY3QEOAIZU5IJGMWCT2TT",
			},
		}
		assert.Equal(t, wantRows, rows)
	}

	// Check the identity rows have been added.
	{
		type row struct {
			AccountID int64      `db:"account_id"`
			ID        int64      `db:"id"`
			CreatedAt time.Time  `db:"created_at"`
			DeletedAt *time.Time `db:"deleted_at"`
			Role      string     `db:"role"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT * FROM identities`)
		require.NoError(t, err)
		wantRows := []row{
			{
				AccountID: 1,
				ID:        1,
				CreatedAt: createdAt.Round(time.Microsecond),
				DeletedAt: nil,
				Role:      "sender",
			},
			{
				AccountID: 1,
				ID:        2,
				CreatedAt: createdAt.Round(time.Microsecond),
				DeletedAt: nil,
				Role:      "receiver",
			},
		}
		assert.Equal(t, wantRows, rows)
	}

	// Check the auth method rows have been added.
	{
		type row struct {
			AccountID  int64      `db:"account_id"`
			IdentityID int64      `db:"identity_id"`
			ID         int64      `db:"id"`
			CreatedAt  time.Time  `db:"created_at"`
			DeletedAt  *time.Time `db:"deleted_at"`
			Type       string     `db:"type"`
			Value      string     `db:"value"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT * FROM auth_methods`)
		require.NoError(t, err)
		wantRows := []row{
			{
				AccountID:  1,
				IdentityID: 1,
				ID:         1,
				CreatedAt:  createdAt.Round(time.Microsecond),
				DeletedAt:  nil,
				Type:       "stellar_address",
				Value:      "GD4NGMOTV4QOXWA6PGPIGVWZYMRCJAKLQJKZIP55C5DGB3GBHHET3YC6",
			},
			{
				AccountID:  1,
				IdentityID: 1,
				ID:         2,
				CreatedAt:  createdAt.Round(time.Microsecond),
				DeletedAt:  nil,
				Type:       "phone_number",
				Value:      "+10000000000",
			},
			{
				AccountID:  1,
				IdentityID: 1,
				ID:         3,
				CreatedAt:  createdAt.Round(time.Microsecond),
				DeletedAt:  nil,
				Type:       "email",
				Value:      "user1@example.com",
			},
			{
				AccountID:  1,
				IdentityID: 2,
				ID:         4,
				CreatedAt:  createdAt.Round(time.Microsecond),
				DeletedAt:  nil,
				Type:       "stellar_address",
				Value:      "GBJCOYGKIJYX3VUEOZ6GVMFP522UO4OEBI5KB5HHWZAZ2DEJTHS6VOHP",
			},
			{
				AccountID:  1,
				IdentityID: 2,
				ID:         5,
				CreatedAt:  createdAt.Round(time.Microsecond),
				DeletedAt:  nil,
				Type:       "phone_number",
				Value:      "+20000000000",
			},
			{
				AccountID:  1,
				IdentityID: 2,
				ID:         6,
				CreatedAt:  createdAt.Round(time.Microsecond),
				DeletedAt:  nil,
				Type:       "email",
				Value:      "user2@example.com",
			},
		}
		assert.Equal(t, wantRows, rows)
	}
}

func TestAdd_conflict(t *testing.T) {
	db := dbtest.Open(t)
	session := db.Open()

	store := DBStore{
		DB: session,
	}

	a := Account{
		Address: "GCLLT3VG4F6EZAHZEBKWBWV5JGVPCVIKUCGTY3QEOAIZU5IJGMWCT2TT",
	}

	err := store.Add(a)
	require.NoError(t, err)

	err = store.Add(a)
	assert.Equal(t, ErrAlreadyExists, err)
}
