-- +migrate Up

CREATE TABLE public.encrypted_keys (
    user_id text NOT NULL,
    type text NOT NULL,
    pubkey text NOT NULL,
    path text,
    extra bytea,
    encrypter_name text NOT NULL,
    encrypted_seed bytea NOT NULL,
    salt text NOT NULL,
    created_at timestamp with time zone NOT NULL DEFAULT NOW(),
    modified_at timestamp with time zone,
    PRIMARY KEY (user_id, pubkey, encrypter_name)
);

CREATE INDEX encrypted_keys_user_id_idx ON public.encrypted_keys (user_id);

-- +migrate Down

DROP TABLE public.encrypted_keys;
