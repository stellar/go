running recipe
recipe finished, closing ledger
ledger closed
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
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

SET search_path = public, pg_catalog;

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
    CONSTRAINT accounts_balance_check CHECK ((balance >= 0)),
    CONSTRAINT accounts_numsubentries_check CHECK ((numsubentries >= 0))
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
    CONSTRAINT trustlines_balance_check CHECK ((balance >= 0)),
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
-- Data for Name: accountdata; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: accounts; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO accounts VALUES ('GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 999999998999999900, 1, 0, NULL, '', 'AQAAAA==', 0, 2);
INSERT INTO accounts VALUES ('GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 999999900, 8589934593, 0, NULL, '', 'AQAAAA==', 0, 3);


--
-- Data for Name: ban; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: ledgerheaders; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO ledgerheaders VALUES ('63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '0000000000000000000000000000000000000000000000000000000000000000', '572a2e32ff248a07b0e70fd1f6d318c1facd20b6cc08c33d5775259868125a16', 1, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO ledgerheaders VALUES ('755b6a774ab704cbb4bed70a51bb33c5a0dab94aea8ea5c4278fc517afd1b172', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '1cacc9fc144882c3d024d85dbebd2676445b1fbcb3617ff0f425eeb1b53b3f53', 2, 1516640443, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZoHUUqPO1SoXWBQWiQne6+AHcM0CVKtgN/zEmvaDE04AAAAAAWmYYuwAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAAEtgyinnIssW5tBeb2/tNZBp1rtCbIvdoa9kC/D9mgLwcrMn8FEiCw9Ak2F2+vSZ2RFsfvLNhf/D0Je6xtTs/UwAAAAIN4Lazp2QAAAAAAAAAAABkAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('dd99581a2dd963bd36705ecbd37adc00d81276dc7c0daf18d7e1b6678324bd22', '755b6a774ab704cbb4bed70a51bb33c5a0dab94aea8ea5c4278fc517afd1b172', 'b5ad6ca954a8038442bcb82fdb51818da3242e51ff34cb275c860bbec88026df', 3, 1516640444, 'AAAACXVbandKtwTLtL7XClG7M8Wg2rlK6o6lxCePxRev0bFyP4MCUm5BxcxDsxL4yH26EGSIa5P6Ixya2avzJ1RxMdUAAAAAWmYYvAAAAAAAAAAAHwHNsmRY+V05R8QcxzL6y8Pg5b8K5XPCts8of63QFDq1rWypVKgDhEK8uC/bUYGNoyQuUf80yydchgu+yIAm3wAAAAMN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
=======
>>>>>>> add price to trade query and /trades endpoint
<<<<<<< HEAD
INSERT INTO ledgerheaders VALUES ('4c2e0668790cd728940c8d146e2ac88365422c3aa23cae4c54d2c228ec056167', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '1cacc9fc144882c3d024d85dbebd2676445b1fbcb3617ff0f425eeb1b53b3f53', 2, 1513375683, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZoHUUqPO1SoXWBQWiQne6+AHcM0CVKtgN/zEmvaDE04AAAAAAWjRHwwAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAAEtgyinnIssW5tBeb2/tNZBp1rtCbIvdoa9kC/D9mgLwcrMn8FEiCw9Ak2F2+vSZ2RFsfvLNhf/D0Je6xtTs/UwAAAAIN4Lazp2QAAAAAAAAAAABkAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('da7595fa0ce98f3c9ece340371a0860f1f6ec0952e3b553cc363bf4f66da52e2', '4c2e0668790cd728940c8d146e2ac88365422c3aa23cae4c54d2c228ec056167', 'b5ad6ca954a8038442bcb82fdb51818da3242e51ff34cb275c860bbec88026df', 3, 1513375684, 'AAAACUwuBmh5DNcolAyNFG4qyINlQiw6ojyuTFTSwijsBWFnJgCRwuFxQo6ze9/f9XKrDjhtXPW9+pn+mFUtQAYWINAAAAAAWjRHxAAAAAAAAAAAHwHNsmRY+V05R8QcxzL6y8Pg5b8K5XPCts8of63QFDq1rWypVKgDhEK8uC/bUYGNoyQuUf80yydchgu+yIAm3wAAAAMN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
INSERT INTO ledgerheaders VALUES ('39fc80e30ebf69e2262aea6d1fc692c149823a82c0298548ba139bc297db2285', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '1cacc9fc144882c3d024d85dbebd2676445b1fbcb3617ff0f425eeb1b53b3f53', 2, 1513639971, 'AAAACGPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZoHUUqPO1SoXWBQWiQne6+AHcM0CVKtgN/zEmvaDE04AAAAAAWjhQIwAAAAIAAAAIAAAAAQAAAAgAAAAIAAAAAwAAJxAAAAAAEtgyinnIssW5tBeb2/tNZBp1rtCbIvdoa9kC/D9mgLwcrMn8FEiCw9Ak2F2+vSZ2RFsfvLNhf/D0Je6xtTs/UwAAAAIN4Lazp2QAAAAAAAAAAABkAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('983ac7c36cca7d949d76d82382784e36627d5a477daa535847544800ef048a42', '39fc80e30ebf69e2262aea6d1fc692c149823a82c0298548ba139bc297db2285', 'b5ad6ca954a8038442bcb82fdb51818da3242e51ff34cb275c860bbec88026df', 3, 1513639972, 'AAAACDn8gOMOv2niJirqbR/GksFJgjqCwCmFSLoTm8KX2yKFQqmFAl47EwRZJEWCCXG3sr5657znE++bkkyOqq2x15QAAAAAWjhQJAAAAAAAAAAAHwHNsmRY+V05R8QcxzL6y8Pg5b8K5XPCts8of63QFDq1rWypVKgDhEK8uC/bUYGNoyQuUf80yydchgu+yIAm3wAAAAMN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
>>>>>>> add price to trade ingestion
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
=======
INSERT INTO ledgerheaders VALUES ('104637f26964c432182e79cff5bbed3ef184fb7513d5132f66c7da64d70c8028', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '1cacc9fc144882c3d024d85dbebd2676445b1fbcb3617ff0f425eeb1b53b3f53', 2, 1514940883, 'AAAACGPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZoHUUqPO1SoXWBQWiQne6+AHcM0CVKtgN/zEmvaDE04AAAAAAWkwp0wAAAAIAAAAIAAAAAQAAAAgAAAAIAAAAAwAAJxAAAAAAEtgyinnIssW5tBeb2/tNZBp1rtCbIvdoa9kC/D9mgLwcrMn8FEiCw9Ak2F2+vSZ2RFsfvLNhf/D0Je6xtTs/UwAAAAIN4Lazp2QAAAAAAAAAAABkAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('f82d7d87ed6d92c0dcb24603c1c503524b83868856a9655aa095c19e6745c773', '104637f26964c432182e79cff5bbed3ef184fb7513d5132f66c7da64d70c8028', 'b5ad6ca954a8038442bcb82fdb51818da3242e51ff34cb275c860bbec88026df', 3, 1514940884, 'AAAACBBGN/JpZMQyGC55z/W77T7xhPt1E9UTL2bH2mTXDIAoNJkWngipyeD14pwNV6q5eZkbnuQa7YTlNvLwSgnLMsYAAAAAWkwp1AAAAAAAAAAAHwHNsmRY+V05R8QcxzL6y8Pg5b8K5XPCts8of63QFDq1rWypVKgDhEK8uC/bUYGNoyQuUf80yydchgu+yIAm3wAAAAMN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
>>>>>>> add price to trade query and /trades endpoint
>>>>>>> add price to trade query and /trades endpoint


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

<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO scphistory VALUES ('GCXURZRBQFOADUOGKEWCZYFIAREQV4IYQFTJ2OE7VBRV6YMYSEWMJJWS', 2, 'AAAAAK9I5iGBXAHRxlEsLOCoBEkK8RiBZp04n6hjX2GYkSzEAAAAAAAAAAIAAAACAAAAAQAAAEigdRSo87VKhdYFBaJCd7r4AdwzQJUq2A3/MSa9oMTTgAAAAABaZhi7AAAAAgAAAAgAAAABAAAACQAAAAgAAAADAAAnEAAAAAAAAAAB+n7i26Uc6Y+Vr1iXEnyCYrKs9BbqzYBujh8sEyWB12UAAABArCZ8OBaydJRhGDHG4QtVTZe2vk3DI361ROdXiuDcDeJk9zpAvZPFFC1Pd3nb+PRBDUWmUMTxaobBtnmeTqEVBw==');
INSERT INTO scphistory VALUES ('GCXURZRBQFOADUOGKEWCZYFIAREQV4IYQFTJ2OE7VBRV6YMYSEWMJJWS', 3, 'AAAAAK9I5iGBXAHRxlEsLOCoBEkK8RiBZp04n6hjX2GYkSzEAAAAAAAAAAMAAAACAAAAAQAAADA/gwJSbkHFzEOzEvjIfboQZIhrk/ojHJrZq/MnVHEx1QAAAABaZhi8AAAAAAAAAAAAAAAB+n7i26Uc6Y+Vr1iXEnyCYrKs9BbqzYBujh8sEyWB12UAAABAInW06fZil/D4Uey0BrSy7AFC6gmiWlVfOPpeGOcQEwVsoW3W2ihqp7nC9LHiOOnSEtgJBwFMTYvAHXyqtpQHDw==');
=======
=======
>>>>>>> add price to trade query and /trades endpoint
<<<<<<< HEAD
INSERT INTO scphistory VALUES ('GB426VBKIRUOSY3Y4SVKIQIR7NSCGDBIO4KI3NVB7E3KURHEBVRCGYVO', 2, 'AAAAAHmvVCpEaOljeOSqpEER+2QjDCh3FI22ofk2qkTkDWIjAAAAAAAAAAIAAAACAAAAAQAAAEigdRSo87VKhdYFBaJCd7r4AdwzQJUq2A3/MSa9oMTTgAAAAABaNEfDAAAAAgAAAAgAAAABAAAACQAAAAgAAAADAAAnEAAAAAAAAAABuhL9sFSNhYGRKv1eH/KhjvlDqjYl4Jgl0lK6r47NQF8AAABAsy9XbbXQs7+HGyAyxjUtdGrde7uIaXbXHeyR5MSXwHRKjVvB814Fo/YDx4ujatrf+o0s9rFJTWNJL7LIg+iqAg==');
INSERT INTO scphistory VALUES ('GB426VBKIRUOSY3Y4SVKIQIR7NSCGDBIO4KI3NVB7E3KURHEBVRCGYVO', 3, 'AAAAAHmvVCpEaOljeOSqpEER+2QjDCh3FI22ofk2qkTkDWIjAAAAAAAAAAMAAAACAAAAAQAAADAmAJHC4XFCjrN739/1cqsOOG1c9b36mf6YVS1ABhYg0AAAAABaNEfEAAAAAAAAAAAAAAABuhL9sFSNhYGRKv1eH/KhjvlDqjYl4Jgl0lK6r47NQF8AAABAta68YYJ2snffRzY4B3VH/pRivq3w11o2DlUu8LG/w5yoNWs6hvPR3s7PcJt42U1uv2pv5PHf7m2gwYXqTuM8AA==');
=======
INSERT INTO scphistory VALUES ('GDS5BC5DR7ZWX7KYXYAEHMAH3VSRFBA4LZ4EBSW7EVUXLH6WWKBYGFNE', 2, 'AAAAAOXQi6OP82v9WL4AQ7AH3WUShBxeeEDK3yVpdZ/WsoODAAAAAAAAAAIAAAACAAAAAQAAAEigdRSo87VKhdYFBaJCd7r4AdwzQJUq2A3/MSa9oMTTgAAAAABaOFAjAAAAAgAAAAgAAAABAAAACAAAAAgAAAADAAAnEAAAAAAAAAABaXqJtcKeJwt3C/8cD6YEpqTslOwjJUEqLyGaP1OeFAAAAABAiWtHE4X5/QLBjuhihwJdSDruVfD3MH0eYMwk/6dSLRCoB+opFG17fGExikJa6QoETVfmKt7GQKHgFu7V5NOrCw==');
INSERT INTO scphistory VALUES ('GDS5BC5DR7ZWX7KYXYAEHMAH3VSRFBA4LZ4EBSW7EVUXLH6WWKBYGFNE', 3, 'AAAAAOXQi6OP82v9WL4AQ7AH3WUShBxeeEDK3yVpdZ/WsoODAAAAAAAAAAMAAAACAAAAAQAAADBCqYUCXjsTBFkkRYIJcbeyvnrnvOcT75uSTI6qrbHXlAAAAABaOFAkAAAAAAAAAAAAAAABaXqJtcKeJwt3C/8cD6YEpqTslOwjJUEqLyGaP1OeFAAAAABAy1wdhlQuzZvA1zo5jOYmxo5WUQu5PtAKA9PizrBD9Tu19P9aNwyOAVbxhtAT3I/gI9id6p0LBZ5SsMVXV5iuCQ==');
>>>>>>> add price to trade ingestion
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
=======
INSERT INTO scphistory VALUES ('GA56ZPI2LXUIDUGUC6V7LYTHOZQQAI3CGNUNMBMMMRUYG5DXQZ6FB2TI', 2, 'AAAAADvsvRpd6IHQ1Ber9eJndmEAI2IzaNYFjGRpg3R3hnxQAAAAAAAAAAIAAAACAAAAAQAAAEigdRSo87VKhdYFBaJCd7r4AdwzQJUq2A3/MSa9oMTTgAAAAABaTCnTAAAAAgAAAAgAAAABAAAACAAAAAgAAAADAAAnEAAAAAAAAAABUMGCfMY8OmgKkhRX7pRWmJYzktchJEoPaGKkK+kzj4EAAABATBRLnPCaKG+LbyR3xsbTA00soWfT6OegeTJHgCVOztnbL/37THln/BxSBmjJyW6CtbYvOMeFWOhdAapLV8DHAw==');
INSERT INTO scphistory VALUES ('GA56ZPI2LXUIDUGUC6V7LYTHOZQQAI3CGNUNMBMMMRUYG5DXQZ6FB2TI', 3, 'AAAAADvsvRpd6IHQ1Ber9eJndmEAI2IzaNYFjGRpg3R3hnxQAAAAAAAAAAMAAAACAAAAAQAAADA0mRaeCKnJ4PXinA1Xqrl5mRue5BrthOU28vBKCcsyxgAAAABaTCnUAAAAAAAAAAAAAAABUMGCfMY8OmgKkhRX7pRWmJYzktchJEoPaGKkK+kzj4EAAABAriRJloOYnlrw+GoqszMMDSeWgEkQRyYNViLBJljyhzmGwydapbcQgu8Wzx80gnLjOz8luNxbySFapZUvUvq8CA==');
>>>>>>> add price to trade query and /trades endpoint
>>>>>>> add price to trade query and /trades endpoint


--
-- Data for Name: scpquorums; Type: TABLE DATA; Schema: public; Owner: -
--

<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO scpquorums VALUES ('fa7ee2dba51ce98f95af5897127c8262b2acf416eacd806e8e1f2c132581d765', 3, 'AAAAAQAAAAEAAAAAr0jmIYFcAdHGUSws4KgESQrxGIFmnTifqGNfYZiRLMQAAAAA');
=======
=======
>>>>>>> add price to trade query and /trades endpoint
<<<<<<< HEAD
INSERT INTO scpquorums VALUES ('ba12fdb0548d8581912afd5e1ff2a18ef943aa3625e09825d252baaf8ecd405f', 3, 'AAAAAQAAAAEAAAAAea9UKkRo6WN45KqkQRH7ZCMMKHcUjbah+TaqROQNYiMAAAAA');
=======
INSERT INTO scpquorums VALUES ('697a89b5c29e270b770bff1c0fa604a6a4ec94ec2325412a2f219a3f539e1400', 3, 'AAAAAQAAAAEAAAAA5dCLo4/za/1YvgBDsAfdZRKEHF54QMrfJWl1n9ayg4MAAAAA');
>>>>>>> add price to trade ingestion
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
=======
INSERT INTO scpquorums VALUES ('50c1827cc63c3a680a921457ee945698963392d721244a0f6862a42be9338f81', 3, 'AAAAAQAAAAEAAAAAO+y9Gl3ogdDUF6v14md2YQAjYjNo1gWMZGmDdHeGfFAAAAAA');
>>>>>>> add price to trade query and /trades endpoint
>>>>>>> add price to trade query and /trades endpoint


--
-- Data for Name: signers; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: storestate; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO storestate VALUES ('databaseschema                  ', '5');
INSERT INTO storestate VALUES ('networkpassphrase               ', 'Test SDF Network ; September 2015');
INSERT INTO storestate VALUES ('forcescponnextlaunch            ', 'false');
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO storestate VALUES ('lastclosedledger                ', 'dd99581a2dd963bd36705ecbd37adc00d81276dc7c0daf18d7e1b6678324bd22');
=======
=======
>>>>>>> add price to trade query and /trades endpoint
<<<<<<< HEAD
INSERT INTO storestate VALUES ('lastclosedledger                ', 'da7595fa0ce98f3c9ece340371a0860f1f6ec0952e3b553cc363bf4f66da52e2');
>>>>>>> add price to trade ingestion
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v9.0.0-4-g59482f9d",
=======
INSERT INTO storestate VALUES ('lastclosedledger                ', '983ac7c36cca7d949d76d82382784e36627d5a477daa535847544800ef048a42');
=======
INSERT INTO storestate VALUES ('lastclosedledger                ', 'f82d7d87ed6d92c0dcb24603c1c503524b83868856a9655aa095c19e6745c773');
>>>>>>> add price to trade query and /trades endpoint
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v0.6.4-32-g176d30f4",
>>>>>>> add price to trade ingestion
    "currentLedger": 3,
    "currentBuckets": [
        {
            "curr": "a8a9304e6c0912dbcfd8d17f5c985c447f165e61e2cc1ba33fc9bb56e03beee1",
            "next": {
                "state": 0
            },
            "snap": "ef31a20a398ee73ce22275ea8177786bac54656f33dcc4f3fec60d55ddf163d9"
        },
        {
            "curr": "0000000000000000000000000000000000000000000000000000000000000000",
            "next": {
                "state": 1,
                "output": "ef31a20a398ee73ce22275ea8177786bac54656f33dcc4f3fec60d55ddf163d9"
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
<<<<<<< HEAD
<<<<<<< HEAD
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAACvSOYhgVwB0cZRLCzgqARJCvEYgWadOJ+oY19hmJEsxAAAAAAAAAADAAAAA/p+4tulHOmPla9YlxJ8gmKyrPQW6s2Abo4fLBMlgddlAAAAAQAAADA/gwJSbkHFzEOzEvjIfboQZIhrk/ojHJrZq/MnVHEx1QAAAABaZhi8AAAAAAAAAAAAAAABAAAAMD+DAlJuQcXMQ7MS+Mh9uhBkiGuT+iMcmtmr8ydUcTHVAAAAAFpmGLwAAAAAAAAAAAAAAECIRl4jfSzP5adW1PVg4TtDa94gG7rJKXkRfq9Xy0PUKMIYLAI/mOcmp+hDCISPU4IH7SNYKVWdW2KZ+S0X6m8PAAAAAK9I5iGBXAHRxlEsLOCoBEkK8RiBZp04n6hjX2GYkSzEAAAAAAAAAAMAAAACAAAAAQAAADA/gwJSbkHFzEOzEvjIfboQZIhrk/ojHJrZq/MnVHEx1QAAAABaZhi8AAAAAAAAAAAAAAAB+n7i26Uc6Y+Vr1iXEnyCYrKs9BbqzYBujh8sEyWB12UAAABAInW06fZil/D4Uey0BrSy7AFC6gmiWlVfOPpeGOcQEwVsoW3W2ihqp7nC9LHiOOnSEtgJBwFMTYvAHXyqtpQHDwAAAAF1W2p3SrcEy7S+1wpRuzPFoNq5SuqOpcQnj8UXr9GxcgAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAAAAAAAAvrwgAAAAAAAAAABruS+TAAAAECxaubprwZCyJVYV7eWzAQjzwKrnqKPbO2UeeoiJHBaE0FHVgxceU3By4gJfMn7ZK7HotCmOktpu7ANRaEdYhkAAAAAAQAAAAEAAAABAAAAAK9I5iGBXAHRxlEsLOCoBEkK8RiBZp04n6hjX2GYkSzEAAAAAA==');
=======
=======
>>>>>>> add price to trade query and /trades endpoint
<<<<<<< HEAD
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAAB5r1QqRGjpY3jkqqRBEftkIwwodxSNtqH5NqpE5A1iIwAAAAAAAAADAAAAA7oS/bBUjYWBkSr9Xh/yoY75Q6o2JeCYJdJSuq+OzUBfAAAAAQAAADAmAJHC4XFCjrN739/1cqsOOG1c9b36mf6YVS1ABhYg0AAAAABaNEfEAAAAAAAAAAAAAAABAAAAMCYAkcLhcUKOs3vf3/Vyqw44bVz1vfqZ/phVLUAGFiDQAAAAAFo0R8QAAAAAAAAAAAAAAECdGnFZdZIsBL/Er6s9uVAunRF/e2oSCSRxCMy73YrcTuDkJIbOQ4LgPBNWSbRnHdAYCOYvR3l26lc40Z6D7ZECAAAAAHmvVCpEaOljeOSqpEER+2QjDCh3FI22ofk2qkTkDWIjAAAAAAAAAAMAAAACAAAAAQAAADAmAJHC4XFCjrN739/1cqsOOG1c9b36mf6YVS1ABhYg0AAAAABaNEfEAAAAAAAAAAAAAAABuhL9sFSNhYGRKv1eH/KhjvlDqjYl4Jgl0lK6r47NQF8AAABAta68YYJ2snffRzY4B3VH/pRivq3w11o2DlUu8LG/w5yoNWs6hvPR3s7PcJt42U1uv2pv5PHf7m2gwYXqTuM8AAAAAAFMLgZoeQzXKJQMjRRuKsiDZUIsOqI8rkxU0sIo7AVhZwAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAAAAAAAAvrwgAAAAAAAAAABruS+TAAAAECxaubprwZCyJVYV7eWzAQjzwKrnqKPbO2UeeoiJHBaE0FHVgxceU3By4gJfMn7ZK7HotCmOktpu7ANRaEdYhkAAAAAAQAAAAEAAAABAAAAAHmvVCpEaOljeOSqpEER+2QjDCh3FI22ofk2qkTkDWIjAAAAAA==');
=======
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAADl0Iujj/Nr/Vi+AEOwB91lEoQcXnhAyt8laXWf1rKDgwAAAAAAAAADAAAAA2l6ibXCnicLdwv/HA+mBKak7JTsIyVBKi8hmj9TnhQAAAAAAQAAADBCqYUCXjsTBFkkRYIJcbeyvnrnvOcT75uSTI6qrbHXlAAAAABaOFAkAAAAAAAAAAAAAAABAAAAMEKphQJeOxMEWSRFgglxt7K+eue85xPvm5JMjqqtsdeUAAAAAFo4UCQAAAAAAAAAAAAAAED+40tsNkc3DxaRlgwgZF1Rv3w9TwRN9wmijipMucLgCibEi2kMvgup8PamwlFQ5nw/gGQBNDDVqz20/sIHCCkGAAAAAOXQi6OP82v9WL4AQ7AH3WUShBxeeEDK3yVpdZ/WsoODAAAAAAAAAAMAAAACAAAAAQAAADBCqYUCXjsTBFkkRYIJcbeyvnrnvOcT75uSTI6qrbHXlAAAAABaOFAkAAAAAAAAAAAAAAABaXqJtcKeJwt3C/8cD6YEpqTslOwjJUEqLyGaP1OeFAAAAABAy1wdhlQuzZvA1zo5jOYmxo5WUQu5PtAKA9PizrBD9Tu19P9aNwyOAVbxhtAT3I/gI9id6p0LBZ5SsMVXV5iuCQAAAAE5/IDjDr9p4iYq6m0fxpLBSYI6gsAphUi6E5vCl9sihQAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAAAAAAAAvrwgAAAAAAAAAABruS+TAAAAECxaubprwZCyJVYV7eWzAQjzwKrnqKPbO2UeeoiJHBaE0FHVgxceU3By4gJfMn7ZK7HotCmOktpu7ANRaEdYhkAAAAAAQAAAAEAAAABAAAAAOXQi6OP82v9WL4AQ7AH3WUShBxeeEDK3yVpdZ/WsoODAAAAAA==');
>>>>>>> add price to trade ingestion
<<<<<<< HEAD
>>>>>>> add price to trade ingestion
=======
=======
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAAA77L0aXeiB0NQXq/XiZ3ZhACNiM2jWBYxkaYN0d4Z8UAAAAAAAAAADAAAAA1DBgnzGPDpoCpIUV+6UVpiWM5LXISRKD2hipCvpM4+BAAAAAQAAADA0mRaeCKnJ4PXinA1Xqrl5mRue5BrthOU28vBKCcsyxgAAAABaTCnUAAAAAAAAAAAAAAABAAAAMDSZFp4Iqcng9eKcDVequXmZG57kGu2E5Tby8EoJyzLGAAAAAFpMKdQAAAAAAAAAAAAAAECknonBbKd8m3+1Yt9NnPFYkmgGm90AwsdUBUDRPQ0WX0Nxf5022eHi4xwmKq7/Qw2752T6zIFmv3JNFGXLejsOAAAAADvsvRpd6IHQ1Ber9eJndmEAI2IzaNYFjGRpg3R3hnxQAAAAAAAAAAMAAAACAAAAAQAAADA0mRaeCKnJ4PXinA1Xqrl5mRue5BrthOU28vBKCcsyxgAAAABaTCnUAAAAAAAAAAAAAAABUMGCfMY8OmgKkhRX7pRWmJYzktchJEoPaGKkK+kzj4EAAABAriRJloOYnlrw+GoqszMMDSeWgEkQRyYNViLBJljyhzmGwydapbcQgu8Wzx80gnLjOz8luNxbySFapZUvUvq8CAAAAAEQRjfyaWTEMhguec/1u+0+8YT7dRPVEy9mx9pk1wyAKAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAAAAAAAAvrwgAAAAAAAAAABruS+TAAAAECxaubprwZCyJVYV7eWzAQjzwKrnqKPbO2UeeoiJHBaE0FHVgxceU3By4gJfMn7ZK7HotCmOktpu7ANRaEdYhkAAAAAAQAAAAEAAAABAAAAADvsvRpd6IHQ1Ber9eJndmEAI2IzaNYFjGRpg3R3hnxQAAAAAA==');
>>>>>>> add price to trade query and /trades endpoint
>>>>>>> add price to trade query and /trades endpoint


--
-- Data for Name: trustlines; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: txfeehistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txfeehistory VALUES ('2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d', 2, 1, 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('f5971def3ff08c05ce222e7d71bf43703bb98ea1f776ea73085265d35dfd42ab', 3, 1, 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAA7msoAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAA7msmcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');


--
-- Data for Name: txhistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txhistory VALUES ('2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d', 2, 1, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAO5rKAAAAAAAAAAABVvwF9wAAAECDzqvkQBQoNAJifPRXDoLhvtycT3lFPCQ51gkdsFHaBNWw05S/VhW0Xgkr0CBPE4NaFV2Kmcs3ZwLmib4TRrML', 'I3Tpk0m57326ml2zM5t4/ajzR3exrzO6RorVwN+UbU0AAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAADuaygAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2s2vJNZwAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');
INSERT INTO txhistory VALUES ('f5971def3ff08c05ce222e7d71bf43703bb98ea1f776ea73085265d35dfd42ab', 3, 1, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAAAAAAAL68IAAAAAAAAAAAa7kvkwAAABAsWrm6a8GQsiVWFe3lswEI88Cq56ij2ztlHnqIiRwWhNBR1YMXHlNwcuICXzJ+2Sux6LQpjpLabuwDUWhHWIZAA==', '9Zcd7z/wjAXOIi59cb9DcDu5jqH3dupzCFJl0139QqsAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAEAAAAAAAAAAA==', 'AAAAAAAAAAEAAAAA');


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
-- PostgreSQL database dump complete
--

