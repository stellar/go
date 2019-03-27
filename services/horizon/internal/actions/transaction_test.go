package actions

import (
	"context"
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/services/horizon/internal/test"
)

var defaultPage db2.PageQuery = db2.PageQuery{
	Order:  db2.OrderAscending,
	Limit:  db2.DefaultPageSize,
	Cursor: "",
}

func TestTransactionPageByAccount(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	ctx := context.Background()
	page, err := TransactionPageByAccount(ctx, &history.Q{tt.HorizonSession()}, "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(3, len(page.Embedded.Records))
}

func TestLoadTransactionRecordByAccount(t *testing.T) {
	tt := test.Start(t).Scenario("base")
	defer tt.Finish()

	records, err := loadTransactionRecordByAccount(&history.Q{tt.HorizonSession()}, "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(3, len(records))

	records, err = loadTransactionRecordByAccount(&history.Q{tt.HorizonSession()}, "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(1, len(records))

	records, err = loadTransactionRecordByAccount(&history.Q{tt.HorizonSession()}, "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", true, defaultPage)
	tt.Assert.NoError(err)
	tt.Assert.Equal(2, len(records))
}
