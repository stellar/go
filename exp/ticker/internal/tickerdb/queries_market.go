package tickerdb

import (
	"fmt"
	"strings"
)

// RetrieveMarketData retrieves the 24h- and 7d aggregated market data for all
// markets that were active during this period.
func (s *TickerSession) RetrieveMarketData() (markets []Market, err error) {
	err = s.SelectRaw(&markets, marketQuery)
	return
}

// RetrievePartialAggMarkets retrieves the aggregated market data for all
// markets (or for a specific one if PairName != nil) for a given period.
func (s *TickerSession) RetrievePartialAggMarkets(
	pairName *string,
	numHoursAgo int,
) (partialMkts []PartialMarket, err error) {
	var bCode, cCode string
	sqlTrue := new(string)
	*sqlTrue = "TRUE"
	optVars := []optionalVar{
		optionalVar{"bAsset.is_valid", sqlTrue},
		optionalVar{"cAsset.is_valid", sqlTrue},
	}

	// parse base and asset codes and add them as SQL parameters
	if pairName != nil {
		bCode, cCode, err = getBaseAndCounterCodes(*pairName)
		if err != nil {
			return
		}
		optVars = append(optVars, []optionalVar{
			optionalVar{"bAsset.code", &bCode},
			optionalVar{"cAsset.code", &cCode},
		}...)
	}

	where, args := generateWhereClause(optVars)
	where += fmt.Sprintf(
		" AND t.ledger_close_time > now() - interval '%d hours'",
		numHoursAgo,
	)
	q := strings.Replace(aggMarketQuery, "__WHERECLAUSE__", where, -1)
	q = strings.Replace(q, "__NUMHOURS__", fmt.Sprintf("%d", numHoursAgo), -1)

	argsInterface := make([]interface{}, len(args))
	for i, v := range args {
		argsInterface[i] = v
	}
	err = s.SelectRaw(&partialMkts, q, argsInterface...)
	return
}

// RetrievePartialMarkets retrieves data in the PartialMarket format from the database.
// It optionally filters the data according to the provided base and counter asset params
// provided, as well as the numHoursAgo time offset.
func (s *TickerSession) RetrievePartialMarkets(
	baseAssetCode *string,
	baseAssetIssuer *string,
	counterAssetCode *string,
	counterAssetIssuer *string,
	numHoursAgo int,
) (partialMkts []PartialMarket, err error) {
	sqlTrue := new(string)
	*sqlTrue = "TRUE"

	where, args := generateWhereClause([]optionalVar{
		optionalVar{"bAsset.is_valid", sqlTrue},
		optionalVar{"cAsset.is_valid", sqlTrue},
		optionalVar{"bAsset.code", baseAssetCode},
		optionalVar{"bAsset.issuer_account", baseAssetIssuer},
		optionalVar{"cAsset.code", counterAssetCode},
		optionalVar{"cAsset.issuer_account", counterAssetIssuer},
	})
	where += fmt.Sprintf(
		" AND t.ledger_close_time > now() - interval '%d hours'",
		numHoursAgo,
	)

	q := strings.Replace(partialMarketQuery, "__WHERECLAUSE__", where, -1)
	q = strings.Replace(q, "__NUMHOURS__", fmt.Sprintf("%d", numHoursAgo), -1)

	argsInterface := make([]interface{}, len(args))
	for i, v := range args {
		argsInterface[i] = v
	}
	err = s.SelectRaw(&partialMkts, q, argsInterface...)
	return
}

var marketQuery = `
SELECT
	t2.trade_pair_name as trade_pair_name,
	COALESCE(t1.base_volume_24h, 0.0) as base_volume_24h,
	COALESCE(t1.counter_volume_24h, 0.0) as counter_volume_24h,
	COALESCE(t1.trade_count_24h, 0) as trade_count_24h,
	COALESCE(t1.highest_price_24h, 0.0) as highest_price_24h,
	COALESCE(t1.lowest_price_24h, 0.0) as lowest_price_24h,
	COALESCE(t4.price_24h_ago - last_price, 0.0) as price_change_24h,
	COALESCE(t4.price_24h_ago, 0.0) as open_price_24h,

	COALESCE(t2.base_volume_7d, 0) as base_volume_7d,
	COALESCE(t2.counter_volume_7d, 0) as counter_volume_7d,
	COALESCE(t2.trade_count_7d, 0) as trade_count_7d,
	COALESCE(t2.highest_price_7d, 0.0) as highest_price_7d,
	COALESCE(t2.lowest_price_7d, 0.0) as lowest_price_7d,
	COALESCE(t5.price_7d_ago - last_price, 0.0) as price_change_7d,
	COALESCE(t5.price_7d_ago, 0.0) as open_price_7d,

	COALESCE(t3.last_price, 0.0) as last_price,
	COALESCE(t3.last_close_time, now()) as close_time

FROM (
	-- All trades between valid assets in the last 24h aggregated:
	SELECT
		concat(bAsset.code, '_', cAsset.code) as trade_pair_name,
		sum(t.base_amount) as base_volume_24h,
		sum(t.counter_amount) as counter_volume_24h,
		max(t.price) as highest_price_24h,
		min(t.price) as lowest_price_24h,
		count(t.base_amount) as trade_count_24h
	FROM trades as t
		JOIN assets as bAsset
		ON t.base_asset_id = bAsset.id
		JOIN assets as cAsset
		ON t.counter_asset_id = cAsset.id
	WHERE bAsset.is_valid = TRUE
		AND cAsset.is_valid = TRUE
		AND t.ledger_close_time > now() - interval '1 day'
	GROUP BY trade_pair_name
) t1
FULL JOIN (
	-- All trades between valid assets in the last 7d aggregated:
	SELECT
		concat(bAsset.code, '_', cAsset.code) as trade_pair_name,
		sum(t.base_amount) as base_volume_7d,
		sum(t.counter_amount) as counter_volume_7d,
		max(t.price) as highest_price_7d,
		min(t.price) as lowest_price_7d,
		count(t.base_amount) as trade_count_7d
	FROM trades as t
		JOIN assets as bAsset
		ON t.base_asset_id = bAsset.id
		JOIN assets as cAsset
		ON t.counter_asset_id = cAsset.id
	WHERE bAsset.is_valid = TRUE
		AND cAsset.is_valid = TRUE
		AND t.ledger_close_time > now() - interval '7 days'
	GROUP BY trade_pair_name

) t2 ON t1.trade_pair_name = t2.trade_pair_name
INNER JOIN (
	-- Last prices and close times:
	SELECT DISTINCT ON (trade_pair_name)
		concat(bAsset.code, '_', cAsset.code) as trade_pair_name,
		t.price as last_price,
		t.ledger_close_time as last_close_time
	FROM trades as t
		JOIN assets as bAsset
		ON t.base_asset_id = bAsset.id
		JOIN assets as cAsset
		ON t.counter_asset_id = cAsset.id
	WHERE bAsset.is_valid = TRUE
		AND cAsset.is_valid = TRUE
		AND t.ledger_close_time > now() - interval '7 days'
	ORDER BY trade_pair_name, t.ledger_close_time DESC
) t3 ON t2.trade_pair_name = t3.trade_pair_name
LEFT JOIN (
	-- Price 24h ago:
	SELECT DISTINCT ON (trade_pair_name)
		concat(bAsset.code, '_', cAsset.code) as trade_pair_name,
		t.price as price_24h_ago
	FROM trades as t
		JOIN assets as bAsset
		ON t.base_asset_id = bAsset.id
		JOIN assets as cAsset
		ON t.counter_asset_id = cAsset.id
	WHERE bAsset.is_valid = TRUE
		AND cAsset.is_valid = TRUE
		AND t.ledger_close_time > now() - interval '1 days'
	ORDER BY trade_pair_name, t.ledger_close_time ASC
) t4 ON t3.trade_pair_name = t4.trade_pair_name
LEFT JOIN (
	-- Price 7d ago:
	SELECT DISTINCT ON (trade_pair_name)
		concat(bAsset.code, '_', cAsset.code) as trade_pair_name,
		t.price as price_7d_ago
	FROM trades as t
		JOIN assets as bAsset
		ON t.base_asset_id = bAsset.id
		JOIN assets as cAsset
		ON t.counter_asset_id = cAsset.id
	WHERE bAsset.is_valid = TRUE
		AND cAsset.is_valid = TRUE
		AND t.ledger_close_time > now() - interval '7 days'
	ORDER BY trade_pair_name, t.ledger_close_time ASC

) t5 ON t4.trade_pair_name = t5.trade_pair_name
ORDER BY trade_pair_name;
`

var partialMarketQuery = `
SELECT
	concat(bAsset.code, ':', bAsset.issuer_account, ' / ', cAsset.code, ':', cAsset.issuer_account) as trade_pair_name,
	bAsset.code AS base_asset_code,
	bAsset.issuer_account as base_asset_issuer,
	cAsset.code AS counter_asset_code,
	cAsset.issuer_account AS counter_asset_issuer,
	sum(t.base_amount) AS base_volume,
	sum(t.counter_amount) AS counter_volume,
	count(t.base_amount) AS trade_count,
	max(t.price) AS highest_price,
	min(t.price) AS lowest_price,
	(array_agg(t.price ORDER BY t.ledger_close_time ASC))[1] AS open_price,
	(array_agg(t.price ORDER BY t.ledger_close_time DESC))[1] AS last_price,
	((array_agg(t.price ORDER BY t.ledger_close_time DESC))[1] - (array_agg(t.price ORDER BY t.ledger_close_time ASC))[1]) AS price_change,
	(now() - interval '__NUMHOURS__ hours') AS interval_start,
	min(t.ledger_close_time) AS first_ledger_close_time
FROM trades AS t
	JOIN assets AS bAsset ON t.base_asset_id = bAsset.id
	JOIN assets AS cAsset on t.counter_asset_id = cAsset.id
__WHERECLAUSE__
GROUP BY bAsset.code, bAsset.issuer_account, cAsset.code, cAsset.issuer_account;
`

var aggMarketQuery = `
SELECT
	concat(bAsset.code, '_', cAsset.code) as trade_pair_name,
	sum(t.base_amount) AS base_volume,
	sum(t.counter_amount) AS counter_volume,
	count(t.base_amount) AS trade_count,
	max(t.price) AS highest_price,
	min(t.price) AS lowest_price,
	(array_agg(t.price ORDER BY t.ledger_close_time ASC))[1] AS open_price,
	(array_agg(t.price ORDER BY t.ledger_close_time DESC))[1] AS last_price,
	((array_agg(t.price ORDER BY t.ledger_close_time DESC))[1] - (array_agg(t.price ORDER BY t.ledger_close_time ASC))[1]) AS price_change,
	(now() - interval '__NUMHOURS__ hours') AS interval_start,
	min(t.ledger_close_time) AS first_ledger_close_time
FROM trades AS t
	JOIN assets AS bAsset ON t.base_asset_id = bAsset.id
	JOIN assets AS cAsset on t.counter_asset_id = cAsset.id
__WHERECLAUSE__
GROUP BY trade_pair_name;`
