-- +migrate Up
CREATE TABLE received_payment (
  id bigserial,
  transaction_id varchar(64) NOT NULL,
  operation_id varchar(255) UNIQUE NOT NULL,
  processed_at timestamp NOT NULL,
  paging_token varchar(255) NOT NULL,
  status varchar(255) NOT NULL,
  PRIMARY KEY (id)
);

-- +migrate Down
DROP TABLE received_payment;
