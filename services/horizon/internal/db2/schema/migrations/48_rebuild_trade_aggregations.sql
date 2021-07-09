-- +migrate Up

-- Backfill the table with existing data. This takes about 9 minutes.
WITH htrd AS (
  SELECT
    to_millis(h.ledger_closed_at, 60000) as timestamp,
    h.history_operation_id,
    h."order",
    h.base_asset_id,
    h.base_amount,
    h.counter_asset_id,
    h.counter_amount,
    ARRAY[h.price_n, h.price_d] as price
  FROM history_trades h
  ORDER BY h.history_operation_id, h."order"
)
  INSERT INTO history_trades_60000
    (
      SELECT
        timestamp,
        base_asset_id,
        counter_asset_id,
        count(*) as count,
        sum(base_amount) as base_volume,
        sum(counter_amount) as counter_volume,
        sum(counter_amount::numeric)/sum(base_amount::numeric) as avg,
        (max_price(price))[1] as high_n,
        (max_price(price))[2] as high_d,
        (min_price(price))[1] as low_n,
        (min_price(price))[2] as low_d,
        first(history_operation_id) as open_ledger_toid,
        (first(price))[1] as open_n,
        (first(price))[2] as open_d,
        last(history_operation_id) as close_ledger_toid,
        (last(price))[1] as close_n,
        (last(price))[2] as close_d
      FROM htrd
      GROUP by base_asset_id, counter_asset_id, timestamp
      ORDER BY timestamp
    );


-- +migrate Down
