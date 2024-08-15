SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: file_chunk_messages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.file_chunk_messages (
    id bigint NOT NULL,
    files_to_sync_fk bigint NOT NULL,
    chunk_number integer NOT NULL,
    discord_message_id character varying(20) NOT NULL
);


--
-- Name: file_chunk_messages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.file_chunk_messages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: file_chunk_messages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.file_chunk_messages_id_seq OWNED BY public.file_chunk_messages.id;


--
-- Name: files_to_sync; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.files_to_sync (
    file_to_sync_uri character varying(512) DEFAULT ''::character varying NOT NULL,
    discord_guild_snowflake character varying(20) NOT NULL,
    discord_channel_snowflake character varying(20) NOT NULL,
    id bigint NOT NULL,
    file_contents text DEFAULT ''::text NOT NULL
);


--
-- Name: files_to_sync_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.files_to_sync_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: files_to_sync_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.files_to_sync_id_seq OWNED BY public.files_to_sync.id;


--
-- Name: github_repo_files; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.github_repo_files (
    id bigint NOT NULL,
    github_repo_url character varying(512) NOT NULL,
    file_to_sync_fk bigint NOT NULL
);


--
-- Name: github_repo_files_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.github_repo_files_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: github_repo_files_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.github_repo_files_id_seq OWNED BY public.github_repo_files.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying(128) NOT NULL
);


--
-- Name: file_chunk_messages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_chunk_messages ALTER COLUMN id SET DEFAULT nextval('public.file_chunk_messages_id_seq'::regclass);


--
-- Name: files_to_sync id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files_to_sync ALTER COLUMN id SET DEFAULT nextval('public.files_to_sync_id_seq'::regclass);


--
-- Name: github_repo_files id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repo_files ALTER COLUMN id SET DEFAULT nextval('public.github_repo_files_id_seq'::regclass);


--
-- Name: file_chunk_messages file_chunk_messages_chunk_number_discord_message_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_chunk_messages
    ADD CONSTRAINT file_chunk_messages_chunk_number_discord_message_id_key UNIQUE (chunk_number, discord_message_id);


--
-- Name: file_chunk_messages file_chunk_messages_discord_message_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_chunk_messages
    ADD CONSTRAINT file_chunk_messages_discord_message_id_key UNIQUE (discord_message_id);


--
-- Name: file_chunk_messages file_chunk_messages_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_chunk_messages
    ADD CONSTRAINT file_chunk_messages_pkey PRIMARY KEY (id);


--
-- Name: files_to_sync files_to_sync_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files_to_sync
    ADD CONSTRAINT files_to_sync_pkey PRIMARY KEY (id);


--
-- Name: files_to_sync files_to_sync_unique_channel; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.files_to_sync
    ADD CONSTRAINT files_to_sync_unique_channel UNIQUE (discord_guild_snowflake, discord_channel_snowflake);


--
-- Name: github_repo_files github_repo_files_file_to_sync_fk_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repo_files
    ADD CONSTRAINT github_repo_files_file_to_sync_fk_key UNIQUE (file_to_sync_fk);


--
-- Name: github_repo_files github_repo_files_github_repo_url_file_to_sync_fk_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repo_files
    ADD CONSTRAINT github_repo_files_github_repo_url_file_to_sync_fk_key UNIQUE (github_repo_url, file_to_sync_fk);


--
-- Name: github_repo_files github_repo_files_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repo_files
    ADD CONSTRAINT github_repo_files_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: file_chunk_messages file_chunk_messages_files_to_sync_fk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.file_chunk_messages
    ADD CONSTRAINT file_chunk_messages_files_to_sync_fk_fkey FOREIGN KEY (files_to_sync_fk) REFERENCES public.files_to_sync(id);


--
-- Name: github_repo_files github_repo_files_file_to_sync_fk_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.github_repo_files
    ADD CONSTRAINT github_repo_files_file_to_sync_fk_fkey FOREIGN KEY (file_to_sync_fk) REFERENCES public.files_to_sync(id);


--
-- PostgreSQL database dump complete
--


--
-- Dbmate schema migrations
--

INSERT INTO public.schema_migrations (version) VALUES
    ('20240808225441'),
    ('20240811003207'),
    ('20240814073257');
