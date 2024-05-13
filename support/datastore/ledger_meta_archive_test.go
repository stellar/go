package datastore

import (
	"fmt"
	"testing"

	"github.com/stellar/go/xdr"
	"github.com/stretchr/testify/require"
)

func createLedgerCloseMeta(ledgerSeq uint32) xdr.LedgerCloseMeta {
	return xdr.LedgerCloseMeta{
		V0: &xdr.LedgerCloseMetaV0{
			LedgerHeader: xdr.LedgerHeaderHistoryEntry{
				Header: xdr.LedgerHeader{
					LedgerSeq: xdr.Uint32(ledgerSeq),
				},
			},
		},
	}
}

func TestLedgerMetaArchive_AddLedgerValidRange(t *testing.T) {

	tests := []struct {
		name     string
		startSeq uint32
		endSeq   uint32
		seqNum   uint32
		errMsg   string
	}{
		{startSeq: 10, endSeq: 100, seqNum: 10, errMsg: ""},
		{startSeq: 10, endSeq: 100, seqNum: 11, errMsg: ""},
		{startSeq: 10, endSeq: 100, seqNum: 99, errMsg: ""},
		{startSeq: 10, endSeq: 100, seqNum: 100, errMsg: ""},
		{startSeq: 10, endSeq: 100, seqNum: 9, errMsg: "ledger sequence 9 is outside valid range [10, 100]"},
		{startSeq: 10, endSeq: 100, seqNum: 101, errMsg: "ledger sequence 101 is outside valid range [10, 100]"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("range [%d, %d]: Add seq %d", tt.startSeq, tt.endSeq, tt.seqNum),
			func(t *testing.T) {
				f := NewLedgerMetaArchive("", tt.startSeq, tt.endSeq)
				err := f.AddLedger(createLedgerCloseMeta(tt.seqNum))
				if tt.errMsg != "" {
					require.EqualError(t, err, tt.errMsg)
				} else {
					require.NoError(t, err)
				}
			})
	}
}
func TestLedgerMetaArchive_AddLedgerSequential(t *testing.T) {
	var start, end uint32 = 1, 100
	f := NewLedgerMetaArchive("", start, end+100)

	// Add ledgers sequentially
	for i := start; i <= end; i++ {
		require.NoError(t, f.AddLedger(createLedgerCloseMeta(i)))
	}

	// Test out of sequence
	testCases := []struct {
		ledgerSeq      uint32
		expectedErrMsg string
	}{
		{
			end + 2,
			fmt.Sprintf("ledgers must be added sequentially: expected sequence %d, got %d", end+1, end+2),
		},
		{
			end,
			fmt.Sprintf("ledgers must be added sequentially: expected sequence %d, got %d", end+1, end),
		},
		{
			end - 1,
			fmt.Sprintf("ledgers must be added sequentially: expected sequence %d, got %d", end+1, end-1),
		},
	}

	for _, tc := range testCases {
		err := f.AddLedger(createLedgerCloseMeta(tc.ledgerSeq))
		require.EqualError(t, err, tc.expectedErrMsg)
	}
}

func TestGetLedger(t *testing.T) {
	var start, end uint32 = 121, 1300
	f := NewLedgerMetaArchive("TestGetLedger.xdr", start, end)

	for i := start; i <= end-10; i++ {
		require.NoError(t, f.AddLedger(createLedgerCloseMeta(i)))
	}

	testCases := []struct {
		name           string
		ledgerSeq      uint32
		expectedErrMsg string
	}{
		{
			name:           "LedgerSequenceInRange",
			ledgerSeq:      start,
			expectedErrMsg: "",
		},
		{
			name:           "LedgerSequenceInRange",
			ledgerSeq:      start + 10,
			expectedErrMsg: "",
		},
		{
			name:      "LedgerSequenceAboveRange",
			ledgerSeq: end + 1,
			expectedErrMsg: fmt.Sprintf("ledger sequence %d is outside the valid range "+
				"of ledger sequences [%d, %d] this meta archive holds", end+1, start, end),
		},
		{
			name:      "LedgerSequenceBelowRange",
			ledgerSeq: start - 1,
			expectedErrMsg: fmt.Sprintf("ledger sequence %d is outside the valid range "+
				"of ledger sequences [%d, %d] this meta archive holds", start-1, start, end),
		},
		{
			name:      "LedgerCloseMetaNotFound",
			ledgerSeq: end - 5,
			expectedErrMsg: fmt.Sprintf("LedgerCloseMeta for sequence %d is not found in %s "+
				"meta archive", end-5, "TestGetLedger.xdr"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			archive, err := f.GetLedger(tc.ledgerSeq)
			if tc.expectedErrMsg != "" {
				require.EqualError(t, err, tc.expectedErrMsg)
				require.Equal(t, archive, xdr.LedgerCloseMeta{})
			} else {
				require.NoError(t, err)
				require.Equal(t, archive.LedgerSequence(), tc.ledgerSeq)
			}
		})
	}
}
