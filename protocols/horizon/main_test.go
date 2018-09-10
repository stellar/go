package horizon

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAccount(t *testing.T) {
	account := Account{
		Data: map[string]string{
			"test":    "aGVsbG8=",
			"invalid": "a_*&^*",
		},
	}

	Convey("Account.GetData", t, func() {
		Convey("Returns decoded value if the key exists", func() {
			decoded, err := account.GetData("test")
			So(err, ShouldBeNil)
			So(string(decoded), ShouldEqual, "hello")
		})

		Convey("Returns empty slice if key doesn't exist", func() {
			decoded, err := account.GetData("test2")
			So(err, ShouldBeNil)
			So(len(decoded), ShouldEqual, 0)
		})

		Convey("Returns error slice if value is invalid", func() {
			_, err := account.GetData("invalid")
			So(err, ShouldNotBeNil)
		})
	})

	Convey("Account.MustGetData", t, func() {
		Convey("Returns decoded value if the key exists", func() {
			decoded := account.MustGetData("test")
			So(string(decoded), ShouldEqual, "hello")
		})

		Convey("Returns empty slice if key doesn't exist", func() {
			decoded := account.MustGetData("test2")
			So(len(decoded), ShouldEqual, 0)
		})

		Convey("Returns error slice if value is invalid", func() {
			So(func() { account.MustGetData("invalid") }, ShouldPanic)
		})
	})
}

//After marshalling and unmarshalling, the resulting struct should be the exact same as the original
func TestTransactionJSONMarshal(t *testing.T) {
	transaction := Transaction{
		ID:       "12345",
		FeePaid:  10,
		MemoType: "text",
		Memo:     "",
	}
	marshaledTransaction, marshalErr := json.Marshal(transaction)
	assert.Nil(t, marshalErr)
	var result Transaction
	json.Unmarshal(marshaledTransaction, &result)
	assert.Equal(t, result, transaction)
}

//For text memos, even if memo is an empty string, the resulting JSON should
// still include memo as a field
func TestTransactionEmptyMemoText(t *testing.T) {
	transaction := Transaction{
		MemoType: "text",
		Memo:     "",
	}
	marshaledTransaction, marshalErr := json.Marshal(transaction)
	assert.Nil(t, marshalErr)
	var result struct {
		Memo *string
	}
	json.Unmarshal(marshaledTransaction, &result)
	assert.NotNil(t, result.Memo)
}

// If a transaction's memo type is None, then the memo field should be omitted from JSON
func TestTransactionMemoTypeNone(t *testing.T) {
	transaction := Transaction{
		MemoType: "none",
	}
	marshaledTransaction, marshalErr := json.Marshal(transaction)
	assert.Nil(t, marshalErr)
	var result struct {
		Memo *string
	}
	json.Unmarshal(marshaledTransaction, &result)
	assert.Nil(t, result.Memo)
}
