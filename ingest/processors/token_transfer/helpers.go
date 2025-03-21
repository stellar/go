package token_transfer

import (
	"crypto/sha256"
	"fmt"
	"github.com/stellar/go/ingest"
	"github.com/stellar/go/xdr"
)

/*
Helper function to convert LiquidityPoolId to generate ClaimableBalanceIds in a deterministic way as per
https://github.com/stellar/stellar-protocol/blob/master/core/cap-0038.md#settrustlineflagsop-and-allowtrustop
Whenever a trustline is revoked for an asset for an account, via setTrustlineFalgs or allowTrust operations,
if that account has deposited into a Liquidity Pool, then, claimable balances are created for those assets from the LP

The generated ClaimableBalanceId is derived from:
- Liquidity Pool Id from which the pool shares are withdrawn
- The asset for which CB is created
- the accountSequence number of the transaction account
- The transaction accountId
- The operationIndex of the setTrustlineFlags or allowTrust operation within the transaction
*/
func ClaimableBalanceIdFromRevocation(liquidityPoolId xdr.PoolId, asset xdr.Asset, accountSequence xdr.SequenceNumber, accountId xdr.AccountId, opIndex uint32) (xdr.ClaimableBalanceId, error) {
	preImageId := xdr.HashIdPreimage{
		Type: xdr.EnvelopeTypeEnvelopeTypePoolRevokeOpId,
		RevokeId: &xdr.HashIdPreimageRevokeId{
			SourceAccount:   accountId,
			SeqNum:          accountSequence,
			OpNum:           xdr.Uint32(opIndex),
			LiquidityPoolId: liquidityPoolId,
			Asset:           asset,
		},
	}
	binaryDump, e := preImageId.MarshalBinary()
	if e != nil {
		return xdr.ClaimableBalanceId{}, fmt.Errorf("failed to convert HashIdPreimage to claimable balanceId:%w", e)
	}
	sha256hash := xdr.Hash(sha256.Sum256(binaryDump))
	cbId := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &sha256hash,
	}
	return cbId, nil
}

// Helper functions
func operationSourceAccount(tx ingest.LedgerTransaction, op xdr.Operation) xdr.MuxedAccount {
	acc := op.SourceAccount
	if acc != nil {
		return *acc
	}
	res := tx.Envelope.SourceAccount()
	return res
}

// Even though these functions simply call the corresponding proto, these are helpful to reduce clutter when being used in the unit test
// otherwise the entire imported path alias needs to be added and it is distracting
func protoAddressFromAccount(account xdr.MuxedAccount) string {
	return account.ToAccountId().Address()
}

// TODO convert to strkey for LpId
func lpIdToStrkey(lpId xdr.PoolId) string {
	return xdr.Hash(lpId).HexString()
}

// TODO convert to strkey for CbId
func cbIdToStrkey(cbId xdr.ClaimableBalanceId) string {
	return cbId.MustV0().HexString()
}

// This operation is used to only find CB entries that are either created or deleted, not updated
func getClaimableBalanceEntriesFromOperationChanges(changeType xdr.LedgerEntryChangeType, tx ingest.LedgerTransaction, opIndex uint32) ([]xdr.ClaimableBalanceEntry, error) {
	if changeType == xdr.LedgerEntryChangeTypeLedgerEntryUpdated {
		return nil, fmt.Errorf("LEDGER_ENTRY_UPDATED is not a valid filter")
	}

	changes, err := tx.GetOperationChanges(opIndex)
	if err != nil {
		return nil, err
	}

	var entries []xdr.ClaimableBalanceEntry
	/*
		This function is expected to be called only to get details of newly created claimable balance
		(for e.g as a result of allowTrust or setTrustlineFlags  operations)
		or claimable balances that are be deleted
		(for e.g due to clawback claimable balance operation)
	*/
	var cb xdr.ClaimableBalanceEntry
	for _, change := range changes {
		if change.Type != xdr.LedgerEntryTypeClaimableBalance || change.LedgerEntryChangeType() != changeType {
			continue
		}
		if change.Pre != nil {
			cb = change.Pre.Data.MustClaimableBalance()
			entries = append(entries, cb)
		} else if change.Post != nil {
			cb = change.Post.Data.MustClaimableBalance()
			entries = append(entries, cb)
		}
	}

	return entries, nil
}
