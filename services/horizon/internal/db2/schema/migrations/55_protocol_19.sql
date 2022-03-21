-- +migrate Up
ALTER TABLE history_transactions ADD ledger_bounds           int8range;
ALTER TABLE history_transactions ADD min_account_sequence    integer;
ALTER TABLE history_transactions ADD min_sequence_age        integer;
ALTER TABLE history_transactions ADD min_sequence_ledger_gap integer;
ALTER TABLE history_transactions ADD extra_signers           character varying(165)[];

-- +migrate Down
ALTER TABLE history_transactions DROP ledger_bounds;
ALTER TABLE history_transactions DROP min_account_sequence;
ALTER TABLE history_transactions DROP min_sequence_age;
ALTER TABLE history_transactions DROP min_sequence_ledger_gap;
ALTER TABLE history_transactions DROP extra_signers;
