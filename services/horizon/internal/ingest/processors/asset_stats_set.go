package processors

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/stellar/go/ingest"
	"github.com/stellar/go/services/horizon/internal/db2/history"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

type assetStatKey struct {
	assetType   xdr.AssetType
	assetCode   string
	assetIssuer string
}

type assetStatValue struct {
	assetStatKey
	balances assetStatBalances
	accounts history.ExpAssetStatAccounts
}

type assetStatBalances struct {
	Authorized                      *big.Int
	AuthorizedToMaintainLiabilities *big.Int
	ClaimableBalances               *big.Int
	LiquidityPools                  *big.Int
	Unauthorized                    *big.Int
	Contracts                       *big.Int
}

func newAssetStatBalance() assetStatBalances {
	return assetStatBalances{
		Authorized:                      big.NewInt(0),
		AuthorizedToMaintainLiabilities: big.NewInt(0),
		ClaimableBalances:               big.NewInt(0),
		LiquidityPools:                  big.NewInt(0),
		Unauthorized:                    big.NewInt(0),
		Contracts:                       big.NewInt(0),
	}
}

func (a *assetStatBalances) Parse(b *history.ExpAssetStatBalances) error {
	authorized, ok := new(big.Int).SetString(b.Authorized, 10)
	if !ok {
		return errors.New("Error parsing: " + b.Authorized)
	}
	a.Authorized = authorized

	authorizedToMaintainLiabilities, ok := new(big.Int).SetString(b.AuthorizedToMaintainLiabilities, 10)
	if !ok {
		return errors.New("Error parsing: " + b.AuthorizedToMaintainLiabilities)
	}
	a.AuthorizedToMaintainLiabilities = authorizedToMaintainLiabilities

	claimableBalances, ok := new(big.Int).SetString(b.ClaimableBalances, 10)
	if !ok {
		return errors.New("Error parsing: " + b.ClaimableBalances)
	}
	a.ClaimableBalances = claimableBalances

	liquidityPools, ok := new(big.Int).SetString(b.LiquidityPools, 10)
	if !ok {
		return errors.New("Error parsing: " + b.LiquidityPools)
	}
	a.LiquidityPools = liquidityPools

	unauthorized, ok := new(big.Int).SetString(b.Unauthorized, 10)
	if !ok {
		return errors.New("Error parsing: " + b.Unauthorized)
	}
	a.Unauthorized = unauthorized

	contracts, ok := new(big.Int).SetString(b.Contracts, 10)
	if !ok {
		return errors.New("Error parsing: " + b.Contracts)
	}
	a.Contracts = contracts

	return nil
}

func (a assetStatBalances) Add(b assetStatBalances) assetStatBalances {
	return assetStatBalances{
		Authorized:                      big.NewInt(0).Add(a.Authorized, b.Authorized),
		AuthorizedToMaintainLiabilities: big.NewInt(0).Add(a.AuthorizedToMaintainLiabilities, b.AuthorizedToMaintainLiabilities),
		ClaimableBalances:               big.NewInt(0).Add(a.ClaimableBalances, b.ClaimableBalances),
		LiquidityPools:                  big.NewInt(0).Add(a.LiquidityPools, b.LiquidityPools),
		Unauthorized:                    big.NewInt(0).Add(a.Unauthorized, b.Unauthorized),
		Contracts:                       big.NewInt(0).Add(a.Contracts, b.Contracts),
	}
}

func (a assetStatBalances) IsZero() bool {
	return a.Authorized.Cmp(big.NewInt(0)) == 0 &&
		a.AuthorizedToMaintainLiabilities.Cmp(big.NewInt(0)) == 0 &&
		a.ClaimableBalances.Cmp(big.NewInt(0)) == 0 &&
		a.LiquidityPools.Cmp(big.NewInt(0)) == 0 &&
		a.Unauthorized.Cmp(big.NewInt(0)) == 0 &&
		a.Contracts.Cmp(big.NewInt(0)) == 0
}

func (a assetStatBalances) ConvertToHistoryObject() history.ExpAssetStatBalances {
	return history.ExpAssetStatBalances{
		Authorized:                      a.Authorized.String(),
		AuthorizedToMaintainLiabilities: a.AuthorizedToMaintainLiabilities.String(),
		ClaimableBalances:               a.ClaimableBalances.String(),
		LiquidityPools:                  a.LiquidityPools.String(),
		Unauthorized:                    a.Unauthorized.String(),
		Contracts:                       a.Contracts.String(),
	}
}

func (value assetStatValue) ConvertToHistoryObject() history.ExpAssetStat {
	balances := value.balances.ConvertToHistoryObject()
	return history.ExpAssetStat{
		AssetType:   value.assetType,
		AssetCode:   value.assetCode,
		AssetIssuer: value.assetIssuer,
		Accounts:    value.accounts,
		Balances:    balances,
		Amount:      balances.Authorized,
		NumAccounts: value.accounts.Authorized,
	}
}

type contractAssetStatValue struct {
	balance    *big.Int
	numHolders int32
}

// AssetStatSet represents a collection of asset stats and a mapping
// of Soroban contract IDs to classic assets (which is unique to each
// network).
type AssetStatSet struct {
	classicAssetStats  map[assetStatKey]*assetStatValue
	contractToAsset    map[[32]byte]*xdr.Asset
	contractAssetStats map[[32]byte]contractAssetStatValue
	networkPassphrase  string
}

// NewAssetStatSet constructs a new AssetStatSet instance
func NewAssetStatSet(networkPassphrase string) AssetStatSet {
	return AssetStatSet{
		classicAssetStats:  map[assetStatKey]*assetStatValue{},
		contractToAsset:    map[[32]byte]*xdr.Asset{},
		contractAssetStats: map[[32]byte]contractAssetStatValue{},
		networkPassphrase:  networkPassphrase,
	}
}

type delta struct {
	Authorized                      int64
	AuthorizedToMaintainLiabilities int64
	Unauthorized                    int64
	ClaimableBalances               int64
	LiquidityPools                  int64
}

func (d *delta) addByFlags(flags xdr.Uint32, amount int64) {
	f := xdr.TrustLineFlags(flags)
	if f.IsAuthorized() {
		d.Authorized += amount
	} else if f.IsAuthorizedToMaintainLiabilitiesFlag() {
		d.AuthorizedToMaintainLiabilities += amount
	} else {
		d.Unauthorized += amount
	}
}

func (d delta) isEmpty() bool {
	return d == delta{}
}

// addDelta adds a delta balance and delta accounts to a given asset trustline.
func (s AssetStatSet) addDelta(asset xdr.Asset, deltaBalances, deltaAccounts delta) error {
	if deltaBalances.isEmpty() && deltaAccounts.isEmpty() {
		return nil
	}

	var key assetStatKey
	if err := asset.Extract(&key.assetType, &key.assetCode, &key.assetIssuer); err != nil {
		return errors.Wrap(err, "could not extract asset info from trustline")
	}

	current, ok := s.classicAssetStats[key]
	if !ok {
		current = &assetStatValue{assetStatKey: key, balances: newAssetStatBalance()}
		s.classicAssetStats[key] = current
	}

	current.accounts.Authorized += int32(deltaAccounts.Authorized)
	current.accounts.AuthorizedToMaintainLiabilities += int32(deltaAccounts.AuthorizedToMaintainLiabilities)
	current.accounts.ClaimableBalances += int32(deltaAccounts.ClaimableBalances)
	current.accounts.LiquidityPools += int32(deltaAccounts.LiquidityPools)
	current.accounts.Unauthorized += int32(deltaAccounts.Unauthorized)

	current.balances.Authorized.Add(current.balances.Authorized, big.NewInt(deltaBalances.Authorized))
	current.balances.AuthorizedToMaintainLiabilities.Add(current.balances.AuthorizedToMaintainLiabilities, big.NewInt(deltaBalances.AuthorizedToMaintainLiabilities))
	current.balances.ClaimableBalances.Add(current.balances.ClaimableBalances, big.NewInt(deltaBalances.ClaimableBalances))
	current.balances.LiquidityPools.Add(current.balances.LiquidityPools, big.NewInt(deltaBalances.LiquidityPools))
	current.balances.Unauthorized.Add(current.balances.Unauthorized, big.NewInt(deltaBalances.Unauthorized))

	// Note: it's possible that after operations above:
	// numAccounts != 0 && amount == 0 (ex. two accounts send some of their assets to third account)
	//  OR
	// numAccounts == 0 && amount != 0 (ex. issuer issued an asset)
	if current.balances.IsZero() && current.accounts.IsZero() {
		delete(s.classicAssetStats, key)
	}

	return nil
}

// AddTrustline updates the set to account for how a given trustline has changed.
// change must be a xdr.LedgerEntryTypeTrustLine type.
func (s AssetStatSet) AddTrustline(change ingest.Change) error {
	var pre, post *xdr.TrustLineEntry
	if change.Pre != nil {
		pre = change.Pre.Data.TrustLine
	}
	if change.Post != nil {
		post = change.Post.Data.TrustLine
	}

	deltaAccounts := delta{}
	deltaBalances := delta{}

	if pre == nil && post == nil {
		return ingest.NewStateError(errors.New("both pre and post trustlines cannot be nil"))
	}

	var asset xdr.TrustLineAsset
	if pre != nil {
		asset = pre.Asset
		deltaAccounts.addByFlags(pre.Flags, -1)
		deltaBalances.addByFlags(pre.Flags, -int64(pre.Balance))
	}
	if post != nil {
		asset = post.Asset
		deltaAccounts.addByFlags(post.Flags, 1)
		deltaBalances.addByFlags(post.Flags, int64(post.Balance))
	}
	if asset.Type == xdr.AssetTypeAssetTypePoolShare || asset.Type == xdr.AssetTypeAssetTypeNative {
		return nil
	}

	err := s.addDelta(asset.ToAsset(), deltaBalances, deltaAccounts)
	if err != nil {
		return errors.Wrap(err, "error running AssetStatSet.addDelta")
	}
	return nil
}

// AddLiquidityPool updates the set to account for how a given liquidity pool has changed.
// change must be a xdr.LedgerEntryTypeLiqidityPool type.
func (s AssetStatSet) AddLiquidityPool(change ingest.Change) error {
	var pre, post *xdr.LiquidityPoolEntry
	if change.Pre != nil {
		pre = change.Pre.Data.LiquidityPool
	}
	if change.Post != nil {
		post = change.Post.Data.LiquidityPool
	}

	assetAdeltaNum := delta{}
	assetAdeltaBalances := delta{}
	assetBdeltaNum := delta{}
	assetBdeltaBalances := delta{}

	if pre == nil && post == nil {
		return ingest.NewStateError(errors.New("both pre and post liquidity pools cannot be nil"))
	}

	lpType, err := change.GetLiquidityPoolType()
	if err != nil {
		return ingest.NewStateError(err)
	}

	var assetA, assetB xdr.Asset
	switch lpType {
	case xdr.LiquidityPoolTypeLiquidityPoolConstantProduct:
		if pre != nil {
			assetA = pre.Body.ConstantProduct.Params.AssetA
			assetAdeltaNum.LiquidityPools--
			assetAdeltaBalances.LiquidityPools -= int64(pre.Body.ConstantProduct.ReserveA)

			assetB = pre.Body.ConstantProduct.Params.AssetB
			assetBdeltaNum.LiquidityPools--
			assetBdeltaBalances.LiquidityPools -= int64(pre.Body.ConstantProduct.ReserveB)
		}
		if post != nil {
			assetA = post.Body.ConstantProduct.Params.AssetA
			assetAdeltaNum.LiquidityPools++
			assetAdeltaBalances.LiquidityPools += int64(post.Body.ConstantProduct.ReserveA)

			assetB = post.Body.ConstantProduct.Params.AssetB
			assetBdeltaNum.LiquidityPools++
			assetBdeltaBalances.LiquidityPools += int64(post.Body.ConstantProduct.ReserveB)
		}
	default:
		return errors.Errorf("Unknown liquidity pool type=%d", lpType)
	}

	if assetA.Type != xdr.AssetTypeAssetTypeNative {
		err := s.addDelta(assetA, assetAdeltaBalances, assetAdeltaNum)
		if err != nil {
			return errors.Wrap(err, "error running AssetStatSet.addDelta using AssetA")
		}
	}

	if assetB.Type != xdr.AssetTypeAssetTypeNative {
		err := s.addDelta(assetB, assetBdeltaBalances, assetBdeltaNum)
		if err != nil {
			return errors.Wrap(err, "error running AssetStatSet.addDelta using AssetB")
		}
	}

	return nil
}

// AddClaimableBalance updates the set to account for how a given claimable balance has changed.
// change must be a xdr.LedgerEntryTypeClaimableBalance type.
func (s AssetStatSet) AddClaimableBalance(change ingest.Change) error {
	var pre, post *xdr.ClaimableBalanceEntry
	if change.Pre != nil {
		pre = change.Pre.Data.ClaimableBalance
	}
	if change.Post != nil {
		post = change.Post.Data.ClaimableBalance
	}

	deltaAccounts := delta{}
	deltaBalances := delta{}

	if pre == nil && post == nil {
		return ingest.NewStateError(errors.New("both pre and post claimable balances cannot be nil"))
	}

	var asset xdr.Asset
	if pre != nil {
		asset = pre.Asset
		deltaAccounts.ClaimableBalances--
		deltaBalances.ClaimableBalances -= int64(pre.Amount)
	}
	if post != nil {
		asset = post.Asset
		deltaAccounts.ClaimableBalances++
		deltaBalances.ClaimableBalances += int64(post.Amount)
	}

	if asset.Type == xdr.AssetTypeAssetTypeNative {
		return nil
	}

	err := s.addDelta(asset, deltaBalances, deltaAccounts)
	if err != nil {
		return errors.Wrap(err, "error running AssetStatSet.addDelta")
	}
	return nil
}

// AddContractData updates the set to account for how a given contract data entry has changed.
// change must be a xdr.LedgerEntryTypeContractData type.
func (s AssetStatSet) AddContractData(change ingest.Change) error {
	if err := s.ingestAssetContractMetadata(change); err != nil {
		return err
	}
	s.ingestAssetContractBalance(change)
	return nil
}

func (s AssetStatSet) ingestAssetContractMetadata(change ingest.Change) error {
	if change.Pre != nil {
		asset := AssetFromContractData(*change.Pre, s.networkPassphrase)
		if asset == nil {
			return nil
		}
		contractID := change.Pre.Data.MustContractData().ContractId
		if change.Post == nil {
			s.contractToAsset[contractID] = nil
			return nil
		}
		// The contract id for a stellar asset should never change and
		// therefore we return a fatal ingestion error if we encounter
		// a stellar asset changing contract ids.
		postAsset := AssetFromContractData(*change.Post, s.networkPassphrase)
		if postAsset == nil || !(*postAsset).Equals(*asset) {
			return ingest.NewStateError(fmt.Errorf("asset contract changed asset"))
		}
	} else if change.Post != nil {
		asset := AssetFromContractData(*change.Post, s.networkPassphrase)
		if asset == nil {
			return nil
		}
		contractID := change.Post.Data.MustContractData().ContractId
		s.contractToAsset[contractID] = asset
	}
	return nil
}

func (s AssetStatSet) ingestAssetContractBalance(change ingest.Change) {
	if change.Pre != nil {
		contractID := change.Pre.Data.MustContractData().ContractId
		holder, amt, ok := ContractBalanceFromContractData(*change.Pre, s.networkPassphrase)
		if !ok {
			return
		}
		stats, ok := s.contractAssetStats[contractID]
		if !ok {
			stats = contractAssetStatValue{
				balance:    big.NewInt(0),
				numHolders: 0,
			}
		}

		if change.Post == nil {
			// the balance was removed so we need to deduct from
			// contract holders and contract balance amount
			stats.balance = new(big.Int).Sub(stats.balance, amt)
			// only decrement holders if the removed balance
			// contained a positive amount of the asset.
			if amt.Cmp(big.NewInt(0)) > 0 {
				stats.numHolders--
			}
			s.maybeAddContractAssetStat(contractID, stats)
			return
		}
		// if the updated ledger entry is not in the expected format then this
		// cannot be emitted by the stellar asset contract, so ignore it
		postHolder, postAmt, postOk := ContractBalanceFromContractData(*change.Post, s.networkPassphrase)
		if !postOk || postHolder != holder {
			return
		}

		delta := new(big.Int).Sub(postAmt, amt)
		stats.balance.Add(stats.balance, delta)
		if postAmt.Cmp(big.NewInt(0)) == 0 && amt.Cmp(big.NewInt(0)) > 0 {
			// if the pre amount is equal to the post amount it means the balance was wiped out so
			// we can decrement the number of contract holders
			stats.numHolders--
		} else if amt.Cmp(big.NewInt(0)) == 0 && postAmt.Cmp(big.NewInt(0)) > 0 {
			// if the pre amount was zero and the post amount is positive the number of
			// contract holders increased
			stats.numHolders++
		}
		s.maybeAddContractAssetStat(contractID, stats)
		return
	}
	// in this case there was no balance before the change
	contractID := change.Post.Data.MustContractData().ContractId
	_, amt, ok := ContractBalanceFromContractData(*change.Post, s.networkPassphrase)
	if !ok {
		return
	}

	// ignore zero balance amounts
	if amt.Cmp(big.NewInt(0)) == 0 {
		return
	}

	// increase the number of contract holders because previously
	// there was no balance
	stats, ok := s.contractAssetStats[contractID]
	if !ok {
		stats = contractAssetStatValue{
			balance:    amt,
			numHolders: 1,
		}
	} else {
		stats.balance = new(big.Int).Add(stats.balance, amt)
		stats.numHolders++
	}

	s.maybeAddContractAssetStat(contractID, stats)
}

func (s AssetStatSet) maybeAddContractAssetStat(contractID [32]byte, stat contractAssetStatValue) {
	if stat.numHolders == 0 && stat.balance.Cmp(big.NewInt(0)) == 0 {
		delete(s.contractAssetStats, contractID)
	} else {
		s.contractAssetStats[contractID] = stat
	}
}

// All returns a list of all `history.ExpAssetStat` contained within the set
// along with all contract id attribution changes in the set.
func (s AssetStatSet) All() ([]history.ExpAssetStat, map[[32]byte]*xdr.Asset, map[[32]byte]contractAssetStatValue) {
	assetStats := make([]history.ExpAssetStat, 0, len(s.classicAssetStats))
	for _, value := range s.classicAssetStats {
		assetStats = append(assetStats, value.ConvertToHistoryObject())
	}
	contractToAsset := make(map[[32]byte]*xdr.Asset, len(s.contractToAsset))
	for key, val := range s.contractToAsset {
		contractToAsset[key] = val
	}
	contractAssetStats := make(map[[32]byte]contractAssetStatValue, len(s.contractAssetStats))
	for key, val := range s.contractAssetStats {
		contractAssetStats[key] = val
	}
	return assetStats, contractToAsset, contractAssetStats
}

// AllFromSnapshot returns a list of all `history.ExpAssetStat` contained within the set.
// AllFromSnapshot should only be invoked when the AssetStatSet has been derived from ledger
// entry changes consisting of only inserts (no updates) reflecting the current state of
// the ledger without any missing entries (e.g. history archives).
func (s AssetStatSet) AllFromSnapshot() ([]history.ExpAssetStat, error) {
	// merge assetStatsDeltas and contractToAsset into one list of history.ExpAssetStat.
	assetStatsDeltas, contractToAsset, contractAssetStats := s.All()

	// modify the asset stat row to update the contract_id column whenever we encounter a
	// contract data ledger entry with the Stellar asset metadata.
	for i, assetStatDelta := range assetStatsDeltas {
		// asset stats only supports non-native assets
		asset := xdr.MustNewCreditAsset(assetStatDelta.AssetCode, assetStatDelta.AssetIssuer)
		contractID, err := asset.ContractID(s.networkPassphrase)
		if err != nil {
			return nil, errors.Wrap(err, "cannot compute contract id for asset")
		}
		if asset, ok := contractToAsset[contractID]; ok && asset == nil {
			return nil, ingest.NewStateError(fmt.Errorf(
				"unexpected contract data removal in history archives: %s",
				hex.EncodeToString(contractID[:]),
			))
		} else if ok {
			assetStatDelta.SetContractID(contractID)
			delete(contractToAsset, contractID)
		}

		if stats, ok := contractAssetStats[contractID]; ok {
			assetStatDelta.Accounts.Contracts = stats.numHolders
			assetStatDelta.Balances.Contracts = stats.balance.String()
		}
		assetStatsDeltas[i] = assetStatDelta
	}

	// There is also a corner case where a Stellar Asset contract is initialized before there exists any
	// trustlines / claimable balances for the Stellar Asset. In this case, when ingesting contract data
	// ledger entries, there will be no existing asset stat row. We handle this case by inserting a row
	// with zero stats just so we can populate the contract id.
	for contractID, asset := range contractToAsset {
		if asset == nil {
			return nil, ingest.NewStateError(fmt.Errorf(
				"unexpected contract data removal in history archives: %s",
				hex.EncodeToString(contractID[:]),
			))
		}
		var assetType xdr.AssetType
		var assetCode, assetIssuer string
		asset.MustExtract(&assetType, &assetCode, &assetIssuer)
		row := history.ExpAssetStat{
			AssetType:   assetType,
			AssetCode:   assetCode,
			AssetIssuer: assetIssuer,
			Accounts:    history.ExpAssetStatAccounts{},
			Balances:    newAssetStatBalance().ConvertToHistoryObject(),
			Amount:      "0",
			NumAccounts: 0,
		}
		if stats, ok := contractAssetStats[contractID]; ok {
			row.Accounts.Contracts = stats.numHolders
			row.Balances.Contracts = stats.balance.String()
		}
		row.SetContractID(contractID)
		assetStatsDeltas = append(assetStatsDeltas, row)
	}
	// all balances remaining in contractAssetStats do not belong to
	// stellar asset contracts (because all stellar asset contracts must
	// be in contractToAsset) so we can ignore them
	return assetStatsDeltas, nil
}
