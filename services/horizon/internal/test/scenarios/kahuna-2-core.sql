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
    lastmodified integer NOT NULL
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
    type integer NOT NULL,
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
INSERT INTO ledgerheaders VALUES ('cd2bcf2ac6d4ffc16a5f6fe4161c0065676eed7a20b203cbaacfba68af404334', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '735227ed398461291237687b08446aa2c9b096e0c98a462dadda569f05dd2484', 2, 1550591292, 'AAAACmPZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZuZRHr9UdXKbTKiclfOjy72YZFJUkJPVcKT5htvorm1QAAAAAXGwlPAAAAAIAAAAIAAAAAQAAAAoAAAAIAAAAAwAAJxAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERlzUiftOYRhKRI3aHsIRGqiybCW4MmKRi2t2lafBd0khAAAAAIN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('3f059cc45f90a4d33275705bfaa115ee3d57c24ef0cfdf84824494428ca9569d', 'cd2bcf2ac6d4ffc16a5f6fe4161c0065676eed7a20b203cbaacfba68af404334', 'da14da6b0c81b426f580632f565e93a528b11589dc1aee5f5c4e23a0f1eaaa7d', 3, 1550591293, 'AAAACs0rzyrG1P/Bal9v5BYcAGVnbu16ILIDy6rPumivQEM0f1zfD/A1YVA9QDB4hROSuRLHn+6+DvDVGrMRew2HGbMAAAAAXGwlPQAAAAAAAAAAlzJ1vISHXzElAf05LhN7qiqWqKvjHhTijb/BgG6FsuLaFNprDIG0JvWAYy9WXpOlKLEVidwa7l9cTiOg8eqqfQAAAAMN4Lazp2QAAAAAAAAAAABkAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('8d9f1ad151ae9105c51f7597817993b851958edd214a85053429fc9e942d4930', '3f059cc45f90a4d33275705bfaa115ee3d57c24ef0cfdf84824494428ca9569d', 'bf7edb6c1979a8b7b7c4255be766c86dcaac665c236df658df13d39a27eafcba', 4, 1550591294, 'AAAACj8FnMRfkKTTMnVwW/qhFe49V8JO8M/fhIJElEKMqVad/JK8d+LoMdGZBa8sGrlZtu+VcZH9h2qM+sLoegiFdWoAAAAAXGwlPgAAAAAAAAAApOCJokUPy1wA/XbpkumCKr9Nv3B8+fRGt4d1ygPrKBy/fttsGXmot7fEJVvnZshtyqxmXCNt9ljfE9OaJ+r8ugAAAAQN4Lazp2QAAAAAAAAAAADIAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('668003779a38278dd68da6233096575ad8011146369126c02bfe4b819fdf5d4a', '8d9f1ad151ae9105c51f7597817993b851958edd214a85053429fc9e942d4930', 'a486f9d25d7c2d7148e558cea733112d5c6e2f0c31476e980f9461fb553d4d43', 5, 1550591295, 'AAAACo2fGtFRrpEFxR91l4F5k7hRlY7dIUqFBTQp/J6ULUkwCJ2mgSVmSAuZlM5C3G5KpusLucQnk1+jkD+MxiCZW2YAAAAAXGwlPwAAAAAAAAAAnzt6UJX5by6BwcesLvsJTMJPn/Y/s/aBHlEOtmT5mwekhvnSXXwtcUjlWM6nMxEtXG4vDDFHbpgPlGH7VT1NQwAAAAUN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('ed6316ac6ba335e8e955d1bb7438c6ec7a88ad2e4ca0d86c1994a6499b66fec7', '668003779a38278dd68da6233096575ad8011146369126c02bfe4b819fdf5d4a', 'c2d5be06f487f5351e3376163c5166f367883682e402f702329a8f1e8a4397b2', 6, 1550591296, 'AAAACmaAA3eaOCeN1o2mIzCWV1rYARFGNpEmwCv+S4Gf311KjaYvpf/cH+4QP0PdoHQRj7aOKXXpvBRenWsT71P8WdUAAAAAXGwlQAAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERnC1b4G9If1NR4zdhY8UWbzZ4g2guQC9wIymo8eikOXsgAAAAYN4Lazp2QAAAAAAAAAAAEsAAAAAAAAAAAAAAAAAAAAZAX14QAAACcQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');


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

INSERT INTO scphistory VALUES ('GBOS3VGVT3GFIGKSSK2QH7GYMELYQRKETDRGISAMRUQYHIVFNLWRCUHZ', 2, 'AAAAAF0t1NWezFQZUpK1A/zYYReIRUSY4mRIDI0hg6Klau0RAAAAAAAAAAIAAAACAAAAAQAAAEi5lEev1R1cptMqJyV86PLvZhkUlSQk9VwpPmG2+iubVAAAAABcbCU8AAAAAgAAAAgAAAABAAAACgAAAAgAAAADAAAnEAAAAAAAAAABnNObRkCgez99nmQ7YS5XN9MVU4pjM9tn6cm+3GHgLiQAAABAhDhF7xdLvxrI73o1xYk8TSpu0+Ify8wk1p2DTGHMzUOcTQxENz/TcZjBjSGLOYIQd3x+KjsbxM7AdZzkpFC9DQ==');
INSERT INTO scphistory VALUES ('GBOS3VGVT3GFIGKSSK2QH7GYMELYQRKETDRGISAMRUQYHIVFNLWRCUHZ', 3, 'AAAAAF0t1NWezFQZUpK1A/zYYReIRUSY4mRIDI0hg6Klau0RAAAAAAAAAAMAAAACAAAAAQAAADB/XN8P8DVhUD1AMHiFE5K5Esef7r4O8NUasxF7DYcZswAAAABcbCU9AAAAAAAAAAAAAAABnNObRkCgez99nmQ7YS5XN9MVU4pjM9tn6cm+3GHgLiQAAABAUozASjbHrpGI8GArf5LpecnmVPpl7l/5GmV2gS/D5ZDjuAXjSheGnCJfqFb/en9XArq3enVG3QnEtaSKcLIZAA==');
INSERT INTO scphistory VALUES ('GBOS3VGVT3GFIGKSSK2QH7GYMELYQRKETDRGISAMRUQYHIVFNLWRCUHZ', 4, 'AAAAAF0t1NWezFQZUpK1A/zYYReIRUSY4mRIDI0hg6Klau0RAAAAAAAAAAQAAAACAAAAAQAAADD8krx34ugx0ZkFrywauVm275Vxkf2Haoz6wuh6CIV1agAAAABcbCU+AAAAAAAAAAAAAAABnNObRkCgez99nmQ7YS5XN9MVU4pjM9tn6cm+3GHgLiQAAABAp9LVlT7YCQ2e6jMQgmqUmFvTbe0EgywIPvlJD9uZ+zopDZrWf3cACSZVhap1NG2XrEKDG6bZbT8Ffa1KwqQSCg==');
INSERT INTO scphistory VALUES ('GBOS3VGVT3GFIGKSSK2QH7GYMELYQRKETDRGISAMRUQYHIVFNLWRCUHZ', 5, 'AAAAAF0t1NWezFQZUpK1A/zYYReIRUSY4mRIDI0hg6Klau0RAAAAAAAAAAUAAAACAAAAAQAAADAInaaBJWZIC5mUzkLcbkqm6wu5xCeTX6OQP4zGIJlbZgAAAABcbCU/AAAAAAAAAAAAAAABnNObRkCgez99nmQ7YS5XN9MVU4pjM9tn6cm+3GHgLiQAAABAjlDA807jVEKmYNAXMc8wZT+rZUYA4hXj53BT2WcSV0zkFGgguchu9RKwXWRB2RzoKl4mSLQVVdMeQ38CfPwbDA==');
INSERT INTO scphistory VALUES ('GBOS3VGVT3GFIGKSSK2QH7GYMELYQRKETDRGISAMRUQYHIVFNLWRCUHZ', 6, 'AAAAAF0t1NWezFQZUpK1A/zYYReIRUSY4mRIDI0hg6Klau0RAAAAAAAAAAYAAAACAAAAAQAAADCNpi+l/9wf7hA/Q92gdBGPto4pdem8FF6daxPvU/xZ1QAAAABcbCVAAAAAAAAAAAAAAAABnNObRkCgez99nmQ7YS5XN9MVU4pjM9tn6cm+3GHgLiQAAABAsH2jji8Pk0PkXLqrNoIdHNtL2YQ8+xNhddz9PTU+TM0nAMGXXNVpmIhJ0H74jk0MT/nG63/cGowCW6Sknt7UDg==');


--
-- Data for Name: scpquorums; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO scpquorums VALUES ('9cd39b4640a07b3f7d9e643b612e5737d315538a6333db67e9c9bedc61e02e24', 6, 'AAAAAQAAAAEAAAAAXS3U1Z7MVBlSkrUD/NhhF4hFRJjiZEgMjSGDoqVq7REAAAAA');


--
-- Data for Name: signers; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO signers VALUES ('GD6NTRJW5Z6NCWH4USWMNEYF77RUR2MTO6NP4KEDVJATTCUXDRO3YIFS', 'XC6GFVFYBWPDNWRJYFWF2TM7CFZR6NQFFRZEAGTWYI6A7NNJW5CCGCON', 1);


--
-- Data for Name: storestate; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO storestate VALUES ('databaseschema                  ', '8');
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
INSERT INTO storestate VALUES ('lastclosedledger                ', 'ed6316ac6ba335e8e955d1bb7438c6ec7a88ad2e4ca0d86c1994a6499b66fec7');
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v10.2.0rc2",
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
INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAABdLdTVnsxUGVKStQP82GEXiEVEmOJkSAyNIYOipWrtEQAAAAAAAAAGAAAAA5zTm0ZAoHs/fZ5kO2EuVzfTFVOKYzPbZ+nJvtxh4C4kAAAAAQAAADCNpi+l/9wf7hA/Q92gdBGPto4pdem8FF6daxPvU/xZ1QAAAABcbCVAAAAAAAAAAAAAAAABAAAAMI2mL6X/3B/uED9D3aB0EY+2jil16bwUXp1rE+9T/FnVAAAAAFxsJUAAAAAAAAAAAAAAAEAug8RWf3dZdVwJZCGOGHp2kOQJTPo/s/iN8FIkN8UZ7mQsiEpU8RJU0tLkcUrmodr5xghWyl7czjL1xFduQYUJAAAAAF0t1NWezFQZUpK1A/zYYReIRUSY4mRIDI0hg6Klau0RAAAAAAAAAAYAAAACAAAAAQAAADCNpi+l/9wf7hA/Q92gdBGPto4pdem8FF6daxPvU/xZ1QAAAABcbCVAAAAAAAAAAAAAAAABnNObRkCgez99nmQ7YS5XN9MVU4pjM9tn6cm+3GHgLiQAAABAsH2jji8Pk0PkXLqrNoIdHNtL2YQ8+xNhddz9PTU+TM0nAMGXXNVpmIhJ0H74jk0MT/nG63/cGowCW6Sknt7UDgAAAAFmgAN3mjgnjdaNpiMwllda2AERRjaRJsAr/kuBn99dSgAAAAAAAAABAAAAAQAAAAEAAAAAXS3U1Z7MVBlSkrUD/NhhF4hFRJjiZEgMjSGDoqVq7REAAAAA');


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

INSERT INTO txhistory VALUES ('54c49533d937a906c0e6e501322bb600ffe332bf888cb474bd4261d42f542470', 3, 1, 'AAAAAGL8HQvQkbK2HA3WVjRrKmjX00fG8sLI7m0ERwJW/AX3AAAAZAAAAAAAAAABAAAAAAAAAAAAAAABAAAAAAAAAAAAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAACVAvkAAAAAAAAAAABVvwF9wAAAEDi0+98/ltS2PZOxNCNogdC0ctkOWrNnQ+3eVu2PI3+LNdVssYOrw4gwvZFULsMpS166y7rVfyn6AIp7gqV5pMD', 'VMSVM9k3qQbA5uUBMiu2AP/jMr+IjLR0vUJh1C9UJHAAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAA==', 'AAAAAQAAAAIAAAADAAAAAwAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcN4Lazp2P/nAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAAAAAAAYvwdC9CRsrYcDdZWNGsqaNfTR8bywsjubQRHAlb8BfcN4Lazp2P/nAAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAwAAAAMAAAADAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrOnY/+cAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAADAAAAAAAAAABi/B0L0JGythwN1lY0aypo19NHxvLCyO5tBEcCVvwF9w3gtrFTWBucAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAADAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+QAAAAAAwAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAA==');
INSERT INTO txhistory VALUES ('59de4450bcc1830b6da83fb481aab833db04dadf1e88d568a00274d4038d531e', 4, 1, 'AAAAAPzZxTbufNFY/KSsxpMF/+NI6ZN3mv4og6pBOYqXHF28AAAAZAAAAAMAAAABAAAAAAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAQAAAAAAAAABAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAAAAK8YtS4DZ422inBbF1NnxFzHzYFLHJAGnbCPA+1qbdEIwAAAAEAAAAAAAAAAZccXbwAAABAeVu+uPT8KOhoKoNJidCWqVs71WAIqQns2Zq4mM3LBluMVDHej/SJhUxiKlsSR5MJwQU3trQbPsOAwb56BinbCw==', 'Wd5EULzBgwttqD+0gaq4M9sE2t8eiNVooAJ01AONUx4AAAAAAAAAZAAAAAAAAAABAAAAAAAAAAUAAAAAAAAAAA==', 'AAAAAQAAAAIAAAADAAAABAAAAAAAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAACVAvjnAAAAAMAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAABAAAAAAAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAACVAvjnAAAAAMAAAABAAAAAAAAAAAAAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAABAAAAAgAAAAMAAAAEAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+OcAAAAAwAAAAEAAAAAAAAAAAAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAEAAAAEAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+OcAAAAAwAAAAEAAAABAAAAAAAAAAAAAAAAAQAAAAAAAAEAAAACvGLUuA2eNtopwWxdTZ8Rcx82BSxyQBp2wjwPtam3RCMAAAABAAAAAAAAAAA=');
INSERT INTO txhistory VALUES ('b0879add9f3957c9796d1d1fb23720dbed15d07793c742773455f2706c0e9a25', 5, 1, 'AAAAAPzZxTbufNFY/KSsxpMF/+NI6ZN3mv4og6pBOYqXHF28AAAAZAAAAAMAAAACAAAAAAAAAAAAAAABAAAAAAAAAAoAAAAEZG9uZQAAAAEAAAAEdHJ1ZQAAAAAAAAABqbdEIwAAACC5TSe5k00+CKUuUtfafav6xITv43pTgO6QiPes4u/N6Q==', 'sIea3Z85V8l5bR0fsjcg2+0V0HeTx0J3NFXycGwOmiUAAAAAAAAAZAAAAAAAAAABAAAAAAAAAAoAAAAAAAAAAA==', 'AAAAAQAAAAIAAAADAAAABQAAAAAAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAACVAvjOAAAAAMAAAABAAAAAQAAAAAAAAAAAAAAAAEAAAAAAAABAAAAArxi1LgNnjbaKcFsXU2fEXMfNgUsckAadsI8D7Wpt0QjAAAAAQAAAAAAAAAAAAAAAQAAAAUAAAAAAAAAAPzZxTbufNFY/KSsxpMF/+NI6ZN3mv4og6pBOYqXHF28AAAAAlQL4zgAAAADAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAQAAAAK8YtS4DZ422inBbF1NnxFzHzYFLHJAGnbCPA+1qbdEIwAAAAEAAAAAAAAAAAAAAAEAAAADAAAAAwAAAAUAAAAAAAAAAPzZxTbufNFY/KSsxpMF/+NI6ZN3mv4og6pBOYqXHF28AAAAAlQL4zgAAAADAAAAAgAAAAEAAAAAAAAAAAAAAAABAAAAAAAAAQAAAAK8YtS4DZ422inBbF1NnxFzHzYFLHJAGnbCPA+1qbdEIwAAAAEAAAAAAAAAAAAAAAEAAAAFAAAAAAAAAAD82cU27nzRWPykrMaTBf/jSOmTd5r+KIOqQTmKlxxdvAAAAAJUC+M4AAAAAwAAAAIAAAACAAAAAAAAAAAAAAAAAQAAAAAAAAEAAAACvGLUuA2eNtopwWxdTZ8Rcx82BSxyQBp2wjwPtam3RCMAAAABAAAAAAAAAAAAAAAAAAAABQAAAAMAAAAA/NnFNu580Vj8pKzGkwX/40jpk3ea/iiDqkE5ipccXbwAAAAEZG9uZQAAAAR0cnVlAAAAAAAAAAA=');


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

