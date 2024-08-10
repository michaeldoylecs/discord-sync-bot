package commands

import (
	"github.com/bwmarrin/discordgo"
	"github.com/michaeldoylecs/discord-sync-bot/config"
)

var commandConfigWrite = CommandConfig{
	info: &discordgo.ApplicationCommand{
		Name:        "write-markdown",
		Description: "Writes messages given link to a Markdown file",
	},
	handler: func(discordSession *discordgo.Session, _ *config.AppCtx) {
		discordSession.AddHandler(func(session *discordgo.Session, interaction *discordgo.InteractionCreate) {
			if interaction.ApplicationCommandData().Name != "write-markdown" {
				return
			}

			session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Test message!",
				},
			})
		})
	},
}
