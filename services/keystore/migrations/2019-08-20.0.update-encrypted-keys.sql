-- +migrate Up

ALTER TABLE public.encrypted_keys
	DROP COLUMN salt,
	DROP COLUMN encrypter_name,

TRUNCATE encrypted_keys;

ALTER TABLE public.encrypted_keys
	ALTER COLUMN encoded_keys_data TYPE jsonb NOT NULL;

-- +migrate Down

ALTER TABLE public.encrypted_keys
	ADD COLUMN salt text NOT NULL,
	ADD COLUMN encrypter_name text NOT NULL,

ALTER TABLE public.encrypted_keys
	ALTER COLUMN encoded_keys_data TYPE bytea NOT NULL;
