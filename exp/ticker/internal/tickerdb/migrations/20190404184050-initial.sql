
-- +migrate Up

--
-- Name: assets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.assets (
    id integer NOT NULL,
    code character varying(12) NOT NULL,
    issuer text NOT NULL,
    type character varying(64) NOT NULL,
    num_accounts integer NOT NULL,
    auth_required boolean NOT NULL,
    auth_revocable boolean NOT NULL,
    amount double precision NOT NULL,
    asset_controlled_by_domain boolean NOT NULL,
    anchor_asset_code character varying(12) NOT NULL,
    anchor_asset_type character varying(64) NOT NULL,
    is_valid boolean NOT NULL,
    validation_error text NOT NULL,
    last_valid timestamp with time zone NOT NULL,
    last_checked timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: assets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.assets_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: assets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.assets_id_seq OWNED BY public.assets.id;


--
-- Name: assets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets ALTER COLUMN id SET DEFAULT nextval('public.assets_id_seq'::regclass);


--
-- Name: assets assets_code_issuer_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_code_issuer_key UNIQUE (code, issuer);


--
-- Name: assets assets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.assets
    ADD CONSTRAINT assets_pkey PRIMARY KEY (id);


-- +migrate Down
