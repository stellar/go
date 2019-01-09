running recipe
recipe finished, closing ledger
ledger closed
--
-- PostgreSQL database dump
--

-- Dumped from database version 9.6.1
-- Dumped by pg_dump version 9.6.1

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

SET search_path = public, pg_catalog;

DROP INDEX IF EXISTS public.upgradehistbyseq;
DROP INDEX IF EXISTS public.signersaccount;
DROP INDEX IF EXISTS public.sellingissuerindex;
DROP INDEX IF EXISTS public.scpquorumsbyseq;
DROP INDEX IF EXISTS public.scpenvsbyseq;
DROP INDEX IF EXISTS public.priceindex;
DROP INDEX IF EXISTS public.ledgersbyseq;
DROP INDEX IF EXISTS public.histfeebyseq;
DROP INDEX IF EXISTS public.histbyseq;
DROP INDEX IF EXISTS public.buyingissuerindex;
DROP INDEX IF EXISTS public.accountbalances;
ALTER TABLE IF EXISTS ONLY public.upgradehistory DROP CONSTRAINT IF EXISTS upgradehistory_pkey;
ALTER TABLE IF EXISTS ONLY public.txhistory DROP CONSTRAINT IF EXISTS txhistory_pkey;
ALTER TABLE IF EXISTS ONLY public.txfeehistory DROP CONSTRAINT IF EXISTS txfeehistory_pkey;
ALTER TABLE IF EXISTS ONLY public.trustlines DROP CONSTRAINT IF EXISTS trustlines_pkey;
ALTER TABLE IF EXISTS ONLY public.storestate DROP CONSTRAINT IF EXISTS storestate_pkey;
ALTER TABLE IF EXISTS ONLY public.signers DROP CONSTRAINT IF EXISTS signers_pkey;
ALTER TABLE IF EXISTS ONLY public.scpquorums DROP CONSTRAINT IF EXISTS scpquorums_pkey;
ALTER TABLE IF EXISTS ONLY public.pubsub DROP CONSTRAINT IF EXISTS pubsub_pkey;
ALTER TABLE IF EXISTS ONLY public.publishqueue DROP CONSTRAINT IF EXISTS publishqueue_pkey;
ALTER TABLE IF EXISTS ONLY public.peers DROP CONSTRAINT IF EXISTS peers_pkey;
ALTER TABLE IF EXISTS ONLY public.offers DROP CONSTRAINT IF EXISTS offers_pkey;
ALTER TABLE IF EXISTS ONLY public.ledgerheaders DROP CONSTRAINT IF EXISTS ledgerheaders_pkey;
ALTER TABLE IF EXISTS ONLY public.ledgerheaders DROP CONSTRAINT IF EXISTS ledgerheaders_ledgerseq_key;
ALTER TABLE IF EXISTS ONLY public.ban DROP CONSTRAINT IF EXISTS ban_pkey;
ALTER TABLE IF EXISTS ONLY public.accounts DROP CONSTRAINT IF EXISTS accounts_pkey;
ALTER TABLE IF EXISTS ONLY public.accountdata DROP CONSTRAINT IF EXISTS accountdata_pkey;
DROP TABLE IF EXISTS public.upgradehistory;
DROP TABLE IF EXISTS public.txhistory;
DROP TABLE IF EXISTS public.txfeehistory;
DROP TABLE IF EXISTS public.trustlines;
DROP TABLE IF EXISTS public.storestate;
DROP TABLE IF EXISTS public.signers;
DROP TABLE IF EXISTS public.scpquorums;
DROP TABLE IF EXISTS public.scphistory;
DROP TABLE IF EXISTS public.pubsub;
DROP TABLE IF EXISTS public.publishqueue;
DROP TABLE IF EXISTS public.peers;
DROP TABLE IF EXISTS public.offers;
DROP TABLE IF EXISTS public.ledgerheaders;
DROP TABLE IF EXISTS public.ban;
DROP TABLE IF EXISTS public.accounts;
DROP TABLE IF EXISTS public.accountdata;
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

SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: accountdata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE accountdata (
    accountid character varying(56) NOT NULL,
    dataname character varying(64) NOT NULL,
    datavalue character varying(112) NOT NULL,
    lastmodified integer DEFAULT 0 NOT NULL
);


--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE accounts (
    accountid character varying(56) NOT NULL,
    balance bigint NOT NULL,
    seqnum bigint NOT NULL,
    numsubentries integer NOT NULL,
    inflationdest character varying(56),
    homedomain character varying(32) NOT NULL,
    thresholds text NOT NULL,
    flags integer NOT NULL,
    lastmodified integer NOT NULL,
    buyingliabilities bigint,
    sellingliabilities bigint,
    CONSTRAINT accounts_balance_check CHECK ((balance >= 0)),
    CONSTRAINT accounts_buyingliabilities_check CHECK ((buyingliabilities >= 0)),
    CONSTRAINT accounts_numsubentries_check CHECK ((numsubentries >= 0)),
    CONSTRAINT accounts_sellingliabilities_check CHECK ((sellingliabilities >= 0))
);


--
-- Name: ban; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE ban (
    nodeid character(56) NOT NULL
);


--
-- Name: ledgerheaders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE ledgerheaders (
    ledgerhash character(64) NOT NULL,
    prevhash character(64) NOT NULL,
    bucketlisthash character(64) NOT NULL,
    ledgerseq integer,
    closetime bigint NOT NULL,
    data text NOT NULL,
    CONSTRAINT ledgerheaders_closetime_check CHECK ((closetime >= 0)),
    CONSTRAINT ledgerheaders_ledgerseq_check CHECK ((ledgerseq >= 0))
);


--
-- Name: offers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE offers (
    sellerid character varying(56) NOT NULL,
    offerid bigint NOT NULL,
    sellingassettype integer NOT NULL,
    sellingassetcode character varying(12),
    sellingissuer character varying(56),
    buyingassettype integer NOT NULL,
    buyingassetcode character varying(12),
    buyingissuer character varying(56),
    amount bigint NOT NULL,
    pricen integer NOT NULL,
    priced integer NOT NULL,
    price double precision NOT NULL,
    flags integer NOT NULL,
    lastmodified integer NOT NULL,
    CONSTRAINT offers_amount_check CHECK ((amount >= 0)),
    CONSTRAINT offers_offerid_check CHECK ((offerid >= 0))
);


--
-- Name: peers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE peers (
    ip character varying(15) NOT NULL,
    port integer DEFAULT 0 NOT NULL,
    nextattempt timestamp without time zone NOT NULL,
    numfailures integer DEFAULT 0 NOT NULL,
    flags integer DEFAULT 0 NOT NULL,
    CONSTRAINT peers_numfailures_check CHECK ((numfailures >= 0)),
    CONSTRAINT peers_port_check CHECK (((port > 0) AND (port <= 65535)))
);


--
-- Name: publishqueue; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE publishqueue (
    ledger integer NOT NULL,
    state text
);


--
-- Name: pubsub; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE pubsub (
    resid character(32) NOT NULL,
    lastread integer
);


--
-- Name: scphistory; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE scphistory (
    nodeid character(56) NOT NULL,
    ledgerseq integer NOT NULL,
    envelope text NOT NULL,
    CONSTRAINT scphistory_ledgerseq_check CHECK ((ledgerseq >= 0))
);


--
-- Name: scpquorums; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE scpquorums (
    qsethash character(64) NOT NULL,
    lastledgerseq integer NOT NULL,
    qset text NOT NULL,
    CONSTRAINT scpquorums_lastledgerseq_check CHECK ((lastledgerseq >= 0))
);


--
-- Name: signers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE signers (
    accountid character varying(56) NOT NULL,
    publickey character varying(56) NOT NULL,
    weight integer NOT NULL
);


--
-- Name: storestate; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE storestate (
    statename character(32) NOT NULL,
    state text
);


--
-- Name: trustlines; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE trustlines (
    accountid character varying(56) NOT NULL,
    assettype integer NOT NULL,
    issuer character varying(56) NOT NULL,
    assetcode character varying(12) NOT NULL,
    tlimit bigint NOT NULL,
    balance bigint NOT NULL,
    flags integer NOT NULL,
    lastmodified integer NOT NULL,
    buyingliabilities bigint,
    sellingliabilities bigint,
    CONSTRAINT trustlines_balance_check CHECK ((balance >= 0)),
    CONSTRAINT trustlines_buyingliabilities_check CHECK ((buyingliabilities >= 0)),
    CONSTRAINT trustlines_sellingliabilities_check CHECK ((sellingliabilities >= 0)),
    CONSTRAINT trustlines_tlimit_check CHECK ((tlimit > 0))
);


--
-- Name: txfeehistory; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE txfeehistory (
    txid character(64) NOT NULL,
    ledgerseq integer NOT NULL,
    txindex integer NOT NULL,
    txchanges text NOT NULL,
    CONSTRAINT txfeehistory_ledgerseq_check CHECK ((ledgerseq >= 0))
);


--
-- Name: txhistory; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE txhistory (
    txid character(64) NOT NULL,
    ledgerseq integer NOT NULL,
    txindex integer NOT NULL,
    txbody text NOT NULL,
    txresult text NOT NULL,
    txmeta text NOT NULL,
    CONSTRAINT txhistory_ledgerseq_check CHECK ((ledgerseq >= 0))
);


--
-- Name: upgradehistory; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE upgradehistory (
    ledgerseq integer NOT NULL,
    upgradeindex integer NOT NULL,
    upgrade text NOT NULL,
    changes text NOT NULL,
    CONSTRAINT upgradehistory_ledgerseq_check CHECK ((ledgerseq >= 0))
);


--
-- Data for Name: accountdata; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO accountdata VALUES ('GD6NTRJW5Z6NCWH4USWMNEYF77RUR2MTO6NP4KEDVJATTCUXDRO3YIFS', 'done', 'dHJ1ZQ==', 5);


--
-- Data for Name: accounts; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO accounts VALUES ('GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 999999989999999900, 1, 0, NULL, '', 'AQAAAA==', 0, 3, NULL, NULL);
INSERT INTO accounts VALUES ('GD6NTRJW5Z6NCWH4USWMNEYF77RUR2MTO6NP4KEDVJATTCUXDRO3YIFS', 9999999800, 12884901890, 2, NULL, '', 'AQAAAA==', 0, 5, NULL, NULL);


--
-- Data for Name: ban; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: ledgerheaders; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO ledgerheaders VALUES ('63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '0000000000000000000000000000000000000000000000000000000000000000', '572a2e32ff248a07b0e70fd1f6d318c1facd20b6cc08c33d5775259868125a16', 1, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('55b2913e7fb60ca5c5855d28d00755c2131c90e48d14dd8eb8794a366dc7d0b5', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '735227ed398461291237687b08446aa2c9b096e0c98a462dadda569f05dd2484', 2, 1547043323, 'AAAACmPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZuZRHr9UdXKbTKiclfOjy72YZFJUkJPVcKT5htvorm1QAAAAAXDYB+wAAAAIAAAAIAAAAAQAAAAoAAAAIAAAAAwAAJxAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERlzUiftOYRhKRI3aHsIRGqiybCW4MmKRi2t2lafBd0khAAAAAIN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('5bafb5897f61ee815d81106928c2441514ab216af6fdcc022593fc1e00f0fdf8', '55b2913e7fb60ca5c5855d28d00755c2131c90e48d14dd8eb8794a366dc7d0b5', 'da14da6b0c81b426f580632f565e93a528b11589dc1aee5f5c4e23a0f1eaaa7d', 3, 1547043324, 'AAAAClWykT5/tgylxYVdKNAHVcITHJDkjRTdjrh5SjZtx9C1KHT2JUVx/rHmrpUcHaFv4SUEgRh3ieTwL+Kp7aW3GCwAAAAAXDYB/AAAAAAAAAAAlzJ1vISHXzElAf05LhN7qiqWqKvjHhTijb/BgG6FsuLaFNprDIG0JvWAYy9WXpOlKLEVidwa7l9cTiOg8eqqfQAAAAMN4Lazp2QAAAAAAAAAAABkAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('a59f9604989c23af62bd7c61550773e9ad7d8d990fc70e5e1bc4ecc2410709a3', '5bafb5897f61ee815d81106928c2441514ab216af6fdcc022593fc1e00f0fdf8', 'bf7edb6c1979a8b7b7c4255be766c86dcaac665c236df658df13d39a27eafcba', 4, 1547043325, 'AAAACluvtYl/Ye6BXYEQaSjCRBUUqyFq9v3MAiWT/B4A8P34KprrJK9jq12hJgJoC6wZuh/8A4FTSL2ZKU0d4URWArgAAAAAXDYB/QAAAAAAAAAApOCJokUPy1wA/XbpkumCKr9Nv3B8+fRGt4d1ygPrKBy/fttsGXmot7fEJVvnZshtyqxmXCNt9ljfE9OaJ+r8ugAAAAQN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('9a18f9b3afffc7d0ff07fea62dcec05a2e364ebacee08945d75063e2921812ba', 'a59f9604989c23af62bd7c61550773e9ad7d8d990fc70e5e1bc4ecc2410709a3', 'a486f9d25d7c2d7148e558cea733112d5c6e2f0c31476e980f9461fb553d4d43', 5, 1547043326, 'AAAACqWflgSYnCOvYr18YVUHc+mtfY2ZD8cOXhvE7MJBBwmjocqtJFY3+Zu9Xkp2Cb+9QFY9z8DStnSLd66+8i+cG0gAAAAAXDYB/gAAAAAAAAAAnzt6UJX5by6BwcesLvsJTMJPn/Y/s/aBHlEOtmT5mwekhvnSXXwtcUjlWM6nMxEtXG4vDDFHbpgPlGH7VT1NQwAAAAUN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('3ed05fda7fa1ec62f72e57c94844756be867429486f25f2460bf5db69904498a', '9a18f9b3afffc7d0ff07fea62dcec05a2e364ebacee08945d75063e2921812ba', 'c2d5be06f487f5351e3376163c5166f367883682e402f702329a8f1e8a4397b2', 6, 1547043327, 'AAAACpoY+bOv/8fQ/wf+pi3OwFouNk66zuCJRddQY+KSGBK6rNwXoEYZNnKnLY45dMWsRxMD+636TFgfRGS1lnP3qm4AAAAAXDYB/wAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERnC1b4G9If1NR4zdhY8UWbzZ4g2guQC9wIymo8eikOXsgAAAAYN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');


--
-- Data for Name: offers; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: peers; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: publishqueue; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: pubsub; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: scphistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO scphistory VALUES ('GAS7S7DKBMNZENFUK5V4PSPJWOUAH4PZWOBCXER3XCIINFDWH2V6CCBK', 2, 'AAAAACX5fGoLG5I0tFdrx8nps6gD8fmzgiuSO7iQhpR2PqvhAAAAAAAAAAIAAAACAAAAAQAAAEi5lEev1R1cptMqJyV86PLvZhkUlSQk9VwpPmG2+iubVAAAAABcNgH7AAAAAgAAAAgAAAABAAAACgAAAAgAAAADAAAnEAAAAAAAAAABrxvBIxaX+93YTVJq37jKyoHdJrOfDrMCbrfTRQ3u0KoAAABAARmWwQ7P8RSM2OgePcPmQOnNojYEZgtPxYszVSpvrLJSYu5lEJcAJzWUoxFCNnBXUWNwIh4inc//J405z+lIDg==');
INSERT INTO scphistory VALUES ('GAS7S7DKBMNZENFUK5V4PSPJWOUAH4PZWOBCXER3XCIINFDWH2V6CCBK', 3, 'AAAAACX5fGoLG5I0tFdrx8nps6gD8fmzgiuSO7iQhpR2PqvhAAAAAAAAAAMAAAACAAAAAQAAADAodPYlRXH+seaulRwdoW/hJQSBGHeJ5PAv4qntpbcYLAAAAABcNgH8AAAAAAAAAAAAAAABrxvBIxaX+93YTVJq37jKyoHdJrOfDrMCbrfTRQ3u0KoAAABAmaaaSvcNxkGEAg5gk+sKN+mHt0LIfpgNlooT3qv6Ui2FpKr4v3wNyWuZMPZDKvQnxg/Zu3zVkBJO8TvjRh/dAw==');
INSERT INTO scphistory VALUES ('GAS7S7DKBMNZENFUK5V4PSPJWOUAH4PZWOBCXER3XCIINFDWH2V6CCBK', 4, 'AAAAACX5fGoLG5I0tFdrx8nps6gD8fmzgiuSO7iQhpR2PqvhAAAAAAAAAAQAAAACAAAAAQAAADAqmuskr2OrXaEmAmgLrBm6H/wDgVNIvZkpTR3hRFYCuAAAAABcNgH9AAAAAAAAAAAAAAABrxvBIxaX+93YTVJq37jKyoHdJrOfDrMCbrfTRQ3u0KoAAABAj8eQDO2MnUBbrOfq7nY2RiMUaklwguHGSnaRWP1w4AAH8+citzogynM8C+zJgCfGsROeLftMsKvvv/7m+ST5AA==');
INSERT INTO scphistory VALUES ('GAS7S7DKBMNZENFUK5V4PSPJWOUAH4PZWOBCXER3XCIINFDWH2V6CCBK', 5, 'AAAAACX5fGoLG5I0tFdrx8nps6gD8fmzgiuSO7iQhpR2PqvhAAAAAAAAAAUAAAACAAAAAQAAADChyq0kVjf5m71eSnYJv71AVj3PwNK2dIt3rr7yL5wbSAAAAABcNgH+AAAAAAAAAAAAAAABrxvBIxaX+93YTVJq37jKyoHdJrOfDrMCbrfTRQ3u0KoAAABA2YV34Gkh2e2ZdFQX0EfuFK848sfwM0xq2qZrBtFYvBtP0YnbhnPTXG/V9tfQ/ObgOTRq4jHDH/8E3GMx/HjMAg==');
INSERT INTO scphistory VALUES ('GAS7S7DKBMNZENFUK5V4PSPJWOUAH4PZWOBCXER3XCIINFDWH2V6CCBK', 6, 'AAAAACX5fGoLG5I0tFdrx8nps6gD8fmzgiuSO7iQhpR2PqvhAAAAAAAAAAYAAAACAAAAAQAAADCs3BegRhk2cqctjjl0xaxHEwP7rfpMWB9EZLWWc/eqbgAAAABcNgH/AAAAAAAAAAAAAAABrxvBIxaX+93YTVJq37jKyoHdJrOfDrMCbrfTRQ3u0KoAAABA+cJROmemMPJmLYAW81zIMrjhOTCND812mwP4dQPGHxyKl7jaTJJUTJI/qJIW/VVXqEYb8Z3ZQXGka+S5vJpCCw==');


--
-- Data for Name: scpquorums; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO scpquorums VALUES ('af1bc1231697fbddd84d526adfb8caca81dd26b39f0eb3026eb7d3450deed0aa', 6, 'AAAAAQAAAAEAAAAAJfl8agsbkjS0V2vHyemzqAPx+bOCK5I7uJCGlHY+q+EAAAAA');


--
-- Data for Name: signers; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO signers VALUES ('GD6NTRJW5Z6NCWH4USWMNEYF77RUR2MTO6NP4KEDVJATTCUXDRO3YIFS', 'XC6GFVFYBWPDNWRJYFWF2TM7CFZR6NQFFRZEAGTWYI6A7NNJW5CCGCON', 1);


--
-- Data for Name: storestate; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO storestate VALUES ('databaseschema                  ', '7');
INSERT INTO storestate VALUES ('networkpassphrase               ', 'Test SDF Network ; September 2015');
INSERT INTO storestate VALUES ('forcescponnextlaunch            ', 'false');
INSERT INTO storestate VALUES ('ledgerupgrades                  ', '{
    "time": 0,
    "version": {
        "has": false
    },
    "fee": {
        "has": false
    },
    "maxtxsize": {
        "has": false
    },
    "reserve": {
        "has": false
    }
}');
INSERT INTO storestate VALUES ('lastclosedledger                ', '3ed05fda7fa1ec62f72e57c94844756be867429486f25f2460bf5db69904498a');
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v10.0.0",
    "currentLedger": 6,
    "currentBuckets": [
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "aaa4881b8995adbb5ab737472f1af7f351c296fe6dd094ff84e6c630bd9037ff"
        },
        {
            "curr": "8aa5e6da375080b09395c35b99f54b16e836d297fe492cab9b94fa3fa1cf7d52",
            "next": {
                "state": 1,
                "output": "aaa4881b8995adbb5ab737472f1af7f351c296fe6dd094ff84e6c630bd9037ff"
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 0
            },
            "snap": "0000000000000000000000000000000000000000000000000000000000000000"
        }
    ]
}');
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAAAl+XxqCxuSNLRXa8fJ6bOoA/H5s4Irkju4kIaUdj6r4QAAAAAAAAAGAAAAA68bwSMWl/vd2E1Sat+4ysqB3Saznw6zAm6300UN7tCqAAAAAQAAADCs3BegRhk2cqctjjl0xaxHEwP7rfpMWB9EZLWWc/eqbgAAAABcNgH/AAAAAAAAAAAAAAABAAAAMKzcF6BGGTZypy2OOXTFrEcTA/ut+kxYH0RktZZz96puAAAAAFw2Af8AAAAAAAAAAAAAAEC47CnoouAB+NUFNJgSy29tlv414/u0qLUaEMtIKBwZTqntqQhgtn1X0U0wK5I35r6hqdCNOk/e1jBUMimeOgsFAAAAACX5fGoLG5I0tFdrx8nps6gD8fmzgiuSO7iQhpR2PqvhAAAAAAAAAAYAAAACAAAAAQAAADCs3BegRhk2cqctjjl0xaxHEwP7rfpMWB9EZLWWc/eqbgAAAABcNgH/AAAAAAAAAAAAAAABrxvBIxaX+93YTVJq37jKyoHdJrOfDrMCbrfTRQ3u0KoAAABA+cJROmemMPJmLYAW81zIMrjhOTCND812mwP4dQPGHxyKl7jaTJJUTJI/qJIW/VVXqEYb8Z3ZQXGka+S5vJpCCwAAAAGaGPmzr//H0P8H/qYtzsBaLjZOus7giUXXUGPikhgSugAAAAAAAAABAAAAAQAAAAEAAAAAJfl8agsbkjS0V2vHyemzqAPx+bOCK5I7uJCGlHY+q+EAAAAA');


--
-- Data for Name: trustlines; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: txfeehistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txfeehistory VALUES ('54c49533d937a906c0e6e501322bb600ffe332bf888cb474bd4261d42f542470', 3, 1, 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('59de4450bcc1830b6da83fb481aab833db04dadf1e88d568a00274d4038d531e', 4, 1, 'AAAAAgAAAAMAAAADAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+QAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+OcAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('b0879add9f3957c9796d1d1fb23720dbed15d07793c742773455f2706c0e9a25', 5, 1, 'AAAAAgAAAAMAAAAEAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+OcAAAAAwAAAAEAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAEAAAACvGLUuA2eNtopwWxdTZ8Rcx82BSxyQBp2wjwPtam3RCMAAAABAAAAAAAAAAAAAAABAAAABQAAAAAAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAACVAvjOAAAAAMAAAABAAAAAQAAAAAAAAAAAAAAAAEAAAAAAAABAAAAArxi1LgNnjbaKcFsXU2fEXMfNgUsckAadsI8D7Wpt0QjAAAAAQAAAAAAAAAA');


--
-- Data for Name: txhistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txhistory VALUES ('54c49533d937a906c0e6e501322bb600ffe332bf888cb474bd4261d42f542470', 3, 1, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEDi0+98/ltS2PZOxNCNogdC0ctkOWrNnQ+3eVu2PI3+LNdVssYOrw4gwvZFULsMpS166y7rVfyn6AIp7gqV5pMD', 'VMSVM9k3qQbA5uUBMiu2AP/jMr+IjLR0vUJh1C9UJHAAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAQAAAAIAAAADAAAAAwAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcN4Lazp2P/nAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcN4Lazp2P/nAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAAAAAADAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+QAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAMAAAADAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrFTWBucAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txhistory VALUES ('59de4450bcc1830b6da83fb481aab833db04dadf1e88d568a00274d4038d531e', 4, 1, 'AAAAAPzZxTbufNFY/KSsxpMF/+NI6ZN3mv4og6pBOYqXHF28AAAAZAAAAAMAAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAK8YtS4DZ422inBbF1NnxFzHzYFLHJAGnbCPA+1qbdEIwAAAAEAAAAAAAAAAZccXbwAAABAeVu+uPT8KOhoKoNJidCWqVs71WAIqQns2Zq4mM3LBluMVDHej/SJhUxiKlsSR5MJwQU3trQbPsOAwb56BinbCw==', 'Wd5EULzBgwttqD+0gaq4M9sE2t8eiNVooAJ01AONUx4AAAAAAAAAZAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAA==', 'AAAAAQAAAAIAAAADAAAABAAAAAAAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAACVAvjnAAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAABAAAAAAAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAACVAvjnAAAAAMAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAgAAAAMAAAAEAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+OcAAAAAwAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+OcAAAAAwAAAAEAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAEAAAACvGLUuA2eNtopwWxdTZ8Rcx82BSxyQBp2wjwPtam3RCMAAAABAAAAAAAAAAA=');
INSERT INTO txhistory VALUES ('b0879add9f3957c9796d1d1fb23720dbed15d07793c742773455f2706c0e9a25', 5, 1, 'AAAAAPzZxTbufNFY/KSsxpMF/+NI6ZN3mv4og6pBOYqXHF28AAAAZAAAAAMAAAACAAAAAAAAAAAAAAABAAAAAAAAAAoAAAAEZG9uZQAAAAEAAAAEdHJ1ZQAAAAAAAAABqbdEIwAAACC5TSe5k00+CKUuUtfafav6xITv43pTgO6QiPes4u/N6Q==', 'sIea3Z85V8l5bR0fsjcg2+0V0HeTx0J3NFXycGwOmiUAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAoAAAAAAAAAAA==', 'AAAAAQAAAAIAAAADAAAABQAAAAAAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAACVAvjOAAAAAMAAAABAAAAAQAAAAAAAAAAAAAAAAEAAAAAAAABAAAAArxi1LgNnjbaKcFsXU2fEXMfNgUsckAadsI8D7Wpt0QjAAAAAQAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAAPzZxTbufNFY/KSsxpMF/+NI6ZN3mv4og6pBOYqXHF28AAAAAlQL4zgAAAADAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAQAAAAK8YtS4DZ422inBbF1NnxFzHzYFLHJAGnbCPA+1qbdEIwAAAAEAAAAAAAAAAAAAAAEAAAADAAAAAAAAAAUAAAADAAAAAPzZxTbufNFY/KSsxpMF/+NI6ZN3mv4og6pBOYqXHF28AAAABGRvbmUAAAAEdHJ1ZQAAAAAAAAAAAAAAAwAAAAUAAAAAAAAAAPzZxTbufNFY/KSsxpMF/+NI6ZN3mv4og6pBOYqXHF28AAAAAlQL4zgAAAADAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAQAAAAK8YtS4DZ422inBbF1NnxFzHzYFLHJAGnbCPA+1qbdEIwAAAAEAAAAAAAAAAAAAAAEAAAAFAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+M4AAAAAwAAAAIAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAEAAAACvGLUuA2eNtopwWxdTZ8Rcx82BSxyQBp2wjwPtam3RCMAAAABAAAAAAAAAAA=');


--
-- Data for Name: upgradehistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO upgradehistory VALUES (2, 1, 'AAAAAQAAAAo=', 'AAAAAA==');
INSERT INTO upgradehistory VALUES (2, 2, 'AAAAAwAAJxA=', 'AAAAAA==');


--
-- Name: accountdata accountdata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accountdata
    ADD CONSTRAINT accountdata_pkey PRIMARY KEY (accountid, dataname);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (accountid);


--
-- Name: ban ban_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY ban
    ADD CONSTRAINT ban_pkey PRIMARY KEY (nodeid);


--
-- Name: ledgerheaders ledgerheaders_ledgerseq_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY ledgerheaders
    ADD CONSTRAINT ledgerheaders_ledgerseq_key UNIQUE (ledgerseq);


--
-- Name: ledgerheaders ledgerheaders_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY ledgerheaders
    ADD CONSTRAINT ledgerheaders_pkey PRIMARY KEY (ledgerhash);


--
-- Name: offers offers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY offers
    ADD CONSTRAINT offers_pkey PRIMARY KEY (offerid);


--
-- Name: peers peers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY peers
    ADD CONSTRAINT peers_pkey PRIMARY KEY (ip, port);


--
-- Name: publishqueue publishqueue_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY publishqueue
    ADD CONSTRAINT publishqueue_pkey PRIMARY KEY (ledger);


--
-- Name: pubsub pubsub_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY pubsub
    ADD CONSTRAINT pubsub_pkey PRIMARY KEY (resid);


--
-- Name: scpquorums scpquorums_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY scpquorums
    ADD CONSTRAINT scpquorums_pkey PRIMARY KEY (qsethash);


--
-- Name: signers signers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY signers
    ADD CONSTRAINT signers_pkey PRIMARY KEY (accountid, publickey);


--
-- Name: storestate storestate_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY storestate
    ADD CONSTRAINT storestate_pkey PRIMARY KEY (statename);


--
-- Name: trustlines trustlines_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY trustlines
    ADD CONSTRAINT trustlines_pkey PRIMARY KEY (accountid, issuer, assetcode);


--
-- Name: txfeehistory txfeehistory_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY txfeehistory
    ADD CONSTRAINT txfeehistory_pkey PRIMARY KEY (ledgerseq, txindex);


--
-- Name: txhistory txhistory_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY txhistory
    ADD CONSTRAINT txhistory_pkey PRIMARY KEY (ledgerseq, txindex);


--
-- Name: upgradehistory upgradehistory_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY upgradehistory
    ADD CONSTRAINT upgradehistory_pkey PRIMARY KEY (ledgerseq, upgradeindex);


--
-- Name: accountbalances; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX accountbalances ON accounts USING btree (balance) WHERE (balance >= 1000000000);


--
-- Name: buyingissuerindex; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX buyingissuerindex ON offers USING btree (buyingissuer);


--
-- Name: histbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX histbyseq ON txhistory USING btree (ledgerseq);


--
-- Name: histfeebyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX histfeebyseq ON txfeehistory USING btree (ledgerseq);


--
-- Name: ledgersbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ledgersbyseq ON ledgerheaders USING btree (ledgerseq);


--
-- Name: priceindex; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX priceindex ON offers USING btree (price);


--
-- Name: scpenvsbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX scpenvsbyseq ON scphistory USING btree (ledgerseq);


--
-- Name: scpquorumsbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX scpquorumsbyseq ON scpquorums USING btree (lastledgerseq);


--
-- Name: sellingissuerindex; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX sellingissuerindex ON offers USING btree (sellingissuer);


--
-- Name: signersaccount; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX signersaccount ON signers USING btree (accountid);


--
-- Name: upgradehistbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX upgradehistbyseq ON upgradehistory USING btree (ledgerseq);


--
-- PostgreSQL database dump complete
--

