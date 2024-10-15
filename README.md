# Discord Sync Bot
> Better name to be determined 

> [!WARNING]
> This discord bot is not recommended for production use. It is a learning project and not finished to my standards. Use at your own discretion. 

## What is this?
A discord bot that allows you to sync the contents of an external (HTTP resource) file to messages in a discord channel. One command to register the external file, another command to update registered files. Optionally, you can register a github repo to send repo push notifications to update files.

## Building and Deploying

### Requirements
  - Docker
  - [just](https://github.com/casey/just) (If using to run build commands)

### Local Use
  1) Create a `.env` file, populated with the contents of `.env.example`, filling in the blank values
      - Alternatively, specify the environment variables in the calling environment.
  2) Run `just run` to build the necessary docker images and run a local database and discord bot instance.


### "Production" Use
In production, I suggest having an existing postgres database, and skip using the `just` command runner. Instead, build and deploy the `Dockerfile` directly. Like in local use, the environment variables will need to be available in the deployment environment.

## Commands

### add-sync
```
add-sync <file-url> <channel-id> [github-repo-url]
    file-url:         URL of the file to be synced.                        (e.g. https://raw.githubusercontent.com/michaeldoylecs/discord-sync-bot/refs/heads/main/README.md)
    channel-id:       Snowflake of the channel to store file contents.     (e.g. 612810906505407562)
    github-repo-url:  (Optional) URL of the gihub repo to associate with.  (e.g. https://github.com/michaeldoylecs/discord-sync-bot) 
```

The github-repo-url will associate a given file with the given github url. If the github repo is configured to send a webhook message to the discord bot on repo updates, then the discord bot will listen and check for file changes on the associated files.

### sync
```
sync <channel-id>
    channel-id:  Snowflake of the channel sync.  (e.g. 612810906505407562)
```
