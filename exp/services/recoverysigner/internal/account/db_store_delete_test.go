package account

import (
	"testing"

	"github.com/stellar/go/exp/services/recoverysigner/internal/db/dbtest"
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
		Signers: []Signer{
			{
				PublicKey:          "GDANXMIFFAQY33KESJWSWFYKRMNIRHVVJVTLJZR6YCGCDELJWXS7TOHJ",
				EncryptedSecretKey: []byte("encrypted(SBTUBPV5B5WRYN7IH2DTAX6HDG5H6AN6BOQOJEXOOPSBK6Y7FBVLRUVS)"),
			},
			{
				PublicKey:          "GCTOKLPWCQ4COBTRLEG5OBERD3HI2PCFPONPMID64EL3RYRQNHGV6QVC",
				EncryptedSecretKey: []byte("encrypted(SAZ6EGTXXJVABVORLHIVDOGTZX4KTDCKJPXY46XK3ZDSCFMAOCMHDHBI)"),
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
		Signers: []Signer{
			{
				PublicKey:          "GCBVZBR6YHOKHCGLWXXOFJKQBKBHKY2NDZWPCP546MAOZE7JR2O3IWIC",
				EncryptedSecretKey: []byte("encrypted(SCNHWRBHMFM6G4UMIHAMCJ47ZXR6Z5OTZH3XLE3UZHXUQYCFC7VS6RYW)"),
			},
			{
				PublicKey:          "GA5CX7BFI5T2TDLI7CINFUUDKNV257UQXADQOGC3IK6HCLTWYMOMBQB2",
				EncryptedSecretKey: []byte("encrypted(SAHLGVJK3YTNM7MT4DWO7NQEVN3UDVEZMGSAXO6F3P6W2AD23I3MOOWI)"),
			},
		},
	}
	err = store.Add(a2)
	require.NoError(t, err)

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

	// Check that account 1 is gone and account 2 remains
	{
		type row struct {
			Address string `db:"address"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT address FROM accounts`)
		require.NoError(t, err)
		wantRows := []row{
			{Address: a2Address},
		}
		assert.ElementsMatch(t, wantRows, rows)
	}
	// Check that account 1 delete has been audited
	{
		type row struct {
			AuditOp string `db:"audit_op"`
			Address string `db:"address"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT audit_op, address FROM accounts_audit`)
		require.NoError(t, err)
		wantRows := []row{
			{AuditOp: "INSERT", Address: a1Address},
			{AuditOp: "INSERT", Address: a2Address},
			{AuditOp: "DELETE", Address: a1Address},
		}
		assert.Equal(t, wantRows, rows)
	}

	// Check that account 1's identities are gone and account 2's remain
	{
		type row struct {
			Role string `db:"role"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT role FROM identities`)
		require.NoError(t, err)
		wantRows := []row{
			// Identities for account 2
			{Role: "owner"},
		}
		assert.ElementsMatch(t, wantRows, rows)
	}
	// Check that account 1's identity's deletes have been audited
	{
		type row struct {
			AuditOp string `db:"audit_op"`
			Role    string `db:"role"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT audit_op, role FROM identities_audit`)
		require.NoError(t, err)
		wantRows := []row{
			{AuditOp: "INSERT", Role: "sender"},
			{AuditOp: "INSERT", Role: "receiver"},
			{AuditOp: "INSERT", Role: "owner"},
			{AuditOp: "DELETE", Role: "sender"},
			{AuditOp: "DELETE", Role: "receiver"},
		}
		assert.Equal(t, wantRows, rows)
	}

	// Check that account 1's auth methods are gone and account 2's remain
	{
		type row struct {
			Value string `db:"value"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT value FROM auth_methods`)
		require.NoError(t, err)
		wantRows := []row{
			// Auth methods for account 2
			{Value: "GAA5TI5BXVNJTA6UEDF7UTMA5FXHR2TFRCJ2G7QT6FJCJ7WD5ITIKQNE"},
			{Value: "+30000000000"},
			{Value: "user3@example.com"},
		}
		assert.ElementsMatch(t, wantRows, rows)
	}
	// Check that account 1's auth methods's deletes have been audited
	{
		type row struct {
			AuditOp string `db:"audit_op"`
			Value   string `db:"value"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT audit_op, value FROM auth_methods_audit`)
		require.NoError(t, err)
		wantRows := []row{
			{AuditOp: "INSERT", Value: "GD4NGMOTV4QOXWA6PGPIGVWZYMRCJAKLQJKZIP55C5DGB3GBHHET3YC6"},
			{AuditOp: "INSERT", Value: "+10000000000"},
			{AuditOp: "INSERT", Value: "user1@example.com"},
			{AuditOp: "INSERT", Value: "GBJCOYGKIJYX3VUEOZ6GVMFP522UO4OEBI5KB5HHWZAZ2DEJTHS6VOHP"},
			{AuditOp: "INSERT", Value: "+20000000000"},
			{AuditOp: "INSERT", Value: "user2@example.com"},
			{AuditOp: "INSERT", Value: "GAA5TI5BXVNJTA6UEDF7UTMA5FXHR2TFRCJ2G7QT6FJCJ7WD5ITIKQNE"},
			{AuditOp: "INSERT", Value: "+30000000000"},
			{AuditOp: "INSERT", Value: "user3@example.com"},
			{AuditOp: "DELETE", Value: "GD4NGMOTV4QOXWA6PGPIGVWZYMRCJAKLQJKZIP55C5DGB3GBHHET3YC6"},
			{AuditOp: "DELETE", Value: "+10000000000"},
			{AuditOp: "DELETE", Value: "user1@example.com"},
			{AuditOp: "DELETE", Value: "GBJCOYGKIJYX3VUEOZ6GVMFP522UO4OEBI5KB5HHWZAZ2DEJTHS6VOHP"},
			{AuditOp: "DELETE", Value: "+20000000000"},
			{AuditOp: "DELETE", Value: "user2@example.com"},
		}
		assert.Equal(t, wantRows, rows)
	}

	// Check that account 1's signers are gone and account 2's remain
	{
		type row struct {
			PublicKey string `db:"public_key"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT public_key FROM signers`)
		require.NoError(t, err)
		wantRows := []row{
			// Signers for account 2
			{PublicKey: "GCBVZBR6YHOKHCGLWXXOFJKQBKBHKY2NDZWPCP546MAOZE7JR2O3IWIC"},
			{PublicKey: "GA5CX7BFI5T2TDLI7CINFUUDKNV257UQXADQOGC3IK6HCLTWYMOMBQB2"},
		}
		assert.ElementsMatch(t, wantRows, rows)
	}
	// Check that account 1's signers's deletes have been audited
	{
		type row struct {
			AuditOp   string `db:"audit_op"`
			PublicKey string `db:"public_key"`
		}
		rows := []row{}
		err = session.Select(&rows, `SELECT audit_op, public_key FROM signers_audit`)
		require.NoError(t, err)
		wantRows := []row{
			// Signers for account 1
			{AuditOp: "INSERT", PublicKey: "GDANXMIFFAQY33KESJWSWFYKRMNIRHVVJVTLJZR6YCGCDELJWXS7TOHJ"},
			{AuditOp: "INSERT", PublicKey: "GCTOKLPWCQ4COBTRLEG5OBERD3HI2PCFPONPMID64EL3RYRQNHGV6QVC"},
			// Signers for account 2
			{AuditOp: "INSERT", PublicKey: "GCBVZBR6YHOKHCGLWXXOFJKQBKBHKY2NDZWPCP546MAOZE7JR2O3IWIC"},
			{AuditOp: "INSERT", PublicKey: "GA5CX7BFI5T2TDLI7CINFUUDKNV257UQXADQOGC3IK6HCLTWYMOMBQB2"},
			// Signers for account 1 - removed
			{AuditOp: "DELETE", PublicKey: "GDANXMIFFAQY33KESJWSWFYKRMNIRHVVJVTLJZR6YCGCDELJWXS7TOHJ"},
			{AuditOp: "DELETE", PublicKey: "GCTOKLPWCQ4COBTRLEG5OBERD3HI2PCFPONPMID64EL3RYRQNHGV6QVC"},
		}
		assert.Equal(t, wantRows, rows)
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
