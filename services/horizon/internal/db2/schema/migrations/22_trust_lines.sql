-- +migrate Up

CREATE TABLE trust_lines (
    accountid character varying(56) NOT NULL,
    assettype int NOT NULL,
    assetissuer character varying(56) NOT NULL,
    assetcode character varying(12) NOT NULL,
    balance bigint NOT NULL,
    tlimit bigint NOT NULL,
    buyingliabilities bigint NOT NULL,
    sellingliabilities bigint NOT NULL,
    flags int NOT NULL,
    last_modified_ledger INT NOT NULL
);

CREATE INDEX trust_lines_by_accountid ON trust_lines USING BTREE(accountid);
CREATE INDEX trust_lines_by_type_code_issuer ON trust_lines USING BTREE(assettype, assetcode, assetissuer);
CREATE INDEX trust_lines_by_issuer ON trust_lines USING BTREE(assetissuer);

-- +migrate Down

DROP TABLE trust_lines cascade;