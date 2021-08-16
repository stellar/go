package xdr

import "fmt"

func (a ClaimAtom) OfferId() Int64 {
	switch a.Type {
	case ClaimAtomTypeClaimAtomTypeV0:
		return a.V0.OfferId
	case ClaimAtomTypeClaimAtomTypeOrderBook:
		return a.OrderBook.OfferId
	default:
		panic(fmt.Errorf("Unknown ClaimAtom type: %v", a.Type))
	}
}

func (a ClaimAtom) SellerId() AccountId {
	switch a.Type {
	case ClaimAtomTypeClaimAtomTypeV0:
		return AccountId{
			Type:    PublicKeyTypePublicKeyTypeEd25519,
			Ed25519: &a.V0.SellerEd25519,
		}
	case ClaimAtomTypeClaimAtomTypeOrderBook:
		return a.OrderBook.SellerId
	default:
		panic(fmt.Errorf("Unknown ClaimAtom type: %v", a.Type))
	}
}

func (a ClaimAtom) AssetBought() Asset {
	switch a.Type {
	case ClaimAtomTypeClaimAtomTypeV0:
		return a.V0.AssetBought
	case ClaimAtomTypeClaimAtomTypeOrderBook:
		return a.OrderBook.AssetBought
	default:
		panic(fmt.Errorf("Unknown ClaimAtom type: %v", a.Type))
	}
}

func (a ClaimAtom) AmountBought() Int64 {
	switch a.Type {
	case ClaimAtomTypeClaimAtomTypeV0:
		return a.V0.AmountBought
	case ClaimAtomTypeClaimAtomTypeOrderBook:
		return a.OrderBook.AmountBought
	default:
		panic(fmt.Errorf("Unknown ClaimAtom type: %v", a.Type))
	}
}

func (a ClaimAtom) AssetSold() Asset {
	switch a.Type {
	case ClaimAtomTypeClaimAtomTypeV0:
		return a.V0.AssetSold
	case ClaimAtomTypeClaimAtomTypeOrderBook:
		return a.OrderBook.AssetSold
	default:
		panic(fmt.Errorf("Unknown ClaimAtom type: %v", a.Type))
	}
}

func (a ClaimAtom) AmountSold() Int64 {
	switch a.Type {
	case ClaimAtomTypeClaimAtomTypeV0:
		return a.V0.AmountSold
	case ClaimAtomTypeClaimAtomTypeOrderBook:
		return a.OrderBook.AmountSold
	default:
		panic(fmt.Errorf("Unknown ClaimAtom type: %v", a.Type))
	}
}
