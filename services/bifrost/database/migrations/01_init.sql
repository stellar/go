CREATE TYPE chain AS ENUM ('ethereum');

CREATE TABLE address_association (
  chain chain NOT NULL,
  address_index bigint NOT NULL UNIQUE,
  address varchar(42) NOT NULL UNIQUE,
  stellar_public_key varchar(56) NOT NULL UNIQUE,
  created_at timestamp NOT NULL,
  PRIMARY KEY (chain, address_index, address, stellar_public_key),
  CONSTRAINT valid_address_index CHECK (address_index >= 0)
);

CREATE TABLE key_value_store (
  key varchar(255) NOT NULL,
  value varchar(255) NOT NULL,
  PRIMARY KEY (key)
);

INSERT INTO key_value_store (key, value) VALUES ('ethereum_address_index', '0');
INSERT INTO key_value_store (key, value) VALUES ('ethereum_last_block', '0');

CREATE TABLE processed_transaction (
  chain chain NOT NULL,
  /* Ethereum: "0x"+hash (so 64+2) */
  transaction_id varchar(66) NOT NULL,
  PRIMARY KEY (chain, transaction_id)
);

/* If using DB storage for the queue not AWS FIFO */
CREATE TABLE transactions_queue (
  /* Ethereum: "0x"+hash (so 64+2) */
  transaction_id varchar(66) NOT NULL,
  asset_code varchar(10) NOT NULL,
  /* ethereum: 100000000 in year 2128 1 Wei     = 0.000000000000000001 */
  /* bitcoin:   21000000              1 Satoshi = 0.00000001 */
  amount varchar(30) NOT NULL,
  stellar_public_key varchar(56) NOT NULL,
  pooled boolean NOT NULL DEFAULT false,
  PRIMARY KEY (transaction_id, asset_code),
  CONSTRAINT valid_asset_code CHECK (char_length(asset_code) >= 3),
  CONSTRAINT valid_stellar_public_key CHECK (char_length(stellar_public_key) = 56)
);
