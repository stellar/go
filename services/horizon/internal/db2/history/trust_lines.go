package history

import (
	"context"
	sq "github.com/Masterminds/squirrel"
	"github.com/guregu/null"
	"github.com/lib/pq"
	"github.com/stellar/go/support/db"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// IsAuthorized returns true if issuer has authorized account to perform
// transactions with its credit
func (trustLine TrustLine) IsAuthorized() bool {
	return xdr.TrustLineFlags(trustLine.Flags).IsAuthorized()
}

// IsAuthorizedToMaintainLiabilities returns true if issuer has authorized the account to maintain
// liabilities with its credit
func (trustLine TrustLine) IsAuthorizedToMaintainLiabilities() bool {
	return xdr.TrustLineFlags(trustLine.Flags).IsAuthorizedToMaintainLiabilitiesFlag()
}

// IsClawbackEnabled returns true if issuer has authorized the account to claw
// assets back
func (trustLine TrustLine) IsClawbackEnabled() bool {
	return xdr.TrustLineFlags(trustLine.Flags).IsClawbackEnabledFlag()
}

func (q *Q) CountTrustLines(ctx context.Context) (int, error) {
	sql := sq.Select("count(*)").From("trust_lines")

	var count int
	if err := q.Get(ctx, &count, sql); err != nil {
		return 0, errors.Wrap(err, "could not run select query")
	}

	return count, nil
}

func (q *Q) GetSortedTrustLinesByAccountID(ctx context.Context, id string) ([]TrustLine, error) {
	return q.GetSortedTrustLinesByAccountIDs(ctx, []string{id})
}

// GetTrustLinesByKeys loads a row from the `trust_lines` table, selected by multiple keys.
func (q *Q) GetTrustLinesByKeys(ctx context.Context, ledgerKeys []string) ([]TrustLine, error) {
	var trustLines []TrustLine
	sql := selectTrustLines.Where(map[string]interface{}{"trust_lines.ledger_key": ledgerKeys})
	err := q.Select(ctx, &trustLines, sql)
	return trustLines, err
}

// UpsertTrustLines upserts a batch of trust lines in the trust lines table.
// There's currently no limit of the number of trust lines this method can
// accept other than 2GB limit of the query string length what should be enough
// for each ledger with the current limits.
func (q *Q) UpsertTrustLines(ctx context.Context, trustLines []TrustLine) error {
	var ledgerKey, accountID, assetIssuer, assetCode []string
	var balance, limit, buyingLiabilities, sellingLiabilities []int64
	var flags, lastModifiedLedger []uint32
	var assetType []xdr.AssetType
	var liquidityPoolIds, sponsor []null.String

	for _, entry := range trustLines {
		ledgerKey = append(ledgerKey, entry.LedgerKey)
		accountID = append(accountID, entry.AccountID)
		assetType = append(assetType, entry.AssetType)
		assetIssuer = append(assetIssuer, entry.AssetIssuer)
		assetCode = append(assetCode, entry.AssetCode)
		liquidityPoolIds = append(liquidityPoolIds, null.NewString(entry.LiquidityPoolID, len(entry.LiquidityPoolID) > 0))
		balance = append(balance, entry.Balance)
		limit = append(limit, entry.Limit)
		buyingLiabilities = append(buyingLiabilities, entry.BuyingLiabilities)
		sellingLiabilities = append(sellingLiabilities, entry.SellingLiabilities)
		flags = append(flags, entry.Flags)
		lastModifiedLedger = append(lastModifiedLedger, entry.LastModifiedLedger)
		sponsor = append(sponsor, entry.Sponsor)
	}

	sql := `
	WITH r AS
		(SELECT
			unnest(?::text[]),
			unnest(?::text[]),
			unnest(?::int[]),
			unnest(?::text[]),
			unnest(?::text[]),
			unnest(?::text[]),
			unnest(?::bigint[]),
			unnest(?::bigint[]),
			unnest(?::bigint[]),
			unnest(?::bigint[]),
			unnest(?::int[]),
			unnest(?::int[]),
			unnest(?::text[])
		)
	INSERT INTO trust_lines ( 
		ledger_key,
		account_id,
		asset_type,
		asset_issuer,
		asset_code,
		liquidity_pool_id,
		balance,
		trust_line_limit,
		buying_liabilities,
		selling_liabilities,
		flags,
		last_modified_ledger,
		sponsor
	)
	SELECT * from r 
	ON CONFLICT (ledger_key) DO UPDATE SET 
		ledger_key = excluded.ledger_key,
		account_id = excluded.account_id,
		asset_type = excluded.asset_type,
		asset_issuer = excluded.asset_issuer,
		asset_code = excluded.asset_code,
		liquidity_pool_id = excluded.liquidity_pool_id,
		balance = excluded.balance,
		trust_line_limit = excluded.trust_line_limit,
		buying_liabilities = excluded.buying_liabilities,
		selling_liabilities = excluded.selling_liabilities,
		flags = excluded.flags,
		last_modified_ledger = excluded.last_modified_ledger,
		sponsor = excluded.sponsor`

	_, err := q.ExecRaw(
		context.WithValue(ctx, &db.QueryTypeContextKey, db.UpsertQueryType),
		sql,
		pq.Array(ledgerKey),
		pq.Array(accountID),
		pq.Array(assetType),
		pq.Array(assetIssuer),
		pq.Array(assetCode),
		pq.Array(liquidityPoolIds),
		pq.Array(balance),
		pq.Array(limit),
		pq.Array(buyingLiabilities),
		pq.Array(sellingLiabilities),
		pq.Array(flags),
		pq.Array(lastModifiedLedger),
		pq.Array(sponsor))
	return err
}

// RemoveTrustLine deletes a row in the trust lines table.
// Returns number of rows affected and error.
func (q *Q) RemoveTrustLine(ctx context.Context, ledgerKey string) (int64, error) {
	sql := sq.Delete("trust_lines").
		Where(map[string]interface{}{"ledger_key": ledgerKey})
	result, err := q.Exec(ctx, sql)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// GetSortedTrustLinesByAccountIDs loads trust lines for a list of accounts ID, ordered by asset and issuer
func (q *Q) GetSortedTrustLinesByAccountIDs(ctx context.Context, id []string) ([]TrustLine, error) {
	var data []TrustLine
	sql := selectTrustLines.Where(sq.Eq{"account_id": id}).
		OrderBy("asset_code", "asset_issuer", "liquidity_pool_id")
	err := q.Select(ctx, &data, sql)
	return data, err
}

var selectTrustLines = sq.Select(`
	ledger_key,
	account_id,
	asset_type,
	asset_issuer,
	asset_code,
	COALESCE(liquidity_pool_id, '') as liquidity_pool_id,
	balance,
	trust_line_limit,
	buying_liabilities,
	selling_liabilities,
	flags,
	last_modified_ledger,
	sponsor
`).From("trust_lines")
