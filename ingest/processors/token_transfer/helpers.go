package token_transfer

import (
	"crypto/sha256"
	"github.com/stellar/go/support/errors"
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
		return xdr.ClaimableBalanceId{}, errors.Wrapf(e, "Failed to convert HashIdPreimage to claimable balanceId")
	}
	sha256hash := xdr.Hash(sha256.Sum256(binaryDump))
	cbId := xdr.ClaimableBalanceId{
		Type: xdr.ClaimableBalanceIdTypeClaimableBalanceIdTypeV0,
		V0:   &sha256hash,
	}
	return cbId, nil
}
