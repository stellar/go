package operation

import (
	"fmt"

	"github.com/stellar/go/xdr"
)

type SetOptionsDetails struct {
	InflationDestination string   `json:"inflation_destination"`
	MasterKeyWeight      uint32   `json:"master_key_weight"`
	LowThreshold         uint32   `json:"low_threshold"`
	MediumThreshold      uint32   `json:"medium_threshold"`
	HighThreshold        uint32   `json:"high_threshold"`
	HomeDomain           string   `json:"home_domain"`
	SignerKey            string   `json:"signer_key"`
	SignerWeight         uint32   `json:"signer_weight"`
	SetFlags             []int32  `json:"set_flags"`
	SetFlagsString       []string `json:"set_flags_string"`
	ClearFlags           []int32  `json:"clear_flags"`
	ClearFlagsString     []string `json:"clear_flags_string"`
}

func (o *LedgerOperation) SetOptionsDetails() (SetOptionsDetails, error) {
	op, ok := o.Operation.Body.GetSetOptionsOp()
	if !ok {
		return SetOptionsDetails{}, fmt.Errorf("could not access GetSetOptions info for this operation (index %d)", o.OperationIndex)
	}

	var setOptionsDetail SetOptionsDetails

	if op.InflationDest != nil {
		setOptionsDetail.InflationDestination = op.InflationDest.Address()
	}

	if op.SetFlags != nil && *op.SetFlags > 0 {
		setOptionsDetail.SetFlags, setOptionsDetail.SetFlagsString = addOperationFlagToOperation(uint32(*op.SetFlags))
	}

	if op.ClearFlags != nil && *op.ClearFlags > 0 {
		setOptionsDetail.ClearFlags, setOptionsDetail.ClearFlagsString = addOperationFlagToOperation(uint32(*op.ClearFlags))
	}

	if op.MasterWeight != nil {
		setOptionsDetail.MasterKeyWeight = uint32(*op.MasterWeight)
	}

	if op.LowThreshold != nil {
		setOptionsDetail.LowThreshold = uint32(*op.LowThreshold)
	}

	if op.MedThreshold != nil {
		setOptionsDetail.MediumThreshold = uint32(*op.MedThreshold)
	}

	if op.HighThreshold != nil {
		setOptionsDetail.HighThreshold = uint32(*op.HighThreshold)
	}

	if op.HomeDomain != nil {
		setOptionsDetail.HomeDomain = string(*op.HomeDomain)
	}

	if op.Signer != nil {
		setOptionsDetail.SignerKey = op.Signer.Key.Address()
		setOptionsDetail.SignerWeight = uint32(op.Signer.Weight)
	}

	return setOptionsDetail, nil
}

func addOperationFlagToOperation(flag uint32) ([]int32, []string) {
	intFlags := make([]int32, 0)
	stringFlags := make([]string, 0)

	if (int64(flag) & int64(xdr.AccountFlagsAuthRequiredFlag)) > 0 {
		intFlags = append(intFlags, int32(xdr.AccountFlagsAuthRequiredFlag))
		stringFlags = append(stringFlags, "auth_required")
	}

	if (int64(flag) & int64(xdr.AccountFlagsAuthRevocableFlag)) > 0 {
		intFlags = append(intFlags, int32(xdr.AccountFlagsAuthRevocableFlag))
		stringFlags = append(stringFlags, "auth_revocable")
	}

	if (int64(flag) & int64(xdr.AccountFlagsAuthImmutableFlag)) > 0 {
		intFlags = append(intFlags, int32(xdr.AccountFlagsAuthImmutableFlag))
		stringFlags = append(stringFlags, "auth_immutable")
	}

	if (int64(flag) & int64(xdr.AccountFlagsAuthClawbackEnabledFlag)) > 0 {
		intFlags = append(intFlags, int32(xdr.AccountFlagsAuthClawbackEnabledFlag))
		stringFlags = append(stringFlags, "auth_clawback_enabled")
	}

	return intFlags, stringFlags
}
