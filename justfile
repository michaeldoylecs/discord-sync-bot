set dotenv-load := true

program_name := "discord-sync-bot"
build_dir := "build"
build_path := "." / build_dir / program_name

build:
    go build -o {{ build_path }}

run:
    docker compose up --build --force-recreate

sqlc:
    docker compose up --build --force-recreate --abort-on-container-exit database dbmate && sqlc generate

clean:
    docker compose rm -f

format:
    just --unstable --fmt
