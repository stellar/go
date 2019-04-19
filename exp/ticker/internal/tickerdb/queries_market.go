package tickerdb

func (s *TickerSession) RetrieveMarketData() (markets []Market, err error) {
	err = s.SelectRaw(&markets, marketQuery)
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
