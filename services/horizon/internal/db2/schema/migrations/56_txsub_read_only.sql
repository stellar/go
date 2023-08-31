-- +migrate Up

DROP TABLE txsub_results;

-- +migrate Down

CREATE TABLE txsub_results (
    transaction_hash       varchar(64) NOT NULL UNIQUE,
    inner_transaction_hash varchar(64),
    tx_result              text, -- serialized history.Transaction
    submitted_at           timestamp NOT NULL DEFAULT NOW()
);
