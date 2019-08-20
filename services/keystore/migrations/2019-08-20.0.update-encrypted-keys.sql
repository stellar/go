-- +migrate Up

ALTER TABLE public.encrypted_keys
	DROP COLUMN salt,
	DROP COLUMN encrypter_name,
	DROP COLUMN encrypted_keys_data;

TRUNCATE encrypted_keys;

ALTER TABLE public.encrypted_keys
	ADD COLUMN encoded_keys_blob text NOT NULL;

-- +migrate Down

ALTER TABLE public.encrypted_keys
	ADD COLUMN salt text NOT NULL,
	ADD COLUMN encrypter_name text NOT NULL,
	ADD COLUMN encrypted_keys_data bytea NOT NULL;

ALTER TABLE public.encrypted_keys
	DROP COLUMN encoded_keys_blob;
