-- migrate:up
create table files_to_sync (
  file_to_sync_uri varchar(512)
  ,discord_guild_snowflake varchar(20)
  ,discord_channel_snowflake varchar(20)
  ,primary key(discord_guild_snowflake, discord_channel_snowflake)
);

-- migrate:down
drop table if exists files_to_sync;
