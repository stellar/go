-- +migrate Up notransaction

-- Create the new table
CREATE TABLE history_trades_60000 (
  timestamp bigint not null,
  base_asset_id bigint not null,
  counter_asset_id bigint not null,
  count integer not null,
  base_volume numeric not null,
  counter_volume numeric not null,
  avg numeric not null,
  high_n numeric not null,
  high_d numeric not null,
  low_n numeric not null,
  low_d numeric not null,
  open_ledger bigint not null,
  open_n numeric not null,
  open_d numeric not null,
  close_ledger bigint not null,
  close_n numeric not null,
  close_d numeric not null,

  PRIMARY KEY(base_asset_id, counter_asset_id, timestamp)
);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION to_millis(t timestamp without time zone, trun numeric DEFAULT 1)
  RETURNS bigint AS $$
  BEGIN
    RETURN div(cast((extract(epoch from t) * 1000 ) as bigint), trun)*trun;
  END;
$$ LANGUAGE plpgsql IMMUTABLE;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION to_millis(t timestamp with time zone, trun numeric DEFAULT 1)
  RETURNS bigint AS $$
  BEGIN
    RETURN to_millis(t::timestamp, trun);
  END;
$$ LANGUAGE plpgsql IMMUTABLE;
-- +migrate StatementEnd

CREATE INDEX CONCURRENTLY htrd_agg_bucket_lookup ON history_trades
  USING btree (to_millis(ledger_closed_at, '60000'::numeric));

CREATE INDEX CONCURRENTLY htrd_agg_open_ledger ON history_trades_60000 USING btree (open_ledger);

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
        first(history_operation_id) as open_ledger,
        (first(price))[1] as open_n,
        (first(price))[2] as open_d,
        last(history_operation_id) as close_ledger,
        (last(price))[1] as close_n,
        (last(price))[2] as close_d
      FROM htrd
      GROUP by base_asset_id, counter_asset_id, timestamp
      ORDER BY timestamp
    );


-- +migrate Down

DROP INDEX htrd_agg_open_ledger;
DROP INDEX htrd_agg_bucket_lookup;
DROP TABLE history_trades_60000;
DROP FUNCTION to_millis(timestamp with time zone, numeric);
DROP FUNCTION to_millis(timestamp without time zone, numeric);
