package config

import "github.com/michaeldoylecs/discord-sync-bot/db"

type AppCtx struct {
	DB *db.Queries
}
