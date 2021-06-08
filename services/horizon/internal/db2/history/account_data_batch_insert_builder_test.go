package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestDataBatchInsertBuilderAdd(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}

	accountID := xdr.MustAddress("GAOQJGUAB7NI7K7I62ORBXMN3J4SSWQUQ7FOEPSDJ322W2HMCNWPHXFB")
	data := xdr.DataEntry{
		AccountId: accountID,
		DataName:  "foo",
		DataValue: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	}
	entry := xdr.LedgerEntry{
		Data: xdr.LedgerEntryData{
			Type: xdr.LedgerEntryTypeData,
			Data: &data,
		},
		LastModifiedLedgerSeq: 1234,
		Ext: xdr.LedgerEntryExt{
			V: 1,
			V1: &xdr.LedgerEntryExtensionV1{
				SponsoringId: xdr.MustAddressPtr("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
			},
		},
	}

	builder := q.NewAccountDataBatchInsertBuilder(2)

	err := builder.Add(tt.Ctx, entry)
	tt.Assert.NoError(err)

	err = builder.Exec(tt.Ctx)
	tt.Assert.NoError(err)

	record, err := q.GetAccountDataByName(tt.Ctx, accountID.Address(), "foo")
	tt.Assert.NoError(err)

	tt.Assert.Equal(data.DataName, xdr.String64(record.Name))
	tt.Assert.Equal([]byte(data.DataValue), []byte(record.Value))
	tt.Assert.Equal(accountID.Address(), record.AccountID)
	tt.Assert.Equal(uint32(1234), record.LastModifiedLedger)
	tt.Assert.Equal("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML", record.Sponsor.String)
}
