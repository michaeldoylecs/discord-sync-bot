package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx"
	"github.com/michaeldoylecs/discord-sync-bot/config"
	"github.com/michaeldoylecs/discord-sync-bot/db"
)

var commandConfigAddSync = CommandConfig{
	info: &discordgo.ApplicationCommand{
		Name:        "add-sync",
		Description: "Sync a channel's messages with a given file URI's contents.",
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
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "github-repo-url",
				Description: "GitHub repo URL",
				Required:    false,
			},
		},
	},
	handler: func(discordSession *discordgo.Session, appCtx *config.AppCtx) {
		discordSession.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
			if interaction.ApplicationCommandData().Name != "add-sync" {
				return
			}

			// Create logger with command relevant info
			logger := newInteractionLogger(interaction.Interaction)
			defer logExecutionTime(logger, "Command finished executing.")()
			logger.Info().Msg("Command started.")

			// Build options map
			options := interaction.ApplicationCommandData().Options
			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			fileUri := optionMap["file-uri"].StringValue()
			channelId := optionMap["channel-id"].StringValue()

			// Handle channel not existing within current guild.
			channels, err := session.GuildChannels(interaction.GuildID)
			if err != nil {
				logger.Error().Err(err).Msg("")
				sendErrorResponse(session, interaction.Interaction)
				return
			}

			if !slices.ContainsFunc(channels, func(c *discordgo.Channel) bool {
				return c.ID == channelId
			}) {
				msg := fmt.Sprintf("Channel: '%s' does not exist in this guild.", channelId)
				sendEphemeralResponse(session, interaction.Interaction, msg)
				return
			}

			// Add sync record to database
			recordInfo := db.AddChannelSyncParams{
				FileToSyncUri:           fileUri,
				DiscordGuildSnowflake:   interaction.GuildID,
				DiscordChannelSnowflake: channelId,
			}
			syncRecord, err := appCtx.DB.AddChannelSync(context.Background(), recordInfo)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					logger.Error().Interface("record_info", recordInfo).Msg("Failed to add sync record to database.")
				} else {
					logger.Error().Err(err).Msg("")
				}
				sendErrorResponse(session, interaction.Interaction)
				return
			}

			// Associate file with GitHub repo if provided
			if opt, ok := optionMap["github-repo-url"]; ok {
				githubRepoUrl := opt.StringValue()
				_, err = appCtx.DB.AddGithubRepoFile(context.Background(), db.AddGithubRepoFileParams{
					GithubRepoUrl: githubRepoUrl,
					FileToSyncFk:  syncRecord.ID,
				})
				if err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
						logger.Error().Interface("record_info", recordInfo).Msg("Failed to add github repo record to database.")
					} else {
						logger.Error().Err(err).Msg("")
					}
					sendErrorResponse(session, interaction.Interaction)
					return
				}
			}

			// Respond to command
			msg := fmt.Sprintf("Added Sync Record.\n%s\n<#%s>", syncRecord.FileToSyncUri, syncRecord.DiscordChannelSnowflake)
			sendEphemeralResponse(session, interaction.Interaction, msg)
		})
	},
}
