package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/test"
)

func TestQueryEmptyAccountSigners(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	signer := "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO0"
	results, err := q.AccountsForSigner(signer, db2.PageQuery{Order: "asc", Limit: 10})
	tt.Assert.NoError(err)
	tt.Assert.Len(results, 0)
}

func TestInsertAccountSigner(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	account := "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2"
	signer := "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"
	weight := int32(123)
	rowsAffected, err := q.CreateAccountSigner(account, signer, weight)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), rowsAffected)

	expected := AccountSigner{
		Account: account,
		Signer:  signer,
		Weight:  weight,
	}
	results, err := q.AccountsForSigner(signer, db2.PageQuery{Order: "asc", Limit: 10})
	tt.Assert.NoError(err)
	tt.Assert.Len(results, 1)
	tt.Assert.Equal(expected, results[0])

	weight = 321
	_, err = q.CreateAccountSigner(account, signer, weight)
	tt.Assert.Error(err)
	tt.Assert.EqualError(err, `exec failed: pq: duplicate key value violates unique constraint "accounts_signers_pkey"`)
}

func TestMultipleAccountsForSigner(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	account := "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH1"
	signer := "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO2"
	weight := int32(123)
	rowsAffected, err := q.CreateAccountSigner(account, signer, weight)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), rowsAffected)

	anotherAccount := "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"
	anotherWeight := int32(321)
	rowsAffected, err = q.CreateAccountSigner(anotherAccount, signer, anotherWeight)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), rowsAffected)

	expected := []AccountSigner{
		AccountSigner{
			Account: account,
			Signer:  signer,
			Weight:  weight,
		},
		AccountSigner{
			Account: anotherAccount,
			Signer:  signer,
			Weight:  anotherWeight,
		},
	}
	results, err := q.AccountsForSigner(signer, db2.PageQuery{Order: "asc", Limit: 10})
	tt.Assert.NoError(err)
	tt.Assert.Len(results, 2)
	tt.Assert.Equal(expected, results)
}

func TestRemoveNonExistantAccountSigner(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	account := "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH3"
	signer := "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO5"
	rowsAffected, err := q.RemoveAccountSigner(account, signer)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(0), rowsAffected)
}

func TestRemoveAccountSigner(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()
	q := &Q{tt.HorizonSession()}

	account := "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH6"
	signer := "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO7"
	weight := int32(123)
	_, err := q.CreateAccountSigner(account, signer, weight)
	tt.Assert.NoError(err)

	expected := AccountSigner{
		Account: account,
		Signer:  signer,
		Weight:  weight,
	}
	results, err := q.AccountsForSigner(signer, db2.PageQuery{Order: "asc", Limit: 10})
	tt.Assert.NoError(err)
	tt.Assert.Len(results, 1)
	tt.Assert.Equal(expected, results[0])

	rowsAffected, err := q.RemoveAccountSigner(account, signer)
	tt.Assert.NoError(err)
	tt.Assert.Equal(int64(1), rowsAffected)

	results, err = q.AccountsForSigner(signer, db2.PageQuery{Order: "asc", Limit: 10})
	tt.Assert.NoError(err)
	tt.Assert.Len(results, 0)
}
