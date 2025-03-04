package ingest

import (
	"context"
	"fmt"

	"github.com/stellar/go/xdr"
)

// StatsChangeProcessor is a state processors that counts number of changes types
// and entry types.
type StatsChangeProcessor struct {
	results StatsChangeProcessorResults
}

// StatsChangeProcessorResults contains results after running StatsChangeProcessor.
type StatsChangeProcessorResults struct {
	AccountsCreated int64
	AccountsUpdated int64
	AccountsRemoved int64

	ClaimableBalancesCreated int64
	ClaimableBalancesUpdated int64
	ClaimableBalancesRemoved int64

	DataCreated int64
	DataUpdated int64
	DataRemoved int64

	OffersCreated int64
	OffersUpdated int64
	OffersRemoved int64

	TrustLinesCreated int64
	TrustLinesUpdated int64
	TrustLinesRemoved int64

	LiquidityPoolsCreated int64
	LiquidityPoolsUpdated int64
	LiquidityPoolsRemoved int64

	ContractDataCreated  int64
	ContractDataUpdated  int64
	ContractDataRemoved  int64
	ContractDataRestored int64

	ContractCodeCreated  int64
	ContractCodeUpdated  int64
	ContractCodeRemoved  int64
	ContractCodeRestored int64

	ConfigSettingsCreated int64
	ConfigSettingsUpdated int64
	ConfigSettingsRemoved int64

	TtlCreated  int64
	TtlUpdated  int64
	TtlRemoved  int64
	TtlRestored int64

	LedgerEntriesEvicted int64
}

func (p *StatsChangeProcessor) ProcessChange(ctx context.Context, change Change) error {
	switch change.Type {
	case xdr.LedgerEntryTypeAccount:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.AccountsCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.AccountsUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.AccountsRemoved++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	case xdr.LedgerEntryTypeClaimableBalance:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.ClaimableBalancesCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.ClaimableBalancesUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.ClaimableBalancesRemoved++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	case xdr.LedgerEntryTypeData:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.DataCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.DataUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.DataRemoved++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	case xdr.LedgerEntryTypeOffer:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.OffersCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.OffersUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.OffersRemoved++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	case xdr.LedgerEntryTypeTrustline:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.TrustLinesCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.TrustLinesUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.TrustLinesRemoved++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	case xdr.LedgerEntryTypeLiquidityPool:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.LiquidityPoolsCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.LiquidityPoolsUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.LiquidityPoolsRemoved++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	case xdr.LedgerEntryTypeContractData:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.ContractDataCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.ContractDataUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.ContractDataRemoved++
		case xdr.LedgerEntryChangeTypeLedgerEntryRestored:
			p.results.ContractDataRestored++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	case xdr.LedgerEntryTypeContractCode:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.ContractCodeCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.ContractCodeUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.ContractCodeRemoved++
		case xdr.LedgerEntryChangeTypeLedgerEntryRestored:
			p.results.ContractCodeRestored++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	case xdr.LedgerEntryTypeConfigSetting:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.ConfigSettingsCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.ConfigSettingsUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.ConfigSettingsRemoved++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	case xdr.LedgerEntryTypeTtl:
		switch change.ChangeType {
		case xdr.LedgerEntryChangeTypeLedgerEntryCreated:
			p.results.TtlCreated++
		case xdr.LedgerEntryChangeTypeLedgerEntryUpdated:
			p.results.TtlUpdated++
		case xdr.LedgerEntryChangeTypeLedgerEntryRemoved:
			p.results.TtlRemoved++
		case xdr.LedgerEntryChangeTypeLedgerEntryRestored:
			p.results.TtlRestored++
		default:
			return fmt.Errorf("unsupported ledger entry change type %s for %s",
				change.ChangeType, change.Type.String())
		}
	default:
		return fmt.Errorf("unsupported ledger entry type: %s", change.Type.String())
	}

	return nil
}

func (p *StatsChangeProcessor) ProcessEvictions(evictions []xdr.LedgerKey) {
	p.results.LedgerEntriesEvicted += int64(len(evictions))
}

func (p *StatsChangeProcessor) GetResults() StatsChangeProcessorResults {
	return p.results
}

func (stats *StatsChangeProcessorResults) Map() map[string]interface{} {
	return map[string]interface{}{
		"stats_accounts_created": stats.AccountsCreated,
		"stats_accounts_updated": stats.AccountsUpdated,
		"stats_accounts_removed": stats.AccountsRemoved,

		"stats_claimable_balances_created": stats.ClaimableBalancesCreated,
		"stats_claimable_balances_updated": stats.ClaimableBalancesUpdated,
		"stats_claimable_balances_removed": stats.ClaimableBalancesRemoved,

		"stats_data_created": stats.DataCreated,
		"stats_data_updated": stats.DataUpdated,
		"stats_data_removed": stats.DataRemoved,

		"stats_offers_created": stats.OffersCreated,
		"stats_offers_updated": stats.OffersUpdated,
		"stats_offers_removed": stats.OffersRemoved,

		"stats_trust_lines_created": stats.TrustLinesCreated,
		"stats_trust_lines_updated": stats.TrustLinesUpdated,
		"stats_trust_lines_removed": stats.TrustLinesRemoved,

		"stats_liquidity_pools_created": stats.LiquidityPoolsCreated,
		"stats_liquidity_pools_updated": stats.LiquidityPoolsUpdated,
		"stats_liquidity_pools_removed": stats.LiquidityPoolsRemoved,

		"stats_contract_data_created":  stats.ContractDataCreated,
		"stats_contract_data_updated":  stats.ContractDataUpdated,
		"stats_contract_data_removed":  stats.ContractDataRemoved,
		"stats_contract_data_restored": stats.ContractDataRestored,

		"stats_contract_code_created":  stats.ContractCodeCreated,
		"stats_contract_code_updated":  stats.ContractCodeUpdated,
		"stats_contract_code_removed":  stats.ContractCodeRemoved,
		"stats_contract_code_restored": stats.ContractCodeRestored,

		"stats_config_settings_created": stats.ConfigSettingsCreated,
		"stats_config_settings_updated": stats.ConfigSettingsUpdated,
		"stats_config_settings_removed": stats.ConfigSettingsRemoved,

		"stats_ttl_created":  stats.TtlCreated,
		"stats_ttl_updated":  stats.TtlUpdated,
		"stats_ttl_removed":  stats.TtlRemoved,
		"stats_ttl_restored": stats.TtlRestored,

		"stats_ledger_entries_evicted": stats.LedgerEntriesEvicted,
	}
}
