package operation

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type SetTrustlineFlagsDetail struct {
	AssetCode        string   `json:"asset_code"`
	AssetIssuer      string   `json:"asset_issuer"`
	AssetType        string   `json:"asset_type"`
	Trustor          string   `json:"trustor"`
	SetFlags         []int32  `json:"set_flags"`
	SetFlagsString   []string `json:"set_flags_string"`
	ClearFlags       []int32  `json:"clear_flags"`
	ClearFlagsString []string `json:"clear_flags_string"`
}

func (o *LedgerOperation) SetTrustlineFlagsDetails() (SetTrustlineFlagsDetail, error) {
	op, ok := o.Operation.Body.GetSetTrustLineFlagsOp()
	if !ok {
		return SetTrustlineFlagsDetail{}, fmt.Errorf("could not access SetTrustLineFlags info for this operation (index %d)", o.OperationIndex)
	}

	setTrustLineFlagsDetail := SetTrustlineFlagsDetail{
		Trustor: op.Trustor.Address(),
	}

	var err error
	var assetCode, assetIssuer, assetType string
	err = op.Asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return SetTrustlineFlagsDetail{}, err
	}

	setTrustLineFlagsDetail.AssetCode = assetCode
	setTrustLineFlagsDetail.AssetIssuer = assetIssuer
	setTrustLineFlagsDetail.AssetType = assetType

	if op.SetFlags > 0 {
		setTrustLineFlagsDetail.SetFlags, setTrustLineFlagsDetail.SetFlagsString = getTrustLineFlagToDetails(xdr.TrustLineFlags(op.SetFlags))
	}
	if op.ClearFlags > 0 {
		setTrustLineFlagsDetail.ClearFlags, setTrustLineFlagsDetail.ClearFlagsString = getTrustLineFlagToDetails(xdr.TrustLineFlags(op.ClearFlags))
	}

	return setTrustLineFlagsDetail, nil
}

func getTrustLineFlagToDetails(f xdr.TrustLineFlags) ([]int32, []string) {
	var (
		n []int32
		s []string
	)

	if f.IsAuthorized() {
		n = append(n, int32(xdr.TrustLineFlagsAuthorizedFlag))
		s = append(s, "authorized")
	}

	if f.IsAuthorizedToMaintainLiabilitiesFlag() {
		n = append(n, int32(xdr.TrustLineFlagsAuthorizedToMaintainLiabilitiesFlag))
		s = append(s, "authorized_to_maintain_liabilities")
	}

	if f.IsClawbackEnabledFlag() {
		n = append(n, int32(xdr.TrustLineFlagsTrustlineClawbackEnabledFlag))
		s = append(s, "clawback_enabled")
	}

	return n, s
}
