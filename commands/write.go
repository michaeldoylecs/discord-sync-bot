package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func addWriteCommand(discordSession *discordgo.Session) *discordgo.ApplicationCommand {
	var applicationCommand = discordgo.ApplicationCommand{
		Name:        "write-markdown",
		Description: "Writes messages given link to a Markdown file",
	}

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

	command, err := discordSession.ApplicationCommandCreate(discordSession.State.User.ID, "", &applicationCommand)
	if err != nil {
		log.Fatalf("Failed to create command 'WriteCommand': %s\n", err)
	}

	return command
}
