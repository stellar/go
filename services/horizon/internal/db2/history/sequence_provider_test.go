package history

import (
	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSequenceProviderEmptyDB(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	addresses := []string{
		"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	}
	results, err := q.GetSequenceNumbers(addresses)
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestSequenceProviderGet(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	batch := q.NewAccountsBatchInsertBuilder(0)
	err := batch.Add(account1, 1234)
	assert.NoError(t, err)
	err = batch.Add(account2, 1235)
	assert.NoError(t, err)
	assert.NoError(t, batch.Exec())

	results, err := q.GetSequenceNumbers([]string{
		"GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB",
		"GCT2NQM5KJJEF55NPMY444C6M6CA7T33HRNCMA6ZFBIIXKNCRO6J25K7",
		"GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
	})
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, uint64(account1.SeqNum), results["GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB"])
	assert.Equal(t, uint64(account2.SeqNum), results["GCT2NQM5KJJEF55NPMY444C6M6CA7T33HRNCMA6ZFBIIXKNCRO6J25K7"])
}
