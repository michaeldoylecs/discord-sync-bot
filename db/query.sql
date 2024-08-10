-- name: AddChannelSync :one
INSERT INTO files_to_sync (file_to_sync_uri, discord_guild_snowflake, discord_channel_snowflake)
VALUES ($1, $2, $3)
ON CONFLICT (discord_guild_snowflake, discord_channel_snowflake)
DO UPDATE SET file_to_sync_uri = $1
RETURNING *
;

-- name: GetGuildSyncs :many
SELECT * FROM files_to_sync
WHERE discord_guild_snowflake = $1;