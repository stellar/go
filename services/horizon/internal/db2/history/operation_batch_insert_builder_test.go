package history

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/test"
	"github.com/stellar/go/xdr"
)

func TestAddOperation(t *testing.T) {
	tt := test.Start(t)
	defer tt.Finish()
	test.ResetHorizonDB(t, tt.HorizonDB)
	q := &Q{tt.HorizonSession()}
	builder := q.NewOperationBatchInsertBuilder(1)
	err := builder.Add(
		261993009153,
		261993009152,
		1,
		11,
		map[string]interface{}{"bump_to": "300000000003"},
		xdr.MustAddress("GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN"),
	)
	tt.Assert.NoError(err)

	err = builder.Exec()
	tt.Assert.NoError(err)
}
