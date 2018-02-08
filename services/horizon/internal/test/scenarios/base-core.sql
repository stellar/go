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

INSERT INTO accounts VALUES ('GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 1000000000, 8589934592, 0, NULL, '', 'AQAAAA==', 0, 2);
INSERT INTO accounts VALUES ('GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 999999996999999700, 3, 0, NULL, '', 'AQAAAA==', 0, 2);
INSERT INTO accounts VALUES ('GBXGQJWVLWOYHFLVTKWV5FGHA3LNYY2JQKM7OAJAUEQFU6LPCSEFVXON', 1050000000, 8589934592, 0, NULL, '', 'AQAAAA==', 0, 3);
INSERT INTO accounts VALUES ('GCXKG6RN4ONIEPCMNFB732A436Z5PNDSRLGWK7GBLCMQLIFO4S7EYWVU', 949999900, 8589934593, 0, NULL, '', 'AQAAAA==', 0, 3);


--
-- Data for Name: ban; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: ledgerheaders; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO ledgerheaders VALUES ('63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '0000000000000000000000000000000000000000000000000000000000000000', '572a2e32ff248a07b0e70fd1f6d318c1facd20b6cc08c33d5775259868125a16', 1, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
<<<<<<< HEAD
INSERT INTO ledgerheaders VALUES ('f04f6d64e6c7ded7f1168f8c4d02d65c3eeeb72d1631ba0c3490daf0904ae302', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '8eb63d15a9e8c24469fc0382b02678bb9ea79abbfd04861fc693cc840e6ee71e', 2, 1516640389, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZlmEdOpVCM5HLr9FNj55qa6w2HKMtqTPFLvG8yPU/aAoAAAAAWmYYhQAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAARUAVxJm1lDMwwqujKcyQzs97F/AETiCgQPrw63wqaPGOtj0VqejCRGn8A4KwJni7nqeau/0Ehh/Gk8yEDm7nHgAAAAIN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('9fe46e192193c73b9b70493330a00854da093c1a5828f9dee64445a060604d33', 'f04f6d64e6c7ded7f1168f8c4d02d65c3eeeb72d1631ba0c3490daf0904ae302', '205d6ca1f76a7635564b509b3df0b5db6571ebb66c04366b1930473c99992ae8', 3, 1516640390, 'AAAACfBPbWTmx97X8RaPjE0C1lw+7rctFjG6DDSQ2vCQSuMCzp+rgwCEKlS60gGNqEggjEPNxUO7qsNdLbUQCShj4WsAAAAAWmYYhgAAAAAAAAAAFMKJva6QmOlDLtejYbhpYI7SUKOfeJbIdkqj9wO1AtogXWyh92p2NVZLUJs98LXbZXHrtmwENmsZMEc8mZkq6AAAAAMN4Lazp2QAAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
<<<<<<< HEAD
INSERT INTO ledgerheaders VALUES ('53d185b8962698607bcf8130dec8813b56c89f7784d214477813fd92289392d0', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '8eb63d15a9e8c24469fc0382b02678bb9ea79abbfd04861fc693cc840e6ee71e', 2, 1513375617, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZlmEdOpVCM5HLr9FNj55qa6w2HKMtqTPFLvG8yPU/aAoAAAAAWjRHgQAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAARUAVxJm1lDMwwqujKcyQzs97F/AETiCgQPrw63wqaPGOtj0VqejCRGn8A4KwJni7nqeau/0Ehh/Gk8yEDm7nHgAAAAIN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('280b1d207329e17239d6b9763121fa25b4f2a76dcbd3bb6b1a86e064b3b7b4b7', '53d185b8962698607bcf8130dec8813b56c89f7784d214477813fd92289392d0', '205d6ca1f76a7635564b509b3df0b5db6571ebb66c04366b1930473c99992ae8', 3, 1513375618, 'AAAACVPRhbiWJphge8+BMN7IgTtWyJ93hNIUR3gT/ZIok5LQIcn8oz35eaHC+560m71fL2PEhx+jsbHtSfzuM85t7soAAAAAWjRHggAAAAAAAAAAFMKJva6QmOlDLtejYbhpYI7SUKOfeJbIdkqj9wO1AtogXWyh92p2NVZLUJs98LXbZXHrtmwENmsZMEc8mZkq6AAAAAMN4Lazp2QAAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
INSERT INTO ledgerheaders VALUES ('65e702a54d61069b953310cde0ff394510383d2584d1c8aceb0ff42abd7529ea', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '8eb63d15a9e8c24469fc0382b02678bb9ea79abbfd04861fc693cc840e6ee71e', 2, 1513640025, 'AAAACGPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZlmEdOpVCM5HLr9FNj55qa6w2HKMtqTPFLvG8yPU/aAoAAAAAWjhQWQAAAAIAAAAIAAAAAQAAAAgAAAAIAAAAAwAAJxAAAAAARUAVxJm1lDMwwqujKcyQzs97F/AETiCgQPrw63wqaPGOtj0VqejCRGn8A4KwJni7nqeau/0Ehh/Gk8yEDm7nHgAAAAIN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('221bff64d6fb5c96b24a8e46dd319f1db589e21357c8ddc0d3c94446bdf0d40e', '65e702a54d61069b953310cde0ff394510383d2584d1c8aceb0ff42abd7529ea', '205d6ca1f76a7635564b509b3df0b5db6571ebb66c04366b1930473c99992ae8', 3, 1513640026, 'AAAACGXnAqVNYQablTMQzeD/OUUQOD0lhNHIrOsP9Cq9dSnq8HeIJ5XiVDXueqLdjp+AlrSrEPi5epFnQRgsrnSc2vYAAAAAWjhQWgAAAAAAAAAAFMKJva6QmOlDLtejYbhpYI7SUKOfeJbIdkqj9wO1AtogXWyh92p2NVZLUJs98LXbZXHrtmwENmsZMEc8mZkq6AAAAAMN4Lazp2QAAAAAAAAAAAGQAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
>>>>>>> add price to trade ingestion
>>>>>>> add price to trade ingestion


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
INSERT INTO scphistory VALUES ('GDLWVGYLK56IEF34IP3WJEPVJGVI42UIJSU74MBRCMSILPJDB7VDNNXB', 2, 'AAAAANdqmwtXfIIXfEP3ZJH1SaqOaohMqf4wMRMkhb0jD+o2AAAAAAAAAAIAAAACAAAAAQAAAEiWYR06lUIzkcuv0U2PnmprrDYcoy2pM8Uu8bzI9T9oCgAAAABaZhiFAAAAAgAAAAgAAAABAAAACQAAAAgAAAADAAAnEAAAAAAAAAABWTvaoY3lLz0ngNGRTi7NdWlOmWJhEnqdBKfDe23UHhoAAABA+IuApN50hyYmx+OJ78qNikGe7HjDzWvuBnI59KlciYQBKiEEL1cAo0+PtjHtuXJ/5Pao4kzd0t9vzbgWJ75GCA==');
INSERT INTO scphistory VALUES ('GDLWVGYLK56IEF34IP3WJEPVJGVI42UIJSU74MBRCMSILPJDB7VDNNXB', 3, 'AAAAANdqmwtXfIIXfEP3ZJH1SaqOaohMqf4wMRMkhb0jD+o2AAAAAAAAAAMAAAACAAAAAQAAADDOn6uDAIQqVLrSAY2oSCCMQ83FQ7uqw10ttRAJKGPhawAAAABaZhiGAAAAAAAAAAAAAAABWTvaoY3lLz0ngNGRTi7NdWlOmWJhEnqdBKfDe23UHhoAAABArRiGTg7a+BeQ9GUlWlhwt0+56vBfJTxNKznGPeA+7qkzU15Xq5Gl8CInQ69syo5TnOWhk9Y0PPImyhnliDuKBg==');
=======
<<<<<<< HEAD
INSERT INTO scphistory VALUES ('GAEDS4CTKRPE2ATMOD7YAP24UR4V2K3K442JD25MQBL2A7QAB6QFWBBE', 2, 'AAAAAAg5cFNUXk0CbHD/gD9cpHldK2rnNJHrrIBXoH4AD6BbAAAAAAAAAAIAAAACAAAAAQAAAEiWYR06lUIzkcuv0U2PnmprrDYcoy2pM8Uu8bzI9T9oCgAAAABaNEeBAAAAAgAAAAgAAAABAAAACQAAAAgAAAADAAAnEAAAAAAAAAABtwMzlX9r5fEyULxIohGx++2WY5pnoXg1X67tobLibicAAABAWWcXpCJwRC9H1vV7qxJ2mlYYP3GEZhSQJVTtbsqUIaKAfIIm2cSknVCx2Spn5kpOL60Gwn29xYaVcflDbLG1Cw==');
INSERT INTO scphistory VALUES ('GAEDS4CTKRPE2ATMOD7YAP24UR4V2K3K442JD25MQBL2A7QAB6QFWBBE', 3, 'AAAAAAg5cFNUXk0CbHD/gD9cpHldK2rnNJHrrIBXoH4AD6BbAAAAAAAAAAMAAAACAAAAAQAAADAhyfyjPfl5ocL7nrSbvV8vY8SHH6Oxse1J/O4zzm3uygAAAABaNEeCAAAAAAAAAAAAAAABtwMzlX9r5fEyULxIohGx++2WY5pnoXg1X67tobLibicAAABA1BSv3F3mEWIvizpw943e+f+sOqWGsGd0P9yMt+yHESopmOYNWmPA5gr7tsal0Q4NjZffLv6Q5YZwKLWabHOIDg==');
=======
INSERT INTO scphistory VALUES ('GCTZE3AO5SJGVOAPNHK54PWR3LTA76G2G7CP65U5J3NHFQ74BUEEZWDI', 2, 'AAAAAKeSbA7skmq4D2nV3j7R2uYP+No3xP92nU7acsP8DQhMAAAAAAAAAAIAAAACAAAAAQAAAEiWYR06lUIzkcuv0U2PnmprrDYcoy2pM8Uu8bzI9T9oCgAAAABaOFBZAAAAAgAAAAgAAAABAAAACAAAAAgAAAADAAAnEAAAAAAAAAABu5og2a4vEy0dlnrph+1nluoMFhIM/6LAw/c/wrGpFE8AAABA/sIJ/NIogDOFZVD7h16bp02yeX59I1gx9/EMw0TIk23RynOU3kA2NSwOGk+UC2SP+I5uNKsezAVCX+glWf3TDA==');
INSERT INTO scphistory VALUES ('GCTZE3AO5SJGVOAPNHK54PWR3LTA76G2G7CP65U5J3NHFQ74BUEEZWDI', 3, 'AAAAAKeSbA7skmq4D2nV3j7R2uYP+No3xP92nU7acsP8DQhMAAAAAAAAAAMAAAACAAAAAQAAADDwd4gnleJUNe56ot2On4CWtKsQ+Ll6kWdBGCyudJza9gAAAABaOFBaAAAAAAAAAAAAAAABu5og2a4vEy0dlnrph+1nluoMFhIM/6LAw/c/wrGpFE8AAABAxd/r2SouyWDYfa+48R4Ci+Ax5W6V5yTcS5ODFjmHw+7HpQIY+8QvfKYOUe19rH/yzMqkdrKRe4+8F88p/KPiDg==');
>>>>>>> add price to trade ingestion
>>>>>>> add price to trade ingestion


--
-- Data for Name: scpquorums; Type: TABLE DATA; Schema: public; Owner: -
--

<<<<<<< HEAD
INSERT INTO scpquorums VALUES ('593bdaa18de52f3d2780d1914e2ecd75694e996261127a9d04a7c37b6dd41e1a', 3, 'AAAAAQAAAAEAAAAA12qbC1d8ghd8Q/dkkfVJqo5qiEyp/jAxEySFvSMP6jYAAAAA');
=======
<<<<<<< HEAD
INSERT INTO scpquorums VALUES ('b70333957f6be5f13250bc48a211b1fbed96639a67a178355faeeda1b2e26e27', 3, 'AAAAAQAAAAEAAAAACDlwU1ReTQJscP+AP1ykeV0rauc0keusgFegfgAPoFsAAAAA');
=======
INSERT INTO scpquorums VALUES ('bb9a20d9ae2f132d1d967ae987ed6796ea0c16120cffa2c0c3f73fc2b1a9144f', 3, 'AAAAAQAAAAEAAAAAp5JsDuySargPadXePtHa5g/42jfE/3adTtpyw/wNCEwAAAAA');
>>>>>>> add price to trade ingestion
>>>>>>> add price to trade ingestion


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
INSERT INTO storestate VALUES ('lastclosedledger                ', '9fe46e192193c73b9b70493330a00854da093c1a5828f9dee64445a060604d33');
=======
<<<<<<< HEAD
INSERT INTO storestate VALUES ('lastclosedledger                ', '280b1d207329e17239d6b9763121fa25b4f2a76dcbd3bb6b1a86e064b3b7b4b7');
>>>>>>> add price to trade ingestion
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v9.0.0-4-g59482f9d",
=======
INSERT INTO storestate VALUES ('lastclosedledger                ', '221bff64d6fb5c96b24a8e46dd319f1db589e21357c8ddc0d3c94446bdf0d40e');
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v0.6.4-32-g176d30f4",
>>>>>>> add price to trade ingestion
    "currentLedger": 3,
    "currentBuckets": [
        {
            "curr": "554613ed35728190bdae94709f59b195fbe9f90bb0841f5de57c15172f90f06a",
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
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAADXapsLV3yCF3xD92SR9UmqjmqITKn+MDETJIW9Iw/qNgAAAAAAAAADAAAAA1k72qGN5S89J4DRkU4uzXVpTpliYRJ6nQSnw3tt1B4aAAAAAQAAADDOn6uDAIQqVLrSAY2oSCCMQ83FQ7uqw10ttRAJKGPhawAAAABaZhiGAAAAAAAAAAAAAAABAAAAMM6fq4MAhCpUutIBjahIIIxDzcVDu6rDXS21EAkoY+FrAAAAAFpmGIYAAAAAAAAAAAAAAEAa2ZS1pll4rGv66p6xUBQddG8cCBfnKKUg770LT6O1xTnAwQ0+WOfnHIsaqtSMbpSanpZF3dPgF9lCNLdECLgNAAAAANdqmwtXfIIXfEP3ZJH1SaqOaohMqf4wMRMkhb0jD+o2AAAAAAAAAAMAAAACAAAAAQAAADDOn6uDAIQqVLrSAY2oSCCMQ83FQ7uqw10ttRAJKGPhawAAAABaZhiGAAAAAAAAAAAAAAABWTvaoY3lLz0ngNGRTi7NdWlOmWJhEnqdBKfDe23UHhoAAABArRiGTg7a+BeQ9GUlWlhwt0+56vBfJTxNKznGPeA+7qkzU15Xq5Gl8CInQ69syo5TnOWhk9Y0PPImyhnliDuKBgAAAAHwT21k5sfe1/EWj4xNAtZcPu63LRYxugw0kNrwkErjAgAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAABuaCbVXZ2DlXWarV6UxwbW3GNJgpn3ASChIFp5bxSIWgAAAAAAAAAAAvrwgAAAAAAAAAABruS+TAAAAED0+72mPKRxFLrSWo4uo3wUfPbjhA/xtpg15NMlkiWvdJtELXeoSv24/g5EODIIH+By6DYYqsMy4rRJPdA5opQHAAAAAQAAAAEAAAABAAAAANdqmwtXfIIXfEP3ZJH1SaqOaohMqf4wMRMkhb0jD+o2AAAAAA==');
=======
<<<<<<< HEAD
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAAAIOXBTVF5NAmxw/4A/XKR5XStq5zSR66yAV6B+AA+gWwAAAAAAAAADAAAAA7cDM5V/a+XxMlC8SKIRsfvtlmOaZ6F4NV+u7aGy4m4nAAAAAQAAADAhyfyjPfl5ocL7nrSbvV8vY8SHH6Oxse1J/O4zzm3uygAAAABaNEeCAAAAAAAAAAAAAAABAAAAMCHJ/KM9+XmhwvuetJu9Xy9jxIcfo7Gx7Un87jPObe7KAAAAAFo0R4IAAAAAAAAAAAAAAEBIxjcwz24XoIo4MkkRLtUekKBoKEMVpYb8PNsvMGHJro/kXBL/vEew9fcYAJxgW3tQtw53WmBjR9lVtUxRYjcCAAAAAAg5cFNUXk0CbHD/gD9cpHldK2rnNJHrrIBXoH4AD6BbAAAAAAAAAAMAAAACAAAAAQAAADAhyfyjPfl5ocL7nrSbvV8vY8SHH6Oxse1J/O4zzm3uygAAAABaNEeCAAAAAAAAAAAAAAABtwMzlX9r5fEyULxIohGx++2WY5pnoXg1X67tobLibicAAABA1BSv3F3mEWIvizpw943e+f+sOqWGsGd0P9yMt+yHESopmOYNWmPA5gr7tsal0Q4NjZffLv6Q5YZwKLWabHOIDgAAAAFT0YW4liaYYHvPgTDeyIE7Vsifd4TSFEd4E/2SKJOS0AAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAABuaCbVXZ2DlXWarV6UxwbW3GNJgpn3ASChIFp5bxSIWgAAAAAAAAAAAvrwgAAAAAAAAAABruS+TAAAAED0+72mPKRxFLrSWo4uo3wUfPbjhA/xtpg15NMlkiWvdJtELXeoSv24/g5EODIIH+By6DYYqsMy4rRJPdA5opQHAAAAAQAAAAEAAAABAAAAAAg5cFNUXk0CbHD/gD9cpHldK2rnNJHrrIBXoH4AD6BbAAAAAA==');
=======
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAACnkmwO7JJquA9p1d4+0drmD/jaN8T/dp1O2nLD/A0ITAAAAAAAAAADAAAAA7uaINmuLxMtHZZ66YftZ5bqDBYSDP+iwMP3P8KxqRRPAAAAAQAAADDwd4gnleJUNe56ot2On4CWtKsQ+Ll6kWdBGCyudJza9gAAAABaOFBaAAAAAAAAAAAAAAABAAAAMPB3iCeV4lQ17nqi3Y6fgJa0qxD4uXqRZ0EYLK50nNr2AAAAAFo4UFoAAAAAAAAAAAAAAEBKPVSPH7HSkiSUdO+Hi96lrxeG08zmj4/Igxo2ZLknW5qpK8Ok6ivy7wYuR8iTqXOGM9/SeS2kiEjIBsKRqQkDAAAAAKeSbA7skmq4D2nV3j7R2uYP+No3xP92nU7acsP8DQhMAAAAAAAAAAMAAAACAAAAAQAAADDwd4gnleJUNe56ot2On4CWtKsQ+Ll6kWdBGCyudJza9gAAAABaOFBaAAAAAAAAAAAAAAABu5og2a4vEy0dlnrph+1nluoMFhIM/6LAw/c/wrGpFE8AAABAxd/r2SouyWDYfa+48R4Ci+Ax5W6V5yTcS5ODFjmHw+7HpQIY+8QvfKYOUe19rH/yzMqkdrKRe4+8F88p/KPiDgAAAAFl5wKlTWEGm5UzEM3g/zlFEDg9JYTRyKzrD/QqvXUp6gAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAAAQAAAABuaCbVXZ2DlXWarV6UxwbW3GNJgpn3ASChIFp5bxSIWgAAAAAAAAAAAvrwgAAAAAAAAAABruS+TAAAAED0+72mPKRxFLrSWo4uo3wUfPbjhA/xtpg15NMlkiWvdJtELXeoSv24/g5EODIIH+By6DYYqsMy4rRJPdA5opQHAAAAAQAAAAEAAAABAAAAAKeSbA7skmq4D2nV3j7R2uYP+No3xP92nU7acsP8DQhMAAAAAA==');
>>>>>>> add price to trade ingestion
>>>>>>> add price to trade ingestion


--
-- Data for Name: trustlines; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: txfeehistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txfeehistory VALUES ('2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d', 2, 1, 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('164a5064eba64f2cdbadb856bf3448485fc626247ada3ed39cddf0f6902133b6', 2, 2, 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('2b2e82dbabb024b27a0c3140ca71d8ac9bc71831f9f5a3bd69eca3d88fb0ec5c', 2, 3, 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/7UAAAAAAAAAAMAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('cebb875a00ff6e1383aef0fd251a76f22c1f9ab2a2dffcb077855736ade2659a', 3, 1, 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAA7msoAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAA7msmcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');


--
-- Data for Name: txhistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txhistory VALUES ('2374e99349b9ef7dba9a5db3339b78fda8f34777b1af33ba468ad5c0df946d4d', 2, 1, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAAAO5rKAAAAAAAAAAABVvwF9wAAAECDzqvkQBQoNAJifPRXDoLhvtycT3lFPCQ51gkdsFHaBNWw05S/VhW0Xgkr0CBPE4NaFV2Kmcs3ZwLmib4TRrML', 'I3Tpk0m57326ml2zM5t4/ajzR3exrzO6RorVwN+UbU0AAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAADuaygAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2s2vJNNQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');
INSERT INTO txhistory VALUES ('164a5064eba64f2cdbadb856bf3448485fc626247ada3ed39cddf0f6902133b6', 2, 2, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAAAO5rKAAAAAAAAAAABVvwF9wAAAEASEZiZbeFwCsrKBnKIus/05VtJDBrgosuhLQ/U6XUj4twWyhs7UtS4CMexOM6JqcfqJK10WlBkkwn4g8PIfjIG', 'FkpQZOumTyzbrbhWvzRISF/GJiR62j7TnN3w9pAhM7YAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAADuaygAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2szAuatQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');
INSERT INTO txhistory VALUES ('2b2e82dbabb024b27a0c3140ca71d8ac9bc71831f9f5a3bd69eca3d88fb0ec5c', 2, 3, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAADAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAbmgm1V2dg5V1mq1elMcG1txjSYKZ9wEgoSBaeW8UiFoAAAAAO5rKAAAAAAAAAAABVvwF9wAAAEDJul1tLGLF4Vxwt0dDCVEf6tb5l4byMrGgCp+lVZMmxct54iNf2mxtjx6Md5ZJ4E4Dlcsf46EAhBGSUPsn8fYD', 'Ky6C26uwJLJ6DDFAynHYrJvHGDH59aO9aeyj2I+w7FwAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAG5oJtVdnYOVdZqtXpTHBtbcY0mCmfcBIKEgWnlvFIhaAAAAADuaygAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2svSToNQAAAAAAAAAAwAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');
INSERT INTO txhistory VALUES ('cebb875a00ff6e1383aef0fd251a76f22c1f9ab2a2dffcb077855736ade2659a', 3, 1, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAEAAAAAbmgm1V2dg5V1mq1elMcG1txjSYKZ9wEgoSBaeW8UiFoAAAAAAAAAAAL68IAAAAAAAAAAAa7kvkwAAABA9Pu9pjykcRS60lqOLqN8FHz244QP8baYNeTTJZIlr3SbRC13qEr9uP4ORDgyCB/gcug2GKrDMuK0ST3QOaKUBw==', 'zruHWgD/bhODrvD9JRp28iwfmrKi3/ywd4VXNq3iZZoAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAEAAAAAAAAAAA==', 'AAAAAAAAAAEAAAAEAAAAAwAAAAIAAAAAAAAAAG5oJtVdnYOVdZqtXpTHBtbcY0mCmfcBIKEgWnlvFIhaAAAAADuaygAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAAG5oJtVdnYOVdZqtXpTHBtbcY0mCmfcBIKEgWnlvFIhaAAAAAD6VuoAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAwAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAADuayZwAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAADif2RwAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');


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

