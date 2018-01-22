-- +migrate Up

ALTER TABLE transactions_queue
  ADD COLUMN locked_until TIMESTAMP;

ALTER TABLE transactions_queue
  ADD COLUMN locked_token VARCHAR(16);

ALTER TABLE transactions_queue
  ADD COLUMN failure_count INT;


CREATE TYPE SUBMISSION_TYPE AS ENUM ('submission_create_account', 'submission_send_tokens');

CREATE TABLE transaction_submissions (
  id             BIGSERIAL,
  transaction_id VARCHAR(66)     NOT NULL,
  asset_code     VARCHAR(10)     NOT NULL,
  type           SUBMISSION_TYPE NOT NULL,
  xdr            TEXT            NOT NULL,
  created_at     TIMESTAMP       NOT NULL
);

CREATE UNIQUE INDEX transaction_submissions_idx_type_queue_id
  ON transaction_submissions (transaction_id, asset_code, type);

-- +migrate Down

ALTER TABLE transactions_queue
  DROP COLUMN locked_until;

ALTER TABLE transactions_queue
  DROP COLUMN locked_token;

ALTER TABLE transactions_queue
  DROP COLUMN failure_count;

DROP TABLE transaction_submissions CASCADE;
DROP TYPE SUBMISSION_TYPE;
