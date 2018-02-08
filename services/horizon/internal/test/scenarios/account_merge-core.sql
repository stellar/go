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

INSERT INTO accounts VALUES ('GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 999999979999999800, 2, 0, NULL, '', 'AQAAAA==', 0, 2);
INSERT INTO accounts VALUES ('GA5WBPYA5Y4WAEHXWR2UKO2UO4BUGHUQ74EUPKON2QHV4WRHOIRNKKH2', 19999999900, 8589934592, 0, NULL, '', 'AQAAAA==', 0, 3);


--
-- Data for Name: ban; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: ledgerheaders; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO ledgerheaders VALUES ('63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '0000000000000000000000000000000000000000000000000000000000000000', '572a2e32ff248a07b0e70fd1f6d318c1facd20b6cc08c33d5775259868125a16', 1, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
<<<<<<< HEAD
INSERT INTO ledgerheaders VALUES ('7fd31bfb1bf3a1f70f145cd55e31a547133ff9969bebabb4bf985f44786791e0', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 'f5b7b3993e6721891a7e1d083e3679c232512d63231476ce6826bc1caf56aecb', 2, 1516640327, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZUeFWh3+LY3YTHkhb6t+XikT48AKVUrh+INl3MY9NNtUAAAAAWmYYRwAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAAwWNZfI5WyFWx2m7ToNQChYZ5zqFrwog2j0kXqQNLArv1t7OZPmchiRp+HQg+NnnCMlEtYyMUds5oJrwcr1auywAAAAIN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('70285edabe95af774e27af2dcaa21ed1dd5565519cada12542687dfed7e70b91', '7fd31bfb1bf3a1f70f145cd55e31a547133ff9969bebabb4bf985f44786791e0', 'e6416678d086830ebdfb266fe3c757df011b4082ba5677ecb5542d2ed5369c42', 3, 1516640328, 'AAAACX/TG/sb86H3DxRc1V4xpUcTP/mWm+urtL+YX0R4Z5Hgm88CKvtpL3QJJ2EihvLyLN7m2jAcK0HFeg/6r/ZmidIAAAAAWmYYSAAAAAAAAAAAl3fNvfZdYBxPGC86jq3dI5KKUxZcop2bXz/KHY7Ox/TmQWZ40IaDDr37Jm/jx1ffARtAgrpWd+y1VC0u1TacQgAAAAMN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
<<<<<<< HEAD
INSERT INTO ledgerheaders VALUES ('5eb94da4423469ecd32a028c8571135978f57d41b9510c991947169eb1c3551b', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 'f5b7b3993e6721891a7e1d083e3679c232512d63231476ce6826bc1caf56aecb', 2, 1513375505, 'AAAACWPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZUeFWh3+LY3YTHkhb6t+XikT48AKVUrh+INl3MY9NNtUAAAAAWjRHEQAAAAIAAAAIAAAAAQAAAAkAAAAIAAAAAwAAJxAAAAAAwWNZfI5WyFWx2m7ToNQChYZ5zqFrwog2j0kXqQNLArv1t7OZPmchiRp+HQg+NnnCMlEtYyMUds5oJrwcr1auywAAAAIN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('d31840d2ab69b30accae9017be46afccb981b06342262b8214ac2ba70deb6f1f', '5eb94da4423469ecd32a028c8571135978f57d41b9510c991947169eb1c3551b', 'e6416678d086830ebdfb266fe3c757df011b4082ba5677ecb5542d2ed5369c42', 3, 1513375506, 'AAAACV65TaRCNGns0yoCjIVxE1l49X1BuVEMmRlHFp6xw1UbqQ6YXiMR5qYXw7VJkALyvQZHtsPDRbxZtLnl2JjswPYAAAAAWjRHEgAAAAAAAAAAl3fNvfZdYBxPGC86jq3dI5KKUxZcop2bXz/KHY7Ox/TmQWZ40IaDDr37Jm/jx1ffARtAgrpWd+y1VC0u1TacQgAAAAMN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
=======
INSERT INTO ledgerheaders VALUES ('8150660b7516f2771b521ff4a25ded6a4d41c3ca350f493fcdcd5b36e1e0f415', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 'f5b7b3993e6721891a7e1d083e3679c232512d63231476ce6826bc1caf56aecb', 2, 1513639941, 'AAAACGPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZUeFWh3+LY3YTHkhb6t+XikT48AKVUrh+INl3MY9NNtUAAAAAWjhQBQAAAAIAAAAIAAAAAQAAAAgAAAAIAAAAAwAAJxAAAAAAwWNZfI5WyFWx2m7ToNQChYZ5zqFrwog2j0kXqQNLArv1t7OZPmchiRp+HQg+NnnCMlEtYyMUds5oJrwcr1auywAAAAIN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('9042b7ca4a783045266db7231cb9450acf2d46df11ae9813d26b16435e91850f', '8150660b7516f2771b521ff4a25ded6a4d41c3ca350f493fcdcd5b36e1e0f415', 'e6416678d086830ebdfb266fe3c757df011b4082ba5677ecb5542d2ed5369c42', 3, 1513639942, 'AAAACIFQZgt1FvJ3G1If9KJd7WpNQcPKNQ9JP83NWzbh4PQVh9PHC3LlGgc9MCWdBrLfK4U1yIhCHORK1/KOn2OZ2DoAAAAAWjhQBgAAAAAAAAAAl3fNvfZdYBxPGC86jq3dI5KKUxZcop2bXz/KHY7Ox/TmQWZ40IaDDr37Jm/jx1ffARtAgrpWd+y1VC0u1TacQgAAAAMN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
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
INSERT INTO scphistory VALUES ('GBZ5HNUYFQCIGPFATHBTFTH5JSTCYW23SGHU7DQWKPWUKDXTWFGPR5JA', 2, 'AAAAAHPTtpgsBIM8oJnDMsz9TKYsW1uRj0+OFlPtRQ7zsUz4AAAAAAAAAAIAAAACAAAAAQAAAEhR4VaHf4tjdhMeSFvq35eKRPjwApVSuH4g2Xcxj0021QAAAABaZhhHAAAAAgAAAAgAAAABAAAACQAAAAgAAAADAAAnEAAAAAAAAAABPb3ITpHvffSyfdD17Mgi2V2zv6/SxGOfPC9Gpaer14oAAABA0pWBVjKZ/iue6ZU/YCUSoj9h0CbWZK5elaW+UDsq94c+ZmbKKE0IwzBHqyzpn6pgbleLLxXrPg+gKDDHH6CmDA==');
INSERT INTO scphistory VALUES ('GBZ5HNUYFQCIGPFATHBTFTH5JSTCYW23SGHU7DQWKPWUKDXTWFGPR5JA', 3, 'AAAAAHPTtpgsBIM8oJnDMsz9TKYsW1uRj0+OFlPtRQ7zsUz4AAAAAAAAAAMAAAACAAAAAQAAADCbzwIq+2kvdAknYSKG8vIs3ubaMBwrQcV6D/qv9maJ0gAAAABaZhhIAAAAAAAAAAAAAAABPb3ITpHvffSyfdD17Mgi2V2zv6/SxGOfPC9Gpaer14oAAABADJtgDnqgEckxjTDVJF6862GIR5enXmLQZKpnYSA+ocqT08rYkSs/u0MGmUxPrOIm2eJ0lfX7GEPTFdD9u7IGCA==');
=======
<<<<<<< HEAD
INSERT INTO scphistory VALUES ('GAHJDP3SMJLNBFXEF523BNSRU2JAMUBCUT6QCWULBDIJHNLPJ47674W5', 2, 'AAAAAA6Rv3JiVtCW5C91sLZRppIGUCKk/QFaiwjQk7VvTz/vAAAAAAAAAAIAAAACAAAAAQAAAEhR4VaHf4tjdhMeSFvq35eKRPjwApVSuH4g2Xcxj0021QAAAABaNEcRAAAAAgAAAAgAAAABAAAACQAAAAgAAAADAAAnEAAAAAAAAAABSI1cAwXlLuCFTcThaPWJIXukIAI9+Wn/C5RvSr5DI70AAABAgB+ZRz2ZPA+RzTTwICi2HYg4eVTV1X/+pARDwUYhOBYqENEmC4TY2HvUdJNU2PELASaF5IXW+PUXL3Lzit7eAA==');
INSERT INTO scphistory VALUES ('GAHJDP3SMJLNBFXEF523BNSRU2JAMUBCUT6QCWULBDIJHNLPJ47674W5', 3, 'AAAAAA6Rv3JiVtCW5C91sLZRppIGUCKk/QFaiwjQk7VvTz/vAAAAAAAAAAMAAAACAAAAAQAAADCpDpheIxHmphfDtUmQAvK9Bke2w8NFvFm0ueXYmOzA9gAAAABaNEcSAAAAAAAAAAAAAAABSI1cAwXlLuCFTcThaPWJIXukIAI9+Wn/C5RvSr5DI70AAABA+u95he3C1kg3EvMLg8toKUGYsFvZLZP2HANlIsuNg0ZbsF3ljWP4jlMfl/ZKgaWVkOOENXO8XiBsqHxHAPRSCw==');
=======
INSERT INTO scphistory VALUES ('GABD4LSHGS723DXQBF6WRG364FVXKUJWR4V32NDGD2YWS4C7TZE6XVCU', 2, 'AAAAAAI+Lkc0v62O8Al9aJt+4Wt1UTaPK700Zh6xaXBfnknrAAAAAAAAAAIAAAACAAAAAQAAAEhR4VaHf4tjdhMeSFvq35eKRPjwApVSuH4g2Xcxj0021QAAAABaOFAFAAAAAgAAAAgAAAABAAAACAAAAAgAAAADAAAnEAAAAAAAAAABUQ8z9qT/v3Lbw4zU0L7Dy3ipoF2rWXyyZI7xH/D5d/IAAABAeK6TbhyLLjdS9J4EPRTqM8fwBbg0OBQucZHAj5rWOMYQZMzpCMvIRIjdUqahP4Afc8oduwbKiXT3dhuX8R9WBg==');
INSERT INTO scphistory VALUES ('GABD4LSHGS723DXQBF6WRG364FVXKUJWR4V32NDGD2YWS4C7TZE6XVCU', 3, 'AAAAAAI+Lkc0v62O8Al9aJt+4Wt1UTaPK700Zh6xaXBfnknrAAAAAAAAAAMAAAACAAAAAQAAADCH08cLcuUaBz0wJZ0Gst8rhTXIiEIc5ErX8o6fY5nYOgAAAABaOFAGAAAAAAAAAAAAAAABUQ8z9qT/v3Lbw4zU0L7Dy3ipoF2rWXyyZI7xH/D5d/IAAABAZUgiupkdzxUNr/s3/bUTf8QX/mB15/kvlGc/y0NBlQyo1onvd6zY8SD0HGbcV+YZH7BOeTFdnZ881mozac0bAw==');
>>>>>>> add price to trade ingestion
>>>>>>> add price to trade ingestion


--
-- Data for Name: scpquorums; Type: TABLE DATA; Schema: public; Owner: -
--

<<<<<<< HEAD
INSERT INTO scpquorums VALUES ('3dbdc84e91ef7df4b27dd0f5ecc822d95db3bfafd2c4639f3c2f46a5a7abd78a', 3, 'AAAAAQAAAAEAAAAAc9O2mCwEgzygmcMyzP1MpixbW5GPT44WU+1FDvOxTPgAAAAA');
=======
<<<<<<< HEAD
INSERT INTO scpquorums VALUES ('488d5c0305e52ee0854dc4e168f589217ba420023df969ff0b946f4abe4323bd', 3, 'AAAAAQAAAAEAAAAADpG/cmJW0JbkL3WwtlGmkgZQIqT9AVqLCNCTtW9PP+8AAAAA');
=======
INSERT INTO scpquorums VALUES ('510f33f6a4ffbf72dbc38cd4d0bec3cb78a9a05dab597cb2648ef11ff0f977f2', 3, 'AAAAAQAAAAEAAAAAAj4uRzS/rY7wCX1om37ha3VRNo8rvTRmHrFpcF+eSesAAAAA');
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
INSERT INTO storestate VALUES ('lastclosedledger                ', '70285edabe95af774e27af2dcaa21ed1dd5565519cada12542687dfed7e70b91');
=======
<<<<<<< HEAD
INSERT INTO storestate VALUES ('lastclosedledger                ', 'd31840d2ab69b30accae9017be46afccb981b06342262b8214ac2ba70deb6f1f');
>>>>>>> add price to trade ingestion
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v9.0.0-4-g59482f9d",
=======
INSERT INTO storestate VALUES ('lastclosedledger                ', '9042b7ca4a783045266db7231cb9450acf2d46df11ae9813d26b16435e91850f');
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v0.6.4-32-g176d30f4",
>>>>>>> add price to trade ingestion
    "currentLedger": 3,
    "currentBuckets": [
        {
            "curr": "02264183de252bcb4f21b3172373876f05f8ea24fdb3a246d748997f90e1a1b7",
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
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAABz07aYLASDPKCZwzLM/UymLFtbkY9PjhZT7UUO87FM+AAAAAAAAAADAAAAAz29yE6R7330sn3Q9ezIItlds7+v0sRjnzwvRqWnq9eKAAAAAQAAADCbzwIq+2kvdAknYSKG8vIs3ubaMBwrQcV6D/qv9maJ0gAAAABaZhhIAAAAAAAAAAAAAAABAAAAMJvPAir7aS90CSdhIoby8ize5towHCtBxXoP+q/2ZonSAAAAAFpmGEgAAAAAAAAAAAAAAEAVn8PCF9EVh+zJha8tptIUF25YPKlh3NoOlTF7Y9afhoiTY7fUnoayfNEsAQjaQl3Y0R34PveghKsWBE7a+NUGAAAAAHPTtpgsBIM8oJnDMsz9TKYsW1uRj0+OFlPtRQ7zsUz4AAAAAAAAAAMAAAACAAAAAQAAADCbzwIq+2kvdAknYSKG8vIs3ubaMBwrQcV6D/qv9maJ0gAAAABaZhhIAAAAAAAAAAAAAAABPb3ITpHvffSyfdD17Mgi2V2zv6/SxGOfPC9Gpaer14oAAABADJtgDnqgEckxjTDVJF6862GIR5enXmLQZKpnYSA+ocqT08rYkSs/u0MGmUxPrOIm2eJ0lfX7GEPTFdD9u7IGCAAAAAF/0xv7G/Oh9w8UXNVeMaVHEz/5lpvrq7S/mF9EeGeR4AAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAACAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAAAAAABruS+TAAAAEAz8O4X3ay1CjSNB+2sS69FGvYVi1ryD8P1ZuZZQnOJTyPtn9IrYaH/+uB7SPRdDzKRvcPwuf3N+ms8rtP5TLMBAAAAAQAAAAEAAAABAAAAAHPTtpgsBIM8oJnDMsz9TKYsW1uRj0+OFlPtRQ7zsUz4AAAAAA==');
=======
<<<<<<< HEAD
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAAAOkb9yYlbQluQvdbC2UaaSBlAipP0BWosI0JO1b08/7wAAAAAAAAADAAAAA0iNXAMF5S7ghU3E4Wj1iSF7pCACPflp/wuUb0q+QyO9AAAAAQAAADCpDpheIxHmphfDtUmQAvK9Bke2w8NFvFm0ueXYmOzA9gAAAABaNEcSAAAAAAAAAAAAAAABAAAAMKkOmF4jEeamF8O1SZAC8r0GR7bDw0W8WbS55diY7MD2AAAAAFo0RxIAAAAAAAAAAAAAAEBE0Zed9J1sHTie/qdUGzRb6pkX3uN636ZfH29FhvwYgsBZphZ4FQIa16XrCCZRBsdoumED+EVGYEGrDQAFYT4DAAAAAA6Rv3JiVtCW5C91sLZRppIGUCKk/QFaiwjQk7VvTz/vAAAAAAAAAAMAAAACAAAAAQAAADCpDpheIxHmphfDtUmQAvK9Bke2w8NFvFm0ueXYmOzA9gAAAABaNEcSAAAAAAAAAAAAAAABSI1cAwXlLuCFTcThaPWJIXukIAI9+Wn/C5RvSr5DI70AAABA+u95he3C1kg3EvMLg8toKUGYsFvZLZP2HANlIsuNg0ZbsF3ljWP4jlMfl/ZKgaWVkOOENXO8XiBsqHxHAPRSCwAAAAFeuU2kQjRp7NMqAoyFcRNZePV9QblRDJkZRxaescNVGwAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAACAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAAAAAABruS+TAAAAEAz8O4X3ay1CjSNB+2sS69FGvYVi1ryD8P1ZuZZQnOJTyPtn9IrYaH/+uB7SPRdDzKRvcPwuf3N+ms8rtP5TLMBAAAAAQAAAAEAAAABAAAAAA6Rv3JiVtCW5C91sLZRppIGUCKk/QFaiwjQk7VvTz/vAAAAAA==');
=======
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAAACPi5HNL+tjvAJfWibfuFrdVE2jyu9NGYesWlwX55J6wAAAAAAAAADAAAAA1EPM/ak/79y28OM1NC+w8t4qaBdq1l8smSO8R/w+XfyAAAAAQAAADCH08cLcuUaBz0wJZ0Gst8rhTXIiEIc5ErX8o6fY5nYOgAAAABaOFAGAAAAAAAAAAAAAAABAAAAMIfTxwty5RoHPTAlnQay3yuFNciIQhzkStfyjp9jmdg6AAAAAFo4UAYAAAAAAAAAAAAAAEDC56/tJVXkep/n3T2g5+iTgCjpuC5T9MsmDAtIQ/hTPcSwMeFmrIIVeuC1i62pj0EkT+MFYLUQF4ebe9lJmzoCAAAAAAI+Lkc0v62O8Al9aJt+4Wt1UTaPK700Zh6xaXBfnknrAAAAAAAAAAMAAAACAAAAAQAAADCH08cLcuUaBz0wJZ0Gst8rhTXIiEIc5ErX8o6fY5nYOgAAAABaOFAGAAAAAAAAAAAAAAABUQ8z9qT/v3Lbw4zU0L7Dy3ipoF2rWXyyZI7xH/D5d/IAAABAZUgiupkdzxUNr/s3/bUTf8QX/mB15/kvlGc/y0NBlQyo1onvd6zY8SD0HGbcV+YZH7BOeTFdnZ881mozac0bAwAAAAGBUGYLdRbydxtSH/SiXe1qTUHDyjUPST/NzVs24eD0FQAAAAEAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAABkAAAAAgAAAAEAAAAAAAAAAAAAAAEAAAAAAAAACAAAAAA7YL8A7jlgEPe0dUU7VHcDQx6Q/wlHqc3UD15aJ3Ii1QAAAAAAAAABruS+TAAAAEAz8O4X3ay1CjSNB+2sS69FGvYVi1ryD8P1ZuZZQnOJTyPtn9IrYaH/+uB7SPRdDzKRvcPwuf3N+ms8rtP5TLMBAAAAAQAAAAEAAAABAAAAAAI+Lkc0v62O8Al9aJt+4Wt1UTaPK700Zh6xaXBfnknrAAAAAA==');
>>>>>>> add price to trade ingestion
>>>>>>> add price to trade ingestion


--
-- Data for Name: trustlines; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: txfeehistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txfeehistory VALUES ('b2a227c39c64a44fc7abd4c96819456f0399906d12c476d70b402bfdb296d6a3', 2, 1, 'AAAAAgAAAAMAAAABAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('36be70fb7782f9801cdcedc1206e21f99293c99860a15e441f4749747a0a37ab', 2, 2, 'AAAAAgAAAAMAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAACAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/84AAAAAAAAAAIAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txfeehistory VALUES ('734be94762dd4b7f98f644de207273f1a139f53aefc2a1eeb61886118ca7827f', 3, 1, 'AAAAAgAAAAMAAAACAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+QAAAAAAgAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAACuo3ot45qCPExpQ/3oHN+z17Ryis1lfMFYmQWgruS+TAAAAAJUC+OcAAAAAgAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');


--
-- Data for Name: txhistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO txhistory VALUES ('b2a227c39c64a44fc7abd4c96819456f0399906d12c476d70b402bfdb296d6a3', 2, 1, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEDt3KwmaPuPdFSUxdAFeb6OQetyQKIWazlbSMMhmHKNLD4sqhEqUZcQP0l+X/Op+osWmN6+FUYbsz75Q2jG4vMM', 'sqInw5xkpE/Hq9TJaBlFbwOZkG0SxHbXC0Ar/bKW1qMAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2sVNYGzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');
INSERT INTO txhistory VALUES ('36be70fb7782f9801cdcedc1206e21f99293c99860a15e441f4749747a0a37ab', 2, 2, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAACAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAACVAvkAAAAAAAAAAABVvwF9wAAAEA3xWbxPObnZMiBGFKLJQufJLguTsHJxyAsPP5F9Zj561aXnvN/HVRJbFsEcitGbgi9dWVdKRYvmVWCizIdmLID', 'Nr5w+3eC+YAc3O3BIG4h+ZKTyZhgoV5EH0dJdHoKN6sAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAAAAAAEAAAACAAAAAAAAAAIAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAIAAAAAAAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3DeC2rv9MNzgAAAAAAAAAAgAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAA');
INSERT INTO txhistory VALUES ('734be94762dd4b7f98f644de207273f1a139f53aefc2a1eeb61886118ca7827f', 3, 1, 'AAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAZAAAAAIAAAABAAAAAAAAAAAAAAABAAAAAAAAAAgAAAAAO2C/AO45YBD3tHVFO1R3A0MekP8JR6nN1A9eWidyItUAAAAAAAAAAa7kvkwAAABAM/DuF92stQo0jQftrEuvRRr2FYta8g/D9WbmWUJziU8j7Z/SK2Gh//rge0j0XQ8ykb3D8Ln9zfprPK7T+UyzAQ==', 'c0vpR2LdS3+Y9kTeIHJz8aE59TrvwqHuthiGEYyngn8AAAAAAAAAZAAAAAAAAAABAAAAAAAAAAgAAAAAAAAAAlQL45wAAAAA', 'AAAAAAAAAAEAAAAEAAAAAwAAAAIAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAAAlQL5AAAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAMAAAAAAAAAADtgvwDuOWAQ97R1RTtUdwNDHpD/CUepzdQPXlonciLVAAAABKgXx5wAAAACAAAAAAAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAwAAAAMAAAAAAAAAAK6jei3jmoI8TGlD/egc37PXtHKKzWV8wViZBaCu5L5MAAAAAlQL45wAAAACAAAAAQAAAAAAAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAgAAAAAAAAAArqN6LeOagjxMaUP96Bzfs9e0corNZXzBWJkFoK7kvkw=');


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

