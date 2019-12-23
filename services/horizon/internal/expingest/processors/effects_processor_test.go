package processors

import (
	"testing"

	"github.com/stellar/go/services/horizon/internal/db2/history"
	. "github.com/stellar/go/services/horizon/internal/test/transactions"
	"github.com/stretchr/testify/assert"
)

func TestOperationEffects(t *testing.T) {
	testCases := []struct {
		desc          string
		envelopeXDR   string
		resultXDR     string
		metaXDR       string
		feeChangesXDR string
		hash          string
		index         uint32
		expected      []map[string]interface{}
	}{
		{
			desc:          "createAccount",
			envelopeXDR:   "AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAAaAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAoZftFP3p4ifbTm6hQdieotu3Zw9E05GtoSh5MBytEpQAAAACVAvkAAAAAAAAAAABVvwF9wAAAEDHU95E9wxgETD8TqxUrkgC0/7XHyNDts6Q5huRHfDRyRcoHdv7aMp/sPvC3RPkXjOMjgbKJUX7SgExUeYB5f8F",
			resultXDR:     "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=",
			metaXDR:       "AAAAAQAAAAIAAAADAAAAOQAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcLGrZY9dZxbAAAAAAAAAAZAAAAAAAAAAEAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAOQAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcLGrZY9dZxbAAAAAAAAAAaAAAAAAAAAAEAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAMAAAA5AAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wsatlj11nFsAAAAAAAAABoAAAAAAAAAAQAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAA5AAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wsatlahyo1sAAAAAAAAABoAAAAAAAAAAQAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAA5AAAAAAAAAAChl+0U/eniJ9tObqFB2J6i27dnD0TTka2hKHkwHK0SlAAAAAJUC+QAAAAAOQAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
			feeChangesXDR: "AAAAAgAAAAMAAAA3AAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wsatlj11nHQAAAAAAAAABkAAAAAAAAAAQAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAA5AAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wsatlj11nFsAAAAAAAAABkAAAAAAAAAAQAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9wAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
			hash:          "0e5bd332291e3098e49886df2cdb9b5369a5f9e0a9973f0d9e1a9489c6581ba2",
			index:         0,
			expected: []map[string]interface{}{
				map[string]interface{}{
					"address":     "GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN",
					"operationID": int64(240518172673),
					"details": map[string]interface{}{
						"starting_balance": "1000.0000000",
					},
					"effectType": history.EffectAccountCreated,
					"order":      uint32(1),
				},
				map[string]interface{}{
					"address":     "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
					"operationID": int64(240518172673),
					"details": map[string]interface{}{
						"amount":     "1000.0000000",
						"asset_type": "native",
					},
					"effectType": history.EffectAccountDebited,
					"order":      uint32(2),
				},
				map[string]interface{}{
					"address":     "GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN",
					"operationID": int64(240518172673),
					"details": map[string]interface{}{
						"public_key": "GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN",
						"weight":     1,
					},
					"effectType": history.EffectSignerCreated,
					"order":      uint32(3),
				},
			},
		},
		{
			desc:          "payment",
			envelopeXDR:   "AAAAABpcjiETZ0uhwxJJhgBPYKWSVJy2TZ2LI87fqV1cUf/UAAAAZAAAADcAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAAAAAAAAAX14QAAAAAAAAAAAVxR/9QAAABAK6pcXYMzAEmH08CZ1LWmvtNDKauhx+OImtP/Lk4hVTMJRVBOebVs5WEPj9iSrgGT0EswuDCZ2i5AEzwgGof9Ag==",
			resultXDR:     "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=",
			metaXDR:       "AAAAAQAAAAIAAAADAAAAOAAAAAAAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAACVAvjnAAAADcAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAOAAAAAAAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAACVAvjnAAAADcAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAA==",
			feeChangesXDR: "AAAAAgAAAAMAAAA3AAAAAAAAAAAaXI4hE2dLocMSSYYAT2ClklSctk2diyPO36ldXFH/1AAAAAJUC+QAAAAANwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAA4AAAAAAAAAAAaXI4hE2dLocMSSYYAT2ClklSctk2diyPO36ldXFH/1AAAAAJUC+OcAAAANwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==",
			hash:          "2a805712c6d10f9e74bb0ccf54ae92a2b4b1e586451fe8133a2433816f6b567c",
			index:         0,
			expected: []map[string]interface{}{
				map[string]interface{}{
					"address": "GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y",
					"details": map[string]interface{}{
						"amount":     "10.0000000",
						"asset_type": "native",
					},
					"effectType":  history.EffectAccountCredited,
					"operationID": int64(240518172673),
					"order":       uint32(1),
				},
				map[string]interface{}{
					"address": "GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y",
					"details": map[string]interface{}{
						"amount":     "10.0000000",
						"asset_type": "native",
					},
					"effectType":  history.EffectAccountDebited,
					"operationID": int64(240518172673),
					"order":       uint32(2),
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tt := assert.New(t)
			transaction := BuildLedgerTransaction(
				t,
				TestTransaction{
					Index:         1,
					EnvelopeXDR:   tc.envelopeXDR,
					ResultXDR:     tc.resultXDR,
					MetaXDR:       tc.metaXDR,
					FeeChangesXDR: tc.feeChangesXDR,
					Hash:          tc.hash,
				},
			)

			operation := transactionOperationWrapper{
				index:          tc.index,
				transaction:    transaction,
				operation:      transaction.Envelope.Tx.Operations[tc.index],
				ledgerSequence: 56,
			}

			effects, err := operation.Effects()
			tt.NoError(err)
			tt.Equal(tc.expected, effects)
		})
	}
}
