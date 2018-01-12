--
-- PostgreSQL database dump
--

<<<<<<< HEAD
-- Dumped from database version 9.6.3
-- Dumped by pg_dump version 9.6.3
=======
-- Dumped from database version 10.1
-- Dumped by pg_dump version 10.1
>>>>>>> add price to trade ingestion

SET statement_timeout = 0;
SET lock_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;

SET search_path = public, pg_catalog;

ALTER TABLE IF EXISTS ONLY public.history_trades DROP CONSTRAINT IF EXISTS history_trades_counter_asset_id_fkey;
ALTER TABLE IF EXISTS ONLY public.history_trades DROP CONSTRAINT IF EXISTS history_trades_counter_account_id_fkey;
ALTER TABLE IF EXISTS ONLY public.history_trades DROP CONSTRAINT IF EXISTS history_trades_base_asset_id_fkey;
ALTER TABLE IF EXISTS ONLY public.history_trades DROP CONSTRAINT IF EXISTS history_trades_base_account_id_fkey;
ALTER TABLE IF EXISTS ONLY public.asset_stats DROP CONSTRAINT IF EXISTS asset_stats_id_fkey;
DROP INDEX IF EXISTS public.trade_effects_by_order_book;
DROP INDEX IF EXISTS public.index_history_transactions_on_id;
DROP INDEX IF EXISTS public.index_history_operations_on_type;
DROP INDEX IF EXISTS public.index_history_operations_on_transaction_id;
DROP INDEX IF EXISTS public.index_history_operations_on_id;
DROP INDEX IF EXISTS public.index_history_ledgers_on_sequence;
DROP INDEX IF EXISTS public.index_history_ledgers_on_previous_ledger_hash;
DROP INDEX IF EXISTS public.index_history_ledgers_on_ledger_hash;
DROP INDEX IF EXISTS public.index_history_ledgers_on_importer_version;
DROP INDEX IF EXISTS public.index_history_ledgers_on_id;
DROP INDEX IF EXISTS public.index_history_ledgers_on_closed_at;
DROP INDEX IF EXISTS public.index_history_effects_on_type;
DROP INDEX IF EXISTS public.index_history_accounts_on_id;
DROP INDEX IF EXISTS public.index_history_accounts_on_address;
DROP INDEX IF EXISTS public.htrd_time_lookup;
DROP INDEX IF EXISTS public.htrd_pid;
DROP INDEX IF EXISTS public.htrd_pair_time_lookup;
DROP INDEX IF EXISTS public.htrd_counter_lookup;
DROP INDEX IF EXISTS public.htrd_by_offer;
DROP INDEX IF EXISTS public.htp_by_htid;
DROP INDEX IF EXISTS public.hs_transaction_by_id;
DROP INDEX IF EXISTS public.hs_ledger_by_id;
DROP INDEX IF EXISTS public.hop_by_hoid;
DROP INDEX IF EXISTS public.hist_tx_p_id;
DROP INDEX IF EXISTS public.hist_op_p_id;
DROP INDEX IF EXISTS public.hist_e_id;
DROP INDEX IF EXISTS public.hist_e_by_order;
DROP INDEX IF EXISTS public.by_ledger;
DROP INDEX IF EXISTS public.by_hash;
DROP INDEX IF EXISTS public.by_account;
DROP INDEX IF EXISTS public.asset_by_issuer;
DROP INDEX IF EXISTS public.asset_by_code;
ALTER TABLE IF EXISTS ONLY public.history_transaction_participants DROP CONSTRAINT IF EXISTS history_transaction_participants_pkey;
ALTER TABLE IF EXISTS ONLY public.history_operation_participants DROP CONSTRAINT IF EXISTS history_operation_participants_pkey;
ALTER TABLE IF EXISTS ONLY public.history_assets DROP CONSTRAINT IF EXISTS history_assets_pkey;
ALTER TABLE IF EXISTS ONLY public.history_assets DROP CONSTRAINT IF EXISTS history_assets_asset_code_asset_type_asset_issuer_key;
ALTER TABLE IF EXISTS ONLY public.gorp_migrations DROP CONSTRAINT IF EXISTS gorp_migrations_pkey;
ALTER TABLE IF EXISTS ONLY public.asset_stats DROP CONSTRAINT IF EXISTS asset_stats_pkey;
ALTER TABLE IF EXISTS public.history_transaction_participants ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS public.history_operation_participants ALTER COLUMN id DROP DEFAULT;
ALTER TABLE IF EXISTS public.history_assets ALTER COLUMN id DROP DEFAULT;
DROP TABLE IF EXISTS public.history_transactions;
DROP SEQUENCE IF EXISTS public.history_transaction_participants_id_seq;
DROP TABLE IF EXISTS public.history_transaction_participants;
DROP TABLE IF EXISTS public.history_trades;
DROP TABLE IF EXISTS public.history_operations;
DROP SEQUENCE IF EXISTS public.history_operation_participants_id_seq;
DROP TABLE IF EXISTS public.history_operation_participants;
DROP TABLE IF EXISTS public.history_ledgers;
DROP TABLE IF EXISTS public.history_effects;
DROP SEQUENCE IF EXISTS public.history_assets_id_seq;
DROP TABLE IF EXISTS public.history_assets;
DROP TABLE IF EXISTS public.history_accounts;
DROP SEQUENCE IF EXISTS public.history_accounts_id_seq;
DROP TABLE IF EXISTS public.gorp_migrations;
DROP TABLE IF EXISTS public.asset_stats;
DROP AGGREGATE IF EXISTS public.min_price(numeric[]);
DROP AGGREGATE IF EXISTS public.max_price(numeric[]);
DROP AGGREGATE IF EXISTS public.last(anyelement);
DROP AGGREGATE IF EXISTS public.first(anyelement);
DROP FUNCTION IF EXISTS public.min_price_agg(numeric[], numeric[]);
DROP FUNCTION IF EXISTS public.max_price_agg(numeric[], numeric[]);
DROP FUNCTION IF EXISTS public.last_agg(anyelement, anyelement);
DROP FUNCTION IF EXISTS public.first_agg(anyelement, anyelement);
DROP EXTENSION IF EXISTS plpgsql;
DROP SCHEMA IF EXISTS public;
--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA public;


--
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET search_path = public, pg_catalog;

--
-- Name: first_agg(anyelement, anyelement); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION first_agg(anyelement, anyelement) RETURNS anyelement
    LANGUAGE sql IMMUTABLE STRICT
    AS $_$ SELECT $1 $_$;


--
-- Name: last_agg(anyelement, anyelement); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION last_agg(anyelement, anyelement) RETURNS anyelement
    LANGUAGE sql IMMUTABLE STRICT
    AS $_$ SELECT $2 $_$;


--
-- Name: max_price_agg(numeric[], numeric[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION max_price_agg(numeric[], numeric[]) RETURNS numeric[]
    LANGUAGE sql IMMUTABLE STRICT
    AS $_$ SELECT (
  CASE WHEN $1[1]/$1[2]>$2[1]/$2[2] THEN $1 ELSE $2 END) $_$;


--
-- Name: min_price_agg(numeric[], numeric[]); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION min_price_agg(numeric[], numeric[]) RETURNS numeric[]
    LANGUAGE sql IMMUTABLE STRICT
    AS $_$ SELECT (
  CASE WHEN $1[1]/$1[2]<$2[1]/$2[2] THEN $1 ELSE $2 END) $_$;


--
-- Name: first(anyelement); Type: AGGREGATE; Schema: public; Owner: -
--

CREATE AGGREGATE first(anyelement) (
    SFUNC = first_agg,
    STYPE = anyelement
);


--
-- Name: last(anyelement); Type: AGGREGATE; Schema: public; Owner: -
--

CREATE AGGREGATE last(anyelement) (
    SFUNC = last_agg,
    STYPE = anyelement
);


--
-- Name: max_price(numeric[]); Type: AGGREGATE; Schema: public; Owner: -
--

CREATE AGGREGATE max_price(numeric[]) (
    SFUNC = max_price_agg,
    STYPE = numeric[]
);


--
-- Name: min_price(numeric[]); Type: AGGREGATE; Schema: public; Owner: -
--

CREATE AGGREGATE min_price(numeric[]) (
    SFUNC = min_price_agg,
    STYPE = numeric[]
);


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: asset_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE asset_stats (
    id bigint NOT NULL,
    amount bigint NOT NULL,
    num_accounts integer NOT NULL,
    flags smallint NOT NULL,
    toml character varying(64) NOT NULL
);


--
-- Name: gorp_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE gorp_migrations (
    id text NOT NULL,
    applied_at timestamp with time zone
);


--
-- Name: history_accounts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE history_accounts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: history_accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_accounts (
    id bigint DEFAULT nextval('history_accounts_id_seq'::regclass) NOT NULL,
    address character varying(64)
);


--
-- Name: history_assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_assets (
    id integer NOT NULL,
    asset_type character varying(64) NOT NULL,
    asset_code character varying(12) NOT NULL,
    asset_issuer character varying(56) NOT NULL
);


--
-- Name: history_assets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE history_assets_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: history_assets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE history_assets_id_seq OWNED BY history_assets.id;


--
-- Name: history_effects; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_effects (
    history_account_id bigint NOT NULL,
    history_operation_id bigint NOT NULL,
    "order" integer NOT NULL,
    type integer NOT NULL,
    details jsonb
);


--
-- Name: history_ledgers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_ledgers (
    sequence integer NOT NULL,
    ledger_hash character varying(64) NOT NULL,
    previous_ledger_hash character varying(64),
    transaction_count integer DEFAULT 0 NOT NULL,
    operation_count integer DEFAULT 0 NOT NULL,
    closed_at timestamp without time zone NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    id bigint,
    importer_version integer DEFAULT 1 NOT NULL,
    total_coins bigint NOT NULL,
    fee_pool bigint NOT NULL,
    base_fee integer NOT NULL,
    base_reserve integer NOT NULL,
    max_tx_set_size integer NOT NULL,
    protocol_version integer DEFAULT 0 NOT NULL,
    ledger_header text
);


--
-- Name: history_operation_participants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_operation_participants (
    id integer NOT NULL,
    history_operation_id bigint NOT NULL,
    history_account_id bigint NOT NULL
);


--
-- Name: history_operation_participants_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE history_operation_participants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: history_operation_participants_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE history_operation_participants_id_seq OWNED BY history_operation_participants.id;


--
-- Name: history_operations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_operations (
    id bigint NOT NULL,
    transaction_id bigint NOT NULL,
    application_order integer NOT NULL,
    type integer NOT NULL,
    details jsonb,
    source_account character varying(64) DEFAULT ''::character varying NOT NULL
);


--
-- Name: history_trades; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_trades (
    history_operation_id bigint NOT NULL,
    "order" integer NOT NULL,
    ledger_closed_at timestamp without time zone NOT NULL,
    offer_id bigint NOT NULL,
    base_account_id bigint NOT NULL,
    base_asset_id bigint NOT NULL,
    base_amount bigint NOT NULL,
    counter_account_id bigint NOT NULL,
    counter_asset_id bigint NOT NULL,
    counter_amount bigint NOT NULL,
    base_is_seller boolean,
    price_n bigint,
    price_d bigint,
    CONSTRAINT history_trades_base_amount_check CHECK ((base_amount > 0)),
    CONSTRAINT history_trades_check CHECK ((base_asset_id < counter_asset_id)),
    CONSTRAINT history_trades_counter_amount_check CHECK ((counter_amount > 0))
);


--
-- Name: history_transaction_participants; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_transaction_participants (
    id integer NOT NULL,
    history_transaction_id bigint NOT NULL,
    history_account_id bigint NOT NULL
);


--
-- Name: history_transaction_participants_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE history_transaction_participants_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: history_transaction_participants_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE history_transaction_participants_id_seq OWNED BY history_transaction_participants.id;


--
-- Name: history_transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE history_transactions (
    transaction_hash character varying(64) NOT NULL,
    ledger_sequence integer NOT NULL,
    application_order integer NOT NULL,
    account character varying(64) NOT NULL,
    account_sequence bigint NOT NULL,
    fee_paid integer NOT NULL,
    operation_count integer NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    id bigint,
    tx_envelope text NOT NULL,
    tx_result text NOT NULL,
    tx_meta text NOT NULL,
    tx_fee_meta text NOT NULL,
    signatures character varying(96)[] DEFAULT '{}'::character varying[] NOT NULL,
    memo_type character varying DEFAULT 'none'::character varying NOT NULL,
    memo character varying,
    time_bounds int8range
);


--
-- Name: history_assets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_assets ALTER COLUMN id SET DEFAULT nextval('history_assets_id_seq'::regclass);


--
-- Name: history_operation_participants id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_operation_participants ALTER COLUMN id SET DEFAULT nextval('history_operation_participants_id_seq'::regclass);


--
-- Name: history_transaction_participants id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_transaction_participants ALTER COLUMN id SET DEFAULT nextval('history_transaction_participants_id_seq'::regclass);


--
-- Data for Name: asset_stats; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO asset_stats VALUES (1, 3000010434000, 2, 1, 'https://test.com/.well-known/stellar.toml');
INSERT INTO asset_stats VALUES (2, 10000000000, 1, 2, '');
INSERT INTO asset_stats VALUES (3, 1009876000, 1, 1, 'https://test.com/.well-known/stellar.toml');


--
-- Data for Name: gorp_migrations; Type: TABLE DATA; Schema: public; Owner: -
--

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO gorp_migrations VALUES ('1_initial_schema.sql', '2017-12-15 13:25:00.27835-06');
INSERT INTO gorp_migrations VALUES ('2_index_participants_by_toid.sql', '2017-12-15 13:25:00.284847-06');
INSERT INTO gorp_migrations VALUES ('3_use_sequence_in_history_accounts.sql', '2017-12-15 13:25:00.287761-06');
INSERT INTO gorp_migrations VALUES ('4_add_protocol_version.sql', '2017-12-15 13:25:00.296332-06');
INSERT INTO gorp_migrations VALUES ('5_create_trades_table.sql', '2017-12-15 13:25:00.303579-06');
INSERT INTO gorp_migrations VALUES ('6_create_assets_table.sql', '2017-12-15 13:25:00.308494-06');
INSERT INTO gorp_migrations VALUES ('7_modify_trades_table.sql', '2017-12-15 13:25:00.315896-06');
INSERT INTO gorp_migrations VALUES ('8_add_aggregators.sql', '2017-12-15 13:25:00.317593-06');
INSERT INTO gorp_migrations VALUES ('8_create_asset_stats_table.sql', '2017-12-15 13:25:00.322275-06');
INSERT INTO gorp_migrations VALUES ('9_add_header_xdr.sql', '2017-12-15 13:25:00.324723-06');
=======
INSERT INTO gorp_migrations VALUES ('1_initial_schema.sql', '2017-12-15 16:14:47.270734-08');
INSERT INTO gorp_migrations VALUES ('2_index_participants_by_toid.sql', '2017-12-15 16:14:47.283665-08');
INSERT INTO gorp_migrations VALUES ('3_use_sequence_in_history_accounts.sql', '2017-12-15 16:14:47.288912-08');
INSERT INTO gorp_migrations VALUES ('4_add_protocol_version.sql', '2017-12-15 16:14:47.328449-08');
INSERT INTO gorp_migrations VALUES ('5_create_trades_table.sql', '2017-12-15 16:14:47.350416-08');
INSERT INTO gorp_migrations VALUES ('6_create_assets_table.sql', '2017-12-15 16:14:47.36914-08');
INSERT INTO gorp_migrations VALUES ('7_modify_trades_table.sql', '2017-12-15 16:14:47.399731-08');
INSERT INTO gorp_migrations VALUES ('8_add_aggregators.sql', '2017-12-15 16:14:47.403942-08');
INSERT INTO gorp_migrations VALUES ('9_create_asset_stats_table.sql', '2017-12-15 16:14:47.415594-08');
INSERT INTO gorp_migrations VALUES ('10_add_price_to_trades.sql', '2017-12-15 16:14:47.418237-08');
>>>>>>> add price to trade ingestion
=======
INSERT INTO gorp_migrations VALUES ('1_initial_schema.sql', '2018-01-11 15:48:21.39163-08');
INSERT INTO gorp_migrations VALUES ('2_index_participants_by_toid.sql', '2018-01-11 15:48:21.404608-08');
INSERT INTO gorp_migrations VALUES ('3_use_sequence_in_history_accounts.sql', '2018-01-11 15:48:21.410406-08');
INSERT INTO gorp_migrations VALUES ('4_add_protocol_version.sql', '2018-01-11 15:48:21.446171-08');
INSERT INTO gorp_migrations VALUES ('5_create_trades_table.sql', '2018-01-11 15:48:21.471319-08');
INSERT INTO gorp_migrations VALUES ('6_create_assets_table.sql', '2018-01-11 15:48:21.490441-08');
INSERT INTO gorp_migrations VALUES ('7_modify_trades_table.sql', '2018-01-11 15:48:21.530711-08');
INSERT INTO gorp_migrations VALUES ('8_add_aggregators.sql', '2018-01-11 15:48:21.536427-08');
INSERT INTO gorp_migrations VALUES ('9_create_asset_stats_table.sql', '2018-01-11 15:48:21.55437-08');
INSERT INTO gorp_migrations VALUES ('10_add_price_to_trades.sql', '2018-01-11 15:48:21.560019-08');
>>>>>>> wip aggregation test passing
=======
INSERT INTO gorp_migrations VALUES ('1_initial_schema.sql', '2018-01-12 10:34:04.739103-08');
INSERT INTO gorp_migrations VALUES ('2_index_participants_by_toid.sql', '2018-01-12 10:34:04.752764-08');
INSERT INTO gorp_migrations VALUES ('3_use_sequence_in_history_accounts.sql', '2018-01-12 10:34:04.758694-08');
INSERT INTO gorp_migrations VALUES ('4_add_protocol_version.sql', '2018-01-12 10:34:04.794738-08');
INSERT INTO gorp_migrations VALUES ('5_create_trades_table.sql', '2018-01-12 10:34:04.821058-08');
INSERT INTO gorp_migrations VALUES ('6_create_assets_table.sql', '2018-01-12 10:34:04.840122-08');
INSERT INTO gorp_migrations VALUES ('7_modify_trades_table.sql', '2018-01-12 10:34:04.874846-08');
INSERT INTO gorp_migrations VALUES ('8_add_aggregators.sql', '2018-01-12 10:34:04.879979-08');
INSERT INTO gorp_migrations VALUES ('9_create_asset_stats_table.sql', '2018-01-12 10:34:04.893319-08');
INSERT INTO gorp_migrations VALUES ('10_add_price_to_trades.sql', '2018-01-12 10:34:04.898473-08');
>>>>>>> tests passing


--
-- Data for Name: history_accounts; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_accounts VALUES (1, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H');
INSERT INTO history_accounts VALUES (2, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_accounts VALUES (3, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_accounts VALUES (4, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2');


--
-- Data for Name: history_assets; Type: TABLE DATA; Schema: public; Owner: -
--

<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_assets VALUES (1, 'credit_alphanum4', 'USD', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
<<<<<<< HEAD
INSERT INTO history_assets VALUES (2, 'credit_alphanum4', 'BTC', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_assets VALUES (3, 'credit_alphanum4', 'SCOT', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
<<<<<<< HEAD


--
-- Name: history_assets_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('history_assets_id_seq', 3, true);
=======
>>>>>>> add price to trade ingestion
=======
INSERT INTO history_assets VALUES (1, 'credit_alphanum4', 'SCOT', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_assets VALUES (2, 'credit_alphanum4', 'USD', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_assets VALUES (3, 'credit_alphanum4', 'BTC', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
>>>>>>> add price to trade query and /trades endpoint
=======
INSERT INTO history_assets VALUES (1, 'credit_alphanum4', 'USD', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_assets VALUES (2, 'credit_alphanum4', 'BTC', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_assets VALUES (3, 'credit_alphanum4', 'SCOT', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
>>>>>>> wip aggregation test passing
=======
INSERT INTO history_assets VALUES (2, 'credit_alphanum4', 'SCOT', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_assets VALUES (3, 'credit_alphanum4', 'BTC', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
>>>>>>> tests passing


--
-- Data for Name: history_effects; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_effects VALUES (2, 8589938689, 1, 0, '{"starting_balance": "1000.0000000"}');
INSERT INTO history_effects VALUES (1, 8589938689, 2, 3, '{"amount": "1000.0000000", "asset_type": "native"}');
INSERT INTO history_effects VALUES (2, 8589938689, 3, 10, '{"weight": 1, "public_key": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 8589942785, 1, 0, '{"starting_balance": "1000.0000000"}');
INSERT INTO history_effects VALUES (1, 8589942785, 2, 3, '{"amount": "1000.0000000", "asset_type": "native"}');
INSERT INTO history_effects VALUES (3, 8589942785, 3, 10, '{"weight": 1, "public_key": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
INSERT INTO history_effects VALUES (4, 8589946881, 1, 0, '{"starting_balance": "1000.0000000"}');
INSERT INTO history_effects VALUES (1, 8589946881, 2, 3, '{"amount": "1000.0000000", "asset_type": "native"}');
INSERT INTO history_effects VALUES (4, 8589946881, 3, 10, '{"weight": 1, "public_key": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2"}');
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_effects VALUES (2, 12884905985, 1, 5, '{"home_domain": "test.com"}');
INSERT INTO history_effects VALUES (2, 12884905985, 2, 12, '{"weight": 1, "public_key": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 12884910081, 1, 6, '{"auth_revocable_flag": true}');
INSERT INTO history_effects VALUES (3, 12884910081, 2, 12, '{"weight": 1, "public_key": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
INSERT INTO history_effects VALUES (2, 12884914177, 1, 6, '{"auth_required_flag": true}');
INSERT INTO history_effects VALUES (2, 12884914177, 2, 12, '{"weight": 1, "public_key": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
=======
INSERT INTO history_effects VALUES (1, 12884905985, 1, 5, '{"home_domain": "test.com"}');
INSERT INTO history_effects VALUES (1, 12884905985, 2, 12, '{"weight": 1, "public_key": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 12884910081, 1, 6, '{"auth_revocable_flag": true}');
INSERT INTO history_effects VALUES (3, 12884910081, 2, 12, '{"weight": 1, "public_key": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
INSERT INTO history_effects VALUES (1, 12884914177, 1, 6, '{"auth_required_flag": true}');
INSERT INTO history_effects VALUES (1, 12884914177, 2, 12, '{"weight": 1, "public_key": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
>>>>>>> add price to trade ingestion
=======
INSERT INTO history_effects VALUES (2, 12884905985, 1, 5, '{"home_domain": "test.com"}');
INSERT INTO history_effects VALUES (2, 12884905985, 2, 12, '{"weight": 1, "public_key": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 12884910081, 1, 6, '{"auth_revocable_flag": true}');
INSERT INTO history_effects VALUES (3, 12884910081, 2, 12, '{"weight": 1, "public_key": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
=======
INSERT INTO history_effects VALUES (3, 12884905985, 1, 6, '{"auth_revocable_flag": true}');
INSERT INTO history_effects VALUES (3, 12884905985, 2, 12, '{"weight": 1, "public_key": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
INSERT INTO history_effects VALUES (2, 12884910081, 1, 5, '{"home_domain": "test.com"}');
INSERT INTO history_effects VALUES (2, 12884910081, 2, 12, '{"weight": 1, "public_key": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
>>>>>>> wip aggregation test passing
INSERT INTO history_effects VALUES (2, 12884914177, 1, 6, '{"auth_required_flag": true}');
INSERT INTO history_effects VALUES (2, 12884914177, 2, 12, '{"weight": 1, "public_key": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
<<<<<<< HEAD
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_effects VALUES (4, 17179873281, 1, 20, '{"limit": "922337203685.4775807", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 17179877377, 1, 20, '{"limit": "922337203685.4775807", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 17179881473, 1, 20, '{"limit": "922337203685.4775807", "asset_code": "BTC", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 17179885569, 1, 20, '{"limit": "922337203685.4775807", "asset_code": "SCOT", "asset_type": "credit_alphanum4", "asset_issuer": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
=======
INSERT INTO history_effects VALUES (3, 17179873281, 1, 20, '{"limit": "922337203685.4775807", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 17179877377, 1, 20, '{"limit": "922337203685.4775807", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 17179881473, 1, 20, '{"limit": "922337203685.4775807", "asset_code": "SCOT", "asset_type": "credit_alphanum4", "asset_issuer": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
INSERT INTO history_effects VALUES (3, 17179885569, 1, 20, '{"limit": "922337203685.4775807", "asset_code": "BTC", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
>>>>>>> tests passing
INSERT INTO history_effects VALUES (2, 21474840577, 1, 23, '{"trustor": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (2, 21474844673, 1, 23, '{"trustor": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "asset_code": "BTC", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (2, 21474848769, 1, 23, '{"trustor": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 25769807873, 1, 2, '{"amount": "100000.1200000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (2, 25769807873, 2, 3, '{"amount": "100000.1200000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 25769811969, 1, 2, '{"amount": "1000.0000000", "asset_code": "SCOT", "asset_type": "credit_alphanum4", "asset_issuer": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
INSERT INTO history_effects VALUES (3, 25769811969, 2, 3, '{"amount": "1000.0000000", "asset_code": "SCOT", "asset_type": "credit_alphanum4", "asset_issuer": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
INSERT INTO history_effects VALUES (3, 25769816065, 1, 2, '{"amount": "100.9876000", "asset_code": "BTC", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (2, 25769816065, 2, 3, '{"amount": "100.9876000", "asset_code": "BTC", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 25769820161, 1, 2, '{"amount": "200000.9234000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_effects VALUES (2, 25769820161, 2, 3, '{"amount": "200000.9234000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 30064775169, 1, 2, '{"amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 30064775169, 2, 3, '{"amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 30064779265, 1, 22, '{"limit": "1000000000.0000000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
=======
=======
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_effects VALUES (1, 25769820161, 2, 3, '{"amount": "200000.9234000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 30064775169, 1, 22, '{"limit": "1000000000.0000000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 30064779265, 1, 2, '{"amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 30064779265, 2, 3, '{"amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
=======
INSERT INTO history_effects VALUES (2, 25769820161, 2, 3, '{"amount": "200000.9234000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
<<<<<<< HEAD
INSERT INTO history_effects VALUES (4, 30064775169, 1, 2, '{"amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 30064775169, 2, 3, '{"amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 30064779265, 1, 20, '{"limit": "1000000000.0000000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
>>>>>>> add price to trade query and /trades endpoint
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_effects VALUES (4, 34359742465, 1, 2, '{"amount": "31.6577680", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 34359742465, 2, 3, '{"amount": "31.6577680", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 34359746561, 1, 2, '{"amount": "1.3623000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 34359746561, 2, 3, '{"amount": "1.3623000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
=======
INSERT INTO history_effects VALUES (4, 30064775169, 1, 22, '{"limit": "1000000000.0000000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 30064779265, 1, 2, '{"amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 30064779265, 2, 3, '{"amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 34359742465, 1, 2, '{"amount": "1.3623000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 34359742465, 2, 3, '{"amount": "1.3623000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (4, 34359746561, 1, 2, '{"amount": "31.6577680", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 34359746561, 2, 3, '{"amount": "31.6577680", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
>>>>>>> wip aggregation test passing


--
-- Data for Name: history_ledgers; Type: TABLE DATA; Schema: public; Owner: -
--

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2018-01-22 17:00:02.856645', '2018-01-22 17:00:02.856645', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (2, '0f0d3b3f3dbcdbb3483171e5bd7e53757ca2e0ad47cdb28c84e95a87735767f0', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 3, 3, '2018-01-22 17:00:00', '2018-01-22 17:00:02.865512', '2018-01-22 17:00:02.865512', 8589934592, 11, 1000000000000000000, 300, 100, 100000000, 10000, 9, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76Z9GwX6q8KzxXUDk3+fdZshszCWX2SbuAnJNMiSoulCykAAAAAWmYYkAAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAACyk/eWGAYEOhla9SxZHjkxGIQ61pIijmPF9hVp1Qv5rM3994NzVGUEstpSkpCIvCg0ArBNhvBZ2713Y2Pc8QigAAAAIN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (3, '0fb88d691f06d4b8ad399e9819d9af0b99f3d3118208f4d76f9cc70fdb5bf99e', '0f0d3b3f3dbcdbb3483171e5bd7e53757ca2e0ad47cdb28c84e95a87735767f0', 3, 3, '2018-01-22 17:00:01', '2018-01-22 17:00:02.8897', '2018-01-22 17:00:02.8897', 12884901888, 11, 1000000000000000000, 600, 100, 100000000, 10000, 9, 'AAAACQ8NOz89vNuzSDFx5b1+U3V8ouCtR82yjITpWodzV2fwHjqDs/CKW0EgbuHtXV+cEcz0bup7he/Kc8LaplYQ3FMAAAAAWmYYkQAAAAAAAAAAYo1bX9bKktPx0YMALq8NmybvVjjSFWQirDV5TTHKxb676GAUV5Afh+w64l628RxE4K9UyT5wbks55EP9g1kMvQAAAAMN4Lazp2QAAAAAAAAAAAJYAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (4, '3e8fbd526990787238c3d5e341be448af0165d39b736c2b6374c79ba553adfd9', '0fb88d691f06d4b8ad399e9819d9af0b99f3d3118208f4d76f9cc70fdb5bf99e', 4, 4, '2018-01-22 17:00:02', '2018-01-22 17:00:02.903383', '2018-01-22 17:00:02.903383', 17179869184, 11, 1000000000000000000, 1000, 100, 100000000, 10000, 9, 'AAAACQ+4jWkfBtS4rTmemBnZrwuZ89MRggj012+cxw/bW/menMkstSUgQm2XL3eiQ4klEMM9sW/rlY479AY9zn3zNBMAAAAAWmYYkgAAAAAAAAAAg4bDLmPB7NuPhcFNu33PjSojcPo2TH0x5yLLubhqMjzdAELFQ/T5G+cw7BCn1PJwi31BZgH7MuejEgdQxxJuvQAAAAQN4Lazp2QAAAAAAAAAAAPoAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (5, '4ab4ad718ef7cc2ab40b55d20a9ee54caa937c89e7bcd9961bbbf7ebff3e907c', '3e8fbd526990787238c3d5e341be448af0165d39b736c2b6374c79ba553adfd9', 3, 3, '2018-01-22 17:00:03', '2018-01-22 17:00:02.923083', '2018-01-22 17:00:02.923083', 21474836480, 11, 1000000000000000000, 1300, 100, 100000000, 10000, 9, 'AAAACT6PvVJpkHhyOMPV40G+RIrwFl05tzbCtjdMebpVOt/ZmD40fk1swdHfiIewRCfPcKqEenUKpDgaheizFbbEUVoAAAAAWmYYkwAAAAAAAAAAAvTWoAZ8sb3n9zINDA1hqoanXiN/pcjfEtKLDrzVvDPbpmBSNgpvamJJTRCorTK25NOHSHs5jMHMrm9HCdj1+QAAAAUN4Lazp2QAAAAAAAAAAAUUAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (6, '23c6795a3728fa108c5a7dd20b1d94161220413af9d1d2cf494b59f9097fdfe7', '4ab4ad718ef7cc2ab40b55d20a9ee54caa937c89e7bcd9961bbbf7ebff3e907c', 4, 4, '2018-01-22 17:00:04', '2018-01-22 17:00:02.940622', '2018-01-22 17:00:02.940622', 25769803776, 11, 1000000000000000000, 1700, 100, 100000000, 10000, 9, 'AAAACUq0rXGO98wqtAtV0gqe5Uyqk3yJ57zZlhu79+v/PpB8RTm0BbIugxOSkgnGb1CUKnBVw4o0XVDJ66douFqOXywAAAAAWmYYlAAAAAAAAAAABfxQX88Y3BtlGniNGV3IwuGKHnvuXPcncSX+LhVbCbKzETnjomxnM5VVggq5b1St/qfoDymXNP0fCnwxn2XJ7wAAAAYN4Lazp2QAAAAAAAAAAAakAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (7, '358f42957b811a166f2163c1b8d83d80c74015ad30141e8bec48c5b0fe24c645', '23c6795a3728fa108c5a7dd20b1d94161220413af9d1d2cf494b59f9097fdfe7', 2, 2, '2018-01-22 17:00:05', '2018-01-22 17:00:02.957217', '2018-01-22 17:00:02.957217', 30064771072, 11, 1000000000000000000, 1900, 100, 100000000, 10000, 9, 'AAAACSPGeVo3KPoQjFp90gsdlBYSIEE6+dHSz0lLWfkJf9/nglQ5qC5ge3MLZc/2IBQJ/EuM3XRjKATkpW4h8T2MRUcAAAAAWmYYlQAAAAAAAAAAyV7/FHiyeDdevKIVtmnMBln7BwHFA4k6e+bc5hANeMJ4N31ge+2KodcF5a59vYuRjQEuoeCFDASBG31UpAKCyQAAAAcN4Lazp2QAAAAAAAAAAAdsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (8, 'd38d6b1d0f534c6a295f1bd59f01e131beea80225f54fc2d4c92470f65fd7d87', '358f42957b811a166f2163c1b8d83d80c74015ad30141e8bec48c5b0fe24c645', 2, 2, '2018-01-22 17:00:06', '2018-01-22 17:00:02.973098', '2018-01-22 17:00:02.973098', 34359738368, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 9, 'AAAACTWPQpV7gRoWbyFjwbjYPYDHQBWtMBQei+xIxbD+JMZFrl9VpgqR6KOOEczEWnMqOzQeHXNMBOVilEmI9E5FQHkAAAAAWmYYlgAAAAAAAAAAIFcEvXLqTKqnn4ZU/O/25YWxOdUJ85OkZ+D/6OUKuebVg7ETUj+cmbsqHUBMQAxZYLoIN3r2GvHvfaqNkLyKawAAAAgN4Lazp2QAAAAAAAAAAAg0AAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (9, 'b3432e998f3e6093711a14dc48a2d7c06c49cf3170749facd977b733150f8a48', 'd38d6b1d0f534c6a295f1bd59f01e131beea80225f54fc2d4c92470f65fd7d87', 0, 0, '2018-01-22 17:00:07', '2018-01-22 17:00:02.997381', '2018-01-22 17:00:02.997381', 38654705664, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 9, 'AAAACdONax0PU0xqKV8b1Z8B4TG+6oAiX1T8LUySRw9l/X2H1DAzplF9naZ0FTY5nOzrLcTfc0qNc+yPFIjDz47L09YAAAAAWmYYlwAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERnVg7ETUj+cmbsqHUBMQAxZYLoIN3r2GvHvfaqNkLyKawAAAAkN4Lazp2QAAAAAAAAAAAg0AAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
=======
>>>>>>> add price to trade query and /trades endpoint
=======
>>>>>>> wip aggregation test passing
=======
>>>>>>> tests passing
<<<<<<< HEAD
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2017-12-15 22:07:19.599196', '2017-12-15 22:07:19.599196', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (2, 'c94e0349989eacb815db79c6c3f210be860e9c385c9aa3e934ec949b0c2c7b99', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 3, 3, '2017-12-15 22:07:16', '2017-12-15 22:07:19.611936', '2017-12-15 22:07:19.611937', 8589934592, 11, 1000000000000000000, 300, 100, 100000000, 10000, 9, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76Z9GwX6q8KzxXUDk3+fdZshszCWX2SbuAnJNMiSoulCykAAAAAWjRHlAAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAACyk/eWGAYEOhla9SxZHjkxGIQ61pIijmPF9hVp1Qv5rM3994NzVGUEstpSkpCIvCg0ArBNhvBZ2713Y2Pc8QigAAAAIN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (3, '4890d70e50cec28dff6d45e27e4d10436f078b19da964cb48c089db564bb6ff4', 'c94e0349989eacb815db79c6c3f210be860e9c385c9aa3e934ec949b0c2c7b99', 3, 3, '2017-12-15 22:07:17', '2017-12-15 22:07:19.644474', '2017-12-15 22:07:19.644474', 12884901888, 11, 1000000000000000000, 600, 100, 100000000, 10000, 9, 'AAAACclOA0mYnqy4Fdt5xsPyEL6GDpw4XJqj6TTslJsMLHuZy2vLEkO+vfA1laJSw6gNqrYXMjQqxZQSClQGhsG5Hi0AAAAAWjRHlQAAAAAAAAAAz4e0cNoSrSyJbiU9ww4Xz/C47nFVIBEHkR11kgme+Dy76GAUV5Afh+w64l628RxE4K9UyT5wbks55EP9g1kMvQAAAAMN4Lazp2QAAAAAAAAAAAJYAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (4, 'cabd1c4e6c5159be46a3b7d357a484dcbb4418c3f74c9f1ac73977c5dae146a8', '4890d70e50cec28dff6d45e27e4d10436f078b19da964cb48c089db564bb6ff4', 4, 4, '2017-12-15 22:07:18', '2017-12-15 22:07:19.658788', '2017-12-15 22:07:19.658788', 17179869184, 11, 1000000000000000000, 1000, 100, 100000000, 10000, 9, 'AAAACUiQ1w5QzsKN/21F4n5NEENvB4sZ2pZMtIwInbVku2/07h9hUR0+OCVShrCloOjAGqXUoJPuJHcqautb6pn4iqMAAAAAWjRHlgAAAAAAAAAAg4bDLmPB7NuPhcFNu33PjSojcPo2TH0x5yLLubhqMjzdAELFQ/T5G+cw7BCn1PJwi31BZgH7MuejEgdQxxJuvQAAAAQN4Lazp2QAAAAAAAAAAAPoAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (5, 'cde70b4059cbdbae9d3082a4c5a55ae1fd760e390e6b675747aced710a582268', 'cabd1c4e6c5159be46a3b7d357a484dcbb4418c3f74c9f1ac73977c5dae146a8', 3, 3, '2017-12-15 22:07:19', '2017-12-15 22:07:19.670524', '2017-12-15 22:07:19.670524', 21474836480, 11, 1000000000000000000, 1300, 100, 100000000, 10000, 9, 'AAAACcq9HE5sUVm+RqO301ekhNy7RBjD90yfGsc5d8Xa4Uaoy2MNl0YUuf1KTGgXfdadaGWme5rUDVNE8z0Un9v6dtIAAAAAWjRHlwAAAAAAAAAAAvTWoAZ8sb3n9zINDA1hqoanXiN/pcjfEtKLDrzVvDPbpmBSNgpvamJJTRCorTK25NOHSHs5jMHMrm9HCdj1+QAAAAUN4Lazp2QAAAAAAAAAAAUUAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (6, '4e6697711f04739913c7a73a01e7c33b226acf46a368483c648632c5050681d6', 'cde70b4059cbdbae9d3082a4c5a55ae1fd760e390e6b675747aced710a582268', 4, 4, '2017-12-15 22:07:20', '2017-12-15 22:07:19.681284', '2017-12-15 22:07:19.681284', 25769803776, 11, 1000000000000000000, 1700, 100, 100000000, 10000, 9, 'AAAACc3nC0BZy9uunTCCpMWlWuH9dg45DmtnV0es7XEKWCJoYfxC6w8yCi8jxGvRrtL/x4UYo5LfpyV28GQXQSa2DRIAAAAAWjRHmAAAAAAAAAAABfxQX88Y3BtlGniNGV3IwuGKHnvuXPcncSX+LhVbCbKzETnjomxnM5VVggq5b1St/qfoDymXNP0fCnwxn2XJ7wAAAAYN4Lazp2QAAAAAAAAAAAakAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (7, '812dffd66cc517045b6ce2b61e53f94463b22c7b109b1b36da13f9b4e09cbf6d', '4e6697711f04739913c7a73a01e7c33b226acf46a368483c648632c5050681d6', 2, 2, '2017-12-15 22:07:21', '2017-12-15 22:07:19.702923', '2017-12-15 22:07:19.702923', 30064771072, 11, 1000000000000000000, 1900, 100, 100000000, 10000, 9, 'AAAACU5ml3EfBHOZE8enOgHnwzsias9Go2hIPGSGMsUFBoHWSsPRrbh2k7nSxBmUsRGuAE4VBhTUCpydYN73IH5ZqpcAAAAAWjRHmQAAAAAAAAAAp6X4a1fQWRN1fT4Oha1VAb23rxjrafhg/8wjyOPQ4el4N31ge+2KodcF5a59vYuRjQEuoeCFDASBG31UpAKCyQAAAAcN4Lazp2QAAAAAAAAAAAdsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (8, 'b5f8410f9fdbfa29c7597211d678f0c6e66eb8f2f76a026a6fdbaba7e39313c1', '812dffd66cc517045b6ce2b61e53f94463b22c7b109b1b36da13f9b4e09cbf6d', 2, 2, '2017-12-15 22:07:22', '2017-12-15 22:07:19.712346', '2017-12-15 22:07:19.712346', 34359738368, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 9, 'AAAACYEt/9ZsxRcEW2zith5T+URjsix7EJsbNtoT+bTgnL9t9BMncaGbf/tpcCs1e40k/4Mmf59NIQIM508B4HYq6zsAAAAAWjRHmgAAAAAAAAAAIFcEvXLqTKqnn4ZU/O/25YWxOdUJ85OkZ+D/6OUKuebVg7ETUj+cmbsqHUBMQAxZYLoIN3r2GvHvfaqNkLyKawAAAAgN4Lazp2QAAAAAAAAAAAg0AAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (9, '4cd31a6fc12952242c6bdca4e371e1c5e060054bcfe4f253e56e393130ceb9ff', 'b5f8410f9fdbfa29c7597211d678f0c6e66eb8f2f76a026a6fdbaba7e39313c1', 0, 0, '2017-12-15 22:07:23', '2017-12-15 22:07:19.722383', '2017-12-15 22:07:19.722383', 38654705664, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 9, 'AAAACbX4QQ+f2/opx1lyEdZ48Mbmbrjy92oCam/bq6fjkxPBawBLMHVqmai3zsZRZL/K2ewPQaxEZFsaeOoCtk4zl0wAAAAAWjRHmwAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERnVg7ETUj+cmbsqHUBMQAxZYLoIN3r2GvHvfaqNkLyKawAAAAkN4Lazp2QAAAAAAAAAAAg0AAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2017-12-18 23:32:08.426711', '2017-12-18 23:32:08.426711', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0);
INSERT INTO history_ledgers VALUES (2, '37959d1f2807ea0ff7a1ea77ffa0805a6bf605ae26288c37314451ffc46ab660', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 3, 3, '2017-12-18 23:32:05', '2017-12-18 23:32:08.433698', '2017-12-18 23:32:08.433698', 8589934592, 11, 1000000000000000000, 300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (3, '1f4e4b1f6698a9970aa110e8e505d3d3edb0b9c1f112d576da2f2349e9c8ed4c', '37959d1f2807ea0ff7a1ea77ffa0805a6bf605ae26288c37314451ffc46ab660', 3, 3, '2017-12-18 23:32:06', '2017-12-18 23:32:08.453845', '2017-12-18 23:32:08.453846', 12884901888, 11, 1000000000000000000, 600, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (4, '3283ab29536df789b1db2ec0f374d783ecdf788553a1f687b92ff856a43c0701', '1f4e4b1f6698a9970aa110e8e505d3d3edb0b9c1f112d576da2f2349e9c8ed4c', 4, 4, '2017-12-18 23:32:07', '2017-12-18 23:32:08.461049', '2017-12-18 23:32:08.461049', 17179869184, 11, 1000000000000000000, 1000, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (5, 'cd55b18db78624bc548aa6948fc57dcf1f9a347151d37556661d2552bfca72b6', '3283ab29536df789b1db2ec0f374d783ecdf788553a1f687b92ff856a43c0701', 3, 3, '2017-12-18 23:32:08', '2017-12-18 23:32:08.48209', '2017-12-18 23:32:08.48209', 21474836480, 11, 1000000000000000000, 1300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (6, 'bf674f80efa354de9e2a2e9e0b92eb419da9fb0394ffc02fa25c05dfa6959683', 'cd55b18db78624bc548aa6948fc57dcf1f9a347151d37556661d2552bfca72b6', 4, 4, '2017-12-18 23:32:09', '2017-12-18 23:32:08.501192', '2017-12-18 23:32:08.501192', 25769803776, 11, 1000000000000000000, 1700, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (7, '8cfcb236c0a01a103874e318b15608a38ee07bdc4cd85ab4a9e77b989febd6d1', 'bf674f80efa354de9e2a2e9e0b92eb419da9fb0394ffc02fa25c05dfa6959683', 2, 2, '2017-12-18 23:32:10', '2017-12-18 23:32:08.520748', '2017-12-18 23:32:08.520749', 30064771072, 11, 1000000000000000000, 1900, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (8, '7c3fc83957297327b60faacb016a6e33abec6edc26b0fc1e493a5d493745a92d', '8cfcb236c0a01a103874e318b15608a38ee07bdc4cd85ab4a9e77b989febd6d1', 2, 2, '2017-12-18 23:32:11', '2017-12-18 23:32:08.530828', '2017-12-18 23:32:08.530828', 34359738368, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (9, '070188f7f8c0a9a0df7b8bdfbce9b271d148151fb7c345b223e443706d3d1a4a', '7c3fc83957297327b60faacb016a6e33abec6edc26b0fc1e493a5d493745a92d', 0, 0, '2017-12-18 23:32:12', '2017-12-18 23:32:08.539098', '2017-12-18 23:32:08.539098', 38654705664, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 8);
>>>>>>> add price to trade ingestion
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
=======
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2018-01-03 00:54:02.610083', '2018-01-03 00:54:02.610083', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0);
INSERT INTO history_ledgers VALUES (2, '349ee99ae06b952c49bfebc0a525c269663abc8078e2450c43e32e1e51e3514e', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 3, 3, '2018-01-03 00:53:59', '2018-01-03 00:54:02.623713', '2018-01-03 00:54:02.623713', 8589934592, 11, 1000000000000000000, 300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (3, '7947d569d33e137f2600b3091d8e858804eb6904135d64444c54e190ae8ba43e', '349ee99ae06b952c49bfebc0a525c269663abc8078e2450c43e32e1e51e3514e', 3, 3, '2018-01-03 00:54:00', '2018-01-03 00:54:02.694206', '2018-01-03 00:54:02.694207', 12884901888, 11, 1000000000000000000, 600, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (4, '7c9809ae92fe39d4ff035b9f361f49543f2e961c47e6c846a4b049a9958c7085', '7947d569d33e137f2600b3091d8e858804eb6904135d64444c54e190ae8ba43e', 4, 4, '2018-01-03 00:54:01', '2018-01-03 00:54:02.731224', '2018-01-03 00:54:02.731224', 17179869184, 11, 1000000000000000000, 1000, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (5, '4fc2bb93fec69ae7abd16c890202e85d2911b7355c9ee1bcbc02c6a21c1824bf', '7c9809ae92fe39d4ff035b9f361f49543f2e961c47e6c846a4b049a9958c7085', 3, 3, '2018-01-03 00:54:02', '2018-01-03 00:54:02.767386', '2018-01-03 00:54:02.767387', 21474836480, 11, 1000000000000000000, 1300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (6, '92d8721efff6af449863a78007dbdd300c31f593a559e4495c9a5586e72d4d9d', '4fc2bb93fec69ae7abd16c890202e85d2911b7355c9ee1bcbc02c6a21c1824bf', 4, 4, '2018-01-03 00:54:03', '2018-01-03 00:54:02.802475', '2018-01-03 00:54:02.802475', 25769803776, 11, 1000000000000000000, 1700, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (7, '761a42d8fd2a77dd0c8b0a902e1b15f434b86182a203b7c8b8fcd66f0ed660c7', '92d8721efff6af449863a78007dbdd300c31f593a559e4495c9a5586e72d4d9d', 2, 2, '2018-01-03 00:54:04', '2018-01-03 00:54:02.848465', '2018-01-03 00:54:02.848466', 30064771072, 11, 1000000000000000000, 1900, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (8, 'c75e59423bdec2a4faa0b9fda35716363894a4660724de695cb379c4961299b1', '761a42d8fd2a77dd0c8b0a902e1b15f434b86182a203b7c8b8fcd66f0ed660c7', 2, 2, '2018-01-03 00:54:05', '2018-01-03 00:54:02.873751', '2018-01-03 00:54:02.873752', 34359738368, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (9, '8d5215365fd99c87caac7c5f629c1884fb1420bc919dedf0724e46d8dde33392', 'c75e59423bdec2a4faa0b9fda35716363894a4660724de695cb379c4961299b1', 0, 0, '2018-01-03 00:54:06', '2018-01-03 00:54:02.901908', '2018-01-03 00:54:02.901908', 38654705664, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 8);
>>>>>>> add price to trade query and /trades endpoint
<<<<<<< HEAD
>>>>>>> add price to trade query and /trades endpoint
=======
=======
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2018-01-12 01:04:47.860729', '2018-01-12 01:04:47.860729', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0);
INSERT INTO history_ledgers VALUES (2, '73fe35c179a9833177e7dca8657c4a15d30c69f74da93d7e1d26b102deb1af38', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 3, 3, '2018-01-12 01:04:45', '2018-01-12 01:04:47.870617', '2018-01-12 01:04:47.870618', 8589934592, 11, 1000000000000000000, 300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (3, 'dfcbfa93446eb24afcb892c5ba31b581847c53d6ed58a67be2cae206ad4d1a72', '73fe35c179a9833177e7dca8657c4a15d30c69f74da93d7e1d26b102deb1af38', 3, 3, '2018-01-12 01:04:46', '2018-01-12 01:04:47.903074', '2018-01-12 01:04:47.903074', 12884901888, 11, 1000000000000000000, 600, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (4, 'c199dec0b47ee23936a9d7e9f8d1b0b585931486b3f0c3ab2ebf11db6d9ef98d', 'dfcbfa93446eb24afcb892c5ba31b581847c53d6ed58a67be2cae206ad4d1a72', 4, 4, '2018-01-12 01:04:47', '2018-01-12 01:04:47.920723', '2018-01-12 01:04:47.920723', 17179869184, 11, 1000000000000000000, 1000, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (5, 'e1641fa750eb83d2c55ec88d954e5811ad1e743e279ad3abcbd950538216b33e', 'c199dec0b47ee23936a9d7e9f8d1b0b585931486b3f0c3ab2ebf11db6d9ef98d', 3, 3, '2018-01-12 01:04:48', '2018-01-12 01:04:47.94468', '2018-01-12 01:04:47.94468', 21474836480, 11, 1000000000000000000, 1300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (6, '3839a0dd46b225dcdbdabf0b58aa503d80dd09b99fc8149eb8d1be7202bcc0a1', 'e1641fa750eb83d2c55ec88d954e5811ad1e743e279ad3abcbd950538216b33e', 4, 4, '2018-01-12 01:04:49', '2018-01-12 01:04:47.972686', '2018-01-12 01:04:47.972686', 25769803776, 11, 1000000000000000000, 1700, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (7, 'eba5d84ed694b0492a7321378f81a4a39c0f1a1bc143faf83de80cd7ccfc4a37', '3839a0dd46b225dcdbdabf0b58aa503d80dd09b99fc8149eb8d1be7202bcc0a1', 2, 2, '2018-01-12 01:04:50', '2018-01-12 01:04:47.995664', '2018-01-12 01:04:47.995664', 30064771072, 11, 1000000000000000000, 1900, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (8, 'beda63f65927a9b66bf254593a893817007d6e19f740b88d48c9d1880c136fdf', 'eba5d84ed694b0492a7321378f81a4a39c0f1a1bc143faf83de80cd7ccfc4a37', 2, 2, '2018-01-12 01:04:51', '2018-01-12 01:04:48.012241', '2018-01-12 01:04:48.012242', 34359738368, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (9, '6770d30f9599e705993ce0e55e7dac6355497623ac19e4875850275f6737fde5', 'beda63f65927a9b66bf254593a893817007d6e19f740b88d48c9d1880c136fdf', 0, 0, '2018-01-12 01:04:52', '2018-01-12 01:04:48.026914', '2018-01-12 01:04:48.026915', 38654705664, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 8);
>>>>>>> wip aggregation test passing
<<<<<<< HEAD
>>>>>>> wip aggregation test passing
=======
=======
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2018-01-12 18:35:14.07969', '2018-01-12 18:35:14.079691', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0);
INSERT INTO history_ledgers VALUES (2, '8269b0a2a7552e5e20dcfc511d5c0b1c0bb7538453741da78bdf64d72b53156f', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 3, 3, '2018-01-12 18:35:11', '2018-01-12 18:35:14.090337', '2018-01-12 18:35:14.090337', 8589934592, 11, 1000000000000000000, 300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (3, '6b5acb6063c49d89e52df76ceb249685120ab3bc54823d7d6c3e9d15d1bdda92', '8269b0a2a7552e5e20dcfc511d5c0b1c0bb7538453741da78bdf64d72b53156f', 3, 3, '2018-01-12 18:35:12', '2018-01-12 18:35:14.113651', '2018-01-12 18:35:14.113651', 12884901888, 11, 1000000000000000000, 600, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (4, '05bcc6f7ce6ce1a1d5f0b9b8ef26fc5b0c9e272aabe02619df12b919cb48208f', '6b5acb6063c49d89e52df76ceb249685120ab3bc54823d7d6c3e9d15d1bdda92', 4, 4, '2018-01-12 18:35:13', '2018-01-12 18:35:14.133754', '2018-01-12 18:35:14.133755', 17179869184, 11, 1000000000000000000, 1000, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (5, 'e28fc7af8999367e67408d39826ca80f5d6d1cda2c62371fb0a5df42e0cb056e', '05bcc6f7ce6ce1a1d5f0b9b8ef26fc5b0c9e272aabe02619df12b919cb48208f', 3, 3, '2018-01-12 18:35:14', '2018-01-12 18:35:14.159557', '2018-01-12 18:35:14.159557', 21474836480, 11, 1000000000000000000, 1300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (6, '8d6fbe14e871251a63f59f2e26aa6564d3c8901882e8a42e70965038aca1ec6d', 'e28fc7af8999367e67408d39826ca80f5d6d1cda2c62371fb0a5df42e0cb056e', 4, 4, '2018-01-12 18:35:15', '2018-01-12 18:35:14.17272', '2018-01-12 18:35:14.17272', 25769803776, 11, 1000000000000000000, 1700, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (7, '7deeedd3bd48b596d56d766b30c121b1c8490a119d72f459aa1b2ebc02cc5c5f', '8d6fbe14e871251a63f59f2e26aa6564d3c8901882e8a42e70965038aca1ec6d', 2, 2, '2018-01-12 18:35:16', '2018-01-12 18:35:14.186008', '2018-01-12 18:35:14.186008', 30064771072, 11, 1000000000000000000, 1900, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (8, 'e9c8d0c855a51a28fe71b65b81370a7a74eb6cf5a794a7c5c3d3c6f77d4e5551', '7deeedd3bd48b596d56d766b30c121b1c8490a119d72f459aa1b2ebc02cc5c5f', 2, 2, '2018-01-12 18:35:17', '2018-01-12 18:35:14.194116', '2018-01-12 18:35:14.194116', 34359738368, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (9, 'b745c52f4fcc8cc2a4de2f35a5d2b1998f24124675c930d13d743ae9908c953d', 'e9c8d0c855a51a28fe71b65b81370a7a74eb6cf5a794a7c5c3d3c6f77d4e5551', 0, 0, '2018-01-12 18:35:18', '2018-01-12 18:35:14.202326', '2018-01-12 18:35:14.202326', 38654705664, 11, 1000000000000000000, 2100, 100, 100000000, 10000, 8);
>>>>>>> tests passing
>>>>>>> tests passing


--
-- Data for Name: history_operation_participants; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_operation_participants VALUES (1, 8589938689, 1);
INSERT INTO history_operation_participants VALUES (2, 8589938689, 2);
INSERT INTO history_operation_participants VALUES (3, 8589942785, 1);
INSERT INTO history_operation_participants VALUES (4, 8589942785, 3);
INSERT INTO history_operation_participants VALUES (5, 8589946881, 1);
INSERT INTO history_operation_participants VALUES (6, 8589946881, 4);
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_operation_participants VALUES (7, 12884905985, 2);
INSERT INTO history_operation_participants VALUES (8, 12884910081, 3);
INSERT INTO history_operation_participants VALUES (9, 12884914177, 2);
=======
INSERT INTO history_operation_participants VALUES (7, 12884905985, 1);
INSERT INTO history_operation_participants VALUES (8, 12884910081, 3);
INSERT INTO history_operation_participants VALUES (9, 12884914177, 1);
>>>>>>> add price to trade ingestion
=======
INSERT INTO history_operation_participants VALUES (7, 12884905985, 2);
INSERT INTO history_operation_participants VALUES (8, 12884910081, 3);
=======
INSERT INTO history_operation_participants VALUES (7, 12884905985, 3);
INSERT INTO history_operation_participants VALUES (8, 12884910081, 2);
>>>>>>> wip aggregation test passing
INSERT INTO history_operation_participants VALUES (9, 12884914177, 2);
<<<<<<< HEAD
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_operation_participants VALUES (10, 17179873281, 4);
INSERT INTO history_operation_participants VALUES (11, 17179877377, 3);
INSERT INTO history_operation_participants VALUES (12, 17179881473, 3);
INSERT INTO history_operation_participants VALUES (13, 17179885569, 4);
=======
INSERT INTO history_operation_participants VALUES (10, 17179873281, 3);
INSERT INTO history_operation_participants VALUES (11, 17179877377, 4);
INSERT INTO history_operation_participants VALUES (12, 17179881473, 4);
INSERT INTO history_operation_participants VALUES (13, 17179885569, 3);
>>>>>>> tests passing
INSERT INTO history_operation_participants VALUES (14, 21474840577, 2);
INSERT INTO history_operation_participants VALUES (15, 21474840577, 3);
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_operation_participants VALUES (16, 21474844673, 3);
INSERT INTO history_operation_participants VALUES (17, 21474844673, 2);
INSERT INTO history_operation_participants VALUES (18, 21474848769, 4);
INSERT INTO history_operation_participants VALUES (19, 21474848769, 2);
INSERT INTO history_operation_participants VALUES (20, 25769807873, 3);
INSERT INTO history_operation_participants VALUES (21, 25769807873, 2);
INSERT INTO history_operation_participants VALUES (22, 25769811969, 3);
INSERT INTO history_operation_participants VALUES (23, 25769811969, 4);
INSERT INTO history_operation_participants VALUES (24, 25769816065, 3);
INSERT INTO history_operation_participants VALUES (25, 25769816065, 2);
INSERT INTO history_operation_participants VALUES (26, 25769820161, 2);
INSERT INTO history_operation_participants VALUES (27, 25769820161, 4);
INSERT INTO history_operation_participants VALUES (28, 30064775169, 3);
INSERT INTO history_operation_participants VALUES (29, 30064775169, 4);
=======
=======
>>>>>>> add price to trade query and /trades endpoint
<<<<<<< HEAD
INSERT INTO history_operation_participants VALUES (16, 21474844673, 2);
INSERT INTO history_operation_participants VALUES (17, 21474844673, 3);
INSERT INTO history_operation_participants VALUES (18, 21474848769, 4);
INSERT INTO history_operation_participants VALUES (19, 21474848769, 2);
INSERT INTO history_operation_participants VALUES (20, 25769807873, 2);
INSERT INTO history_operation_participants VALUES (21, 25769807873, 3);
=======
INSERT INTO history_operation_participants VALUES (16, 21474844673, 3);
INSERT INTO history_operation_participants VALUES (17, 21474844673, 1);
INSERT INTO history_operation_participants VALUES (18, 21474848769, 1);
INSERT INTO history_operation_participants VALUES (19, 21474848769, 4);
INSERT INTO history_operation_participants VALUES (20, 25769807873, 3);
INSERT INTO history_operation_participants VALUES (21, 25769807873, 1);
>>>>>>> add price to trade ingestion
=======
INSERT INTO history_operation_participants VALUES (16, 21474844673, 2);
INSERT INTO history_operation_participants VALUES (17, 21474844673, 3);
INSERT INTO history_operation_participants VALUES (18, 21474848769, 2);
INSERT INTO history_operation_participants VALUES (19, 21474848769, 4);
<<<<<<< HEAD
INSERT INTO history_operation_participants VALUES (20, 25769807873, 3);
<<<<<<< HEAD
INSERT INTO history_operation_participants VALUES (21, 25769807873, 2);
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_operation_participants VALUES (22, 25769811969, 3);
INSERT INTO history_operation_participants VALUES (23, 25769811969, 4);
=======
INSERT INTO history_operation_participants VALUES (21, 25769807873, 4);
INSERT INTO history_operation_participants VALUES (22, 25769811969, 2);
INSERT INTO history_operation_participants VALUES (23, 25769811969, 3);
>>>>>>> wip aggregation test passing
=======
INSERT INTO history_operation_participants VALUES (20, 25769807873, 2);
INSERT INTO history_operation_participants VALUES (21, 25769807873, 3);
INSERT INTO history_operation_participants VALUES (22, 25769811969, 3);
INSERT INTO history_operation_participants VALUES (23, 25769811969, 4);
>>>>>>> tests passing
INSERT INTO history_operation_participants VALUES (24, 25769816065, 2);
INSERT INTO history_operation_participants VALUES (25, 25769816065, 3);
<<<<<<< HEAD
INSERT INTO history_operation_participants VALUES (26, 25769820161, 4);
INSERT INTO history_operation_participants VALUES (27, 25769820161, 1);
INSERT INTO history_operation_participants VALUES (28, 30064775169, 4);
INSERT INTO history_operation_participants VALUES (29, 30064779265, 3);
>>>>>>> add price to trade ingestion
=======
INSERT INTO history_operation_participants VALUES (26, 25769820161, 2);
INSERT INTO history_operation_participants VALUES (27, 25769820161, 4);
<<<<<<< HEAD
INSERT INTO history_operation_participants VALUES (28, 30064775169, 3);
INSERT INTO history_operation_participants VALUES (29, 30064775169, 4);
>>>>>>> add price to trade query and /trades endpoint
=======
INSERT INTO history_operation_participants VALUES (28, 30064775169, 4);
INSERT INTO history_operation_participants VALUES (29, 30064779265, 3);
>>>>>>> wip aggregation test passing
INSERT INTO history_operation_participants VALUES (30, 30064779265, 4);
INSERT INTO history_operation_participants VALUES (31, 34359742465, 4);
INSERT INTO history_operation_participants VALUES (32, 34359742465, 3);
INSERT INTO history_operation_participants VALUES (33, 34359746561, 3);
INSERT INTO history_operation_participants VALUES (34, 34359746561, 4);


--
-- Data for Name: history_operations; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_operations VALUES (8589938689, 8589938688, 1, 0, '{"funder": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", "account": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "starting_balance": "1000.0000000"}', 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H');
INSERT INTO history_operations VALUES (8589942785, 8589942784, 1, 0, '{"funder": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", "account": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "starting_balance": "1000.0000000"}', 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H');
INSERT INTO history_operations VALUES (8589946881, 8589946880, 1, 0, '{"funder": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", "account": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "starting_balance": "1000.0000000"}', 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H');
INSERT INTO history_operations VALUES (12884905985, 12884905984, 1, 5, '{"set_flags": [2], "set_flags_s": ["auth_revocable"]}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_operations VALUES (12884910081, 12884910080, 1, 5, '{"home_domain": "test.com"}', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_operations VALUES (12884914177, 12884914176, 1, 5, '{"set_flags": [1], "set_flags_s": ["auth_required"]}', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_operations VALUES (17179873281, 17179873280, 1, 6, '{"limit": "922337203685.4775807", "trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_operations VALUES (17179877377, 17179877376, 1, 6, '{"limit": "922337203685.4775807", "trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2');
INSERT INTO history_operations VALUES (17179881473, 17179881472, 1, 6, '{"limit": "922337203685.4775807", "trustee": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "trustor": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "asset_code": "SCOT", "asset_type": "credit_alphanum4", "asset_issuer": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}', 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2');
INSERT INTO history_operations VALUES (17179885569, 17179885568, 1, 6, '{"limit": "922337203685.4775807", "trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "asset_code": "BTC", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_operations VALUES (21474840577, 21474840576, 1, 7, '{"trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "authorize": true, "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_operations VALUES (21474844673, 21474844672, 1, 7, '{"trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "authorize": true, "asset_code": "BTC", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_operations VALUES (21474848769, 21474848768, 1, 7, '{"trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "authorize": true, "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_operations VALUES (25769807873, 25769807872, 1, 1, '{"to": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "from": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "amount": "100000.1200000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_operations VALUES (25769811969, 25769811968, 1, 1, '{"to": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "from": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "amount": "1000.0000000", "asset_code": "SCOT", "asset_type": "credit_alphanum4", "asset_issuer": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_operations VALUES (25769816065, 25769816064, 1, 1, '{"to": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "from": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "amount": "100.9876000", "asset_code": "BTC", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
INSERT INTO history_operations VALUES (25769820161, 25769820160, 1, 1, '{"to": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "from": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "amount": "200000.9234000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_operations VALUES (30064775169, 30064775168, 1, 1, '{"to": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "from": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_operations VALUES (30064779265, 30064779264, 1, 6, '{"limit": "1000000000.0000000", "trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2');
=======
=======
>>>>>>> wip aggregation test passing
<<<<<<< HEAD
INSERT INTO history_operations VALUES (30064775169, 30064775168, 1, 6, '{"limit": "1000000000.0000000", "trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2');
INSERT INTO history_operations VALUES (30064779265, 30064779264, 1, 1, '{"to": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "from": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
=======
INSERT INTO history_operations VALUES (30064775169, 30064775168, 1, 1, '{"to": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "from": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_operations VALUES (30064779265, 30064779264, 1, 6, '{"limit": "1000000000.0000000", "trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2');
>>>>>>> add price to trade query and /trades endpoint
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_operations VALUES (34359742465, 34359742464, 1, 1, '{"to": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "from": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "amount": "31.6577680", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_operations VALUES (34359746561, 34359746560, 1, 1, '{"to": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "from": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "amount": "1.3623000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2');
=======
INSERT INTO history_operations VALUES (30064775169, 30064775168, 1, 6, '{"limit": "1000000000.0000000", "trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2');
INSERT INTO history_operations VALUES (30064779265, 30064779264, 1, 1, '{"to": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "from": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "amount": "89.9500000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_operations VALUES (34359742465, 34359742464, 1, 1, '{"to": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "from": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "amount": "1.3623000", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2');
INSERT INTO history_operations VALUES (34359746561, 34359746560, 1, 1, '{"to": "GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2", "from": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "amount": "31.6577680", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
>>>>>>> wip aggregation test passing


--
-- Data for Name: history_trades; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: history_transaction_participants; Type: TABLE DATA; Schema: public; Owner: -
--

<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_transaction_participants VALUES (1, 8589938688, 1);
INSERT INTO history_transaction_participants VALUES (2, 8589938688, 2);
=======
INSERT INTO history_transaction_participants VALUES (1, 8589938688, 2);
INSERT INTO history_transaction_participants VALUES (2, 8589938688, 1);
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
INSERT INTO history_transaction_participants VALUES (1, 8589938688, 2);
INSERT INTO history_transaction_participants VALUES (2, 8589938688, 1);
<<<<<<< HEAD
=======
INSERT INTO history_transaction_participants VALUES (1, 8589938688, 1);
INSERT INTO history_transaction_participants VALUES (2, 8589938688, 2);
>>>>>>> add price to trade query and /trades endpoint
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_transaction_participants VALUES (3, 8589942784, 1);
INSERT INTO history_transaction_participants VALUES (4, 8589942784, 3);
INSERT INTO history_transaction_participants VALUES (5, 8589946880, 1);
INSERT INTO history_transaction_participants VALUES (6, 8589946880, 4);
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_transaction_participants VALUES (7, 12884905984, 2);
INSERT INTO history_transaction_participants VALUES (8, 12884910080, 3);
=======
=======
>>>>>>> wip aggregation test passing
<<<<<<< HEAD
INSERT INTO history_transaction_participants VALUES (7, 12884905984, 3);
INSERT INTO history_transaction_participants VALUES (8, 12884910080, 2);
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_transaction_participants VALUES (9, 12884914176, 2);
=======
INSERT INTO history_transaction_participants VALUES (3, 8589942784, 3);
INSERT INTO history_transaction_participants VALUES (4, 8589942784, 2);
INSERT INTO history_transaction_participants VALUES (5, 8589946880, 4);
INSERT INTO history_transaction_participants VALUES (6, 8589946880, 2);
INSERT INTO history_transaction_participants VALUES (7, 12884905984, 1);
INSERT INTO history_transaction_participants VALUES (8, 12884910080, 3);
INSERT INTO history_transaction_participants VALUES (9, 12884914176, 1);
>>>>>>> add price to trade ingestion
=======
INSERT INTO history_transaction_participants VALUES (7, 12884905984, 2);
INSERT INTO history_transaction_participants VALUES (8, 12884910080, 3);
=======
INSERT INTO history_transaction_participants VALUES (7, 12884905984, 3);
INSERT INTO history_transaction_participants VALUES (8, 12884910080, 2);
>>>>>>> wip aggregation test passing
INSERT INTO history_transaction_participants VALUES (9, 12884914176, 2);
<<<<<<< HEAD
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_transaction_participants VALUES (10, 17179873280, 4);
INSERT INTO history_transaction_participants VALUES (11, 17179877376, 3);
INSERT INTO history_transaction_participants VALUES (12, 17179881472, 3);
INSERT INTO history_transaction_participants VALUES (13, 17179885568, 4);
INSERT INTO history_transaction_participants VALUES (14, 21474840576, 2);
INSERT INTO history_transaction_participants VALUES (15, 21474840576, 3);
=======
INSERT INTO history_transaction_participants VALUES (10, 17179873280, 3);
INSERT INTO history_transaction_participants VALUES (11, 17179877376, 4);
INSERT INTO history_transaction_participants VALUES (12, 17179881472, 4);
INSERT INTO history_transaction_participants VALUES (13, 17179885568, 3);
INSERT INTO history_transaction_participants VALUES (14, 21474840576, 3);
INSERT INTO history_transaction_participants VALUES (15, 21474840576, 2);
>>>>>>> tests passing
INSERT INTO history_transaction_participants VALUES (16, 21474844672, 2);
INSERT INTO history_transaction_participants VALUES (17, 21474844672, 3);
INSERT INTO history_transaction_participants VALUES (18, 21474848768, 2);
INSERT INTO history_transaction_participants VALUES (19, 21474848768, 4);
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_transaction_participants VALUES (20, 25769807872, 3);
INSERT INTO history_transaction_participants VALUES (21, 25769807872, 2);
INSERT INTO history_transaction_participants VALUES (22, 25769811968, 3);
INSERT INTO history_transaction_participants VALUES (23, 25769811968, 4);
INSERT INTO history_transaction_participants VALUES (24, 25769816064, 3);
INSERT INTO history_transaction_participants VALUES (25, 25769816064, 2);
INSERT INTO history_transaction_participants VALUES (26, 25769820160, 2);
INSERT INTO history_transaction_participants VALUES (27, 25769820160, 4);
INSERT INTO history_transaction_participants VALUES (28, 30064775168, 3);
INSERT INTO history_transaction_participants VALUES (29, 30064775168, 4);
=======
INSERT INTO history_transaction_participants VALUES (20, 25769807872, 1);
=======
=======
>>>>>>> wip aggregation test passing
=======
>>>>>>> tests passing
INSERT INTO history_transaction_participants VALUES (20, 25769807872, 2);
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_transaction_participants VALUES (21, 25769807872, 3);
<<<<<<< HEAD
INSERT INTO history_transaction_participants VALUES (22, 25769811968, 4);
INSERT INTO history_transaction_participants VALUES (23, 25769811968, 3);
INSERT INTO history_transaction_participants VALUES (24, 25769816064, 2);
=======
INSERT INTO history_transaction_participants VALUES (22, 25769811968, 3);
INSERT INTO history_transaction_participants VALUES (23, 25769811968, 4);
<<<<<<< HEAD
INSERT INTO history_transaction_participants VALUES (24, 25769816064, 1);
>>>>>>> add price to trade ingestion
=======
=======
INSERT INTO history_transaction_participants VALUES (20, 25769807872, 3);
INSERT INTO history_transaction_participants VALUES (21, 25769807872, 4);
INSERT INTO history_transaction_participants VALUES (22, 25769811968, 2);
=======
INSERT INTO history_transaction_participants VALUES (20, 25769807872, 2);
INSERT INTO history_transaction_participants VALUES (21, 25769807872, 3);
INSERT INTO history_transaction_participants VALUES (22, 25769811968, 4);
>>>>>>> tests passing
INSERT INTO history_transaction_participants VALUES (23, 25769811968, 3);
>>>>>>> wip aggregation test passing
INSERT INTO history_transaction_participants VALUES (24, 25769816064, 2);
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_transaction_participants VALUES (25, 25769816064, 3);
INSERT INTO history_transaction_participants VALUES (26, 25769820160, 4);
INSERT INTO history_transaction_participants VALUES (27, 25769820160, 2);
INSERT INTO history_transaction_participants VALUES (28, 30064775168, 4);
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_transaction_participants VALUES (29, 30064779264, 3);
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
=======
INSERT INTO history_transaction_participants VALUES (29, 30064775168, 3);
>>>>>>> add price to trade query and /trades endpoint
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO history_transaction_participants VALUES (30, 30064779264, 4);
INSERT INTO history_transaction_participants VALUES (31, 34359742464, 3);
INSERT INTO history_transaction_participants VALUES (32, 34359742464, 4);
INSERT INTO history_transaction_participants VALUES (33, 34359746560, 4);
INSERT INTO history_transaction_participants VALUES (34, 34359746560, 3);
<<<<<<< HEAD
=======
=======
>>>>>>> wip aggregation test passing
INSERT INTO history_transaction_participants VALUES (29, 30064779264, 4);
INSERT INTO history_transaction_participants VALUES (30, 30064779264, 3);
=======
INSERT INTO history_transaction_participants VALUES (29, 30064779264, 3);
INSERT INTO history_transaction_participants VALUES (30, 30064779264, 4);
>>>>>>> tests passing
INSERT INTO history_transaction_participants VALUES (31, 34359742464, 4);
INSERT INTO history_transaction_participants VALUES (32, 34359742464, 3);
INSERT INTO history_transaction_participants VALUES (33, 34359746560, 3);
INSERT INTO history_transaction_participants VALUES (34, 34359746560, 4);
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
>>>>>>> add price to trade query and /trades endpoint
=======
>>>>>>> wip aggregation test passing


--
-- Data for Name: history_transactions; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_transactions VALUES ('db398eb4ae89756325643cad21c94e13bfc074b323ee83e141bf701a5d904f1b', 2, 1, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 1, 100, 1, '2018-01-12 18:35:14.091067', '2018-01-12 18:35:14.091067', 8589938688, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAACVAvkAAAAAAAAAAABVvwF9wAAAEAYjQcPT2G5hqnBmgGGeg9J8l4c1EnUlxklElH9sqZr0971F6OLWfe/m4kpFtI+sI0i1qLit5A0JyWnbhYLW5oD', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2sVNYGtQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{GI0HD09huYapwZoBhnoPSfJeHNRJ1JcZJRJR/bKma9Pe9Reji1n3v5uJKRbSPrCNItai4reQNCclp24WC1uaAw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('f97caffab8c16023a37884165cb0b3ff1aa2daf4000fef49d21efc847ddbfbea', 2, 2, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 2, 100, 1, '2018-01-12 18:35:14.099941', '2018-01-12 18:35:14.099941', 8589942784, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBmKpSgvrwKO20XCOfYfXsGEEUtwYaaEfqSu6ymJmlDma+IX6I7IggbUZMocQdZ94IMAfKdQANqXbIO7ysweeMC', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rv9MNtQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAQAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{ZiqUoL68CjttFwjn2H17BhBFLcGGmhH6kruspiZpQ5mviF+iOyIIG1GTKHEHWfeCDAHynUADal2yDu8rMHnjAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('725756b1fbdf83b08127f385efedf0909cc820b6cce71f1c0897d15427cb5add', 2, 3, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 3, 100, 1, '2018-01-12 18:35:14.103655', '2018-01-12 18:35:14.103656', 8589946880, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAADAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBj4gBQ/BAbgqf7qOotatgZUHjDlsOtDNdp7alZR5/Fk9fGj+lxEygAZWzY7/LY1Z3SF6c0qs172LhAkkvV8p0M', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rKtAUtQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAQAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/7UAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{Y+IAUPwQG4Kn+6jqLWrYGVB4w5bDrQzXae2pWUefxZPXxo/pcRMoAGVs2O/y2NWd0henNKrNe9i4QJJL1fKdDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('bd60680a1378ffec739e1ffa2db4cd51f58babfb714e04a52bd2b65bf8a31b4f', 3, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934593, 100, 1, '2018-01-12 18:35:14.113965', '2018-01-12 18:35:14.113965', 12884905984, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABruS+TAAAAEBkz5uRgU5FxqOu8Yak7Bbdc0BtgvEJ0FjurZz/LgGwT2EX91Y81YrdSVu2NPR0lbhSAotGQlvSPYEy5vN67p4C', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAA=', 'AAAAAAAAAAEAAAABAAAAAQAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAAAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{ZM+bkYFORcajrvGGpOwW3XNAbYLxCdBY7q2c/y4BsE9hF/dWPNWK3UlbtjT0dJW4UgKLRkJb0j2BMubzeu6eAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('c780569c402c298b7b5f3f1a6a20ac1219a06df39a78fb3ac6d93ca53ad4e5ed', 3, 2, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934593, 100, 1, '2018-01-12 18:35:14.115877', '2018-01-12 18:35:14.115877', 12884910080, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAACHRlc3QuY29tAAAAAAAAAAAAAAAB+ZAt7wAAAEBHwkZcyIWmaPvEtDlR8Ed4dD1Mep2juLtHF3n5RG0jurJhKq/3MB1zR6bDHr+wow35ijK92ihjHWqTxjzKDhkO', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAA=', 'AAAAAAAAAAEAAAABAAAAAQAAAAMAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAAAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', 'AAAAAgAAAAMAAAACAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{R8JGXMiFpmj7xLQ5UfBHeHQ9THqdo7i7Rxd5+URtI7qyYSqv9zAdc0emwx6/sKMN+YoyvdooYx1qk8Y8yg4ZDg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('2d317dcef8626e639bcaab4a4b1ca1e8e6647eb46d65ca8d98137cd98eb10ae7', 3, 3, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934594, 100, 1, '2018-01-12 18:35:14.117661', '2018-01-12 18:35:14.117661', 12884914176, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB+ZAt7wAAAEB8q5Of+GA0eadw+hTrTCIAoedKyFge/Kv+RUNsq7sv7pSoLAQFWqwFIvxCGBul0XhSxOomG/gWgmIiwj6a1goM', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAA=', 'AAAAAAAAAAEAAAABAAAAAQAAAAMAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', 'AAAAAQAAAAEAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{fKuTn/hgNHmncPoU60wiAKHnSshYHvyr/kVDbKu7L+6UqCwEBVqsBSL8QhgbpdF4UsTqJhv4FoJiIsI+mtYKDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('4486298e04ffb1f3620c521f81adb5207f5d12c21b08a076589d2be3d8dae543', 4, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934594, 100, 1, '2018-01-12 18:35:14.13486', '2018-01-12 18:35:14.13486', 17179873280, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQFp8rsD4Au1oeZkBT1RHIJRyxWayau3f5UjeA0w4+0LzjLEyi9nGMs8elAH4lDhhDJxCJ8HhxbG+XT/cmQsu1QA=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAEAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{WnyuwPgC7Wh5mQFPVEcglHLFZrJq7d/lSN4DTDj7QvOMsTKL2cYyzx6UAfiUOGEMnEInweHFsb5dP9yZCy7VAA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('00ab9cfce2b4c4141d8bb6768dd094bdbb1c7406710dbb3ba0ef98870f63a344', 4, 2, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934593, 100, 1, '2018-01-12 18:35:14.141373', '2018-01-12 18:35:14.141374', 17179877376, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAEnciLVAAAAQLVbII+1LeizxgncDI46KHyBt05+H92n1+R328J9zNl2fgJW2nfn3FIoLVs2qV1+CUpr121a2B7AM6HKr4nBLAI=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{tVsgj7Ut6LPGCdwMjjoofIG3Tn4f3afX5Hfbwn3M2XZ+Albad+fcUigtWzapXX4JSmvXbVrYHsAzocqvicEsAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('647eaba7f3bc5726dc1041553fe4741542ed0a2af2d098d93b0bac5b6f3c624c', 4, 3, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934594, 100, 1, '2018-01-12 18:35:14.148032', '2018-01-12 18:35:14.148032', 17179881472, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABU0NPVAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TH//////////AAAAAAAAAAEnciLVAAAAQHTUKeZaZX/yonQdzrGY0klZqwhUZd7ontUbjpQmLk+XRY8uYos+AI2Z3qqU3QF27EV4VRsVcUUvvn57fqFdzgQ=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVNDT1QAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAIAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAQAAAAEAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{dNQp5lplf/KidB3OsZjSSVmrCFRl3uie1RuOlCYuT5dFjy5iiz4AjZneqpTdAXbsRXhVGxVxRS++fnt+oV3OBA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('1d6308dc6e9617bee39a69f68176cf6f3abcf4d3617db3c766647bd198a5e442', 4, 4, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934595, 100, 1, '2018-01-12 18:35:14.15175', '2018-01-12 18:35:14.15175', 17179885568, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABQlRDAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQLEyHlSQ5gb4aQ7evOl4mZ6lSTIF7kShyso/iyP0uz3ipHocd38/dLiu7lVvMGXwo6ymJ7mixdDuNLIWiI9TbQI=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAIAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAQAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+LUAAAAAgAAAAMAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{sTIeVJDmBvhpDt686XiZnqVJMgXuRKHKyj+LI/S7PeKkehx3fz90uK7uVW8wZfCjrKYnuaLF0O40shaIj1NtAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('3b666a253313fc7a0d241ee28064eec78aaa5ebd0a7c0ae7f85259e80fad029f', 5, 1, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934595, 100, 1, '2018-01-12 18:35:14.159927', '2018-01-12 18:35:14.159927', 21474840576, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAEAAAAAAAAAAfmQLe8AAABAL6czYFvSBhdVeD4fbXOHuXFa2CDqLpFfc+QJnoiPLt/23YViURGLyfg388FKMKsbNJEgmFsCJjtgl3fj7wr/Aw==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAHAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4tQAAAACAAAAAwAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{L6czYFvSBhdVeD4fbXOHuXFa2CDqLpFfc+QJnoiPLt/23YViURGLyfg388FKMKsbNJEgmFsCJjtgl3fj7wr/Aw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('d9d6b816a0a3c640637d48fe33fa00f9ef116103c204834a1c18a9765803fd5d', 5, 2, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934596, 100, 1, '2018-01-12 18:35:14.162581', '2018-01-12 18:35:14.162581', 21474844672, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABQlRDAAAAAAEAAAAAAAAAAfmQLe8AAABAMIB8sKelxTqFOLPILjB0nItcfrGrCwursIhshVeKHSw2IC4pmCeg7KGDOLpfUCLc23n5HeTsxJsb/CrHJF/XDQ==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAHAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAA', 'AAAAAQAAAAEAAAAFAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+JwAAAAAgAAAAQAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAA', '{MIB8sKelxTqFOLPILjB0nItcfrGrCwursIhshVeKHSw2IC4pmCeg7KGDOLpfUCLc23n5HeTsxJsb/CrHJF/XDQ==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('6ab66668ea2801de6a7239c94d44e5d41f361812607748125da372b27b66cd3c', 5, 3, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934597, 100, 1, '2018-01-12 18:35:14.165242', '2018-01-12 18:35:14.165242', 21474848768, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAFAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAEAAAAAAAAAAfmQLe8AAABA78VZpv8Z9a3XM9gv6hyMLt2bBrZ5sKsFRU4GKXYtxY2MkAt9J9ENrSRZn1M0jlx9FFGtCvtFFZi8DhxvqDyaBQ==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAHAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAUAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAA', 'AAAAAQAAAAEAAAAFAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+IMAAAAAgAAAAUAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAA', '{78VZpv8Z9a3XM9gv6hyMLt2bBrZ5sKsFRU4GKXYtxY2MkAt9J9ENrSRZn1M0jlx9FFGtCvtFFZi8DhxvqDyaBQ==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('2300600248f841cd5f50276fc18eb16bc88a734e7a290f287ab3a2aa92684826', 6, 1, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934598, 100, 1, '2018-01-12 18:35:14.173027', '2018-01-12 18:35:14.173027', 25769807872, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAGAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAOjUt1+AAAAAAAAAAAH5kC3vAAAAQIqp3RfP1ueB0TRJRYXnao+kmde4BDh8q0Ep7q14Q8oRNx1R9utncfpoXr7JOcqiwtgarT9k6KmMyjda97H5RgM=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADo1LdfgH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAFAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+IMAAAAAgAAAAUAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAYAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4agAAAACAAAABgAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{iqndF8/W54HRNElFhedqj6SZ17gEOHyrQSnurXhDyhE3HVH262dx+mhevsk5yqLC2BqtP2ToqYzKN1r3sflGAw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('cf0f5fcd46881458ba623f9e6e7c52489d4bd3979a4196819882bb6240b4e855', 6, 2, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934596, 100, 1, '2018-01-12 18:35:14.175383', '2018-01-12 18:35:14.175383', 25769811968, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABU0NPVAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAAAAAAGu5L5MAAAAQLSYQCC1+DGQ8srHLxi6SfnN/dn8t7mAcXlDniU3J+d6Ezg1U6lg9i0jWOsfamioYVbJ9dAiQBZyIsn7TB5cLww=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVNDT1QAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVNDT1QAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+LUAAAAAgAAAAMAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAGAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+JwAAAAAgAAAAQAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{tJhAILX4MZDyyscvGLpJ+c392fy3uYBxeUOeJTcn53oTODVTqWD2LSNY6x9qaKhhVsn10CJAFnIiyftMHlwvDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('5a48a811ec874fc9c5d77c7caeb8abcea076c1baa51b755b2a878391a089c7d1', 6, 3, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934599, 100, 1, '2018-01-12 18:35:14.177488', '2018-01-12 18:35:14.177488', 25769816064, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAHAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABQlRDAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAA8MXwgAAAAAAAAAAH5kC3vAAAAQGEXqpE9OKOxah6oBhR955A4BYmO+yuLNMMtcALlLsKj2M1e9QTlBvAzuwkgECvg2iw8qXZB2kHteYw8qoozcQA=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAPDF8IH//////////AAAAAQAAAAAAAAAA', 'AAAAAQAAAAEAAAAGAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+FEAAAAAgAAAAcAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAA', '{YReqkT04o7FqHqgGFH3nkDgFiY77K4s0wy1wAuUuwqPYzV71BOUG8DO7CSAQK+DaLDypdkHaQe15jDyqijNxAA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('4e4779f0d69db51ec4f7b73387b60c239433804d2747def21b7771e9b71d75be', 6, 4, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934600, 100, 1, '2018-01-12 18:35:14.179721', '2018-01-12 18:35:14.179721', 25769820160, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAIAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAdGp1wZQAAAAAAAAAAH5kC3vAAAAQHQFhOcK6JMPYxfRWB+xO13EkPDqkvvPG/Hp8EWDTIMTpHHi4Mqr3/SreJLUxOi3qGSqYFJHiAoK65rFYQaPEAQ=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAUAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHRqdcGUH//////////AAAAAQAAAAAAAAAA', 'AAAAAQAAAAEAAAAGAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+DgAAAAAgAAAAgAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAA', '{dAWE5wrokw9jF9FYH7E7XcSQ8OqS+88b8enwRYNMgxOkceLgyqvf9Kt4ktTE6LeoZKpgUkeICgrrmsVhBo8QBA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('fa17f7c083fddc53e8e28885be934e19bf637e287c1951be581dd05c0be93b56', 7, 1, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934595, 100, 1, '2018-01-12 18:35:14.186336', '2018-01-12 18:35:14.186336', 30064775168, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAjhvJvwQAAAAAAAAAAAAEnciLVAAAAQNI8SXbUBWJi/xf8bWtBBKonww9YpbLck1/295qxZOYN5vjFDYQLaG3b1aGWqzWZqa9FMHkJ2tAEDPjEHIMkzAw=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAYAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHRqdcGUH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHRqdcGUAAjhvJvwQAAAAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+M4AAAAAgAAAAIAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAHAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+LUAAAAAgAAAAMAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{0jxJdtQFYmL/F/xta0EEqifDD1ilstyTX/b3mrFk5g3m+MUNhAtobdvVoZarNZmpr0UweQna0AQM+MQcgyTMDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('c42c988a72ac8aed3bb9a7b7dfb96b905e33d1506f4e663360135e6c6e115078', 7, 2, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934597, 100, 1, '2018-01-12 18:35:14.187999', '2018-01-12 18:35:14.187999', 30064779264, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAAFAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAA1nUfgAAAAAAAAAAGu5L5MAAAAQAGYynFy2CKfKZyhmWMLfgmhdJtJHXW7ogTdyZ7aviECOHYJSQKPkcnMoG4N76ipkuVH6hjuxDHBJ83+HnyhbAQ=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAQAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR33ROMAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAwAAAAYAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADo1LdfgH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAcAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADonxoXoH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAGAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+JwAAAAAgAAAAQAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAHAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+IMAAAAAgAAAAUAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{AZjKcXLYIp8pnKGZYwt+CaF0m0kddbuiBN3Jntq+IQI4dglJAo+Rycygbg3vqKmS5UfqGO7EMcEnzf4efKFsBA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('142d3dbe5948eb39db1fd62d912ce67131b1b300adb015acf0f17d91a057429d', 8, 1, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934596, 100, 1, '2018-01-12 18:35:14.194461', '2018-01-12 18:35:14.194461', 34359742464, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAAAz97YAAAAAAAAAAEnciLVAAAAQHbmlPqVcxoIqzJFayddJwGRM8Vxm0BYlui3LVu9d/nB2hb/tsUWgUZLCUnNv/CPjsMTAN2LmVkYOMtCdYc+NQ8=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAAEAAAAAwAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR33ROMAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR3qRvWAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAwAAAAcAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADonxoXoH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADon+n2eH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAHAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+LUAAAAAgAAAAMAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAIAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+JwAAAAAgAAAAQAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{duaU+pVzGgirMkVrJ10nAZEzxXGbQFiW6LctW713+cHaFv+2xRaBRksJSc2/8I+OwxMA3YuZWRg4y0J1hz41Dw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('3362c9b76d85a844c739b338dbef4213ce64eca1ceb6c0d70e878975ab1477b1', 8, 2, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934598, 100, 1, '2018-01-12 18:35:14.19703', '2018-01-12 18:35:14.197031', 34359746560, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAAGAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAAS3peQAAAAAAAAAAGu5L5MAAAAQKnjaWS6Rk617nkw1/KuCffaeN1Mymuz8m9Brm0RJ1IYNKdnudV+72HsCM1Vnfnz/+iB6ERFxOsEp1mBHpUMQwk=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAQAAAAgAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR8YMG6AAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADojQte6H//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAHAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+IMAAAAAgAAAAUAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAIAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+GoAAAAAgAAAAYAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{qeNpZLpGTrXueTDX8q4J99p43UzKa7Pyb0GubREnUhg0p2e51X7vYewIzVWd+fP/6IHoREXE6wSnWYEelQxDCQ==}', 'none', NULL, NULL);


--
-- Name: history_accounts_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

<<<<<<< HEAD
INSERT INTO history_transactions VALUES ('db398eb4ae89756325643cad21c94e13bfc074b323ee83e141bf701a5d904f1b', 2, 1, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 1, 100, 1, '2018-01-22 17:00:02.865962', '2018-01-22 17:00:02.865964', 8589938688, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAACVAvkAAAAAAAAAAABVvwF9wAAAEAYjQcPT2G5hqnBmgGGeg9J8l4c1EnUlxklElH9sqZr0971F6OLWfe/m4kpFtI+sI0i1qLit5A0JyWnbhYLW5oD', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2sVNYGtQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{GI0HD09huYapwZoBhnoPSfJeHNRJ1JcZJRJR/bKma9Pe9Reji1n3v5uJKRbSPrCNItai4reQNCclp24WC1uaAw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('f97caffab8c16023a37884165cb0b3ff1aa2daf4000fef49d21efc847ddbfbea', 2, 2, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 2, 100, 1, '2018-01-22 17:00:02.872081', '2018-01-22 17:00:02.872081', 8589942784, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBmKpSgvrwKO20XCOfYfXsGEEUtwYaaEfqSu6ymJmlDma+IX6I7IggbUZMocQdZ94IMAfKdQANqXbIO7ysweeMC', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rv9MNtQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{ZiqUoL68CjttFwjn2H17BhBFLcGGmhH6kruspiZpQ5mviF+iOyIIG1GTKHEHWfeCDAHynUADal2yDu8rMHnjAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('725756b1fbdf83b08127f385efedf0909cc820b6cce71f1c0897d15427cb5add', 2, 3, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 3, 100, 1, '2018-01-22 17:00:02.875905', '2018-01-22 17:00:02.875905', 8589946880, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAADAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBj4gBQ/BAbgqf7qOotatgZUHjDlsOtDNdp7alZR5/Fk9fGj+lxEygAZWzY7/LY1Z3SF6c0qs172LhAkkvV8p0M', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rKtAUtQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/7UAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{Y+IAUPwQG4Kn+6jqLWrYGVB4w5bDrQzXae2pWUefxZPXxo/pcRMoAGVs2O/y2NWd0henNKrNe9i4QJJL1fKdDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('c780569c402c298b7b5f3f1a6a20ac1219a06df39a78fb3ac6d93ca53ad4e5ed', 3, 1, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934593, 100, 1, '2018-01-22 17:00:02.890089', '2018-01-22 17:00:02.890089', 12884905984, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAACHRlc3QuY29tAAAAAAAAAAAAAAAB+ZAt7wAAAEBHwkZcyIWmaPvEtDlR8Ed4dD1Mep2juLtHF3n5RG0jurJhKq/3MB1zR6bDHr+wow35ijK92ihjHWqTxjzKDhkO', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAMAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAAAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', 'AAAAAgAAAAMAAAACAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{R8JGXMiFpmj7xLQ5UfBHeHQ9THqdo7i7Rxd5+URtI7qyYSqv9zAdc0emwx6/sKMN+YoyvdooYx1qk8Y8yg4ZDg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('bd60680a1378ffec739e1ffa2db4cd51f58babfb714e04a52bd2b65bf8a31b4f', 3, 2, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934593, 100, 1, '2018-01-22 17:00:02.894612', '2018-01-22 17:00:02.894612', 12884910080, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABruS+TAAAAEBkz5uRgU5FxqOu8Yak7Bbdc0BtgvEJ0FjurZz/LgGwT2EX91Y81YrdSVu2NPR0lbhSAotGQlvSPYEy5vN67p4C', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAAAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{ZM+bkYFORcajrvGGpOwW3XNAbYLxCdBY7q2c/y4BsE9hF/dWPNWK3UlbtjT0dJW4UgKLRkJb0j2BMubzeu6eAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('2d317dcef8626e639bcaab4a4b1ca1e8e6647eb46d65ca8d98137cd98eb10ae7', 3, 3, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934594, 100, 1, '2018-01-22 17:00:02.897115', '2018-01-22 17:00:02.897115', 12884914176, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB+ZAt7wAAAEB8q5Of+GA0eadw+hTrTCIAoedKyFge/Kv+RUNsq7sv7pSoLAQFWqwFIvxCGBul0XhSxOomG/gWgmIiwj6a1goM', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAMAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAAAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAACVAvjOAAAAAIAAAACAAAAAAAAAAAAAAABAAAACHRlc3QuY29tAQAAAAAAAAAAAAAAAAAAAA==', 'AAAAAgAAAAMAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{fKuTn/hgNHmncPoU60wiAKHnSshYHvyr/kVDbKu7L+6UqCwEBVqsBSL8QhgbpdF4UsTqJhv4FoJiIsI+mtYKDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('00ab9cfce2b4c4141d8bb6768dd094bdbb1c7406710dbb3ba0ef98870f63a344', 4, 1, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934593, 100, 1, '2018-01-22 17:00:02.903861', '2018-01-22 17:00:02.903861', 17179873280, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAEnciLVAAAAQLVbII+1LeizxgncDI46KHyBt05+H92n1+R328J9zNl2fgJW2nfn3FIoLVs2qV1+CUpr121a2B7AM6HKr4nBLAI=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAwAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{tVsgj7Ut6LPGCdwMjjoofIG3Tn4f3afX5Hfbwn3M2XZ+Albad+fcUigtWzapXX4JSmvXbVrYHsAzocqvicEsAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('4486298e04ffb1f3620c521f81adb5207f5d12c21b08a076589d2be3d8dae543', 4, 2, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934594, 100, 1, '2018-01-22 17:00:02.906429', '2018-01-22 17:00:02.90643', 17179877376, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQFp8rsD4Au1oeZkBT1RHIJRyxWayau3f5UjeA0w4+0LzjLEyi9nGMs8elAH4lDhhDJxCJ8HhxbG+XT/cmQsu1QA=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAwAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAAAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAEAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{WnyuwPgC7Wh5mQFPVEcglHLFZrJq7d/lSN4DTDj7QvOMsTKL2cYyzx6UAfiUOGEMnEInweHFsb5dP9yZCy7VAA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('1d6308dc6e9617bee39a69f68176cf6f3abcf4d3617db3c766647bd198a5e442', 4, 3, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934595, 100, 1, '2018-01-22 17:00:02.908406', '2018-01-22 17:00:02.908406', 17179881472, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABQlRDAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQLEyHlSQ5gb4aQ7evOl4mZ6lSTIF7kShyso/iyP0uz3ipHocd38/dLiu7lVvMGXwo6ymJ7mixdDuNLIWiI9TbQI=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAwAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAEAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAIAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+LUAAAAAgAAAAMAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{sTIeVJDmBvhpDt686XiZnqVJMgXuRKHKyj+LI/S7PeKkehx3fz90uK7uVW8wZfCjrKYnuaLF0O40shaIj1NtAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('647eaba7f3bc5726dc1041553fe4741542ed0a2af2d098d93b0bac5b6f3c624c', 4, 4, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934594, 100, 1, '2018-01-22 17:00:02.910763', '2018-01-22 17:00:02.910763', 17179885568, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABU0NPVAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TH//////////AAAAAAAAAAEnciLVAAAAQHTUKeZaZX/yonQdzrGY0klZqwhUZd7ontUbjpQmLk+XRY8uYos+AI2Z3qqU3QF27EV4VRsVcUUvvn57fqFdzgQ=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVNDT1QAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAwAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAIAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{dNQp5lplf/KidB3OsZjSSVmrCFRl3uie1RuOlCYuT5dFjy5iiz4AjZneqpTdAXbsRXhVGxVxRS++fnt+oV3OBA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('3b666a253313fc7a0d241ee28064eec78aaa5ebd0a7c0ae7f85259e80fad029f', 5, 1, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934595, 100, 1, '2018-01-22 17:00:02.92348', '2018-01-22 17:00:02.92348', 21474840576, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAEAAAAAAAAAAfmQLe8AAABAL6czYFvSBhdVeD4fbXOHuXFa2CDqLpFfc+QJnoiPLt/23YViURGLyfg388FKMKsbNJEgmFsCJjtgl3fj7wr/Aw==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAHAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4tQAAAACAAAAAwAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{L6czYFvSBhdVeD4fbXOHuXFa2CDqLpFfc+QJnoiPLt/23YViURGLyfg388FKMKsbNJEgmFsCJjtgl3fj7wr/Aw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('d9d6b816a0a3c640637d48fe33fa00f9ef116103c204834a1c18a9765803fd5d', 5, 2, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934596, 100, 1, '2018-01-22 17:00:02.926127', '2018-01-22 17:00:02.926127', 21474844672, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABQlRDAAAAAAEAAAAAAAAAAfmQLe8AAABAMIB8sKelxTqFOLPILjB0nItcfrGrCwursIhshVeKHSw2IC4pmCeg7KGDOLpfUCLc23n5HeTsxJsb/CrHJF/XDQ==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAHAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAFAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+LUAAAAAgAAAAMAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4nAAAAACAAAABAAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{MIB8sKelxTqFOLPILjB0nItcfrGrCwursIhshVeKHSw2IC4pmCeg7KGDOLpfUCLc23n5HeTsxJsb/CrHJF/XDQ==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('6ab66668ea2801de6a7239c94d44e5d41f361812607748125da372b27b66cd3c', 5, 3, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934597, 100, 1, '2018-01-22 17:00:02.928822', '2018-01-22 17:00:02.928822', 21474848768, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAFAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAEAAAAAAAAAAfmQLe8AAABA78VZpv8Z9a3XM9gv6hyMLt2bBrZ5sKsFRU4GKXYtxY2MkAt9J9ENrSRZn1M0jlx9FFGtCvtFFZi8DhxvqDyaBQ==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAHAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAUAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAFAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+JwAAAAAgAAAAQAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4gwAAAACAAAABQAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{78VZpv8Z9a3XM9gv6hyMLt2bBrZ5sKsFRU4GKXYtxY2MkAt9J9ENrSRZn1M0jlx9FFGtCvtFFZi8DhxvqDyaBQ==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('2300600248f841cd5f50276fc18eb16bc88a734e7a290f287ab3a2aa92684826', 6, 1, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934598, 100, 1, '2018-01-22 17:00:02.941049', '2018-01-22 17:00:02.941049', 25769807872, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAGAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAOjUt1+AAAAAAAAAAAH5kC3vAAAAQIqp3RfP1ueB0TRJRYXnao+kmde4BDh8q0Ep7q14Q8oRNx1R9utncfpoXr7JOcqiwtgarT9k6KmMyjda97H5RgM=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADo1LdfgH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAFAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+IMAAAAAgAAAAUAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAYAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4agAAAACAAAABgAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{iqndF8/W54HRNElFhedqj6SZ17gEOHyrQSnurXhDyhE3HVH262dx+mhevsk5yqLC2BqtP2ToqYzKN1r3sflGAw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('cf0f5fcd46881458ba623f9e6e7c52489d4bd3979a4196819882bb6240b4e855', 6, 2, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934596, 100, 1, '2018-01-22 17:00:02.944292', '2018-01-22 17:00:02.944292', 25769811968, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABU0NPVAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAAAAAAGu5L5MAAAAQLSYQCC1+DGQ8srHLxi6SfnN/dn8t7mAcXlDniU3J+d6Ezg1U6lg9i0jWOsfamioYVbJ9dAiQBZyIsn7TB5cLww=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVNDT1QAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVNDT1QAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+LUAAAAAgAAAAMAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAGAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+JwAAAAAgAAAAQAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{tJhAILX4MZDyyscvGLpJ+c392fy3uYBxeUOeJTcn53oTODVTqWD2LSNY6x9qaKhhVsn10CJAFnIiyftMHlwvDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('5a48a811ec874fc9c5d77c7caeb8abcea076c1baa51b755b2a878391a089c7d1', 6, 3, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934599, 100, 1, '2018-01-22 17:00:02.947168', '2018-01-22 17:00:02.947168', 25769816064, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAHAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABQlRDAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAA8MXwgAAAAAAAAAAH5kC3vAAAAQGEXqpE9OKOxah6oBhR955A4BYmO+yuLNMMtcALlLsKj2M1e9QTlBvAzuwkgECvg2iw8qXZB2kHteYw8qoozcQA=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAPDF8IH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAGAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+GoAAAAAgAAAAYAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAYAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4UQAAAACAAAABwAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{YReqkT04o7FqHqgGFH3nkDgFiY77K4s0wy1wAuUuwqPYzV71BOUG8DO7CSAQK+DaLDypdkHaQe15jDyqijNxAA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('4e4779f0d69db51ec4f7b73387b60c239433804d2747def21b7771e9b71d75be', 6, 4, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934600, 100, 1, '2018-01-22 17:00:02.950108', '2018-01-22 17:00:02.950108', 25769820160, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAIAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAdGp1wZQAAAAAAAAAAH5kC3vAAAAQHQFhOcK6JMPYxfRWB+xO13EkPDqkvvPG/Hp8EWDTIMTpHHi4Mqr3/SreJLUxOi3qGSqYFJHiAoK65rFYQaPEAQ=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAUAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHRqdcGUH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAGAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+FEAAAAAgAAAAcAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAYAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4OAAAAACAAAACAAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{dAWE5wrokw9jF9FYH7E7XcSQ8OqS+88b8enwRYNMgxOkceLgyqvf9Kt4ktTE6LeoZKpgUkeICgrrmsVhBo8QBA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('c42c988a72ac8aed3bb9a7b7dfb96b905e33d1506f4e663360135e6c6e115078', 7, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934597, 100, 1, '2018-01-22 17:00:02.957622', '2018-01-22 17:00:02.957622', 30064775168, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAAFAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAA1nUfgAAAAAAAAAAGu5L5MAAAAQAGYynFy2CKfKZyhmWMLfgmhdJtJHXW7ogTdyZ7aviECOHYJSQKPkcnMoG4N76ipkuVH6hjuxDHBJ83+HnyhbAQ=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAAEAAAAAwAAAAYAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHRqdcGUH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR33ROMH//////////AAAAAQAAAAAAAAAAAAAAAwAAAAYAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADo1LdfgH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAcAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADonxoXoH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAGAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+JwAAAAAgAAAAQAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAHAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+IMAAAAAgAAAAUAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{AZjKcXLYIp8pnKGZYwt+CaF0m0kddbuiBN3Jntq+IQI4dglJAo+Rycygbg3vqKmS5UfqGO7EMcEnzf4efKFsBA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('fa17f7c083fddc53e8e28885be934e19bf637e287c1951be581dd05c0be93b56', 7, 2, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934595, 100, 1, '2018-01-22 17:00:02.960622', '2018-01-22 17:00:02.960622', 30064779264, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAjhvJvwQAAAAAAAAAAAAEnciLVAAAAQNI8SXbUBWJi/xf8bWtBBKonww9YpbLck1/295qxZOYN5vjFDYQLaG3b1aGWqzWZqa9FMHkJ2tAEDPjEHIMkzAw=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR33ROMH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR33ROMAAjhvJvwQAAAAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+M4AAAAAgAAAAIAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAHAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+LUAAAAAgAAAAMAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{0jxJdtQFYmL/F/xta0EEqifDD1ilstyTX/b3mrFk5g3m+MUNhAtobdvVoZarNZmpr0UweQna0AQM+MQcgyTMDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('3362c9b76d85a844c739b338dbef4213ce64eca1ceb6c0d70e878975ab1477b1', 8, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934598, 100, 1, '2018-01-22 17:00:02.973511', '2018-01-22 17:00:02.973511', 34359742464, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAAGAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAAS3peQAAAAAAAAAAGu5L5MAAAAQKnjaWS6Rk617nkw1/KuCffaeN1Mymuz8m9Brm0RJ1IYNKdnudV+72HsCM1Vnfnz/+iB6ERFxOsEp1mBHpUMQwk=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAAEAAAAAwAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR33ROMAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR8lLlwAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAwAAAAcAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADonxoXoH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADojDuAEH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAHAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+IMAAAAAgAAAAUAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAIAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+GoAAAAAgAAAAYAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{qeNpZLpGTrXueTDX8q4J99p43UzKa7Pyb0GubREnUhg0p2e51X7vYewIzVWd+fP/6IHoREXE6wSnWYEelQxDCQ==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('142d3dbe5948eb39db1fd62d912ce67131b1b300adb015acf0f17d91a057429d', 8, 2, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934596, 100, 1, '2018-01-22 17:00:02.976513', '2018-01-22 17:00:02.976513', 34359746560, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAAAz97YAAAAAAAAAAEnciLVAAAAQHbmlPqVcxoIqzJFayddJwGRM8Vxm0BYlui3LVu9d/nB2hb/tsUWgUZLCUnNv/CPjsMTAN2LmVkYOMtCdYc+NQ8=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAAEAAAAAwAAAAgAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR8lLlwAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR8YMG6AAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAwAAAAgAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADojDuAEH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADojQte6H//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAHAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+LUAAAAAgAAAAMAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAIAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+JwAAAAAgAAAAQAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{duaU+pVzGgirMkVrJ10nAZEzxXGbQFiW6LctW713+cHaFv+2xRaBRksJSc2/8I+OwxMA3YuZWRg4y0J1hz41Dw==}', 'none', NULL, NULL);
=======
SELECT pg_catalog.setval('history_accounts_id_seq', 4, true);


--
-- Name: history_assets_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('history_assets_id_seq', 3, true);


--
-- Name: history_operation_participants_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('history_operation_participants_id_seq', 34, true);


--
-- Name: history_transaction_participants_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

<<<<<<< HEAD
INSERT INTO history_transactions VALUES ('db398eb4ae89756325643cad21c94e13bfc074b323ee83e141bf701a5d904f1b', 2, 1, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 1, 100, 1, '2017-12-15 22:07:19.612811', '2017-12-15 22:07:19.612811', 8589938688, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAACVAvkAAAAAAAAAAABVvwF9wAAAEAYjQcPT2G5hqnBmgGGeg9J8l4c1EnUlxklElH9sqZr0971F6OLWfe/m4kpFtI+sI0i1qLit5A0JyWnbhYLW5oD', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2sVNYGtQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{GI0HD09huYapwZoBhnoPSfJeHNRJ1JcZJRJR/bKma9Pe9Reji1n3v5uJKRbSPrCNItai4reQNCclp24WC1uaAw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('f97caffab8c16023a37884165cb0b3ff1aa2daf4000fef49d21efc847ddbfbea', 2, 2, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 2, 100, 1, '2017-12-15 22:07:19.622195', '2017-12-15 22:07:19.622195', 8589942784, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBmKpSgvrwKO20XCOfYfXsGEEUtwYaaEfqSu6ymJmlDma+IX6I7IggbUZMocQdZ94IMAfKdQANqXbIO7ysweeMC', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rv9MNtQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{ZiqUoL68CjttFwjn2H17BhBFLcGGmhH6kruspiZpQ5mviF+iOyIIG1GTKHEHWfeCDAHynUADal2yDu8rMHnjAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('725756b1fbdf83b08127f385efedf0909cc820b6cce71f1c0897d15427cb5add', 2, 3, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 3, 100, 1, '2017-12-15 22:07:19.627679', '2017-12-15 22:07:19.62768', 8589946880, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAADAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBj4gBQ/BAbgqf7qOotatgZUHjDlsOtDNdp7alZR5/Fk9fGj+lxEygAZWzY7/LY1Z3SF6c0qs172LhAkkvV8p0M', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rKtAUtQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/7UAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{Y+IAUPwQG4Kn+6jqLWrYGVB4w5bDrQzXae2pWUefxZPXxo/pcRMoAGVs2O/y2NWd0henNKrNe9i4QJJL1fKdDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('bd60680a1378ffec739e1ffa2db4cd51f58babfb714e04a52bd2b65bf8a31b4f', 3, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934593, 100, 1, '2017-12-15 22:07:19.645405', '2017-12-15 22:07:19.645405', 12884905984, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABruS+TAAAAEBkz5uRgU5FxqOu8Yak7Bbdc0BtgvEJ0FjurZz/LgGwT2EX91Y81YrdSVu2NPR0lbhSAotGQlvSPYEy5vN67p4C', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAAAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{ZM+bkYFORcajrvGGpOwW3XNAbYLxCdBY7q2c/y4BsE9hF/dWPNWK3UlbtjT0dJW4UgKLRkJb0j2BMubzeu6eAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('c780569c402c298b7b5f3f1a6a20ac1219a06df39a78fb3ac6d93ca53ad4e5ed', 3, 2, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934593, 100, 1, '2017-12-15 22:07:19.649446', '2017-12-15 22:07:19.649446', 12884910080, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAACHRlc3QuY29tAAAAAAAAAAAAAAAB+ZAt7wAAAEBHwkZcyIWmaPvEtDlR8Ed4dD1Mep2juLtHF3n5RG0jurJhKq/3MB1zR6bDHr+wow35ijK92ihjHWqTxjzKDhkO', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAMAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAAAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', 'AAAAAgAAAAMAAAACAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{R8JGXMiFpmj7xLQ5UfBHeHQ9THqdo7i7Rxd5+URtI7qyYSqv9zAdc0emwx6/sKMN+YoyvdooYx1qk8Y8yg4ZDg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('2d317dcef8626e639bcaab4a4b1ca1e8e6647eb46d65ca8d98137cd98eb10ae7', 3, 3, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934594, 100, 1, '2017-12-15 22:07:19.65253', '2017-12-15 22:07:19.65253', 12884914176, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB+ZAt7wAAAEB8q5Of+GA0eadw+hTrTCIAoedKyFge/Kv+RUNsq7sv7pSoLAQFWqwFIvxCGBul0XhSxOomG/gWgmIiwj6a1goM', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAFAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAMAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAAAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAACVAvjOAAAAAIAAAACAAAAAAAAAAAAAAABAAAACHRlc3QuY29tAQAAAAAAAAAAAAAAAAAAAA==', 'AAAAAgAAAAMAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{fKuTn/hgNHmncPoU60wiAKHnSshYHvyr/kVDbKu7L+6UqCwEBVqsBSL8QhgbpdF4UsTqJhv4FoJiIsI+mtYKDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('00ab9cfce2b4c4141d8bb6768dd094bdbb1c7406710dbb3ba0ef98870f63a344', 4, 1, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934593, 100, 1, '2017-12-15 22:07:19.659144', '2017-12-15 22:07:19.659144', 17179873280, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAEnciLVAAAAQLVbII+1LeizxgncDI46KHyBt05+H92n1+R328J9zNl2fgJW2nfn3FIoLVs2qV1+CUpr121a2B7AM6HKr4nBLAI=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAwAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{tVsgj7Ut6LPGCdwMjjoofIG3Tn4f3afX5Hfbwn3M2XZ+Albad+fcUigtWzapXX4JSmvXbVrYHsAzocqvicEsAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('4486298e04ffb1f3620c521f81adb5207f5d12c21b08a076589d2be3d8dae543', 4, 2, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934594, 100, 1, '2017-12-15 22:07:19.661903', '2017-12-15 22:07:19.661903', 17179877376, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQFp8rsD4Au1oeZkBT1RHIJRyxWayau3f5UjeA0w4+0LzjLEyi9nGMs8elAH4lDhhDJxCJ8HhxbG+XT/cmQsu1QA=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAwAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAAAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAEAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{WnyuwPgC7Wh5mQFPVEcglHLFZrJq7d/lSN4DTDj7QvOMsTKL2cYyzx6UAfiUOGEMnEInweHFsb5dP9yZCy7VAA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('1d6308dc6e9617bee39a69f68176cf6f3abcf4d3617db3c766647bd198a5e442', 4, 3, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934595, 100, 1, '2017-12-15 22:07:19.663778', '2017-12-15 22:07:19.663778', 17179881472, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABQlRDAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQLEyHlSQ5gb4aQ7evOl4mZ6lSTIF7kShyso/iyP0uz3ipHocd38/dLiu7lVvMGXwo6ymJ7mixdDuNLIWiI9TbQI=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAwAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAEAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4tQAAAACAAAAAwAAAAIAAAAAAAAAAgAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+LUAAAAAgAAAAMAAAAAAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{sTIeVJDmBvhpDt686XiZnqVJMgXuRKHKyj+LI/S7PeKkehx3fz90uK7uVW8wZfCjrKYnuaLF0O40shaIj1NtAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('647eaba7f3bc5726dc1041553fe4741542ed0a2af2d098d93b0bac5b6f3c624c', 4, 4, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934594, 100, 1, '2017-12-15 22:07:19.665543', '2017-12-15 22:07:19.665543', 17179885568, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABU0NPVAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TH//////////AAAAAAAAAAEnciLVAAAAQHTUKeZaZX/yonQdzrGY0klZqwhUZd7ontUbjpQmLk+XRY8uYos+AI2Z3qqU3QF27EV4VRsVcUUvvn57fqFdzgQ=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVNDT1QAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAwAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL4zgAAAACAAAAAgAAAAIAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{dNQp5lplf/KidB3OsZjSSVmrCFRl3uie1RuOlCYuT5dFjy5iiz4AjZneqpTdAXbsRXhVGxVxRS++fnt+oV3OBA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('3b666a253313fc7a0d241ee28064eec78aaa5ebd0a7c0ae7f85259e80fad029f', 5, 1, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934595, 100, 1, '2017-12-15 22:07:19.670865', '2017-12-15 22:07:19.670865', 21474840576, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAEAAAAAAAAAAfmQLe8AAABAL6czYFvSBhdVeD4fbXOHuXFa2CDqLpFfc+QJnoiPLt/23YViURGLyfg388FKMKsbNJEgmFsCJjtgl3fj7wr/Aw==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAHAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAADAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+M4AAAAAgAAAAIAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4tQAAAACAAAAAwAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{L6czYFvSBhdVeD4fbXOHuXFa2CDqLpFfc+QJnoiPLt/23YViURGLyfg388FKMKsbNJEgmFsCJjtgl3fj7wr/Aw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('d9d6b816a0a3c640637d48fe33fa00f9ef116103c204834a1c18a9765803fd5d', 5, 2, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934596, 100, 1, '2017-12-15 22:07:19.673261', '2017-12-15 22:07:19.673261', 21474844672, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABQlRDAAAAAAEAAAAAAAAAAfmQLe8AAABAMIB8sKelxTqFOLPILjB0nItcfrGrCwursIhshVeKHSw2IC4pmCeg7KGDOLpfUCLc23n5HeTsxJsb/CrHJF/XDQ==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAHAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAFAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+LUAAAAAgAAAAMAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4nAAAAACAAAABAAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{MIB8sKelxTqFOLPILjB0nItcfrGrCwursIhshVeKHSw2IC4pmCeg7KGDOLpfUCLc23n5HeTsxJsb/CrHJF/XDQ==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('6ab66668ea2801de6a7239c94d44e5d41f361812607748125da372b27b66cd3c', 5, 3, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934597, 100, 1, '2017-12-15 22:07:19.67552', '2017-12-15 22:07:19.67552', 21474848768, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAFAAAAAAAAAAAAAAABAAAAAAAAAAcAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAEAAAAAAAAAAfmQLe8AAABA78VZpv8Z9a3XM9gv6hyMLt2bBrZ5sKsFRU4GKXYtxY2MkAt9J9ENrSRZn1M0jlx9FFGtCvtFFZi8DhxvqDyaBQ==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAHAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAAAAAAAAAAAAAAAAAQAAAAUAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAFAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+JwAAAAAgAAAAQAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4gwAAAACAAAABQAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{78VZpv8Z9a3XM9gv6hyMLt2bBrZ5sKsFRU4GKXYtxY2MkAt9J9ENrSRZn1M0jlx9FFGtCvtFFZi8DhxvqDyaBQ==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('2300600248f841cd5f50276fc18eb16bc88a734e7a290f287ab3a2aa92684826', 6, 1, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934598, 100, 1, '2017-12-15 22:07:19.681626', '2017-12-15 22:07:19.681626', 25769807872, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAGAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAOjUt1+AAAAAAAAAAAH5kC3vAAAAQIqp3RfP1ueB0TRJRYXnao+kmde4BDh8q0Ep7q14Q8oRNx1R9utncfpoXr7JOcqiwtgarT9k6KmMyjda97H5RgM=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADo1LdfgH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAFAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+IMAAAAAgAAAAUAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAYAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4agAAAACAAAABgAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{iqndF8/W54HRNElFhedqj6SZ17gEOHyrQSnurXhDyhE3HVH262dx+mhevsk5yqLC2BqtP2ToqYzKN1r3sflGAw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('cf0f5fcd46881458ba623f9e6e7c52489d4bd3979a4196819882bb6240b4e855', 6, 2, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934596, 100, 1, '2017-12-15 22:07:19.685584', '2017-12-15 22:07:19.685584', 25769811968, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABU0NPVAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAAAAAAGu5L5MAAAAQLSYQCC1+DGQ8srHLxi6SfnN/dn8t7mAcXlDniU3J+d6Ezg1U6lg9i0jWOsfamioYVbJ9dAiQBZyIsn7TB5cLww=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAQAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVNDT1QAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVNDT1QAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+LUAAAAAgAAAAMAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAGAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+JwAAAAAgAAAAQAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{tJhAILX4MZDyyscvGLpJ+c392fy3uYBxeUOeJTcn53oTODVTqWD2LSNY6x9qaKhhVsn10CJAFnIiyftMHlwvDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('5a48a811ec874fc9c5d77c7caeb8abcea076c1baa51b755b2a878391a089c7d1', 6, 3, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934599, 100, 1, '2017-12-15 22:07:19.688142', '2017-12-15 22:07:19.688142', 25769816064, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAHAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABQlRDAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAA8MXwgAAAAAAAAAAH5kC3vAAAAQGEXqpE9OKOxah6oBhR955A4BYmO+yuLNMMtcALlLsKj2M1e9QTlBvAzuwkgECvg2iw8qXZB2kHteYw8qoozcQA=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAUAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAUJUQwAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAPDF8IH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAGAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+GoAAAAAgAAAAYAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAYAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4UQAAAACAAAABwAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{YReqkT04o7FqHqgGFH3nkDgFiY77K4s0wy1wAuUuwqPYzV71BOUG8DO7CSAQK+DaLDypdkHaQe15jDyqijNxAA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('4e4779f0d69db51ec4f7b73387b60c239433804d2747def21b7771e9b71d75be', 6, 4, 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4', 8589934600, 100, 1, '2017-12-15 22:07:19.690851', '2017-12-15 22:07:19.690851', 25769820160, 'AAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAZAAAAAIAAAAIAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAdGp1wZQAAAAAAAAAAH5kC3vAAAAQHQFhOcK6JMPYxfRWB+xO13EkPDqkvvPG/Hp8EWDTIMTpHHi4Mqr3/SreJLUxOi3qGSqYFJHiAoK65rFYQaPEAQ=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAUAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAYAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHRqdcGUH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAGAAAAAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAJUC+FEAAAAAgAAAAcAAAAAAAAAAAAAAAEAAAAIdGVzdC5jb20BAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAYAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL4OAAAAACAAAACAAAAAAAAAAAAAAAAQAAAAh0ZXN0LmNvbQEAAAAAAAAAAAAAAAAAAAA=', '{dAWE5wrokw9jF9FYH7E7XcSQ8OqS+88b8enwRYNMgxOkceLgyqvf9Kt4ktTE6LeoZKpgUkeICgrrmsVhBo8QBA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('fa17f7c083fddc53e8e28885be934e19bf637e287c1951be581dd05c0be93b56', 7, 1, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934595, 100, 1, '2017-12-15 22:07:19.703329', '2017-12-15 22:07:19.703329', 30064775168, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAADAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAjhvJvwQAAAAAAAAAAAAEnciLVAAAAQNI8SXbUBWJi/xf8bWtBBKonww9YpbLck1/295qxZOYN5vjFDYQLaG3b1aGWqzWZqa9FMHkJ2tAEDPjEHIMkzAw=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAwAAAAYAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHRqdcGUH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHRqdcGUAAjhvJvwQAAAAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAEAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+M4AAAAAgAAAAIAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAHAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+LUAAAAAgAAAAMAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{0jxJdtQFYmL/F/xta0EEqifDD1ilstyTX/b3mrFk5g3m+MUNhAtobdvVoZarNZmpr0UweQna0AQM+MQcgyTMDA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('c42c988a72ac8aed3bb9a7b7dfb96b905e33d1506f4e663360135e6c6e115078', 7, 2, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934597, 100, 1, '2017-12-15 22:07:19.705421', '2017-12-15 22:07:19.705421', 30064779264, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAAFAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAA1nUfgAAAAAAAAAAGu5L5MAAAAQAGYynFy2CKfKZyhmWMLfgmhdJtJHXW7ogTdyZ7aviECOHYJSQKPkcnMoG4N76ipkuVH6hjuxDHBJ83+HnyhbAQ=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAAEAAAAAwAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHRqdcGUAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAQAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR33ROMAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAwAAAAYAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADo1LdfgH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAcAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADonxoXoH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAGAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+JwAAAAAgAAAAQAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAHAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+IMAAAAAgAAAAUAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{AZjKcXLYIp8pnKGZYwt+CaF0m0kddbuiBN3Jntq+IQI4dglJAo+Rycygbg3vqKmS5UfqGO7EMcEnzf4efKFsBA==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('3362c9b76d85a844c739b338dbef4213ce64eca1ceb6c0d70e878975ab1477b1', 8, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934598, 100, 1, '2017-12-15 22:07:19.712806', '2017-12-15 22:07:19.712806', 34359742464, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAAGAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAAS3peQAAAAAAAAAAGu5L5MAAAAQKnjaWS6Rk617nkw1/KuCffaeN1Mymuz8m9Brm0RJ1IYNKdnudV+72HsCM1Vnfnz/+iB6ERFxOsEp1mBHpUMQwk=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAAEAAAAAwAAAAcAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR33ROMAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR8lLlwAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAwAAAAcAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADonxoXoH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADojDuAEH//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAHAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+IMAAAAAgAAAAUAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAIAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+GoAAAAAgAAAAYAAAACAAAAAAAAAAIAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{qeNpZLpGTrXueTDX8q4J99p43UzKa7Pyb0GubREnUhg0p2e51X7vYewIzVWd+fP/6IHoREXE6wSnWYEelQxDCQ==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('142d3dbe5948eb39db1fd62d912ce67131b1b300adb015acf0f17d91a057429d', 8, 2, 'GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 8589934596, 100, 1, '2017-12-15 22:07:19.71595', '2017-12-15 22:07:19.71595', 34359746560, 'AAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAZAAAAAIAAAAEAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt7wAAAAAAz97YAAAAAAAAAAEnciLVAAAAQHbmlPqVcxoIqzJFayddJwGRM8Vxm0BYlui3LVu9d/nB2hb/tsUWgUZLCUnNv/CPjsMTAN2LmVkYOMtCdYc+NQ8=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAAEAAAAAwAAAAgAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR8lLlwAAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAHR8YMG6AAjhvJvwQAAAAAAAQAAAAAAAAAAAAAAAwAAAAgAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADojDuAEH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAgAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAADojQte6H//////////AAAAAQAAAAAAAAAA', 'AAAAAgAAAAMAAAAHAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+LUAAAAAgAAAAMAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAIAAAAAAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAJUC+JwAAAAAgAAAAQAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{duaU+pVzGgirMkVrJ10nAZEzxXGbQFiW6LctW713+cHaFv+2xRaBRksJSc2/8I+OwxMA3YuZWRg4y0J1hz41Dw==}', 'none', NULL, NULL);
=======
SELECT pg_catalog.setval('history_transaction_participants_id_seq', 34, true);
>>>>>>> add price to trade ingestion
>>>>>>> add price to trade ingestion


--
-- Name: asset_stats asset_stats_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY asset_stats
    ADD CONSTRAINT asset_stats_pkey PRIMARY KEY (id);


--
-- Name: gorp_migrations gorp_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY gorp_migrations
    ADD CONSTRAINT gorp_migrations_pkey PRIMARY KEY (id);


--
-- Name: history_assets history_assets_asset_code_asset_type_asset_issuer_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_assets
    ADD CONSTRAINT history_assets_asset_code_asset_type_asset_issuer_key UNIQUE (asset_code, asset_type, asset_issuer);


--
-- Name: history_assets history_assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_assets
    ADD CONSTRAINT history_assets_pkey PRIMARY KEY (id);


--
-- Name: history_operation_participants history_operation_participants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_operation_participants
    ADD CONSTRAINT history_operation_participants_pkey PRIMARY KEY (id);


--
-- Name: history_transaction_participants history_transaction_participants_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_transaction_participants
    ADD CONSTRAINT history_transaction_participants_pkey PRIMARY KEY (id);


--
-- Name: asset_by_code; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX asset_by_code ON history_assets USING btree (asset_code);


--
-- Name: asset_by_issuer; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX asset_by_issuer ON history_assets USING btree (asset_issuer);


--
-- Name: by_account; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX by_account ON history_transactions USING btree (account, account_sequence);


--
-- Name: by_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX by_hash ON history_transactions USING btree (transaction_hash);


--
-- Name: by_ledger; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX by_ledger ON history_transactions USING btree (ledger_sequence, application_order);


--
-- Name: hist_e_by_order; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hist_e_by_order ON history_effects USING btree (history_operation_id, "order");


--
-- Name: hist_e_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hist_e_id ON history_effects USING btree (history_account_id, history_operation_id, "order");


--
-- Name: hist_op_p_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hist_op_p_id ON history_operation_participants USING btree (history_account_id, history_operation_id);


--
-- Name: hist_tx_p_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hist_tx_p_id ON history_transaction_participants USING btree (history_account_id, history_transaction_id);


--
-- Name: hop_by_hoid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX hop_by_hoid ON history_operation_participants USING btree (history_operation_id);


--
-- Name: hs_ledger_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hs_ledger_by_id ON history_ledgers USING btree (id);


--
-- Name: hs_transaction_by_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX hs_transaction_by_id ON history_transactions USING btree (id);


--
-- Name: htp_by_htid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX htp_by_htid ON history_transaction_participants USING btree (history_transaction_id);


--
-- Name: htrd_by_offer; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX htrd_by_offer ON history_trades USING btree (offer_id);


--
-- Name: htrd_counter_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX htrd_counter_lookup ON history_trades USING btree (counter_asset_id);


--
-- Name: htrd_pair_time_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX htrd_pair_time_lookup ON history_trades USING btree (base_asset_id, counter_asset_id, ledger_closed_at);


--
-- Name: htrd_pid; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX htrd_pid ON history_trades USING btree (history_operation_id, "order");


--
-- Name: htrd_time_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX htrd_time_lookup ON history_trades USING btree (ledger_closed_at);


--
-- Name: index_history_accounts_on_address; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_accounts_on_address ON history_accounts USING btree (address);


--
-- Name: index_history_accounts_on_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_accounts_on_id ON history_accounts USING btree (id);


--
-- Name: index_history_effects_on_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_effects_on_type ON history_effects USING btree (type);


--
-- Name: index_history_ledgers_on_closed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_ledgers_on_closed_at ON history_ledgers USING btree (closed_at);


--
-- Name: index_history_ledgers_on_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_ledgers_on_id ON history_ledgers USING btree (id);


--
-- Name: index_history_ledgers_on_importer_version; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_ledgers_on_importer_version ON history_ledgers USING btree (importer_version);


--
-- Name: index_history_ledgers_on_ledger_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_ledgers_on_ledger_hash ON history_ledgers USING btree (ledger_hash);


--
-- Name: index_history_ledgers_on_previous_ledger_hash; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_ledgers_on_previous_ledger_hash ON history_ledgers USING btree (previous_ledger_hash);


--
-- Name: index_history_ledgers_on_sequence; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_ledgers_on_sequence ON history_ledgers USING btree (sequence);


--
-- Name: index_history_operations_on_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_operations_on_id ON history_operations USING btree (id);


--
-- Name: index_history_operations_on_transaction_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_operations_on_transaction_id ON history_operations USING btree (transaction_id);


--
-- Name: index_history_operations_on_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX index_history_operations_on_type ON history_operations USING btree (type);


--
-- Name: index_history_transactions_on_id; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX index_history_transactions_on_id ON history_transactions USING btree (id);


--
-- Name: trade_effects_by_order_book; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX trade_effects_by_order_book ON history_effects USING btree (((details ->> 'sold_asset_type'::text)), ((details ->> 'sold_asset_code'::text)), ((details ->> 'sold_asset_issuer'::text)), ((details ->> 'bought_asset_type'::text)), ((details ->> 'bought_asset_code'::text)), ((details ->> 'bought_asset_issuer'::text))) WHERE (type = 33);


--
-- Name: asset_stats asset_stats_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY asset_stats
    ADD CONSTRAINT asset_stats_id_fkey FOREIGN KEY (id) REFERENCES history_assets(id) ON UPDATE RESTRICT ON DELETE CASCADE;


--
-- Name: history_trades history_trades_base_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_trades
    ADD CONSTRAINT history_trades_base_account_id_fkey FOREIGN KEY (base_account_id) REFERENCES history_accounts(id);


--
-- Name: history_trades history_trades_base_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_trades
    ADD CONSTRAINT history_trades_base_asset_id_fkey FOREIGN KEY (base_asset_id) REFERENCES history_assets(id);


--
-- Name: history_trades history_trades_counter_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_trades
    ADD CONSTRAINT history_trades_counter_account_id_fkey FOREIGN KEY (counter_account_id) REFERENCES history_accounts(id);


--
-- Name: history_trades history_trades_counter_asset_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY history_trades
    ADD CONSTRAINT history_trades_counter_asset_id_fkey FOREIGN KEY (counter_asset_id) REFERENCES history_assets(id);


--
-- PostgreSQL database dump complete
--

