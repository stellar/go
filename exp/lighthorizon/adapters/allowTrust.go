package adapters

import (
	"github.com/stellar/go/exp/lighthorizon/common"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

func populateAllowTrustOperation(op *common.Operation, baseOp operations.Base) (operations.AllowTrust, error) {
	allowTrust := op.Get().Body.MustAllowTrustOp()
	baseOp.Type = "allow_trust"

	var (
		assetType string
		code      string
		issuer    string
	)

	err := allowTrust.Asset.ToAsset(op.SourceAccount()).Extract(&assetType, &code, &issuer)
	if err != nil {
		return operations.AllowTrust{}, errors.Wrap(err, "xdr.Asset.Extract error")
	}

	flags := xdr.TrustLineFlags(allowTrust.Authorize)

	return operations.AllowTrust{
		Base: baseOp,
		Asset: base.Asset{
			Type:   assetType,
			Code:   code,
			Issuer: issuer,
		},

		Trustee:                        op.SourceAccount().Address(),
		Trustor:                        allowTrust.Trustor.Address(),
		Authorize:                      flags.IsAuthorized(),
		AuthorizeToMaintainLiabilities: flags.IsAuthorizedToMaintainLiabilitiesFlag(),
		// TODO:
		TrusteeMuxed:   "",
		TrusteeMuxedID: 0,
	}, nil
}
