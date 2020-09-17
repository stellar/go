package txnbuild

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/stellar/go/xdr"
)

type RevokeSponsorshipType int

const (
	RevokeSponsorshipTypeAccount RevokeSponsorshipType = iota + 1
	RevokeSponsorshipTypeTrustLine
	RevokeSponsorshipTypeOffer
	RevokeSponsorshipTypeData
	RevokeSponsorshipTypeClaimableBalance
	RevokeSponsorshipTypeSigner
)

// RevokeSponsorship is a union type representing a RevokeSponsorship Operation.
// SponsorshipType stablishes which sponsorship is being revoked.
// The other fields should be ignored.
type RevokeSponsorship struct {
	SourceAccount    Account
	SponsorshipType  RevokeSponsorshipType
	Account          *string
	TrustLine        *TrustLineId
	Offer            *OfferId
	Data             *DataId
	ClaimableBalance *ClaimableBalanceHash
	Signer           *SignerId
}

type TrustLineId struct {
	AccountAddress string
	Asset          Asset
}

type OfferId struct {
	SellerAccountAddress string
	OfferId              int64
}

type DataId struct {
	AccountAddress string
	DataName       string
}

type ClaimableBalanceHash [32]byte

type SignerId struct {
	AccountAddress string
	SignerAddress  string
}

func (r *RevokeSponsorship) BuildXDR() (xdr.Operation, error) {
	xdrOp := xdr.RevokeSponsorshipOp{}
	switch r.SponsorshipType {
	case RevokeSponsorshipTypeAccount:
		var key xdr.LedgerKeyAccount
		if r.Account == nil {
			return xdr.Operation{}, errors.New("Account can't be nil")
		}
		if err := key.AccountId.SetAddress(*r.Account); err != nil {
			return xdr.Operation{}, errors.Wrap(err, "incorrect Account address")
		}
		xdrOp.Type = xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry
		xdrOp.LedgerKey = &xdr.LedgerKey{
			Type:    xdr.LedgerEntryTypeAccount,
			Account: &key,
		}
	case RevokeSponsorshipTypeTrustLine:
		var key xdr.LedgerKeyTrustLine
		if r.TrustLine == nil {
			return xdr.Operation{}, errors.New("TrustLine can't be nil")
		}
		if err := key.AccountId.SetAddress(r.TrustLine.AccountAddress); err != nil {
			return xdr.Operation{}, errors.Wrap(err, "incorrect Account address")
		}
		asset, err := r.TrustLine.Asset.ToXDR()
		if err != nil {
			return xdr.Operation{}, errors.Wrap(err, "incorrect TrustLine asset")
		}
		key.Asset = asset
		xdrOp.Type = xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry
		xdrOp.LedgerKey = &xdr.LedgerKey{
			Type:      xdr.LedgerEntryTypeTrustline,
			TrustLine: &key,
		}
	case RevokeSponsorshipTypeOffer:
		var key xdr.LedgerKeyOffer
		if r.Offer == nil {
			return xdr.Operation{}, errors.New("Offer can't be nil")
		}
		if err := key.SellerId.SetAddress(r.Offer.SellerAccountAddress); err != nil {
			return xdr.Operation{}, errors.Wrap(err, "incorrect Seller account address")
		}
		key.OfferId = xdr.Int64(r.Offer.OfferId)
		xdrOp.Type = xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry
		xdrOp.LedgerKey = &xdr.LedgerKey{
			Type:  xdr.LedgerEntryTypeOffer,
			Offer: &key,
		}
	case RevokeSponsorshipTypeData:
		var key xdr.LedgerKeyData
		if r.Data == nil {
			return xdr.Operation{}, errors.New("Data can't be nil")
		}
		if err := key.AccountId.SetAddress(r.Data.AccountAddress); err != nil {
			return xdr.Operation{}, errors.Wrap(err, "incorrect Account address")
		}
		// TODO: should we check the size?
		key.DataName = xdr.String64(r.Data.DataName)
		xdrOp.Type = xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry
		xdrOp.LedgerKey = &xdr.LedgerKey{
			Type: xdr.LedgerEntryTypeData,
			Data: &key,
		}
	case RevokeSponsorshipTypeClaimableBalance:
		key := xdr.LedgerKeyClaimableBalance{
			BalanceId: xdr.ClaimableBalanceId{
				Type: 0,
				V0:   &xdr.Hash{},
			},
		}
		if r.ClaimableBalance == nil {
			return xdr.Operation{}, errors.New("ClaimableBalance can't be nil")
		}
		copy(key.BalanceId.V0[:], r.ClaimableBalance[:])
		xdrOp.Type = xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry
		xdrOp.LedgerKey = &xdr.LedgerKey{
			Type:             xdr.LedgerEntryTypeClaimableBalance,
			ClaimableBalance: &key,
		}
	case RevokeSponsorshipTypeSigner:
		var signer xdr.RevokeSponsorshipOpSigner
		if r.Signer == nil {
			return xdr.Operation{}, errors.New("Signer can't be nil")
		}
		if err := signer.AccountId.SetAddress(r.Signer.AccountAddress); err != nil {
			return xdr.Operation{}, errors.New("incorrect Account address")
		}
		if err := signer.SignerKey.SetAddress(r.Signer.SignerAddress); err != nil {
			return xdr.Operation{}, errors.New("incorrect Signer account address")
		}
		xdrOp.Type = xdr.RevokeSponsorshipTypeRevokeSponsorshipSigner
		xdrOp.Signer = &signer
	default:
		return xdr.Operation{}, fmt.Errorf("unknown SponsorshipType: %d", r.SponsorshipType)
	}
	opType := xdr.OperationTypeRevokeSponsorship
	body, err := xdr.NewOperationBody(opType, xdrOp)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "failed to build XDR OperationBody")
	}
	op := xdr.Operation{Body: body}
	SetOpSourceAccount(&op, r.SourceAccount)
	return op, nil
}

func (r *RevokeSponsorship) FromXDR(xdrOp xdr.Operation) error {
	r.SourceAccount = accountFromXDR(xdrOp.SourceAccount)
	op, ok := xdrOp.Body.GetRevokeSponsorshipOp()
	if !ok {
		return errors.New("error parsing revoke_sponsorhip operation from xdr")
	}
	switch op.Type {
	case xdr.RevokeSponsorshipTypeRevokeSponsorshipLedgerEntry:
		lkey := op.LedgerKey
		switch lkey.Type {
		case xdr.LedgerEntryTypeAccount:
			var sponsorshipId string
			sponsorshipId = lkey.Account.AccountId.Address()
			r.SponsorshipType = RevokeSponsorshipTypeAccount
			r.Account = &sponsorshipId
		case xdr.LedgerEntryTypeTrustline:
			var sponsorshipId TrustLineId
			sponsorshipId.AccountAddress = lkey.TrustLine.AccountId.Address()
			asset, err := assetFromXDR(lkey.TrustLine.Asset)
			if err != nil {
				return errors.Wrap(err, "error parsing Trustline Asset")
			}
			sponsorshipId.Asset = asset
			r.SponsorshipType = RevokeSponsorshipTypeTrustLine
			r.TrustLine = &sponsorshipId
		case xdr.LedgerEntryTypeOffer:
			var sponsorshipId OfferId
			sponsorshipId.SellerAccountAddress = lkey.Offer.SellerId.Address()
			sponsorshipId.OfferId = int64(lkey.Offer.OfferId)
			r.SponsorshipType = RevokeSponsorshipTypeOffer
			r.Offer = &sponsorshipId
		case xdr.LedgerEntryTypeData:
			var sponsorshipId DataId
			sponsorshipId.AccountAddress = lkey.Data.AccountId.Address()
			sponsorshipId.DataName = string(lkey.Data.DataName)
			r.SponsorshipType = RevokeSponsorshipTypeData
			r.Data = &sponsorshipId
		case xdr.LedgerEntryTypeClaimableBalance:
			var sponsorshipId ClaimableBalanceHash
			if lkey.ClaimableBalance.BalanceId.Type != 0 {
				return fmt.Errorf(
					"unexpected ClaimableBalance Id Type: %d",
					lkey.ClaimableBalance.BalanceId.Type,
				)
			}
			copy(sponsorshipId[:], lkey.ClaimableBalance.BalanceId.V0[:])
			r.SponsorshipType = RevokeSponsorshipTypeClaimableBalance
			r.ClaimableBalance = &sponsorshipId
		default:
			return fmt.Errorf("unexpected LedgerEntryType: %d", lkey.Type)
		}
	case xdr.RevokeSponsorshipTypeRevokeSponsorshipSigner:
		var sponsorshipId SignerId
		sponsorshipId.AccountAddress = op.Signer.AccountId.Address()
		sponsorshipId.SignerAddress = op.Signer.SignerKey.Address()
		r.SponsorshipType = RevokeSponsorshipTypeSigner
		r.Signer = &sponsorshipId
	default:
		return fmt.Errorf("unexpected RevokeSponsorshipType: %d", op.Type)
	}
	return nil
}

func (r *RevokeSponsorship) Validate() error {
	switch r.SponsorshipType {
	case RevokeSponsorshipTypeAccount:
		if r.Account == nil {
			return errors.New("Account can't be nil")
		}
		return validateStellarPublicKey(*r.Account)
	case RevokeSponsorshipTypeTrustLine:
		if r.TrustLine == nil {
			return errors.New("Trustline can't be nil")
		}
		if err := validateStellarPublicKey(r.TrustLine.AccountAddress); err != nil {
			return errors.Wrap(err, "invalid Account address")
		}
		if err := validateStellarAsset(r.TrustLine.Asset); err != nil {
			return errors.Wrap(err, "invalid TrustLine asset")
		}
	case RevokeSponsorshipTypeOffer:
		if r.Offer == nil {
			return errors.New("Offer can't be nil")
		}
		if err := validateStellarPublicKey(r.Offer.SellerAccountAddress); err != nil {
			return errors.Wrap(err, "invalid Seller account address")
		}
		return validateStellarPublicKey(r.Offer.SellerAccountAddress)
	case RevokeSponsorshipTypeData:
		if r.Data == nil {
			return errors.New("Data can't be nil")
		}
		if err := validateStellarPublicKey(r.Data.AccountAddress); err != nil {
			return errors.Wrap(err, "invalid Account address")
		}
		// TODO: should we check the DataName size?
	case RevokeSponsorshipTypeClaimableBalance:
		if r.ClaimableBalance == nil {
			return errors.New("ClaimableBalance can't be nil")
		}
	case RevokeSponsorshipTypeSigner:
		if r.Signer == nil {
			return errors.New("Signer can't be nil")
		}
		if err := validateStellarPublicKey(r.Signer.AccountAddress); err != nil {
			return errors.New("invalid Account address")
		}
		if err := validateStellarSignerKey(r.Signer.SignerAddress); err != nil {
			return errors.New("invalid Signer account address")
		}
	default:
		return fmt.Errorf("unknown SponsorshipType: %d", r.SponsorshipType)
	}
	return nil
}

func (r *RevokeSponsorship) GetSourceAccount() Account {
	return r.SourceAccount
}
