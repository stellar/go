package operation

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type AllowTrustDetail struct {
	AssetCode                      string `json:"asset_code"`
	AssetIssuer                    string `json:"asset_issuer"`
	AssetType                      string `json:"asset_type"`
	Trustor                        string `json:"trustor"`
	Trustee                        string `json:"trustee"`
	TrusteeMuxed                   string `json:"trustee_muxed"`
	TrusteeMuxedID                 uint64 `json:"trustee_muxed_id,string"`
	Authorize                      bool   `json:"authorize"`
	AuthorizeToMaintainLiabilities bool   `json:"authorize_to_maintain_liabilities"`
	ClawbackEnabled                bool   `json:"clawback_enabled"`
}

func (o *LedgerOperation) AllowTrustDetails() (AllowTrustDetail, error) {
	op, ok := o.Operation.Body.GetAllowTrustOp()
	if !ok {
		return AllowTrustDetail{}, fmt.Errorf("could not access AllowTrust info for this operation (index %d)", o.OperationIndex)
	}

	allowTrustDetail := AllowTrustDetail{
		Trustor:                        op.Trustor.Address(),
		Trustee:                        o.SourceAccount(),
		Authorize:                      xdr.TrustLineFlags(op.Authorize).IsAuthorized(),
		AuthorizeToMaintainLiabilities: xdr.TrustLineFlags(op.Authorize).IsAuthorizedToMaintainLiabilitiesFlag(),
		ClawbackEnabled:                xdr.TrustLineFlags(op.Authorize).IsClawbackEnabledFlag(),
	}

	var err error
	var assetCode, assetIssuer, assetType string
	err = op.Asset.ToAsset(o.sourceAccountXDR().ToAccountId()).Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return AllowTrustDetail{}, err
	}

	allowTrustDetail.AssetCode = assetCode
	allowTrustDetail.AssetIssuer = assetIssuer
	allowTrustDetail.AssetType = assetType

	var trusteeMuxed string
	var trusteeMuxedID uint64
	trusteeMuxed, trusteeMuxedID, err = getMuxedAccountDetails(o.sourceAccountXDR())
	if err != nil {
		return AllowTrustDetail{}, err
	}

	allowTrustDetail.TrusteeMuxed = trusteeMuxed
	allowTrustDetail.TrusteeMuxedID = trusteeMuxedID

	return allowTrustDetail, nil
}
