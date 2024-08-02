program_name := "discord-sync-bot"
build_dir := "build"
build_path := "." / build_dir / program_name

build:
    go build -o {{ build_path }}

run:
    go build -o {{ build_path }} && {{ build_path }}

format:
    just --unstable --fmt
