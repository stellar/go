package ingest

import (
	"strings"

	"github.com/stellar/go/services/horizon/internal/db2/core"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/xdr"
)

func (assetsModified AssetsModified) handlePaymentOp(paymentOp *xdr.PaymentOp, sourceAccount *xdr.AccountId) error {
	err := assetsModified.updateIfAssetIssuerInvolved(paymentOp.Asset, *sourceAccount)
	if err != nil {
		return err
	}
	err = assetsModified.updateIfAssetIssuerInvolved(paymentOp.Asset, paymentOp.Destination)
	return err
}

func (assetsModified AssetsModified) handleManageOfferOp(manageOfferOp *xdr.ManageOfferOp, sourceAccount *xdr.AccountId) error {
	err := assetsModified.updateIfAssetIssuerInvolved(manageOfferOp.Buying, *sourceAccount)
	if err != nil {
		return err
	}
	err = assetsModified.updateIfAssetIssuerInvolved(manageOfferOp.Selling, *sourceAccount)
	return err
}

func (assetsModified AssetsModified) handlePassiveOfferOp(passiveOfferOp *xdr.CreatePassiveOfferOp, sourceAccount *xdr.AccountId) error {
	err := assetsModified.updateIfAssetIssuerInvolved(passiveOfferOp.Buying, *sourceAccount)
	if err != nil {
		return err
	}
	err = assetsModified.updateIfAssetIssuerInvolved(passiveOfferOp.Selling, *sourceAccount)
	return err
}

func defaultSourceAccount(sourceAccount *xdr.AccountId, defaultAddress string) *xdr.AccountId {
	if sourceAccount != nil {
		return sourceAccount
	}

	var accountID xdr.AccountId
	accountID.SetAddress(defaultAddress)
	return &accountID
}

// IngestOperation updates the assetsModified using the passed in operation
func (assetsModified AssetsModified) IngestOperation(err error, op *xdr.Operation, sourceAddress string, coreQ *core.Q) error {
	if err != nil {
		return err
	}

	body := op.Body
	sourceAccount := defaultSourceAccount(op.SourceAccount, sourceAddress)
	switch body.Type {
	// TODO NNS 2 need to fix GetCreateAssetID call when adding assets from account
	// case xdr.OperationTypeSetOptions:
	// 	assetsModified.addAssetsFromAccount(coreQ, sourceAccount)
	case xdr.OperationTypePayment:
		return assetsModified.handlePaymentOp(body.PaymentOp, sourceAccount)
	case xdr.OperationTypePathPayment:
		// if this gets expensive then we can limit it to only include those assets that includes the issuer
		assetsModified[body.PathPaymentOp.DestAsset.String()] = body.PathPaymentOp.DestAsset
		assetsModified[body.PathPaymentOp.SendAsset.String()] = body.PathPaymentOp.SendAsset
		for _, asset := range body.PathPaymentOp.Path {
			assetsModified[asset.String()] = asset
		}
	case xdr.OperationTypeManageOffer:
		return assetsModified.handleManageOfferOp(body.ManageOfferOp, sourceAccount)
	case xdr.OperationTypeCreatePassiveOffer:
		return assetsModified.handlePassiveOfferOp(body.CreatePassiveOfferOp, sourceAccount)
	case xdr.OperationTypeChangeTrust:
		assetsModified[body.ChangeTrustOp.Line.String()] = body.ChangeTrustOp.Line
	case xdr.OperationTypeAllowTrust:
		asset := body.AllowTrustOp.Asset.ToAsset(*sourceAccount)
		assetsModified[asset.String()] = asset
	}

	return nil
}

// UpdateAssetStats updates the db with the latest asset stats for the assets that were modified
func (assetsModified AssetsModified) UpdateAssetStats(is *Session) {
	if is.Err != nil {
		return
	}

	hasValue := false
	for _, asset := range assetsModified {
		assetStat := computeAssetStat(is, &asset)
		if is.Err != nil {
			return
		}

		if assetStat != nil {
			hasValue = true
			is.Ingestion.assetStats = is.Ingestion.assetStats.Values(
				assetStat.ID,
				assetStat.Amount,
				assetStat.NumAccounts,
				assetStat.Flags,
				assetStat.Toml,
			)
		}
	}

	if hasValue {
		is.Ingestion.assetStats = is.Ingestion.assetStats.
			Suffix("ON CONFLICT (id) DO UPDATE SET (amount, num_accounts, flags, toml) = (excluded.amount, excluded.num_accounts, excluded.flags, excluded.toml)")
		_, is.Err = is.Ingestion.DB.Exec(is.Ingestion.assetStats)
	}
}

// func (assetsModified AssetsModified) addAssetsFromAccount(coreQ *core.Q, account *xdr.AccountId) {
// 	if account == nil {
// 		return
// 	}

// 	var assets []xdr.Asset
// 	coreQ.AssetsForAddress(&assets, account.Address())

// 	for _, asset := range assets {
// 		if asset.Type != xdr.AssetTypeAssetTypeNative {
// 			assetsModified[asset.String()] = asset
// 		}
// 	}
// }

func (assetsModified AssetsModified) updateIfAssetIssuerInvolved(asset xdr.Asset, account xdr.AccountId) error {
	var assetType, assetCode, assetIssuer string
	err := asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		return err
	}

	if assetIssuer == account.Address() {
		assetsModified[asset.String()] = asset
	}
	return nil
}

func computeAssetStat(is *Session, asset *xdr.Asset) *history.AssetStat {
	if asset.Type == xdr.AssetTypeAssetTypeNative {
		return nil
	}

	historyQ := history.Q{Session: is.Ingestion.DB}
	assetID, err := historyQ.GetCreateAssetID(*asset)
	if err != nil {
		is.Err = err
		return nil
	}

	var assetType xdr.AssetType
	var assetCode, assetIssuer string
	err = asset.Extract(&assetType, &assetCode, &assetIssuer)
	if err != nil {
		is.Err = err
		return nil
	}

	coreQ := &core.Q{Session: is.Cursor.DB}

	numAccounts, amount, err := statTrustlinesInfo(coreQ, assetType, assetCode, assetIssuer)
	if err != nil {
		is.Err = err
		return nil
	}

	flags, toml, err := statAccountInfo(coreQ, assetIssuer)
	if err != nil {
		is.Err = err
		return nil
	}

	return &history.AssetStat{
		ID:          assetID,
		Amount:      amount,
		NumAccounts: numAccounts,
		Flags:       flags,
		Toml:        toml,
	}
}

// statTrustlinesInfo fetches all the stats from the trustlines table
func statTrustlinesInfo(coreQ *core.Q, assetType xdr.AssetType, assetCode string, assetIssuer string) (int32, int64, error) {
	return coreQ.BalancesForAsset(int32(assetType), assetCode, assetIssuer)
}

// statAccountInfo fetches all the stats from the accounts table
func statAccountInfo(coreQ *core.Q, accountID string) (int8, string, error) {
	var account core.Account
	err := coreQ.AccountByAddress(&account, accountID)
	if err != nil {
		return -1, "", err
	}

	var toml string
	if !account.HomeDomain.Valid {
		toml = ""
	} else {
		trimmed := strings.TrimSpace(account.HomeDomain.String)
		if trimmed != "" {
			toml = "https://" + account.HomeDomain.String + "/.well-known/stellar.toml"
		} else {
			toml = ""
		}
	}

	return int8(account.Flags), toml, nil
}
