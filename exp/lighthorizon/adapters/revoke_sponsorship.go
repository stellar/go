package adapters

import (
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func populateRevokeSponsorshipOperation(op *common.Operation, baseOp operations.Base) (operations.RevokeSponsorship, error) {
	revokeSponsorship := op.Get().Body.MustRevokeSponsorshipOp()

	switch revokeSponsorship.Type {
	case xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry:
		ret := operations.RevokeSponsorship{
			Base: baseOp,
		}

		ledgerKey := revokeSponsorship.LedgerKey

		switch ledgerKey.Type {
		case xdr.LedgerEntryTypeAccount:
			accountID := ledgerKey.Account.AccountId.Address()
			ret.AccountID = &accountID
		case xdr.LedgerEntryTypeClaimableBalance:
			marshalHex, err := xdr.MarshalHex(ledgerKey.ClaimableBalance.BalanceId)
			if err != nil {
				return operations.RevokeSponsorship{}, err
			}
			ret.ClaimableBalanceID = &marshalHex
		case xdr.LedgerEntryTypeData:
			accountID := ledgerKey.Data.AccountId.Address()
			dataName := string(ledgerKey.Data.DataName)
			ret.DataAccountID = &accountID
			ret.DataName = &dataName
		case xdr.LedgerEntryTypeOffer:
			offerID := int64(ledgerKey.Offer.OfferId)
			ret.OfferID = &offerID
		case xdr.LedgerEntryTypeTrustline:
			trustlineAccountID := ledgerKey.TrustLine.AccountId.Address()
			ret.TrustlineAccountID = &trustlineAccountID
			if ledgerKey.TrustLine.Asset.Type == xdr.AssetTypeAssetTypePoolShare {
				trustlineLiquidityPoolID := xdr.Hash(*ledgerKey.TrustLine.Asset.LiquidityPoolId).HexString()
				ret.TrustlineLiquidityPoolID = &trustlineLiquidityPoolID
			} else {
				trustlineAsset := ledgerKey.TrustLine.Asset.ToAsset().StringCanonical()
				ret.TrustlineAsset = &trustlineAsset
			}
		default:
			return operations.RevokeSponsorship{}, errors.Errorf("invalid ledger key type: %d", ledgerKey.Type)
		}

		return ret, nil
	case xdr.RevokeSponsorshipTypeRevokeSponsorshipSigner:
		signerAccountID := revokeSponsorship.Signer.AccountId.Address()
		signerKey := revokeSponsorship.Signer.SignerKey.Address()

		return operations.RevokeSponsorship{
			Base:            baseOp,
			SignerAccountID: &signerAccountID,
			SignerKey:       &signerKey,
		}, nil
	}

	return operations.RevokeSponsorship{}, errors.Errorf("invalid revoke type: %d", revokeSponsorship.Type)
}
