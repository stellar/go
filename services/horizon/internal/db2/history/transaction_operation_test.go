package history

import (
	"testing"

	"github.com/stellar/go/exp/ingest/io"
	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/assert"
)

var (
	ledger = int64(4294967296) // ledger sequence 1
	tx     = int64(4096)       // tx index 1
	op     = int64(1)          // op index 1
)

func TestTransactionOperationID(t *testing.T) {
	tt := assert.New(t)
	transaction, err := buildTransaction(
		1,
		"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
		nil,
	)
	tt.NoError(err)

	operation := TransactionOperation{
		Index:          0,
		Transaction:    transaction,
		Operation:      transaction.Envelope.Tx.Operations[0],
		LedgerSequence: 1,
	}

	tt.Equal(ledger+tx+op, operation.ID())
}

func TestTransactionOperationTransactionID(t *testing.T) {
	tt := assert.New(t)
	transaction, err := buildTransaction(
		1,
		"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
		nil,
	)
	tt.NoError(err)

	operation := TransactionOperation{
		Index:          1,
		Transaction:    transaction,
		Operation:      transaction.Envelope.Tx.Operations[0],
		LedgerSequence: 1,
	}

	tt.Equal(ledger+tx, operation.TransactionID())
}

func TestOperationTransactionSourceAccount(t *testing.T) {
	testCases := []struct {
		desc          string
		sourceAccount string
		expected      string
	}{
		{
			desc:          "Source account is same as transaction",
			sourceAccount: "",
			expected:      "GAUJETIZVEP2NRYLUESJ3LS66NVCEGMON4UDCBCSBEVPIID773P2W6AY",
		},
		{
			desc:          "Source account is different to transaction",
			sourceAccount: "GDMQUXK7ZUCWM5472ZU3YLDP4BMJLQQ76DEMNYDEY2ODEEGGRKLEWGW2",
			expected:      "GDMQUXK7ZUCWM5472ZU3YLDP4BMJLQQ76DEMNYDEY2ODEEGGRKLEWGW2",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tt := assert.New(t)
			transaction, err := buildTransaction(
				1,
				"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
				nil,
			)
			tt.NoError(err)
			op := transaction.Envelope.Tx.Operations[0]
			if len(tc.sourceAccount) > 0 {
				sourceAccount := xdr.MustAddress(tc.sourceAccount)
				op.SourceAccount = &sourceAccount
			}

			operation := TransactionOperation{
				Index:          1,
				Transaction:    transaction,
				Operation:      op,
				LedgerSequence: 1,
			}

			tt.Equal(tc.expected, operation.SourceAccount().Address())
		})
	}
}

func TestTransactionOperationType(t *testing.T) {
	tt := assert.New(t)
	transaction, err := buildTransaction(
		1,
		"AAAAACiSTRmpH6bHC6Ekna5e82oiGY5vKDEEUgkq9CB//t+rAAAAZAEXUhsAADDGAAAAAQAAAAAAAAAAAAAAAF3v3WAAAAABAAAACjEwOTUzMDMyNTAAAAAAAAEAAAAAAAAAAQAAAAAOr5CG1ax6qG2fBEgXJlF0sw5W0irOS6N/NRDbavBm4QAAAAAAAAAAE32fwAAAAAAAAAABf/7fqwAAAEAkWgyAgV5tF3m1y1TIDYkNXP8pZLAwcxhWEi4f3jcZJK7QrKSXhKoawVGrp5NNs4y9dgKt8zHZ8KbJreFBUsIB",
		nil,
	)
	tt.NoError(err)

	operation := TransactionOperation{
		Index:          1,
		Transaction:    transaction,
		Operation:      transaction.Envelope.Tx.Operations[0],
		LedgerSequence: 1,
	}

	tt.Equal(xdr.OperationTypePayment, operation.OperationType())
}
func TestTransactionOperationDetails(t *testing.T) {
	testCases := []struct {
		desc     string
		envelope string
		result   string
		index    uint32
		expected map[string]interface{}
	}{
		{
			desc:     "createAccount",
			envelope: "AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAAaAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAoZftFP3p4ifbTm6hQdieotu3Zw9E05GtoSh5MBytEpQAAAACVAvkAAAAAAAAAAABVvwF9wAAAEDHU95E9wxgETD8TqxUrkgC0/7XHyNDts6Q5huRHfDRyRcoHdv7aMp/sPvC3RPkXjOMjgbKJUX7SgExUeYB5f8F",
			index:    0,
			expected: map[string]interface{}{
				"account":          "GCQZP3IU7XU6EJ63JZXKCQOYT2RNXN3HB5CNHENNUEUHSMA4VUJJJSEN",
				"funder":           "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H",
				"starting_balance": "1000.0000000",
			},
		},
		{
			desc:     "payment",
			envelope: "AAAAABpcjiETZ0uhwxJJhgBPYKWSVJy2TZ2LI87fqV1cUf/UAAAAZAAAADcAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAGlyOIRNnS6HDEkmGAE9gpZJUnLZNnYsjzt+pXVxR/9QAAAAAAAAAAAX14QAAAAAAAAAAAVxR/9QAAABAK6pcXYMzAEmH08CZ1LWmvtNDKauhx+OImtP/Lk4hVTMJRVBOebVs5WEPj9iSrgGT0EswuDCZ2i5AEzwgGof9Ag==",
			index:    0,
			expected: map[string]interface{}{
				"amount":     "10.0000000",
				"asset_type": "native",
				"from":       "GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y",
				"to":         "GANFZDRBCNTUXIODCJEYMACPMCSZEVE4WZGZ3CZDZ3P2SXK4KH75IK6Y",
			},
		},
		{
			desc:     "pathPaymentStrictReceive",
			envelope: "AAAAAONt/6wGI884Zi6sYDYC1GOV/drnh4OcRrTrqJPoOTUKAAAAZAAAABAAAAADAAAAAAAAAAAAAAABAAAAAAAAAAIAAAAAAAAAADuaygAAAAAABAjoBMEUiZNLUjsWXL1iK59D90Li4w56076b8HKxZfIAAAABRVVSAAAAAAAuwvNzNk9twbuJHUBqnX26GYI3MbCdpQU9t4n6EVRXsQAAAAA7msoAAAAAAAAAAAAAAAAB6Dk1CgAAAEB+7jxesBKKrF343onyycjp2tiQLZiGH2ETl+9fuOqotveY2rIgvt9ng+QJ2aDP3+PnDsYEa9ZUaA+Zne2nIGgE",
			result:   "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAACAAAAAAAAAAEAAAAALsLzczZPbcG7iR1Aap19uhmCNzGwnaUFPbeJ+hFUV7EAAAAAAAAAAgAAAAFFVVIAAAAAAC7C83M2T23Bu4kdQGqdfboZgjcxsJ2lBT23ifoRVFexAAAAADuaygAAAAAAAAAAADuaygAAAAAABAjoBMEUiZNLUjsWXL1iK59D90Li4w56076b8HKxZfIAAAABRVVSAAAAAAAuwvNzNk9twbuJHUBqnX26GYI3MbCdpQU9t4n6EVRXsQAAAAA7msoAAAAAAA==",
			index:    0,
			expected: map[string]interface{}{
				"to":                "GACAR2AEYEKITE2LKI5RMXF5MIVZ6Q7XILROGDT22O7JX4DSWFS7FDDP",
				"from":              "GDRW375MAYR46ODGF2WGANQC2RRZL7O246DYHHCGWTV2RE7IHE2QUQLD",
				"path":              []map[string]interface{}{},
				"amount":            "100.0000000",
				"asset_code":        "EUR",
				"asset_type":        "credit_alphanum4",
				"source_max":        "100.0000000",
				"asset_issuer":      "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU",
				"source_amount":     "100.0000000",
				"source_asset_type": "native",
			},
		},
		{
			desc:     "manageSellOffer",
			envelope: "AAAAAC7C83M2T23Bu4kdQGqdfboZgjcxsJ2lBT23ifoRVFexAAAAZAAAABAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAMAAAAAAAAAAVVTRAAAAAAALsLzczZPbcG7iR1Aap19uhmCNzGwnaUFPbeJ+hFUV7EAAAAA7msoAAAAAAEAAAACAAAAAAAAAAAAAAAAAAAAARFUV7EAAABALuai5QxceFbtAiC5nkntNVnvSPeWR+C+FgplPAdRgRS+PPESpUiSCyuiwuhmvuDw7kwxn+A6E0M4ca1s2qzMAg==",
			result:   "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAADAAAAAAAAAAAAAAAAAAAAAC7C83M2T23Bu4kdQGqdfboZgjcxsJ2lBT23ifoRVFexAAAAAAAAAAEAAAAAAAAAAVVTRAAAAAAALsLzczZPbcG7iR1Aap19uhmCNzGwnaUFPbeJ+hFUV7EAAAAA7msoAAAAAAEAAAACAAAAAAAAAAAAAAAA",
			index:    0,
			expected: map[string]interface{}{
				"amount":              "400.0000000",
				"buying_asset_code":   "USD",
				"buying_asset_issuer": "GAXMF43TGZHW3QN3REOUA2U5PW5BTARXGGYJ3JIFHW3YT6QRKRL3CPPU",
				"buying_asset_type":   "credit_alphanum4",
				"offer_id":            xdr.Int64(0),
				"price":               "0.5000000",
				"price_r": map[string]interface{}{
					"d": xdr.Int32(2),
					"n": xdr.Int32(1),
				},
				"selling_asset_type": "native",
			},
		},
		{
			desc:     "createPassiveSellOffer",
			envelope: "AAAAAHxm7WmlvJxH5BuTz9Qn+PnWcTY9zK8s6YgIjqQyboYYAAAAZAAAABkAAAABAAAAAAAAAAAAAAABAAAAAAAAAAQAAAABVVNEAAAAAAB8Zu1ppbycR+Qbk8/UJ/j51nE2PcyvLOmICI6kMm6GGAAAAAAAAAAAdzWUAAAAAAEAAAABAAAAAAAAAAEyboYYAAAAQBqzCYDuLYn/jXhfEVxEGigMCJGoOBCK92lUb3Um15PgwSJ63tNl+FpH8+y5c+mCs/rzcvdyo9uXdodd4LXWiQg=",
			result:   "AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAADAAAAAAAAAAAAAAAAAAAAAHxm7WmlvJxH5BuTz9Qn+PnWcTY9zK8s6YgIjqQyboYYAAAAAAAAAAUAAAABVVNEAAAAAAB8Zu1ppbycR+Qbk8/UJ/j51nE2PcyvLOmICI6kMm6GGAAAAAAAAAAAdzWUAAAAAAEAAAABAAAAAQAAAAAAAAAA",
			index:    0,
			expected: map[string]interface{}{
				"price":  "1.0000000",
				"amount": "200.0000000",
				"price_r": map[string]interface{}{
					"d": xdr.Int32(1),
					"n": xdr.Int32(1),
				},
				"buying_asset_type":    "native",
				"selling_asset_code":   "USD",
				"selling_asset_type":   "credit_alphanum4",
				"selling_asset_issuer": "GB6GN3LJUW6JYR7EDOJ47VBH7D45M4JWHXGK6LHJRAEI5JBSN2DBQY7Q",
			},
		},
		{
			desc:     "setOption -  home domain",
			envelope: "AAAAAJBSxgo7D+SP4ldcjl6iwJjYtL5+AfTMUdled6cwnjBQAAAAZAAAABsAAAAGAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAC2V4YW1wbGUuY29tAAAAAAAAAAAAAAAAATCeMFAAAABAkID6CkBHP9eovLQXkMQJ7QkE6NWlmdKGmLxaiI1YaVKZaKJxz5P85x+6wzpYxxbs6Bd2l4qxVjS7Q36DwRiqBA==",
			index:    0,
			expected: map[string]interface{}{
				"home_domain": xdr.String32("example.com"),
			},
		},
		{
			desc:     "setOption - signer",
			envelope: "AAAAAJBSxgo7D+SP4ldcjl6iwJjYtL5+AfTMUdled6cwnjBQAAAAZAAAABsAAAAMAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAB8ndnLViBPKqPJAcSNhZzc2mH7fQ7RtzGyFA8mFkMTkAAAAAAAAAAAAAAAATCeMFAAAABAOb0qGWnk1WrSUXS6iQFocaIOY/BDmgG1zTmlPyg0boSid3jTBK3z9U8+IPGAOELNLgkQHtgGYFgFGMio1xY+BQ==",
			index:    0,
			expected: map[string]interface{}{
				"signer_key":    "GB6J3WOLKYQE6KVDZEA4JDMFTTONUYP3PUHNDNZRWIKA6JQWIMJZATFE",
				"signer_weight": xdr.Uint32(0),
			},
		},
		{
			desc:     "setOption - inflation dest",
			envelope: "AAAAAOPd2ARCnU3lTd8FI4LH+evle2IKY0nagwlkzH4xgrcnAAAAZAAAAC0AAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAABAAAAAOPd2ARCnU3lTd8FI4LH+evle2IKY0nagwlkzH4xgrcnAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABMYK3JwAAAEAOkGOPTOBDSQ7nW2Zn+bls2PDUebk2/k3/gqHKQ8eYOFsD6nBeEvyMD858vo5BabjQwB9injABIM8esDh7bEkC",
			index:    0,
			expected: map[string]interface{}{
				"inflation_dest": "GDR53WAEIKOU3ZKN34CSHAWH7HV6K63CBJRUTWUDBFSMY7RRQK3SPKOS",
			},
		},
		{
			desc:     "setOption - set flags",
			envelope: "AAAAAPkmOJur5F/mOxTJDb+0bMLCJGDRl3meP2MBEDVKSPP4AAAAZAAAACYAAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABSkjz+AAAAECyjDa1e+jtXukTrHluO7x0Mx7Wj4mRoM4S5UAFmRV+2rVoxjMwqFJhtYnEAUV19+C5ycp5jOLLpWxrCeRKJQUG",
			index:    0,
			expected: map[string]interface{}{
				"set_flags": []int32{1, 2},
				"set_flags_s": []string{
					"auth_required",
					"auth_revocable",
				},
			},
		},
		{
			desc:     "setOption - master weight",
			envelope: "AAAAAJBSxgo7D+SP4ldcjl6iwJjYtL5+AfTMUdled6cwnjBQAAAAZAAAABsAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAEAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAATCeMFAAAABAAd6MzHDjUdRtHozzDnD3jJA+uRDCar3PQtuH/43pnROzk1HkovJPQ1YyzcpOb/NeuU/LKNzseL0PJNasVX1lAQ==",
			index:    0,
			expected: map[string]interface{}{
				"master_key_weight": xdr.Uint32(2),
			},
		},
		{
			desc:     "setOption - thresholds",
			envelope: "AAAAAJBSxgo7D+SP4ldcjl6iwJjYtL5+AfTMUdled6cwnjBQAAAAZAAAABsAAAAFAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAACAAAAAQAAAAIAAAAAAAAAAAAAAAAAAAABMJ4wUAAAAEAnFzc6kqweyIL4TzIDbr+8GUOGGs1W5jcX5iSNw4DeonzQARlejYJ9NOn/XkrcoC9Hvd8hc5lNx+1h991GxJUJ",
			index:    0,
			expected: map[string]interface{}{
				"low_threshold":  xdr.Uint32(0),
				"med_threshold":  xdr.Uint32(2),
				"high_threshold": xdr.Uint32(2),
			},
		},
		{
			desc:     "setOption - clears flags",
			envelope: "AAAAAJBSxgo7D+SP4ldcjl6iwJjYtL5+AfTMUdled6cwnjBQAAAAZAAAABsAAAALAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAMAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABMJ4wUAAAAEAFytUxjxN4bnJMrEJkSprnES9iGpOxAsNOFYrTP/xtGVk/PZ2oThUW+/hLRIk+hYYEgF21Gf58N/abJKFpqlsI",
			index:    0,
			expected: map[string]interface{}{
				"clear_flags": []int32{1, 2},
				"clear_flags_s": []string{
					"auth_required",
					"auth_revocable",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tt := assert.New(t)
			var result *string
			if len(tc.result) > 0 {
				result = &tc.result
			}

			transaction, err := buildTransaction(
				1,
				tc.envelope,
				result,
			)
			tt.NoError(err)

			operation := TransactionOperation{
				Index:          tc.index,
				Transaction:    transaction,
				Operation:      transaction.Envelope.Tx.Operations[tc.index],
				LedgerSequence: 1,
			}

			tt.Equal(tc.expected, operation.Details())
		})
	}
}

func buildTransaction(index uint32, envelope string, result *string) (io.LedgerTransaction, error) {
	transaction := io.LedgerTransaction{
		Index:    1,
		Envelope: xdr.TransactionEnvelope{},
		Result:   xdr.TransactionResultPair{},
	}
	err := xdr.SafeUnmarshalBase64(
		envelope,
		&transaction.Envelope,
	)
	if err != nil {
		return transaction, err
	}

	if result != nil {
		err = xdr.SafeUnmarshalBase64(
			*result,
			&transaction.Result.Result,
		)
		if err != nil {
			return transaction, err
		}
	}

	return transaction, nil
}
