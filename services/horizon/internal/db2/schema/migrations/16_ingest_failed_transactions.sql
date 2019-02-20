-- +migrate Up

-- Check db2/history.Transaction.Successful field comment for more information.
ALTER TABLE history_transactions ADD successful boolean;

-- +migrate Down

-- Remove failed transactions!
DELETE FROM history_transactions WHERE successful = false;
ALTER TABLE history_transactions DROP COLUMN successful;
