-- +migrate Up

ALTER TABLE public.encrypted_keys
	DROP COLUMN salt,
	DROP COLUMN encrypter_name;

ALTER TABLE public.encrypted_keys
	ALTER COLUMN encrypted_keys_data TYPE jsonb;

-- +migrate Down

ALTER TABLE public.encrypted_keys
	ADD COLUMN salt text NOT NULL,
	ADD COLUMN encrypter_name text NOT NULL;

ALTER TABLE public.encrypted_keys
	ALTER COLUMN encrypted_keys_data TYPE bytea;
