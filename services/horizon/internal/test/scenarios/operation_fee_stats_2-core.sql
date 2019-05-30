running recipe
recipe finished
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
DROP INDEX IF EXISTS public.scpquorumsbyseq;
DROP INDEX IF EXISTS public.scpenvsbyseq;
DROP INDEX IF EXISTS public.ledgersbyseq;
DROP INDEX IF EXISTS public.histfeebyseq;
DROP INDEX IF EXISTS public.histbyseq;
DROP INDEX IF EXISTS public.bestofferindex;
DROP INDEX IF EXISTS public.accountbalances;
ALTER TABLE IF EXISTS ONLY public.upgradehistory DROP CONSTRAINT IF EXISTS upgradehistory_pkey;
ALTER TABLE IF EXISTS ONLY public.txhistory DROP CONSTRAINT IF EXISTS txhistory_pkey;
ALTER TABLE IF EXISTS ONLY public.txfeehistory DROP CONSTRAINT IF EXISTS txfeehistory_pkey;
ALTER TABLE IF EXISTS ONLY public.trustlines DROP CONSTRAINT IF EXISTS trustlines_pkey;
ALTER TABLE IF EXISTS ONLY public.storestate DROP CONSTRAINT IF EXISTS storestate_pkey;
ALTER TABLE IF EXISTS ONLY public.scpquorums DROP CONSTRAINT IF EXISTS scpquorums_pkey;
ALTER TABLE IF EXISTS ONLY public.quoruminfo DROP CONSTRAINT IF EXISTS quoruminfo_pkey;
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
DROP TABLE IF EXISTS public.scpquorums;
DROP TABLE IF EXISTS public.scphistory;
DROP TABLE IF EXISTS public.quoruminfo;
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
    dataname character varying(88) NOT NULL,
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
    homedomain character varying(44) NOT NULL,
    thresholds text NOT NULL,
    flags integer NOT NULL,
    lastmodified integer NOT NULL,
    buyingliabilities bigint,
    sellingliabilities bigint,
    signers text,
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
    sellingasset text NOT NULL,
    buyingasset text NOT NULL,
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
-- Name: quoruminfo; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE quoruminfo (
    nodeid character(56) NOT NULL,
    qsethash character(64) NOT NULL
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



--
-- Data for Name: accounts; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO accounts VALUES ('GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H', 1000000000000000000, 0, 0, NULL, '', 'AQAAAA==', 0, 1, NULL, NULL, NULL);


--
-- Data for Name: ban; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: ledgerheaders; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO ledgerheaders VALUES ('63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', '0000000000000000000000000000000000000000000000000000000000000000', '572a2e32ff248a07b0e70fd1f6d318c1facd20b6cc08c33d5775259868125a16', 1, 0, 'AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABXKi4y/ySKB7DnD9H20xjB+s0gtswIwz1XdSWYaBJaFgAAAAEN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAAABkAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('a16d2092e2942994d5519de3ee382f2ae9d1c87611a29381b82b5f99024d58f2', '63d98f536ee68d1b27b5b89f23af5311b7569a24faf1403ad0b52b633b07be99', 'eff8bb6dff2733ff1f3ffa5141f34ae7571ee3d8cae6dbd129bac511fa0bfd64', 2, 1559239417, 'AAAAC2PZj1Nu5o0bJ7W4nyOvUxG3Vpok+vFAOtC1K2M7B76ZuZRHr9UdXKbTKiclfOjy72YZFJUkJPVcKT5htvorm1QAAAAAXPAa+QAAAAIAAAAIAAAAAQAAAAsAAAAIAAAAAwAPQkAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERnv+Ltt/ycz/x8/+lFB80rnVx7j2Mrm29EpusUR+gv9ZAAAAAIN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAD0JAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('238dd9b2e3dba31cca67255e37c1628d9183d5447ea919af2ae0e9500645a5f3', 'a16d2092e2942994d5519de3ee382f2ae9d1c87611a29381b82b5f99024d58f2', 'eff8bb6dff2733ff1f3ffa5141f34ae7571ee3d8cae6dbd129bac511fa0bfd64', 3, 1559239418, 'AAAAC6FtIJLilCmU1VGd4+44Lyrp0ch2EaKTgbgrX5kCTVjyo1XA61Wq7MuDfOxEdOCFt4XjU0MijVfUFMzNZCrBoZoAAAAAXPAa+gAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERnv+Ltt/ycz/x8/+lFB80rnVx7j2Mrm29EpusUR+gv9ZAAAAAMN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAD0JAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('9dee3aa12ac2574c5296dc7edfeb5ec45b0e9eed1883517d16abae0882997d39', '238dd9b2e3dba31cca67255e37c1628d9183d5447ea919af2ae0e9500645a5f3', 'f79fa0bf4f0941a78d93bbee679f206d87fb0da208857e8fac6ce60968444614', 4, 1559239419, 'AAAACyON2bLj26McymclXjfBYo2Rg9VEfqkZryrg6VAGRaXzP96QsrL686qCSa5j+lVX5uT99UsVyXllbBqHNC9r8OMAAAAAXPAa+wAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERn3n6C/TwlBp42Tu+5nnyBth/sNogiFfo+sbOYJaERGFAAAAAQN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAD0JAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('43ed61171591243f1e3ece9f03005f714e1fda6fc28fb17ae116f98ceb589119', '9dee3aa12ac2574c5296dc7edfeb5ec45b0e9eed1883517d16abae0882997d39', 'f79fa0bf4f0941a78d93bbee679f206d87fb0da208857e8fac6ce60968444614', 5, 1559239420, 'AAAAC53uOqEqwldMUpbcft/rXsRbDp7tGINRfRarrgiCmX05/tRsVRmq0XSeA+qfUBHMbRt/13Qoya0aaC77X1ZnJg8AAAAAXPAa/AAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERn3n6C/TwlBp42Tu+5nnyBth/sNogiFfo+sbOYJaERGFAAAAAUN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAD0JAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('5cb519f5f417f362d101b8d9d452a70f661c87a91b716ad5a659115210f11b08', '43ed61171591243f1e3ece9f03005f714e1fda6fc28fb17ae116f98ceb589119', '6a6ce2f01ea7c9b517e0fe337cc3df702d30f312fc7389eceb9dc49db9e7785c', 6, 1559239421, 'AAAAC0PtYRcVkSQ/Hj7OnwMAX3FOH9pvwo+xeuEW+YzrWJEZVCm1Mf5WMv5B7A+ZFIxyh/nxpixKV2XOSgN5YZ+9G3IAAAAAXPAa/QAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERlqbOLwHqfJtRfg/jN8w99wLTDzEvxziezrncSdued4XAAAAAYN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAD0JAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');
INSERT INTO ledgerheaders VALUES ('15366719873b297876551a86e99b6416f34da9d2a31b26b021c49acd21039418', '5cb519f5f417f362d101b8d9d452a70f661c87a91b716ad5a659115210f11b08', '6a6ce2f01ea7c9b517e0fe337cc3df702d30f312fc7389eceb9dc49db9e7785c', 7, 1559239422, 'AAAAC1y1GfX0F/Ni0QG42dRSpw9mHIepG3Fq1aZZEVIQ8RsIiEZp1+gnfOlG5REj2rw43KNm7iYyzbWtXQBfHVU8aOQAAAAAXPAa/gAAAAAAAAAA3z9hmASpL9tAVxktxD3XSOp3itxSvEmM6AUkwBS4ERlqbOLwHqfJtRfg/jN8w99wLTDzEvxziezrncSdued4XAAAAAcN4Lazp2QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAZAX14QAAD0JAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA');


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
-- Data for Name: quoruminfo; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: scphistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO scphistory VALUES ('GCVQFCTJ6DYBQZ7N3YHNTKYYH7JEZII4V5DT7YIWZKHXEY4HKZ4CJADL', 2, 'AAAAAKsCimnw8Bhn7d4O2asYP9JMoRyvRz/hFsqPcmOHVngkAAAAAAAAAAIAAAACAAAAAQAAAEi5lEev1R1cptMqJyV86PLvZhkUlSQk9VwpPmG2+iubVAAAAABc8Br5AAAAAgAAAAgAAAABAAAACwAAAAgAAAADAA9CQAAAAAAAAAAB1VUZG7IJnYR4vkzwZyIWYyZMlk+H2j5dFS86/ejrCBEAAABA6X1ksVRRnhQzAvCu4Gh8KIqPr8y22Wizf6qAvUXzK2+gTWTOnID2wxC8401ziE/Pc2zIy/CJrmplff8iKKWCAw==');
INSERT INTO scphistory VALUES ('GCVQFCTJ6DYBQZ7N3YHNTKYYH7JEZII4V5DT7YIWZKHXEY4HKZ4CJADL', 3, 'AAAAAKsCimnw8Bhn7d4O2asYP9JMoRyvRz/hFsqPcmOHVngkAAAAAAAAAAMAAAACAAAAAQAAADCjVcDrVarsy4N87ER04IW3heNTQyKNV9QUzM1kKsGhmgAAAABc8Br6AAAAAAAAAAAAAAAB1VUZG7IJnYR4vkzwZyIWYyZMlk+H2j5dFS86/ejrCBEAAABAWtv4tjsYGXQj09RM1LeGoXqeNMvQbPI8VZgBYUON81/iUY2fRJ88ZbvtM0Q5On8GRWvUx8wCzQEEhgZ4qToKBw==');
INSERT INTO scphistory VALUES ('GCVQFCTJ6DYBQZ7N3YHNTKYYH7JEZII4V5DT7YIWZKHXEY4HKZ4CJADL', 4, 'AAAAAKsCimnw8Bhn7d4O2asYP9JMoRyvRz/hFsqPcmOHVngkAAAAAAAAAAQAAAACAAAAAQAAADA/3pCysvrzqoJJrmP6VVfm5P31SxXJeWVsGoc0L2vw4wAAAABc8Br7AAAAAAAAAAAAAAAB1VUZG7IJnYR4vkzwZyIWYyZMlk+H2j5dFS86/ejrCBEAAABAYZzo+HpZqz72mHkBRaTMOYCDQrDwIgD2Oo5qFvkN7HSElh5la6dRbbdsyRk/RV5n/h6HrZaYYvMOq8j3Eb8ADQ==');
INSERT INTO scphistory VALUES ('GCVQFCTJ6DYBQZ7N3YHNTKYYH7JEZII4V5DT7YIWZKHXEY4HKZ4CJADL', 5, 'AAAAAKsCimnw8Bhn7d4O2asYP9JMoRyvRz/hFsqPcmOHVngkAAAAAAAAAAUAAAACAAAAAQAAADD+1GxVGarRdJ4D6p9QEcxtG3/XdCjJrRpoLvtfVmcmDwAAAABc8Br8AAAAAAAAAAAAAAAB1VUZG7IJnYR4vkzwZyIWYyZMlk+H2j5dFS86/ejrCBEAAABABnnsIL8o0MDP/7FdUIbct5u0WPrNEycVxm0PfPDQVQ/vqUcfeYiyVnxPo48+Qmh5IGi+JZY94krKL3mFYYh6CA==');
INSERT INTO scphistory VALUES ('GCVQFCTJ6DYBQZ7N3YHNTKYYH7JEZII4V5DT7YIWZKHXEY4HKZ4CJADL', 6, 'AAAAAKsCimnw8Bhn7d4O2asYP9JMoRyvRz/hFsqPcmOHVngkAAAAAAAAAAYAAAACAAAAAQAAADBUKbUx/lYy/kHsD5kUjHKH+fGmLEpXZc5KA3lhn70bcgAAAABc8Br9AAAAAAAAAAAAAAAB1VUZG7IJnYR4vkzwZyIWYyZMlk+H2j5dFS86/ejrCBEAAABAT/w2Q470iSQMadL+OLpqqpTAD5l6demQRf5MZVOMMcj5edKi24kiXqN6te+Do8Lnjqu7QzJH/MLAeVogsXjsBA==');
INSERT INTO scphistory VALUES ('GCVQFCTJ6DYBQZ7N3YHNTKYYH7JEZII4V5DT7YIWZKHXEY4HKZ4CJADL', 7, 'AAAAAKsCimnw8Bhn7d4O2asYP9JMoRyvRz/hFsqPcmOHVngkAAAAAAAAAAcAAAACAAAAAQAAADCIRmnX6Cd86UblESPavDjco2buJjLNta1dAF8dVTxo5AAAAABc8Br+AAAAAAAAAAAAAAAB1VUZG7IJnYR4vkzwZyIWYyZMlk+H2j5dFS86/ejrCBEAAABAvCW5HFgMM0jv+WPET+I0MvKcOZWe3ee1BZaLw2PlhzQHPFhOxKXCvch270+m+XR8vUXJ8oJXfNrVtGpvSpW7BA==');


--
-- Data for Name: scpquorums; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO scpquorums VALUES ('d555191bb2099d8478be4cf067221663264c964f87da3e5d152f3afde8eb0811', 7, 'AAAAAQAAAAEAAAAAqwKKafDwGGft3g7Zqxg/0kyhHK9HP+EWyo9yY4dWeCQAAAAA');


--
-- Data for Name: storestate; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO storestate VALUES ('lastscpdata                     ', 'AAAAAgAAAACrAopp8PAYZ+3eDtmrGD/STKEcr0c/4RbKj3Jjh1Z4JAAAAAAAAAAHAAAAA9VVGRuyCZ2EeL5M8GciFmMmTJZPh9o+XRUvOv3o6wgRAAAAAQAAAJiIRmnX6Cd86UblESPavDjco2buJjLNta1dAF8dVTxo5AAAAABc8Br+AAAAAAAAAAEAAAAAqwKKafDwGGft3g7Zqxg/0kyhHK9HP+EWyo9yY4dWeCQAAABAOelRLZB176cOjfnUPx2Q1pWo5pN0f5hNDjK0ACTIZYhiDC1+tEHbYAKcR6Prf9UzK3PL5vrvVw5bN5BiOuuxCwAAAAEAAACYiEZp1+gnfOlG5REj2rw43KNm7iYyzbWtXQBfHVU8aOQAAAAAXPAa/gAAAAAAAAABAAAAAKsCimnw8Bhn7d4O2asYP9JMoRyvRz/hFsqPcmOHVngkAAAAQDnpUS2Qde+nDo351D8dkNaVqOaTdH+YTQ4ytAAkyGWIYgwtfrRB22ACnEej63/VMytzy+b671cOWzeQYjrrsQsAAABA5oOycf3MoN+B3+p3Jv85mrghJSLVheFKLi1KgSeD2PlE5e5xzBI4EsRGLnu6qSxpEKyoYlW0pnWPqvuPJsN2AAAAAACrAopp8PAYZ+3eDtmrGD/STKEcr0c/4RbKj3Jjh1Z4JAAAAAAAAAAHAAAAAgAAAAEAAAAwiEZp1+gnfOlG5REj2rw43KNm7iYyzbWtXQBfHVU8aOQAAAAAXPAa/gAAAAAAAAAAAAAAAdVVGRuyCZ2EeL5M8GciFmMmTJZPh9o+XRUvOv3o6wgRAAAAQLwluRxYDDNI7/ljxE/iNDLynDmVnt3ntQWWi8Nj5Yc0BzxYTsSlwr3Idu9Ppvl0fL1FyfKCV3za1bRqb0qVuwQAAAABXLUZ9fQX82LRAbjZ1FKnD2Ych6kbcWrVplkRUhDxGwgAAAAAAAAAAQAAAAEAAAABAAAAAKsCimnw8Bhn7d4O2asYP9JMoRyvRz/hFsqPcmOHVngkAAAAAA==');
INSERT INTO storestate VALUES ('databaseschema                  ', '10');
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
INSERT INTO storestate VALUES ('lastclosedledger                ', '15366719873b297876551a86e99b6416f34da9d2a31b26b021c49acd21039418');
INSERT INTO storestate VALUES ('historyarchivestate             ', '{
    "version": 1,
    "server": "v11.1.0",
    "currentLedger": 7,
    "currentBuckets": [
        {
            "curr": "056cd09fa286b2de1ff329b224259b858f94ec1dcc93f0e0512b2f6819348951",
            "next": {
                "state": 0
            },
            "snap": "056cd09fa286b2de1ff329b224259b858f94ec1dcc93f0e0512b2f6819348951"
        },
        {
            "curr": "e3bc65ded24f6c7ee979f9350aaecf9856f52e75490a401f1a66bb3d77d767dd",
            "next": {
                "state": 1,
                "output": "056cd09fa286b2de1ff329b224259b858f94ec1dcc93f0e0512b2f6819348951"
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


--
-- Data for Name: trustlines; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: txfeehistory; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: txhistory; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: upgradehistory; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO upgradehistory VALUES (2, 1, 'AAAAAQAAAAs=', 'AAAAAA==');
INSERT INTO upgradehistory VALUES (2, 2, 'AAAAAwAPQkA=', 'AAAAAA==');


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
-- Name: quoruminfo quoruminfo_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY quoruminfo
    ADD CONSTRAINT quoruminfo_pkey PRIMARY KEY (nodeid);


--
-- Name: scpquorums scpquorums_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY scpquorums
    ADD CONSTRAINT scpquorums_pkey PRIMARY KEY (qsethash);


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
-- Name: bestofferindex; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX bestofferindex ON offers USING btree (sellingasset, buyingasset, price);


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
-- Name: scpenvsbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX scpenvsbyseq ON scphistory USING btree (ledgerseq);


--
-- Name: scpquorumsbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX scpquorumsbyseq ON scpquorums USING btree (lastledgerseq);


--
-- Name: upgradehistbyseq; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX upgradehistbyseq ON upgradehistory USING btree (ledgerseq);


--
-- PostgreSQL database dump complete
--

