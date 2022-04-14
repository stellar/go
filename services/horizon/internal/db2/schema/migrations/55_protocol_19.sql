-- +migrate Up
ALTER TABLE history_transactions ADD ledger_bounds                   int8range;     -- xdr.Uint32s
ALTER TABLE history_transactions ADD min_account_sequence            bigint;        -- xdr.SequenceNumber -> int64
ALTER TABLE history_transactions ADD min_account_sequence_age        varchar(20);   -- xdr.TimePoint -> uint64 -> longest uint64 number
ALTER TABLE history_transactions ADD min_account_sequence_ledger_gap bigint;        -- xdr.Int32
ALTER TABLE history_transactions ADD extra_signers                   text[];

-- +migrate Down
ALTER TABLE history_transactions DROP ledger_bounds;
ALTER TABLE history_transactions DROP min_account_sequence;
ALTER TABLE history_transactions DROP min_account_sequence_age;
ALTER TABLE history_transactions DROP min_account_sequence_ledger_gap;
ALTER TABLE history_transactions DROP extra_signers;
