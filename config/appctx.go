package config

import (
	"github.com/bwmarrin/discordgo"
	"github.com/michaeldoylecs/discord-sync-bot/db"
)

type AppCtx struct {
	DB             *db.Queries
	DiscordSession *discordgo.Session
}
