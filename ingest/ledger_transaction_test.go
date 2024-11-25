package ingest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stellar/go/xdr"
)

func TestChangeAccountChangedExceptSignersInvalidType(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeOffer,
	}

	var err error
	assert.Panics(t, func() {
		_, err = change.AccountChangedExceptSigners()
	})
	// the following is here only to avoid false-positive warning by the linter.
	require.NoError(t, err)
}

func TestGetContractEventsEmpty(t *testing.T) {
	tx := LedgerTransaction{
		FeeChanges: xdr.LedgerEntryChanges{},
		UnsafeMeta: xdr.TransactionMeta{
			V: 3,
			V3: &xdr.TransactionMetaV3{
				SorobanMeta: &xdr.SorobanTransactionMeta{
					Events: []xdr.ContractEvent{},
				},
			},
		},
	}

	events, err := tx.GetDiagnosticEvents()
	assert.NoError(t, err)
	assert.Empty(t, events)
}

func TestGetContractEventsSingle(t *testing.T) {
	value := xdr.Uint32(1)
	tx := LedgerTransaction{
		FeeChanges: xdr.LedgerEntryChanges{},
		UnsafeMeta: xdr.TransactionMeta{
			V: 3,
			V3: &xdr.TransactionMetaV3{
				SorobanMeta: &xdr.SorobanTransactionMeta{
					Events: []xdr.ContractEvent{
						{
							Type: xdr.ContractEventTypeSystem,
							Body: xdr.ContractEventBody{
								V: 0,
								V0: &xdr.ContractEventV0{
									Data: xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &value},
								},
							},
						},
					},
				},
			},
		},
	}

	events, err := tx.GetDiagnosticEvents()
	assert.Len(t, events, 1)
	assert.True(t, events[0].InSuccessfulContractCall)
	assert.Equal(t, *events[0].Event.Body.V0.Data.U32, value)

	tx.UnsafeMeta.V = 0
	_, err = tx.GetDiagnosticEvents()
	assert.EqualError(t, err, "unsupported TransactionMeta version: 0")

	tx.UnsafeMeta.V = 4
	_, err = tx.GetDiagnosticEvents()
	assert.EqualError(t, err, "unsupported TransactionMeta version: 4")

	tx.UnsafeMeta.V = 1
	events, err = tx.GetDiagnosticEvents()
	assert.NoError(t, err)
	assert.Empty(t, events)

	tx.UnsafeMeta.V = 2
	events, err = tx.GetDiagnosticEvents()
	assert.NoError(t, err)
	assert.Empty(t, events)
}

func TestGetContractEventsMultiple(t *testing.T) {
	values := make([]xdr.Uint32, 2)
	for i := range values {
		values[i] = xdr.Uint32(i)
	}
	tx := LedgerTransaction{
		FeeChanges: xdr.LedgerEntryChanges{},
		UnsafeMeta: xdr.TransactionMeta{
			V: 3,
			V3: &xdr.TransactionMetaV3{
				SorobanMeta: &xdr.SorobanTransactionMeta{
					Events: []xdr.ContractEvent{
						{
							Type: xdr.ContractEventTypeSystem,
							Body: xdr.ContractEventBody{
								V: 0,
								V0: &xdr.ContractEventV0{
									Data: xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &values[0]},
								},
							},
						},
						{
							Type: xdr.ContractEventTypeSystem,
							Body: xdr.ContractEventBody{
								V: 0,
								V0: &xdr.ContractEventV0{
									Data: xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &values[1]},
								},
							},
						},
					},
				},
			},
		},
	}
	events, err := tx.GetDiagnosticEvents()
	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.True(t, events[0].InSuccessfulContractCall)
	assert.Equal(t, *events[0].Event.Body.V0.Data.U32, values[0])
	assert.True(t, events[1].InSuccessfulContractCall)
	assert.Equal(t, *events[1].Event.Body.V0.Data.U32, values[1])
}

func TestGetDiagnosticEventsEmpty(t *testing.T) {
	tx := LedgerTransaction{
		FeeChanges: xdr.LedgerEntryChanges{},
		UnsafeMeta: xdr.TransactionMeta{
			V: 3,
			V3: &xdr.TransactionMetaV3{
				SorobanMeta: &xdr.SorobanTransactionMeta{
					DiagnosticEvents: []xdr.DiagnosticEvent{},
				},
			},
		},
	}

	events, err := tx.GetDiagnosticEvents()
	assert.NoError(t, err)
	assert.Empty(t, events)
}

func TestGetDiagnosticEventsSingle(t *testing.T) {
	value := xdr.Uint32(1)
	tx := LedgerTransaction{
		FeeChanges: xdr.LedgerEntryChanges{},
		UnsafeMeta: xdr.TransactionMeta{
			V: 3,
			V3: &xdr.TransactionMetaV3{
				SorobanMeta: &xdr.SorobanTransactionMeta{
					DiagnosticEvents: []xdr.DiagnosticEvent{
						{
							InSuccessfulContractCall: false,
							Event: xdr.ContractEvent{
								Type: xdr.ContractEventTypeSystem,
								Body: xdr.ContractEventBody{
									V: 0,
									V0: &xdr.ContractEventV0{
										Data: xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &value},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	events, err := tx.GetDiagnosticEvents()
	assert.NoError(t, err)
	assert.Len(t, events, 1)
	assert.False(t, events[0].InSuccessfulContractCall)
	assert.Equal(t, *events[0].Event.Body.V0.Data.U32, value)

	tx.UnsafeMeta.V = 0
	_, err = tx.GetDiagnosticEvents()
	assert.EqualError(t, err, "unsupported TransactionMeta version: 0")

	tx.UnsafeMeta.V = 4
	_, err = tx.GetDiagnosticEvents()
	assert.EqualError(t, err, "unsupported TransactionMeta version: 4")

	tx.UnsafeMeta.V = 1
	events, err = tx.GetDiagnosticEvents()
	assert.NoError(t, err)
	assert.Empty(t, events)

	tx.UnsafeMeta.V = 2
	events, err = tx.GetDiagnosticEvents()
	assert.NoError(t, err)
	assert.Empty(t, events)
}

func TestGetDiagnosticEventsMultiple(t *testing.T) {
	values := make([]xdr.Uint32, 2)
	for i := range values {
		values[i] = xdr.Uint32(i)
	}
	tx := LedgerTransaction{
		FeeChanges: xdr.LedgerEntryChanges{},
		UnsafeMeta: xdr.TransactionMeta{
			V: 3,
			V3: &xdr.TransactionMetaV3{
				SorobanMeta: &xdr.SorobanTransactionMeta{
					DiagnosticEvents: []xdr.DiagnosticEvent{
						{
							InSuccessfulContractCall: true,

							Event: xdr.ContractEvent{
								Type: xdr.ContractEventTypeSystem,
								Body: xdr.ContractEventBody{
									V: 0,
									V0: &xdr.ContractEventV0{
										Data: xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &values[0]},
									},
								},
							},
						},
						{
							InSuccessfulContractCall: true,
							Event: xdr.ContractEvent{
								Type: xdr.ContractEventTypeSystem,
								Body: xdr.ContractEventBody{
									V: 0,
									V0: &xdr.ContractEventV0{
										Data: xdr.ScVal{Type: xdr.ScValTypeScvU32, U32: &values[1]},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	events, err := tx.GetDiagnosticEvents()
	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.True(t, events[0].InSuccessfulContractCall)
	assert.Equal(t, *events[0].Event.Body.V0.Data.U32, values[0])
	assert.True(t, events[1].InSuccessfulContractCall)
	assert.Equal(t, *events[1].Event.Body.V0.Data.U32, values[1])
}

func TestFeeMetaAndOperationsChangesSeparate(t *testing.T) {
	tx := LedgerTransaction{
		LedgerVersion: 12,
		FeeChanges: xdr.LedgerEntryChanges{
			xdr.LedgerEntryChange{
				Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
				State: &xdr.LedgerEntry{
					Data: xdr.LedgerEntryData{
						Type: xdr.LedgerEntryTypeAccount,
						Account: &xdr.AccountEntry{
							AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
							Balance:   100,
						},
					},
				},
			},
			xdr.LedgerEntryChange{
				Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
				Updated: &xdr.LedgerEntry{
					Data: xdr.LedgerEntryData{
						Type: xdr.LedgerEntryTypeAccount,
						Account: &xdr.AccountEntry{
							AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
							Balance:   200,
						},
					},
				},
			},
		},
		UnsafeMeta: xdr.TransactionMeta{
			V: 1,
			V1: &xdr.TransactionMetaV1{
				Operations: []xdr.OperationMeta{
					{
						Changes: xdr.LedgerEntryChanges{
							xdr.LedgerEntryChange{
								Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
								State: &xdr.LedgerEntry{
									Data: xdr.LedgerEntryData{
										Type: xdr.LedgerEntryTypeAccount,
										Account: &xdr.AccountEntry{
											AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
											Balance:   300,
										},
									},
								},
							},
							xdr.LedgerEntryChange{
								Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
								Updated: &xdr.LedgerEntry{
									Data: xdr.LedgerEntryData{
										Type: xdr.LedgerEntryTypeAccount,
										Account: &xdr.AccountEntry{
											AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
											Balance:   400,
										},
									},
								},
							},
						},
					},
				},
			},
		}}

	feeChanges := tx.GetFeeChanges()
	assert.Len(t, feeChanges, 1)
	assert.Equal(t, feeChanges[0].Pre.Data.MustAccount().Balance, xdr.Int64(100))
	assert.Equal(t, feeChanges[0].Post.Data.MustAccount().Balance, xdr.Int64(200))

	metaChanges, err := tx.GetChanges()
	assert.NoError(t, err)
	assert.Len(t, metaChanges, 1)
	assert.Equal(t, metaChanges[0].Pre.Data.MustAccount().Balance, xdr.Int64(300))
	assert.Equal(t, metaChanges[0].Post.Data.MustAccount().Balance, xdr.Int64(400))

	operationChanges, err := tx.GetOperationChanges(0)
	assert.NoError(t, err)
	assert.Len(t, operationChanges, 1)
	assert.Equal(t, operationChanges[0].Pre.Data.MustAccount().Balance, xdr.Int64(300))
	assert.Equal(t, operationChanges[0].Post.Data.MustAccount().Balance, xdr.Int64(400))

	// Ignore operation meta if tx result is txInternalError
	// https://github.com/stellar/go/issues/2111
	tx.Result.Result.Result.Code = xdr.TransactionResultCodeTxInternalError
	metaChanges, err = tx.GetChanges()
	assert.NoError(t, err)
	assert.Len(t, metaChanges, 0)

	operationChanges, err = tx.GetOperationChanges(0)
	assert.NoError(t, err)
	assert.Len(t, operationChanges, 0)

	// Starting from protocol 13, we no longer need to ignore txInternalError
	tx.LedgerVersion = 13

	metaChanges, err = tx.GetChanges()
	assert.NoError(t, err)
	assert.Len(t, metaChanges, 1)
	assert.Equal(t, metaChanges[0].Pre.Data.MustAccount().Balance, xdr.Int64(300))
	assert.Equal(t, metaChanges[0].Post.Data.MustAccount().Balance, xdr.Int64(400))

	operationChanges, err = tx.GetOperationChanges(0)
	assert.NoError(t, err)
	assert.Len(t, operationChanges, 1)
	assert.Equal(t, operationChanges[0].Pre.Data.MustAccount().Balance, xdr.Int64(300))
	assert.Equal(t, operationChanges[0].Post.Data.MustAccount().Balance, xdr.Int64(400))
}

func TestFailedTransactionOperationChangesMeta(t *testing.T) {
	testCases := []struct {
		desc string
		meta xdr.TransactionMeta
	}{
		{
			desc: "V0",
			meta: xdr.TransactionMeta{
				Operations: &[]xdr.OperationMeta{},
			},
		},
		{
			desc: "V1",
			meta: xdr.TransactionMeta{
				V:  1,
				V1: &xdr.TransactionMetaV1{},
			},
		},
		{
			desc: "V2",
			meta: xdr.TransactionMeta{
				V:  2,
				V2: &xdr.TransactionMetaV2{},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tx := LedgerTransaction{
				Result: xdr.TransactionResultPair{
					Result: xdr.TransactionResult{
						Result: xdr.TransactionResultResult{
							Code: xdr.TransactionResultCodeTxFailed,
						},
					},
				},
				UnsafeMeta: tc.meta,
			}

			operationChanges, err := tx.GetOperationChanges(0)
			if tx.UnsafeMeta.V == 0 {
				assert.Error(t, err)
				assert.EqualError(t, err, "TransactionMeta.V=0 not supported")
			} else {
				assert.NoError(t, err)
				assert.Len(t, operationChanges, 0)
			}
		})
	}
}
func TestMetaV2Order(t *testing.T) {
	tx := LedgerTransaction{
		UnsafeMeta: xdr.TransactionMeta{
			V: 2,
			V2: &xdr.TransactionMetaV2{
				TxChangesBefore: xdr.LedgerEntryChanges{
					xdr.LedgerEntryChange{
						Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
						State: &xdr.LedgerEntry{
							Data: xdr.LedgerEntryData{
								Type: xdr.LedgerEntryTypeAccount,
								Account: &xdr.AccountEntry{
									AccountId: xdr.MustAddress("GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX"),
									Balance:   100,
								},
							},
						},
					},
					xdr.LedgerEntryChange{
						Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
						Updated: &xdr.LedgerEntry{
							Data: xdr.LedgerEntryData{
								Type: xdr.LedgerEntryTypeAccount,
								Account: &xdr.AccountEntry{
									AccountId: xdr.MustAddress("GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX"),
									Balance:   200,
								},
							},
						},
					},
					xdr.LedgerEntryChange{
						Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
						State: &xdr.LedgerEntry{
							Data: xdr.LedgerEntryData{
								Type: xdr.LedgerEntryTypeAccount,
								Account: &xdr.AccountEntry{
									AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
									Balance:   100,
								},
							},
						},
					},
					xdr.LedgerEntryChange{
						Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
						Updated: &xdr.LedgerEntry{
							Data: xdr.LedgerEntryData{
								Type: xdr.LedgerEntryTypeAccount,
								Account: &xdr.AccountEntry{
									AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
									Balance:   200,
								},
							},
						},
					},
				},
				Operations: []xdr.OperationMeta{
					{
						Changes: xdr.LedgerEntryChanges{
							xdr.LedgerEntryChange{
								Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
								State: &xdr.LedgerEntry{
									Data: xdr.LedgerEntryData{
										Type: xdr.LedgerEntryTypeAccount,
										Account: &xdr.AccountEntry{
											AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
											Balance:   300,
										},
									},
								},
							},
							xdr.LedgerEntryChange{
								Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
								Updated: &xdr.LedgerEntry{
									Data: xdr.LedgerEntryData{
										Type: xdr.LedgerEntryTypeAccount,
										Account: &xdr.AccountEntry{
											AccountId: xdr.MustAddress("GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A"),
											Balance:   400,
										},
									},
								},
							},
						},
					},
				},
				TxChangesAfter: xdr.LedgerEntryChanges{
					xdr.LedgerEntryChange{
						Type: xdr.LedgerEntryChangeTypeLedgerEntryState,
						State: &xdr.LedgerEntry{
							Data: xdr.LedgerEntryData{
								Type: xdr.LedgerEntryTypeAccount,
								Account: &xdr.AccountEntry{
									AccountId: xdr.MustAddress("GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX"),
									Balance:   300,
								},
							},
						},
					},
					xdr.LedgerEntryChange{
						Type: xdr.LedgerEntryChangeTypeLedgerEntryUpdated,
						Updated: &xdr.LedgerEntry{
							Data: xdr.LedgerEntryData{
								Type: xdr.LedgerEntryTypeAccount,
								Account: &xdr.AccountEntry{
									AccountId: xdr.MustAddress("GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX"),
									Balance:   400,
								},
							},
						},
					},
				},
			},
		}}

	metaChanges, err := tx.GetChanges()
	assert.NoError(t, err)
	assert.Len(t, metaChanges, 4)

	change := metaChanges[0]
	id := change.Pre.Data.MustAccount().AccountId
	assert.Equal(t, id.Address(), "GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX")
	assert.Equal(t, change.Pre.Data.MustAccount().Balance, xdr.Int64(100))
	assert.Equal(t, change.Post.Data.MustAccount().Balance, xdr.Int64(200))

	change = metaChanges[1]
	id = change.Pre.Data.MustAccount().AccountId
	assert.Equal(t, id.Address(), "GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A")
	assert.Equal(t, change.Pre.Data.MustAccount().Balance, xdr.Int64(100))
	assert.Equal(t, change.Post.Data.MustAccount().Balance, xdr.Int64(200))

	change = metaChanges[2]
	id = change.Pre.Data.MustAccount().AccountId
	assert.Equal(t, id.Address(), "GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A")
	assert.Equal(t, change.Pre.Data.MustAccount().Balance, xdr.Int64(300))
	assert.Equal(t, change.Post.Data.MustAccount().Balance, xdr.Int64(400))

	change = metaChanges[3]
	id = change.Pre.Data.MustAccount().AccountId
	assert.Equal(t, id.Address(), "GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX")
	assert.Equal(t, change.Pre.Data.MustAccount().Balance, xdr.Int64(300))
	assert.Equal(t, change.Post.Data.MustAccount().Balance, xdr.Int64(400))

	operationChanges, err := tx.GetOperationChanges(0)
	assert.NoError(t, err)
	assert.Len(t, operationChanges, 1)

	// Ignore operations meta and txChangesAfter if txInternalError
	// https://github.com/stellar/go/issues/2111
	tx.Result.Result.Result.Code = xdr.TransactionResultCodeTxInternalError
	metaChanges, err = tx.GetChanges()
	assert.NoError(t, err)
	assert.Len(t, metaChanges, 2)

	change = metaChanges[0]
	id = change.Pre.Data.MustAccount().AccountId
	assert.Equal(t, id.Address(), "GACMZD5VJXTRLKVET72CETCYKELPNCOTTBDC6DHFEUPLG5DHEK534JQX")
	assert.Equal(t, change.Pre.Data.MustAccount().Balance, xdr.Int64(100))
	assert.Equal(t, change.Post.Data.MustAccount().Balance, xdr.Int64(200))

	change = metaChanges[1]
	id = change.Pre.Data.MustAccount().AccountId
	assert.Equal(t, id.Address(), "GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A")
	assert.Equal(t, change.Pre.Data.MustAccount().Balance, xdr.Int64(100))
	assert.Equal(t, change.Post.Data.MustAccount().Balance, xdr.Int64(200))

	operationChanges, err = tx.GetOperationChanges(0)
	assert.NoError(t, err)
	assert.Len(t, operationChanges, 0)

}

func TestMetaV0(t *testing.T) {
	tx := LedgerTransaction{
		UnsafeMeta: xdr.TransactionMeta{
			V: 0,
		}}

	_, err := tx.GetChanges()
	assert.Error(t, err)
	assert.EqualError(t, err, "TransactionMeta.V=0 not supported")
}

func TestChangeAccountChangedExceptSignersLastModifiedLedgerSeq(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 11,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
	}
	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.True(t, changed)
}

func TestChangeAccountChangedExceptSignersNoPre(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre:  nil,
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
	}
	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.True(t, changed)
}

func TestChangeAccountChangedExceptSignersNoPost(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
				},
			},
		},
		Post: nil,
	}
	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.True(t, changed)
}

func TestChangeAccountChangedExceptSignersMasterKeyRemoved(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 1
					Thresholds: [4]byte{1, 1, 1, 1},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					// Master weight = 0
					Thresholds: [4]byte{0, 1, 1, 1},
				},
			},
		},
	}

	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.True(t, changed)
}

func TestChangeAccountChangedExceptSignersSignerChange(t *testing.T) {
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers: []xdr.Signer{
						{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 1,
						},
					},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId: xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Signers: []xdr.Signer{
						{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 2,
						},
					},
				},
			},
		},
	}

	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.False(t, changed)
}

func TestChangeAccountChangedExceptSignersNoChanges(t *testing.T) {
	inflationDest := xdr.MustAddress("GBAH2GBLJB54JAROJ3FVO4ZTTJJI3XKOBTMJOZFUJ3UHYIVNJTLJUYFY")
	change := Change{
		Type: xdr.LedgerEntryTypeAccount,
		Pre: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:     xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Balance:       1000,
					SeqNum:        432894732,
					NumSubEntries: 2,
					InflationDest: &inflationDest,
					Flags:         4,
					HomeDomain:    "stellar.org",
					Thresholds:    [4]byte{1, 1, 1, 1},
					Signers: []xdr.Signer{
						{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 1,
						},
					},
					Ext: xdr.AccountEntryExt{
						V: 1,
						V1: &xdr.AccountEntryExtensionV1{
							Liabilities: xdr.Liabilities{
								Buying:  10,
								Selling: 20,
							},
						},
					},
				},
			},
		},
		Post: &xdr.LedgerEntry{
			LastModifiedLedgerSeq: 10,
			Data: xdr.LedgerEntryData{
				Type: xdr.LedgerEntryTypeAccount,
				Account: &xdr.AccountEntry{
					AccountId:     xdr.MustAddress("GC3C4AKRBQLHOJ45U4XG35ESVWRDECWO5XLDGYADO6DPR3L7KIDVUMML"),
					Balance:       1000,
					SeqNum:        432894732,
					NumSubEntries: 2,
					InflationDest: &inflationDest,
					Flags:         4,
					HomeDomain:    "stellar.org",
					Thresholds:    [4]byte{1, 1, 1, 1},
					Signers: []xdr.Signer{
						{
							Key:    xdr.MustSigner("GCCCU34WDY2RATQTOOQKY6SZWU6J5DONY42SWGW2CIXGW4LICAGNRZKX"),
							Weight: 1,
						},
					},
					Ext: xdr.AccountEntryExt{
						V: 1,
						V1: &xdr.AccountEntryExtensionV1{
							Liabilities: xdr.Liabilities{
								Buying:  10,
								Selling: 20,
							},
						},
					},
				},
			},
		},
	}

	changed, err := change.AccountChangedExceptSigners()
	assert.NoError(t, err)
	assert.False(t, changed)

	// Make sure pre and post not modified
	assert.NotNil(t, change.Pre.Data.Account.Signers)
	assert.Len(t, change.Pre.Data.Account.Signers, 1)

	assert.NotNil(t, change.Post.Data.Account.Signers)
	assert.Len(t, change.Post.Data.Account.Signers, 1)
}
