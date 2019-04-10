
-- +migrate Up
ALTER TABLE public.assets
    ADD COLUMN issuer_account text NOT NULL;


-- +migrate Down
ALTER TABLE public.assets
    DROP COLUMN issuer_account;
