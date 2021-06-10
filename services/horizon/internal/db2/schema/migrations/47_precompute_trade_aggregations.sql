-- +migrate Up

-- Create the new table
--
-- Note: We use a wide jsonb column here, to save disk space. If we stored,
-- timestamp/base_asset_id/counter_asset_id/...values..., then the timestamp,
-- base_asset_id, and counter_asset_id would be duplicated to each row. In
-- experiments (2020-05-26) this saves ~60% of disk space. The tradeoff is that
-- the queries are slightly slower (~15%), and more complex as they have to use
-- the jsonb_each postgres function to get each minute-bucket. If you are
-- changing the contents of the value, make sure it doesn't overflow the
-- maximum column width, as each "values" field can have 1 minute * 31 days
-- elements. The 'values' field has the format:
-- { "1622211420000": {
--     "timestamp": 1622211420000,
--     "count": 1,
--     "base_volume": 123,
--     "counter_volume": 456,
--     "avg": 18.123,
--     "high_n": 200,
--     "high_d": 11,
--     "low_n": 200,
--     "low_d": 11,
--     "open_n": 200,
--     "open_d": 11,
--     "close_n": 200,
--     "close_d": 11
--   },
--   ...
-- }
CREATE TABLE public.history_trades_60000 (
  year integer not null,
  month integer not null,
  base_asset_id bigint not null,
  counter_asset_id bigint not null,
  values jsonb not null,

  UNIQUE(year, month, base_asset_id, counter_asset_id)
);

-- Add the trigger to keep it up to date
-- TODO: This should probably handle updates, not just inserts.
-- TODO: This shouldn't assume we are always inserting in order.
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION update_history_trades_compute_1m()
  RETURNS trigger AS $$
  DECLARE
    timestamp bigint;
    key text;
  BEGIN
    timestamp = div(cast((extract(epoch from NEW.ledger_closed_at) * 1000 ) as bigint), 60000)*60000;
    key = cast(timestamp as text);
    -- make sure we have the row. Means we can just update later. simpler..
    INSERT INTO public.history_trades_60000
      (year, month, base_asset_id, counter_asset_id, values)
      VALUES (
        extract(year from new.ledger_closed_at),
        extract(month from new.ledger_closed_at),
        new.base_asset_id,
        new.counter_asset_id,
        '{}'::jsonb
      )
      ON CONFLICT (year, month, base_asset_id, counter_asset_id) DO NOTHING;

    -- incremental update the minute bucket, and merge the new result into values.
    UPDATE public.history_trades_60000 h
        SET values = values || jsonb_build_object(
            key,
            jsonb_build_object(
                'timestamp', timestamp,
                'count', coalesce((h.values->key->>'count')::integer, 0)+1,
                'base_volume', coalesce((h.values->key->>'base_volume')::bigint, 0)+new.base_amount,
                'counter_volume', coalesce((h.values->key->>'counter_volume')::bigint, 0)+new.counter_amount,
                'avg', (coalesce((h.values->key->>'counter_volume')::numeric, 0)+new.counter_amount)/(coalesce((h.values->key->>'base_volume')::numeric, 0)+new.base_amount),
                'high_n', (public.max_price_agg(ARRAY[(h.values->key->>'high_n')::numeric, (h.values->key->>'high_d')::numeric], ARRAY[new.price_n::numeric, new.price_d::numeric]))[1],
                'high_d', (public.max_price_agg(ARRAY[(h.values->key->>'high_n')::numeric, (h.values->key->>'high_d')::numeric], ARRAY[new.price_n::numeric, new.price_d::numeric]))[2],
                'low_n', (public.min_price_agg(ARRAY[(h.values->key->>'low_n')::numeric, (h.values->key->>'low_d')::numeric], ARRAY[new.price_n::numeric, new.price_d::numeric]))[1],
                'low_d', (public.min_price_agg(ARRAY[(h.values->key->>'low_n')::numeric, (h.values->key->>'low_d')::numeric], ARRAY[new.price_n::numeric, new.price_d::numeric]))[2]
            ) || (
              CASE
                WHEN (h.values->key->>'open_ledger_seq')::bigint < new.history_operation_id THEN
                  jsonb_build_object(
                    'open_ledger_seq', (h.values->key->>'open_ledger_seq')::bigint,
                    'open_n', (h.values->key->>'open_n')::bigint,
                    'open_d', (h.values->key->>'open_d')::bigint
                  )
                ELSE
                  jsonb_build_object(
                    'open_ledger_seq', new.history_operation_id,
                    'open_n', new.price_n,
                    'open_d', new.price_d
                  )
              END
            ) || (
              CASE
                WHEN (h.values->key->>'close_ledger_seq')::bigint > new.history_operation_id THEN
                  jsonb_build_object(
                    'close_ledger_seq', (h.values->key->>'close_ledger_seq')::bigint,
                    'close_n', (h.values->key->>'close_n')::bigint,
                    'close_d', (h.values->key->>'close_d')::bigint
                  )
                ELSE
                  jsonb_build_object(
                    'close_ledger_seq', new.history_operation_id,
                    'close_n', new.price_n,
                    'close_d', new.price_d
                  )
              END
            )
        )
        WHERE h.year = extract(year from new.ledger_closed_at)
          AND h.month = extract(month from new.ledger_closed_at)
          AND h.base_asset_id = new.base_asset_id
          AND h.counter_asset_id = new.counter_asset_id;

    RETURN NULL;
  END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- Wire up the trigger on inserts.
CREATE TRIGGER htrd_compute_1m
  AFTER INSERT ON history_trades
  FOR EACH ROW
  EXECUTE FUNCTION update_history_trades_compute_1m();

-- Backfill the table with existing data. This takes like ~10-15 minutes,
-- maybe.
-- TODO: Confirm how long this takes.
WITH htrd AS (
  SELECT
    div((cast((extract(epoch from h.ledger_closed_at) * 1000 ) as bigint) - 0), 60000)*60000 + 0 as timestamp,
    h.history_operation_id,
    h."order",
    h.base_asset_id,
    h.base_amount,
    h.counter_asset_id,
    h.counter_amount,
    ARRAY[h.price_n, h.price_d] as price
  FROM history_trades h
  ORDER BY h.history_operation_id, h."order"
), buckets AS (
  SELECT
    base_asset_id,
    counter_asset_id,
    timestamp,
    count(*) as count,
    sum(base_amount) as base_volume,
    sum(counter_amount) as counter_volume,
    sum(counter_amount)/sum(base_amount) as avg,
    (max_price(price))[1] as high_n,
    (max_price(price))[2] as high_d,
    (min_price(price))[1] as low_n,
    (min_price(price))[2] as low_d,
    first(history_operation_id) as open_ledger_seq,
    (first(price))[1] as open_n,
    (first(price))[2] as open_d,
    last(history_operation_id) as close_ledger_seq,
    (last(price))[1] as close_n,
    (last(price))[2] as close_d
  FROM htrd
  GROUP by base_asset_id, counter_asset_id, timestamp
  ORDER BY timestamp
) INSERT INTO public.history_trades_60000
  SELECT
    extract(year from to_timestamp(timestamp/1000)) as year,
    extract(month from to_timestamp(timestamp/1000)) as month,
    base_asset_id,
    counter_asset_id,
    jsonb_object_agg(
      timestamp,
      jsonb_build_object(
        'timestamp', timestamp,
        'count', count,
        'base_volume', base_volume,
        'counter_volume', counter_volume,
        'avg', avg,
        'high_n', high_n,
        'high_d', high_d,
        'low_n', low_n,
        'low_d', low_d,
        'open_ledger_seq', open_ledger_seq,
        'open_n', open_n,
        'open_d', open_d,
        'close_ledger_seq', close_ledger_seq,
        'close_n', close_n,
        'close_d', close_d
      )
    ) as values
  FROM buckets
  GROUP BY year, month, base_asset_id, counter_asset_id
  ORDER BY year, month
  ON CONFLICT (year, month, base_asset_id, counter_asset_id) DO UPDATE SET values = public.history_trades_60000.values || EXCLUDED.values;


-- +migrate Down

DROP TRIGGER htrd_compute_1m on history_trades;
DROP FUNCTION update_history_trades_compute_1m;
DROP TABLE history_trades_60000;
