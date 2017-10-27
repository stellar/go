-- +migrate Up
CREATE TABLE `RecoveryTransactions` (
  `source` varchar(56) NOT NULL,
  `envelope_xdr` text NOT NULL,
  );

-- +migrate Down
DROP TABLE `RecoveryTransactions`;
