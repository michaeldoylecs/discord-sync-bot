set dotenv-load

program_name := "discord-sync-bot"
build_dir := "build"
build_path := "." / build_dir / program_name

build:
    go build -o {{ build_path }}

run:
    docker compose up --build --force-recreate

clean:
    docker compose rm -f

format:
    just --unstable --fmt
