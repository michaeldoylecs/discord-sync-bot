-- name: AddChannelSync :one
INSERT INTO files_to_sync (file_to_sync_uri, discord_guild_snowflake, discord_channel_snowflake)
VALUES ($1, $2, $3)
ON CONFLICT (discord_guild_snowflake, discord_channel_snowflake)
DO UPDATE SET file_to_sync_uri = $1
RETURNING *
;

-- name: SetFileSyncContents :exec
UPDATE files_to_sync
SET file_contents = @file_contents
WHERE discord_channel_snowflake = @channel_id
;

-- name: GetGuildSyncs :many
SELECT * FROM files_to_sync
WHERE discord_guild_snowflake = $1;

-- name: GetGuildChannelSync :one
SELECT * FROM files_to_sync
WHERE discord_guild_snowflake = @guild_id
  AND discord_channel_snowflake = @channel_id
;

-- name: GetFileContentChunks :many
SELECT
  fcm.chunk_number
  ,fcm.discord_message_id
FROM file_chunk_messages fcm
JOIN files_to_sync fts ON fts.id = fcm.files_to_sync_fk
WHERE fts.discord_channel_snowflake = @channel_id
;

-- name: AddFileContentChunks :many
INSERT INTO file_chunk_messages (files_to_sync_fk, chunk_number, discord_message_id)
VALUES (@files_to_sync_fk, unnest(@chunk_numbers::int[]), unnest(@discord_message_ids::varchar(20)[]))
ON CONFLICT (discord_message_id)
  DO UPDATE SET
    chunk_number = excluded.chunk_number
    ,discord_message_id = excluded.discord_message_id
RETURNING *
;

-- name: RemoveFileContentChunks :exec
DELETE FROM file_chunk_messages WHERE files_to_sync_fk = @file_to_sync_fk
;