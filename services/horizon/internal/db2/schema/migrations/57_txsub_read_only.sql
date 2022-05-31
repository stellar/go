-- +migrate Up

DROP TABLE txsub_results;

-- used to temporarily store filtered-out transactions
-- needed by the transaction system
CREATE TABLE history_transactions_filtered_tmp (
    created_at timestamp NOT NULL DEFAULT NOW()
) INHERITS (history_transactions);

-- +migrate Down

DROP TABLE history_transactions_filtered_tmp;

CREATE TABLE txsub_results (
    transaction_hash       varchar(64) NOT NULL UNIQUE,
    inner_transaction_hash varchar(64),
    tx_result              text, -- serialized history.Transaction
    submitted_at           timestamp NOT NULL DEFAULT NOW()
);
