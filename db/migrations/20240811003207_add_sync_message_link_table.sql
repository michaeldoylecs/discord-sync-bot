-- migrate:up
ALTER TABLE files_to_sync
  ADD COLUMN id bigserial
  ,ADD COLUMN file_contents text NOT NULL DEFAULT ''
;

ALTER TABLE files_to_sync
  DROP CONSTRAINT files_to_sync_pkey;
;

ALTER TABLE files_to_sync
  ADD PRIMARY KEY (id)
;

ALTER TABLE files_to_sync
  ADD CONSTRAINT files_to_sync_unique_channel
    UNIQUE (discord_guild_snowflake, discord_channel_snowflake)
;

ALTER TABLE files_to_sync
  ALTER COLUMN file_to_sync_uri SET DEFAULT ''
  ,ALTER COLUMN file_to_sync_uri SET NOT NULL
;

create table file_chunk_messages (
  id bigserial PRIMARY KEY
  ,files_to_sync_fk bigint REFERENCES files_to_sync (id) NOT NULL
  ,chunk_number int NOT NULL
  ,discord_message_id varchar(20) NOT NULL UNIQUE
  ,UNIQUE (chunk_number, discord_message_id)
);

-- migrate:down

