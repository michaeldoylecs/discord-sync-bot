-- name: AddChannelSync :one
INSERT INTO files_to_sync (file_to_sync_uri, discord_guild_snowflake, discord_channel_snowflake)
VALUES ($1, $2, $3)
ON CONFLICT (discord_guild_snowflake, discord_channel_snowflake)
DO UPDATE SET file_to_sync_uri = $1
RETURNING *
;

-- name: GetChannelSync :one
SELECT *
FROM files_to_sync
WHERE file_to_sync_uri = @file_uri
;

-- name: SetFileSyncContents :exec
UPDATE files_to_sync
SET file_contents = @file_contents
WHERE discord_channel_snowflake = @channel_id
;

-- name: GetGuildSyncs :many
SELECT * FROM files_to_sync
WHERE discord_guild_snowflake = $1
;

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

-- name: AddGithubRepoFile :one
INSERT INTO github_repo_files (github_repo_url, file_to_sync_fk)
VALUES (@github_repo_url, @file_to_sync_fk)
ON CONFLICT (file_to_sync_fk)
  DO UPDATE SET github_repo_url = @github_repo_url
RETURNING *
;

-- name: GetGithubRepoSyncFiles :many
SELECT
  fts.id AS files_to_sync_id
  ,fts.file_to_sync_uri AS url
  ,fts.file_contents AS file_contents
  ,fts.discord_guild_snowflake AS guild_id
  ,fts.discord_channel_snowflake AS channel_id
FROM github_repo_files grf
  JOIN files_to_sync fts ON fts.id = grf.file_to_sync_fk
WHERE grf.github_repo_url = @github_repo_url
;