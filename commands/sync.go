package commands

import (
	"context"
	"errors"
	"fmt"
	"log"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/michaeldoylecs/discord-sync-bot/config"
	"github.com/michaeldoylecs/discord-sync-bot/db"
)

var commandConfigSync = CommandConfig{
	info: &discordgo.ApplicationCommand{
		Name:        "sync",
		Description: "Sync's a channel's messages with a given file URI's contents.",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "file-uri",
				Description: "File URI",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "channel-id",
				Description: "Channel ID",
				Required:    true,
			},
		},
	},
	handler: func(discordSession *discordgo.Session, appCtx *config.AppCtx) {
		discordSession.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
			if interaction.ApplicationCommandData().Name != "sync" {
				return
			}

			options := interaction.ApplicationCommandData().Options
			fileUri := options[0].Value.(string)
			channelId := options[1].Value.(string)

			// Handle channel not existing within current guild.
			channels, err := session.GuildChannels(interaction.GuildID)
			if err != nil {
				log.Println(err)
			}

			if !slices.ContainsFunc(channels, func(c *discordgo.Channel) bool {
				return c.ID == channelId
			}) {
				msg := fmt.Sprintf("Channel: '%s' does not exist in this guild.", channelId)

				session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: msg,
					},
				})
			}

			// Add sync record to database
			recordInfo := db.AddChannelSyncParams{
				FileToSyncUri:           pgtype.Text{String: fileUri, Valid: true},
				DiscordGuildSnowflake:   interaction.GuildID,
				DiscordChannelSnowflake: channelId,
			}
			sync, err := appCtx.DB.AddChannelSync(context.Background(), recordInfo)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					log.Printf("Failed to add sync record to database: %+v\n", recordInfo)
				} else {
					log.Println(err)
				}
				session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: err.Error(),
					},
				})
				return
			}

			// Respond to command
			msg := fmt.Sprintf("Added Sync Record.\n%s\n<#%s>", sync.FileToSyncUri.String, sync.DiscordChannelSnowflake)
			session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
				},
			})
		})
	},
}
