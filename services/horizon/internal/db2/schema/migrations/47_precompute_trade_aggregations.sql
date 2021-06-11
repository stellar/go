-- +migrate Up

-- Create the new table
CREATE TABLE public.history_trades_60000 (
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
  open_ledger_seq bigint not null,
  open_n numeric not null,
  open_d numeric not null,
  close_ledger_seq bigint not null,
  close_n numeric not null,
  close_d numeric not null,

  PRIMARY KEY(base_asset_id, counter_asset_id, timestamp)
);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION public.to_millis(t timestamp without time zone, trun numeric DEFAULT 1)
  RETURNS bigint AS $$
  BEGIN
    RETURN div(cast((extract(epoch from t) * 1000 ) as bigint), trun)*trun;
  END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION public.to_millis(t timestamp with time zone, trun numeric DEFAULT 1)
  RETURNS bigint AS $$
  BEGIN
    RETURN public.to_millis(t::timestamp, trun);
  END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd


-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION public.history_trades_60000_rebuild_buckets(base bigint, counter bigint, start_time timestamp without time zone, end_time timestamp without time zone)
  RETURNS boolean AS $$
  DECLARE
    tstart bigint := public.to_millis(start_time, 60000);
    tend bigint := public.to_millis(end_time, 60000)+60000;
  BEGIN
    DELETE FROM public.history_trades_60000
      WHERE base_asset_id = base
        AND counter_asset_id = counter
        AND timestamp >= tstart
        AND timestamp < tend;
    WITH htrd AS (
      SELECT
        public.to_millis(ledger_closed_at, 60000) as timestamp,
        history_operation_id,
        "order",
        base_asset_id,
        base_amount,
        counter_asset_id,
        counter_amount,
        ARRAY[price_n, price_d] as price
      FROM history_trades
      WHERE base_asset_id = base
        AND counter_asset_id = counter
      ORDER BY history_operation_id , "order"
    )
    INSERT INTO public.history_trades_60000
      (
        SELECT
          timestamp,
          base_asset_id,
          counter_asset_id,
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
        WHERE timestamp >= tstart
          AND timestamp < tend
        GROUP BY timestamp
      );
    RETURN true;
  END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION public.history_trades_60000_rebuild_buckets(base bigint, counter bigint, start_time timestamp with time zone, end_time timestamp with time zone)
  RETURNS boolean AS $$
  BEGIN
    RETURN public.history_trades_60000_rebuild_buckets(base, counter, start_time::timestamp, end_time::timestamp);
  END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION public.history_trades_60000_rebuild_buckets(base bigint, counter bigint, t timestamp with time zone)
  RETURNS boolean AS $$
  BEGIN
    RETURN public.history_trades_60000_rebuild_buckets(base, counter, t::timestamp, t::timestamp);
  END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION public.history_trades_60000_rebuild_buckets(base bigint, counter bigint, t timestamp without time zone)
  RETURNS boolean AS $$
  BEGIN
    RETURN public.history_trades_60000_rebuild_buckets(base, counter, t, t);
  END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd


-- Add the trigger to keep it up to date
-- TODO: This should probably handle updates, not just inserts.
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION public.history_trades_60000_insert()
  RETURNS trigger AS $$
  BEGIN
    -- make sure we have the row. Means we can just update later. simpler..
    INSERT INTO public.history_trades_60000 as h
      (timestamp, base_asset_id, counter_asset_id, count, base_volume, counter_volume, avg, high_n, high_d, low_n, low_d, open_ledger_seq, open_n, open_d, close_ledger_seq, close_n, close_d)
      VALUES (
        public.to_millis(new.ledger_closed_at, 60000),
        new.base_asset_id,
        new.counter_asset_id,
        1,
        new.base_amount,
        new.counter_amount,
        -- TODO: Is this the right thing when base_amount is 0?
        CASE WHEN new.base_amount = 0 THEN 0 ELSE (new.counter_amount/new.base_amount) END,
        new.price_n,
        new.price_d,
        new.price_n,
        new.price_d,
        new.history_operation_id,
        new.price_n,
        new.price_d,
        new.history_operation_id,
        new.price_n,
        new.price_d
      )
      ON CONFLICT (base_asset_id, counter_asset_id, timestamp)
        DO UPDATE SET
          "count" = h."count"+1,
          base_volume = h.base_volume+excluded.base_volume,
          counter_volume = h.counter_volume+excluded.counter_volume,
          "avg" = (h.counter_volume+excluded.counter_volume)/(h.base_volume+excluded.base_volume),
          high_n = (public.max_price_agg(ARRAY[h.high_n, h.high_d], ARRAY[excluded.high_n::numeric, excluded.high_d::numeric]))[1],
          high_d = (public.max_price_agg(ARRAY[h.high_n, h.high_d], ARRAY[excluded.high_n::numeric, excluded.high_d::numeric]))[2],
          low_n = (public.min_price_agg(ARRAY[h.low_n, h.low_d], ARRAY[excluded.low_n::numeric, excluded.low_d::numeric]))[1],
          low_d = (public.min_price_agg(ARRAY[h.low_n, h.low_d], ARRAY[excluded.low_n::numeric, excluded.low_d::numeric]))[2],
          open_ledger_seq = CASE WHEN h.open_ledger_seq < excluded.open_ledger_seq THEN h.open_ledger_seq ELSE excluded.open_ledger_seq END,
          open_n = CASE WHEN h.open_ledger_seq < excluded.open_ledger_seq THEN h.open_n ELSE excluded.open_n END,
          open_d = CASE WHEN h.open_ledger_seq < excluded.open_ledger_seq THEN h.open_d ELSE excluded.open_d END,
          close_ledger_seq = CASE WHEN h.close_ledger_seq > excluded.close_ledger_seq THEN h.close_ledger_seq ELSE excluded.close_ledger_seq END,
          close_n = CASE WHEN h.close_ledger_seq > excluded.close_ledger_seq THEN h.close_n ELSE excluded.close_n END,
          close_d = CASE WHEN h.close_ledger_seq > excluded.close_ledger_seq THEN h.close_d ELSE excluded.close_d END;

    RETURN NULL;
  END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION public.history_trades_60000_truncate()
  RETURNS trigger AS $$
  BEGIN
    TRUNCATE TABLE public.history_trades_60000;
  END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- Wire up the trigger on inserts.
CREATE TRIGGER htrd_60000_insert
  AFTER INSERT ON history_trades
  FOR EACH ROW
  EXECUTE FUNCTION public.history_trades_60000_insert();

CREATE TRIGGER htrd_60000_truncate
  AFTER TRUNCATE ON history_trades
  FOR EACH STATEMENT
  EXECUTE FUNCTION public.history_trades_60000_truncate();

-- Backfill the table with existing data. This takes about 4 minutes.
WITH htrd AS (
  SELECT
    public.to_millis(h.ledger_closed_at, 60000) as timestamp,
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
  INSERT INTO public.history_trades_60000
    (
      SELECT
        timestamp,
        base_asset_id,
        counter_asset_id,
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
    );


-- +migrate Down

DROP TRIGGER htrd_60000_insert on history_trades;
DROP FUNCTION history_trades_60000_insert;
DROP TRIGGER htrd_60000_truncate on history_trades;
DROP FUNCTION history_trades_60000_truncate;
DROP FUNCTION history_trades_60000_rebuild_buckets(bigint, bigint, timestamp without time zone, timestamp without time zone);
DROP FUNCTION history_trades_60000_rebuild_buckets(bigint, bigint, timestamp with time zone, timestamp with time zone);
DROP TABLE history_trades_60000;
