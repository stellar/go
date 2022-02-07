package adapters

import (
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/xdr"
)

func populateSetOptionsOperation(op *common.Operation, baseOp operations.Base) (operations.SetOptions, error) {
	setOptions := op.Get().Body.MustSetOptionsOp()
	baseOp.Type = "set_options"

	homeDomain := ""
	if setOptions.HomeDomain != nil {
		homeDomain = string(*setOptions.HomeDomain)
	}

	inflationDest := ""
	if setOptions.InflationDest != nil {
		inflationDest = setOptions.InflationDest.Address()
	}

	var signerKey string
	var signerWeight *int
	if setOptions.Signer != nil {
		signerKey = setOptions.Signer.Key.Address()
		signerWeightInt := int(setOptions.Signer.Weight)
		signerWeight = &signerWeightInt
	}

	var masterKeyWeight, lowThreshold, medThreshold, highThreshold *int
	if setOptions.MasterWeight != nil {
		masterKeyWeightInt := int(*setOptions.MasterWeight)
		masterKeyWeight = &masterKeyWeightInt
	}
	if setOptions.LowThreshold != nil {
		lowThresholdInt := int(*setOptions.LowThreshold)
		lowThreshold = &lowThresholdInt
	}
	if setOptions.MedThreshold != nil {
		medThresholdInt := int(*setOptions.MedThreshold)
		medThreshold = &medThresholdInt
	}
	if setOptions.HighThreshold != nil {
		highThresholdInt := int(*setOptions.HighThreshold)
		highThreshold = &highThresholdInt
	}

	var (
		setFlags  []int
		setFlagsS []string

		clearFlags  []int
		clearFlagsS []string
	)

	if setOptions.SetFlags != nil && *setOptions.SetFlags > 0 {
		f := xdr.AccountFlags(*setOptions.SetFlags)

		if f.IsAuthRequired() {
			setFlags = append(setFlags, int(xdr.AccountFlagsAuthRequiredFlag))
			setFlagsS = append(setFlagsS, "auth_required")
		}

		if f.IsAuthRevocable() {
			setFlags = append(setFlags, int(xdr.AccountFlagsAuthRevocableFlag))
			setFlagsS = append(setFlagsS, "auth_revocable")
		}

		if f.IsAuthImmutable() {
			setFlags = append(setFlags, int(xdr.AccountFlagsAuthImmutableFlag))
			setFlagsS = append(setFlagsS, "auth_immutable")
		}

		if f.IsAuthClawbackEnabled() {
			setFlags = append(setFlags, int(xdr.AccountFlagsAuthClawbackEnabledFlag))
			setFlagsS = append(setFlagsS, "auth_clawback_enabled")
		}
	}

	if setOptions.ClearFlags != nil && *setOptions.ClearFlags > 0 {
		f := xdr.AccountFlags(*setOptions.ClearFlags)

		if f.IsAuthRequired() {
			clearFlags = append(clearFlags, int(xdr.AccountFlagsAuthRequiredFlag))
			clearFlagsS = append(clearFlagsS, "auth_required")
		}

		if f.IsAuthRevocable() {
			clearFlags = append(clearFlags, int(xdr.AccountFlagsAuthRevocableFlag))
			clearFlagsS = append(clearFlagsS, "auth_revocable")
		}

		if f.IsAuthImmutable() {
			clearFlags = append(clearFlags, int(xdr.AccountFlagsAuthImmutableFlag))
			clearFlagsS = append(clearFlagsS, "auth_immutable")
		}

		if f.IsAuthClawbackEnabled() {
			clearFlags = append(clearFlags, int(xdr.AccountFlagsAuthClawbackEnabledFlag))
			clearFlagsS = append(clearFlagsS, "auth_clawback_enabled")
		}
	}

	return operations.SetOptions{
		Base:          baseOp,
		HomeDomain:    homeDomain,
		InflationDest: inflationDest,

		MasterKeyWeight: masterKeyWeight,
		SignerKey:       signerKey,
		SignerWeight:    signerWeight,

		SetFlags:    setFlags,
		SetFlagsS:   setFlagsS,
		ClearFlags:  clearFlags,
		ClearFlagsS: clearFlagsS,

		LowThreshold:  lowThreshold,
		MedThreshold:  medThreshold,
		HighThreshold: highThreshold,
	}, nil
}
