-- +migrate Up
CREATE TABLE public.accounts_kyc_status (stellar_address text NOT NULL PRIMARY KEY);
-- +migrate Down
DROP TABLE public.accounts_kyc_status;