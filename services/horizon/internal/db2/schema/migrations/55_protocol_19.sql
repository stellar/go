-- +migrate Up
ALTER TABLE history_transactions ADD ledger_bounds                   int8range;
ALTER TABLE history_transactions ADD min_account_sequence            bigint;
ALTER TABLE history_transactions ADD min_account_sequence_age        bigint;
ALTER TABLE history_transactions ADD min_account_sequence_ledger_gap bigint;
ALTER TABLE history_transactions ADD extra_signers                   text[];

-- +migrate Down
ALTER TABLE history_transactions DROP ledger_bounds;
ALTER TABLE history_transactions DROP min_account_sequence;
ALTER TABLE history_transactions DROP min_account_sequence_age;
ALTER TABLE history_transactions DROP min_account_sequence_ledger_gap;
ALTER TABLE history_transactions DROP extra_signers;
