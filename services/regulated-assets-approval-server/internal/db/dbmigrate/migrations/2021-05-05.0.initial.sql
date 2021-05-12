-- This migration file is intentionally empty and is a first starting point for
-- our migrations before we yet have a schema.
-- +migrate Up
CREATE TABLE public.accounts_kyc_status (
    stellar_address text NOT NULL PRIMARY KEY,
    email_address text,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    kyc_submitted_at timestamp with time zone,
    approved_at timestamp with time zone
);
-- +migrate Down
DROP TABLE public.accounts_kyc_status;