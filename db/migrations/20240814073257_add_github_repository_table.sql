-- migrate:up
CREATE TABLE IF NOT EXISTS github_repo_files (
  id bigserial PRIMARY KEY
  ,github_repo_url varchar(512) NOT NULL 
  ,file_to_sync_fk bigint REFERENCES files_to_sync (id) NOT NULL UNIQUE
  ,UNIQUE (github_repo_url, file_to_sync_fk)
)
;

-- migrate:down

