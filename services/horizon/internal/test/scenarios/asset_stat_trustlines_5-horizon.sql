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

INSERT INTO asset_stats VALUES (1, 0, 1, 0, '');


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


--
-- Data for Name: history_assets; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_assets VALUES (1, 'credit_alphanum4', 'USD', 'GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4');


--
-- Data for Name: history_effects; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_effects VALUES (2, 8589938689, 1, 0, '{"starting_balance": "1000.0000000"}');
INSERT INTO history_effects VALUES (1, 8589938689, 2, 3, '{"amount": "1000.0000000", "asset_type": "native"}');
INSERT INTO history_effects VALUES (2, 8589938689, 3, 10, '{"weight": 1, "public_key": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (3, 8589942785, 1, 0, '{"starting_balance": "1000.0000000"}');
INSERT INTO history_effects VALUES (1, 8589942785, 2, 3, '{"amount": "1000.0000000", "asset_type": "native"}');
INSERT INTO history_effects VALUES (3, 8589942785, 3, 10, '{"weight": 1, "public_key": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU"}');
INSERT INTO history_effects VALUES (3, 12884905985, 1, 20, '{"limit": "922337203685.4775807", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}');
INSERT INTO history_effects VALUES (2, 17179873281, 1, 2, '{"amount": "10.0000000", "asset_type": "native"}');
INSERT INTO history_effects VALUES (3, 17179873281, 2, 3, '{"amount": "10.0000000", "asset_type": "native"}');


--
-- Data for Name: history_ledgers; Type: TABLE DATA; Schema: public; Owner: -
--

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2018-01-22 16:59:35.059336', '2018-01-22 16:59:35.059336', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (2, '9216b7f9bddf2f59b4794bac6eefe743a2ba4120638f1b3250885a5e0d29852a', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 2, 2, '2018-01-22 16:59:32', '2018-01-22 16:59:35.074323', '2018-01-22 16:59:35.074323', 8589934592, 11, 1000000000000000000, 200, 100, 100000000, 10000, 9, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76Zu5H9cHd0qAsZj9Q6Y8yIIc6utkf/CO+Lm/0KbsfrzPUAAAAAWmYYdAAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAAP0PjS3Rf7hs8xVIBg2g8xKvqOO/n/g/XwgPR/HB7qe9psd7TDAx6H99TBuFmQ9oJnYcpBNFtnkR7EZW+HhFy8AAAAAIN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (3, '4f289bdfc2d14c734feb345f3cbe91c8d5b865dd586bf964d403af74495917ea', '9216b7f9bddf2f59b4794bac6eefe743a2ba4120638f1b3250885a5e0d29852a', 1, 1, '2018-01-22 16:59:33', '2018-01-22 16:59:35.095762', '2018-01-22 16:59:35.095762', 12884901888, 11, 1000000000000000000, 300, 100, 100000000, 10000, 9, 'AAAACZIWt/m93y9ZtHlLrG7v50OiukEgY48bMlCIWl4NKYUqn4pOLMN77EV1Y5+cXkpsto15tQXxGhL4snEbA6Kk0RoAAAAAWmYYdQAAAAAAAAAALLKPbMojH+RR+TSBDKGB/tufH2mL12ccCHr1Jn27yPBxAcwOP+mAeEZ1JMAE2HMIT1AhPX0S/h1DXUX7jq0gGwAAAAMN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (4, '44e8ec0e5cd862f636b1155082c691f062cc186f24282dc28dedd933f28a0eee', '4f289bdfc2d14c734feb345f3cbe91c8d5b865dd586bf964d403af74495917ea', 1, 1, '2018-01-22 16:59:34', '2018-01-22 16:59:35.108043', '2018-01-22 16:59:35.108043', 17179869184, 11, 1000000000000000000, 400, 100, 100000000, 10000, 9, 'AAAACU8om9/C0UxzT+s0Xzy+kcjVuGXdWGv5ZNQDr3RJWRfq0KoYC1qq64igLlyMQvdLN327X10X0s/mwNPjG9iuV74AAAAAWmYYdgAAAAAAAAAA6Ap3Xy0qNN834YP/UI/nracOqnsrZT8u288mLAvc1Bw+LAe4Zal58aTFC5O7JGhjNy90IBQybFvf+2pLKRraJQAAAAQN4Lazp2QAAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (5, '60b6e7bc23d2c5a35dc83e76b4af1c4cd690655ef3abc35acecd8f0257ea9d92', '44e8ec0e5cd862f636b1155082c691f062cc186f24282dc28dedd933f28a0eee', 0, 0, '2018-01-22 16:59:35', '2018-01-22 16:59:35.115814', '2018-01-22 16:59:35.115814', 21474836480, 11, 1000000000000000000, 400, 100, 100000000, 10000, 9, 'AAAACUTo7A5c2GL2NrEVUILGkfBizBhvJCgtwo3t2TPyig7uCME2I+3sxsbbILoA5u0k02ZCLrgYBmGfkmggtyYkl7QAAAAAWmYYdwAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERk+LAe4Zal58aTFC5O7JGhjNy90IBQybFvf+2pLKRraJQAAAAUN4Lazp2QAAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
=======
>>>>>>> add price to trade query and /trades endpoint
=======
>>>>>>> wip aggregation test passing
=======
>>>>>>> tests passing
<<<<<<< HEAD
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2017-12-15 22:06:03.829383', '2017-12-15 22:06:03.829383', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (2, 'aba6703555dd751f687d9ffab404430514c27f765926ca44f56f47c3971e0acc', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 2, 2, '2017-12-15 22:06:00', '2017-12-15 22:06:03.846266', '2017-12-15 22:06:03.846267', 8589934592, 11, 1000000000000000000, 200, 100, 100000000, 10000, 9, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76Zu5H9cHd0qAsZj9Q6Y8yIIc6utkf/CO+Lm/0KbsfrzPUAAAAAWjRHSAAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAAP0PjS3Rf7hs8xVIBg2g8xKvqOO/n/g/XwgPR/HB7qe9psd7TDAx6H99TBuFmQ9oJnYcpBNFtnkR7EZW+HhFy8AAAAAIN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (3, '32cf386a2484899c1ab9bcf164300b58c1037c073ae79a0dbf3e494dc54edd74', 'aba6703555dd751f687d9ffab404430514c27f765926ca44f56f47c3971e0acc', 1, 1, '2017-12-15 22:06:01', '2017-12-15 22:06:03.871665', '2017-12-15 22:06:03.871665', 12884901888, 11, 1000000000000000000, 300, 100, 100000000, 10000, 9, 'AAAACaumcDVV3XUfaH2f+rQEQwUUwn92WSbKRPVvR8OXHgrMwWPO5xok9UPy3JL6cPS8+IYJ/Yb4fE/LH9BmRWe4LTYAAAAAWjRHSQAAAAAAAAAALLKPbMojH+RR+TSBDKGB/tufH2mL12ccCHr1Jn27yPBxAcwOP+mAeEZ1JMAE2HMIT1AhPX0S/h1DXUX7jq0gGwAAAAMN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (4, '325236720a1f300e9ad33a1c42327464ec7fe3d7222dae4b6b31f9811256df61', '32cf386a2484899c1ab9bcf164300b58c1037c073ae79a0dbf3e494dc54edd74', 1, 1, '2017-12-15 22:06:02', '2017-12-15 22:06:03.884538', '2017-12-15 22:06:03.884538', 17179869184, 11, 1000000000000000000, 400, 100, 100000000, 10000, 9, 'AAAACTLPOGokhImcGrm88WQwC1jBA3wHOueaDb8+SU3FTt10AuWSp43hQKZzTrZZ/n92/OsQSAGabuzrJ5cT4b9/+igAAAAAWjRHSgAAAAAAAAAA6Ap3Xy0qNN834YP/UI/nracOqnsrZT8u288mLAvc1Bw+LAe4Zal58aTFC5O7JGhjNy90IBQybFvf+2pLKRraJQAAAAQN4Lazp2QAAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO history_ledgers VALUES (5, '921a2f0425eae7491b3613986b5b5ceb649b88f4b3935e5d8bac4256de11ccae', '325236720a1f300e9ad33a1c42327464ec7fe3d7222dae4b6b31f9811256df61', 0, 0, '2017-12-15 22:06:03', '2017-12-15 22:06:03.897884', '2017-12-15 22:06:03.897884', 21474836480, 11, 1000000000000000000, 400, 100, 100000000, 10000, 9, 'AAAACTJSNnIKHzAOmtM6HEIydGTsf+PXIi2uS2sx+YESVt9hTSlG2jaQYJsp5Y3ueUdDbwH+pg23UpJiifTrSDfqSVkAAAAAWjRHSwAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERk+LAe4Zal58aTFC5O7JGhjNy90IBQybFvf+2pLKRraJQAAAAUN4Lazp2QAAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2017-12-18 23:32:35.384116', '2017-12-18 23:32:35.384116', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0);
INSERT INTO history_ledgers VALUES (2, '293cb64f318c85e68a1a0ceb8bebb28dcc046dd0faac63b531db48f6c92321be', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 2, 2, '2017-12-18 23:32:32', '2017-12-18 23:32:35.390529', '2017-12-18 23:32:35.390529', 8589934592, 11, 1000000000000000000, 200, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (3, '6dfb22f5c496ebe9f33647423f2ebb4e401e90fd9d7f7eee1eb579070174d947', '293cb64f318c85e68a1a0ceb8bebb28dcc046dd0faac63b531db48f6c92321be', 1, 1, '2017-12-18 23:32:33', '2017-12-18 23:32:35.403349', '2017-12-18 23:32:35.403349', 12884901888, 11, 1000000000000000000, 300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (4, '0efe81ca3e77c9961bc94201e2d5febd1b9bcb612bc29a18a17bee0676163215', '6dfb22f5c496ebe9f33647423f2ebb4e401e90fd9d7f7eee1eb579070174d947', 1, 1, '2017-12-18 23:32:34', '2017-12-18 23:32:35.407973', '2017-12-18 23:32:35.407973', 17179869184, 11, 1000000000000000000, 400, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (5, 'f4a1a68d48d891fbedac5688cf7d505f689b30f55b5f8b69c9bd56391e73e705', '0efe81ca3e77c9961bc94201e2d5febd1b9bcb612bc29a18a17bee0676163215', 0, 0, '2017-12-18 23:32:35', '2017-12-18 23:32:35.41305', '2017-12-18 23:32:35.41305', 21474836480, 11, 1000000000000000000, 400, 100, 100000000, 10000, 8);
>>>>>>> add price to trade ingestion
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
=======
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2018-01-03 00:54:28.35334', '2018-01-03 00:54:28.35334', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0);
INSERT INTO history_ledgers VALUES (2, 'bf3058a5ce5d675c57a8096eba972824c538d0236c932a21f2367bcbc9dfe3bf', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 2, 2, '2018-01-03 00:54:25', '2018-01-03 00:54:28.358792', '2018-01-03 00:54:28.358792', 8589934592, 11, 1000000000000000000, 200, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (3, '09a0e87ede9c27cf7b86aa27b075fdede73dca7704a3b1a7bab4c43f77b0c1bb', 'bf3058a5ce5d675c57a8096eba972824c538d0236c932a21f2367bcbc9dfe3bf', 1, 1, '2018-01-03 00:54:26', '2018-01-03 00:54:28.371794', '2018-01-03 00:54:28.371794', 12884901888, 11, 1000000000000000000, 300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (4, '3fe42d49e42e7c8d41785ef26ac9cd1b714e1af0e71b8abeeee6f264eddedead', '09a0e87ede9c27cf7b86aa27b075fdede73dca7704a3b1a7bab4c43f77b0c1bb', 1, 1, '2018-01-03 00:54:27', '2018-01-03 00:54:28.376891', '2018-01-03 00:54:28.376891', 17179869184, 11, 1000000000000000000, 400, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (5, '4e5f47f708951a441095516c18e768d6dac1abb584df10d27ccca4f949681d63', '3fe42d49e42e7c8d41785ef26ac9cd1b714e1af0e71b8abeeee6f264eddedead', 0, 0, '2018-01-03 00:54:28', '2018-01-03 00:54:28.382486', '2018-01-03 00:54:28.382486', 21474836480, 11, 1000000000000000000, 400, 100, 100000000, 10000, 8);
>>>>>>> add price to trade query and /trades endpoint
<<<<<<< HEAD
>>>>>>> add price to trade query and /trades endpoint
=======
=======
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2018-01-12 01:05:15.830745', '2018-01-12 01:05:15.830745', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0);
INSERT INTO history_ledgers VALUES (2, 'd304ee99eba3fafe9424d71e6be29743e03092899abe66a6a535dde31b53bab8', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 2, 2, '2018-01-12 01:05:13', '2018-01-12 01:05:15.845972', '2018-01-12 01:05:15.845972', 8589934592, 11, 1000000000000000000, 200, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (3, '049a4a213e7af51f27ce7b189df29d7a4761c46cee142153a0a5f2f8b4a79d08', 'd304ee99eba3fafe9424d71e6be29743e03092899abe66a6a535dde31b53bab8', 1, 1, '2018-01-12 01:05:14', '2018-01-12 01:05:15.866866', '2018-01-12 01:05:15.866866', 12884901888, 11, 1000000000000000000, 300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (4, 'b599d7a7908dd3ddad3f0e0df7bcdba0c4eba3d03729e87d2c4d65da521f845b', '049a4a213e7af51f27ce7b189df29d7a4761c46cee142153a0a5f2f8b4a79d08', 1, 1, '2018-01-12 01:05:15', '2018-01-12 01:05:15.885418', '2018-01-12 01:05:15.885418', 17179869184, 11, 1000000000000000000, 400, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (5, '9bbbe7b87b12d1b21960ae4aaa2e7e8a578e52eebd77c38cddace19633d77a14', 'b599d7a7908dd3ddad3f0e0df7bcdba0c4eba3d03729e87d2c4d65da521f845b', 0, 0, '2018-01-12 01:05:16', '2018-01-12 01:05:15.893556', '2018-01-12 01:05:15.893556', 21474836480, 11, 1000000000000000000, 400, 100, 100000000, 10000, 8);
>>>>>>> wip aggregation test passing
<<<<<<< HEAD
>>>>>>> wip aggregation test passing
=======
=======
INSERT INTO history_ledgers VALUES (1, '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', NULL, 0, 0, '1970-01-01 00:00:00', '2018-01-12 18:35:40.056253', '2018-01-12 18:35:40.056253', 4294967296, 11, 1000000000000000000, 0, 100, 100000000, 100, 0);
INSERT INTO history_ledgers VALUES (2, 'b8af6d3454f4a3b723465ade1b2f7b1000cab67e0dbb1b771081e1ac9a1abf1d', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 2, 2, '2018-01-12 18:35:37', '2018-01-12 18:35:40.063149', '2018-01-12 18:35:40.063149', 8589934592, 11, 1000000000000000000, 200, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (3, 'b025dd39d33abeb2d22ed9c84d3912a555edc893384d3139f53e25eb75eb8109', 'b8af6d3454f4a3b723465ade1b2f7b1000cab67e0dbb1b771081e1ac9a1abf1d', 1, 1, '2018-01-12 18:35:38', '2018-01-12 18:35:40.082696', '2018-01-12 18:35:40.082696', 12884901888, 11, 1000000000000000000, 300, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (4, 'fabbf42422e2b156bc516c79570f176c92e20b222af5315cd0c85d61394f3950', 'b025dd39d33abeb2d22ed9c84d3912a555edc893384d3139f53e25eb75eb8109', 1, 1, '2018-01-12 18:35:39', '2018-01-12 18:35:40.099715', '2018-01-12 18:35:40.099716', 17179869184, 11, 1000000000000000000, 400, 100, 100000000, 10000, 8);
INSERT INTO history_ledgers VALUES (5, '59711afbd4bcb35828192fcbae27d58ba0a69ae936a14be9bd6a21a0d3568ca2', 'fabbf42422e2b156bc516c79570f176c92e20b222af5315cd0c85d61394f3950', 0, 0, '2018-01-12 18:35:40', '2018-01-12 18:35:40.110228', '2018-01-12 18:35:40.110228', 21474836480, 11, 1000000000000000000, 400, 100, 100000000, 10000, 8);
>>>>>>> tests passing
>>>>>>> tests passing


--
-- Data for Name: history_operation_participants; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_operation_participants VALUES (1, 8589938689, 1);
INSERT INTO history_operation_participants VALUES (2, 8589938689, 2);
INSERT INTO history_operation_participants VALUES (3, 8589942785, 1);
INSERT INTO history_operation_participants VALUES (4, 8589942785, 3);
INSERT INTO history_operation_participants VALUES (5, 12884905985, 3);
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_operation_participants VALUES (6, 17179873281, 3);
INSERT INTO history_operation_participants VALUES (7, 17179873281, 2);
=======
=======
>>>>>>> wip aggregation test passing
=======
>>>>>>> tests passing
<<<<<<< HEAD
INSERT INTO history_operation_participants VALUES (6, 17179873281, 2);
INSERT INTO history_operation_participants VALUES (7, 17179873281, 3);
=======
INSERT INTO history_operation_participants VALUES (6, 17179873281, 3);
INSERT INTO history_operation_participants VALUES (7, 17179873281, 1);
>>>>>>> add price to trade query and /trades endpoint
<<<<<<< HEAD
>>>>>>> add price to trade query and /trades endpoint
=======
=======
INSERT INTO history_operation_participants VALUES (6, 17179873281, 2);
INSERT INTO history_operation_participants VALUES (7, 17179873281, 3);
>>>>>>> wip aggregation test passing
<<<<<<< HEAD
>>>>>>> wip aggregation test passing
=======
=======
INSERT INTO history_operation_participants VALUES (6, 17179873281, 3);
INSERT INTO history_operation_participants VALUES (7, 17179873281, 2);
>>>>>>> tests passing
>>>>>>> tests passing


--
-- Data for Name: history_operations; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_operations VALUES (8589938689, 8589938688, 1, 0, '{"funder": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", "account": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "starting_balance": "1000.0000000"}', 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H');
INSERT INTO history_operations VALUES (8589942785, 8589942784, 1, 0, '{"funder": "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H", "account": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "starting_balance": "1000.0000000"}', 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H');
INSERT INTO history_operations VALUES (12884905985, 12884905984, 1, 6, '{"limit": "922337203685.4775807", "trustee": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "trustor": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "asset_code": "USD", "asset_type": "credit_alphanum4", "asset_issuer": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');
INSERT INTO history_operations VALUES (17179873281, 17179873280, 1, 1, '{"to": "GC23QF2HUE52AMXUFUH3AYJAXXGXXV2VHXYYR6EYXETPKDXZSAW67XO4", "from": "GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU", "amount": "10.0000000", "asset_type": "native"}', 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU');


--
-- Data for Name: history_trades; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: history_transaction_participants; Type: TABLE DATA; Schema: public; Owner: -
--

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO history_transaction_participants VALUES (1, 8589938688, 1);
INSERT INTO history_transaction_participants VALUES (2, 8589938688, 2);
<<<<<<< HEAD
INSERT INTO history_transaction_participants VALUES (3, 8589942784, 1);
INSERT INTO history_transaction_participants VALUES (4, 8589942784, 3);
=======
INSERT INTO history_transaction_participants VALUES (3, 8589942784, 3);
INSERT INTO history_transaction_participants VALUES (4, 8589942784, 1);
=======
INSERT INTO history_transaction_participants VALUES (1, 8589938688, 2);
INSERT INTO history_transaction_participants VALUES (2, 8589938688, 1);
INSERT INTO history_transaction_participants VALUES (3, 8589942784, 2);
=======
INSERT INTO history_transaction_participants VALUES (1, 8589938688, 1);
INSERT INTO history_transaction_participants VALUES (2, 8589938688, 2);
=======
INSERT INTO history_transaction_participants VALUES (1, 8589938688, 2);
INSERT INTO history_transaction_participants VALUES (2, 8589938688, 1);
>>>>>>> tests passing
INSERT INTO history_transaction_participants VALUES (3, 8589942784, 1);
>>>>>>> wip aggregation test passing
INSERT INTO history_transaction_participants VALUES (4, 8589942784, 3);
>>>>>>> add price to trade ingestion
>>>>>>> add price to trade ingestion
INSERT INTO history_transaction_participants VALUES (5, 12884905984, 3);
INSERT INTO history_transaction_participants VALUES (6, 17179873280, 3);
INSERT INTO history_transaction_participants VALUES (7, 17179873280, 2);


--
-- Data for Name: history_transactions; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO history_transactions VALUES ('db398eb4ae89756325643cad21c94e13bfc074b323ee83e141bf701a5d904f1b', 2, 1, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 1, 100, 1, '2018-01-12 18:35:40.063654', '2018-01-12 18:35:40.063655', 8589938688, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAACVAvkAAAAAAAAAAABVvwF9wAAAEAYjQcPT2G5hqnBmgGGeg9J8l4c1EnUlxklElH9sqZr0971F6OLWfe/m4kpFtI+sI0i1qLit5A0JyWnbhYLW5oD', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2sVNYGzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{GI0HD09huYapwZoBhnoPSfJeHNRJ1JcZJRJR/bKma9Pe9Reji1n3v5uJKRbSPrCNItai4reQNCclp24WC1uaAw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('f97caffab8c16023a37884165cb0b3ff1aa2daf4000fef49d21efc847ddbfbea', 2, 2, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 2, 100, 1, '2018-01-12 18:35:40.069906', '2018-01-12 18:35:40.069906', 8589942784, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBmKpSgvrwKO20XCOfYfXsGEEUtwYaaEfqSu6ymJmlDma+IX6I7IggbUZMocQdZ94IMAfKdQANqXbIO7ysweeMC', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rv9MNzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAQAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{ZiqUoL68CjttFwjn2H17BhBFLcGGmhH6kruspiZpQ5mviF+iOyIIG1GTKHEHWfeCDAHynUADal2yDu8rMHnjAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('bd486dbdd02d460817671c4a5a7e9d6e865ca29cb41e62d7aaf70a2fee5b36de', 3, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934593, 100, 1, '2018-01-12 18:35:40.083719', '2018-01-12 18:35:40.083719', 12884905984, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQB9kmKW2q3v7Qfy8PMekEb1TTI5ixqkI0BogXrOt7gO162Qbkh2dSTUfeDovc0PAafhDXxthVAlsLujlBmyjBAY=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAMAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{H2SYpbare/tB/Lw8x6QRvVNMjmLGqQjQGiBes63uA7XrZBuSHZ1JNR94Oi9zQ8Bp+ENfG2FUCWwu6OUGbKMEBg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('5eba4195dc8326158c5d87c641a2f17a3a276f8889c1ade84b5c60b462684bc0', 4, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934594, 100, 1, '2018-01-12 18:35:40.100413', '2018-01-12 18:35:40.100413', 17179873280, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAAX14QAAAAAAAAAAAa7kvkwAAABAD8OHQOSeNgKiCX3tTSvuXhy2/pE8FTbrkHZ0FfVBYAjka/2DIQuvVw98shOatgxBUcAAE6v10atB+uEfUIRCAg==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAQAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAk4WAjgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAwAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAloBxQAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{D8OHQOSeNgKiCX3tTSvuXhy2/pE8FTbrkHZ0FfVBYAjka/2DIQuvVw98shOatgxBUcAAE6v10atB+uEfUIRCAg==}', 'none', NULL, NULL);


--
-- Name: history_accounts_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('history_accounts_id_seq', 3, true);


--
-- Name: history_assets_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('history_assets_id_seq', 1, true);


--
-- Name: history_operation_participants_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('history_operation_participants_id_seq', 7, true);


--
-- Name: history_transaction_participants_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

<<<<<<< HEAD
INSERT INTO history_transactions VALUES ('db398eb4ae89756325643cad21c94e13bfc074b323ee83e141bf701a5d904f1b', 2, 1, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 1, 100, 1, '2018-01-22 16:59:35.075193', '2018-01-22 16:59:35.075195', 8589938688, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAACVAvkAAAAAAAAAAABVvwF9wAAAEAYjQcPT2G5hqnBmgGGeg9J8l4c1EnUlxklElH9sqZr0971F6OLWfe/m4kpFtI+sI0i1qLit5A0JyWnbhYLW5oD', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2sVNYGzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{GI0HD09huYapwZoBhnoPSfJeHNRJ1JcZJRJR/bKma9Pe9Reji1n3v5uJKRbSPrCNItai4reQNCclp24WC1uaAw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('f97caffab8c16023a37884165cb0b3ff1aa2daf4000fef49d21efc847ddbfbea', 2, 2, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 2, 100, 1, '2018-01-22 16:59:35.082184', '2018-01-22 16:59:35.082184', 8589942784, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBmKpSgvrwKO20XCOfYfXsGEEUtwYaaEfqSu6ymJmlDma+IX6I7IggbUZMocQdZ94IMAfKdQANqXbIO7ysweeMC', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rv9MNzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{ZiqUoL68CjttFwjn2H17BhBFLcGGmhH6kruspiZpQ5mviF+iOyIIG1GTKHEHWfeCDAHynUADal2yDu8rMHnjAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('bd486dbdd02d460817671c4a5a7e9d6e865ca29cb41e62d7aaf70a2fee5b36de', 3, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934593, 100, 1, '2018-01-22 16:59:35.096285', '2018-01-22 16:59:35.096285', 12884905984, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQB9kmKW2q3v7Qfy8PMekEb1TTI5ixqkI0BogXrOt7gO162Qbkh2dSTUfeDovc0PAafhDXxthVAlsLujlBmyjBAY=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAMAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAwAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{H2SYpbare/tB/Lw8x6QRvVNMjmLGqQjQGiBes63uA7XrZBuSHZ1JNR94Oi9zQ8Bp+ENfG2FUCWwu6OUGbKMEBg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('5eba4195dc8326158c5d87c641a2f17a3a276f8889c1ade84b5c60b462684bc0', 4, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934594, 100, 1, '2018-01-22 16:59:35.108531', '2018-01-22 16:59:35.108531', 17179873280, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAAX14QAAAAAAAAAAAa7kvkwAAABAD8OHQOSeNgKiCX3tTSvuXhy2/pE8FTbrkHZ0FfVBYAjka/2DIQuvVw98shOatgxBUcAAE6v10atB+uEfUIRCAg==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAAEAAAAAwAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4zgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAk4WAjgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAwAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAloBxQAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{D8OHQOSeNgKiCX3tTSvuXhy2/pE8FTbrkHZ0FfVBYAjka/2DIQuvVw98shOatgxBUcAAE6v10atB+uEfUIRCAg==}', 'none', NULL, NULL);
=======
<<<<<<< HEAD
INSERT INTO history_transactions VALUES ('db398eb4ae89756325643cad21c94e13bfc074b323ee83e141bf701a5d904f1b', 2, 1, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 1, 100, 1, '2017-12-15 22:06:03.846926', '2017-12-15 22:06:03.846926', 8589938688, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAACVAvkAAAAAAAAAAABVvwF9wAAAEAYjQcPT2G5hqnBmgGGeg9J8l4c1EnUlxklElH9sqZr0971F6OLWfe/m4kpFtI+sI0i1qLit5A0JyWnbhYLW5oD', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2sVNYGzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{GI0HD09huYapwZoBhnoPSfJeHNRJ1JcZJRJR/bKma9Pe9Reji1n3v5uJKRbSPrCNItai4reQNCclp24WC1uaAw==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('f97caffab8c16023a37884165cb0b3ff1aa2daf4000fef49d21efc847ddbfbea', 2, 2, 'GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 2, 100, 1, '2017-12-15 22:06:03.855032', '2017-12-15 22:06:03.855032', 8589942784, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEBmKpSgvrwKO20XCOfYfXsGEEUtwYaaEfqSu6ymJmlDma+IX6I7IggbUZMocQdZ94IMAfKdQANqXbIO7ysweeMC', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAA=', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rv9MNzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{ZiqUoL68CjttFwjn2H17BhBFLcGGmhH6kruspiZpQ5mviF+iOyIIG1GTKHEHWfeCDAHynUADal2yDu8rMHnjAg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('bd486dbdd02d460817671c4a5a7e9d6e865ca29cb41e62d7aaf70a2fee5b36de', 3, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934593, 100, 1, '2017-12-15 22:06:03.872143', '2017-12-15 22:06:03.872143', 12884905984, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAYAAAABVVNEAAAAAAC1uBdHoTugMvQtD7BhIL3Ne9dVPfGI+Ji5JvUO+ZAt73//////////AAAAAAAAAAGu5L5MAAAAQB9kmKW2q3v7Qfy8PMekEb1TTI5ixqkI0BogXrOt7gO162Qbkh2dSTUfeDovc0PAafhDXxthVAlsLujlBmyjBAY=', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAAGAAAAAAAAAAA=', 'AAAAAAAAAAEAAAADAAAAAAAAAAMAAAABAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAVVTRAAAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAH//////////AAAAAQAAAAAAAAAAAAAAAwAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{H2SYpbare/tB/Lw8x6QRvVNMjmLGqQjQGiBes63uA7XrZBuSHZ1JNR94Oi9zQ8Bp+ENfG2FUCWwu6OUGbKMEBg==}', 'none', NULL, NULL);
INSERT INTO history_transactions VALUES ('5eba4195dc8326158c5d87c641a2f17a3a276f8889c1ade84b5c60b462684bc0', 4, 1, 'GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 8589934594, 100, 1, '2017-12-15 22:06:03.88501', '2017-12-15 22:06:03.88501', 17179873280, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAACAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAtbgXR6E7oDL0LQ+wYSC9zXvXVT3xiPiYuSb1DvmQLe8AAAAAAAAAAAX14QAAAAAAAAAAAa7kvkwAAABAD8OHQOSeNgKiCX3tTSvuXhy2/pE8FTbrkHZ0FfVBYAjka/2DIQuvVw98shOatgxBUcAAE6v10atB+uEfUIRCAg==', 'AAAAAAAAAGQAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAA=', 'AAAAAAAAAAEAAAAEAAAAAwAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL4zgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAk4WAjgAAAACAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAwAAAAIAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAQAAAAAAAAAALW4F0ehO6Ay9C0PsGEgvc1711U98Yj4mLkm9Q75kC3vAAAAAloBxQAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA', 'AAAAAgAAAAMAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+M4AAAAAgAAAAIAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==', '{D8OHQOSeNgKiCX3tTSvuXhy2/pE8FTbrkHZ0FfVBYAjka/2DIQuvVw98shOatgxBUcAAE6v10atB+uEfUIRCAg==}', 'none', NULL, NULL);
=======
SELECT pg_catalog.setval('history_transaction_participants_id_seq', 7, true);
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

