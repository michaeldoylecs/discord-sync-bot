package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/michaeldoylecs/discord-sync-bot/config"
	"github.com/samber/lo"
)

type CommandHandler func(*discordgo.Session, *config.AppCtx)

type CommandConfig struct {
	info    discordgo.ApplicationCommand
	handler CommandHandler
}

var commandConfigs = []CommandConfig{
	commandConfigSync,
}

func AddAllCommands(session *discordgo.Session, appCtx *config.AppCtx) {
	guildIds := lo.Map(session.State.Guilds, func(guild *discordgo.Guild, _ int) string {
		return guild.ID
	})

	guildCommands := make([][]*discordgo.ApplicationCommand, len(guildIds))
	for i, guildId := range guildIds {
		commands, err := session.ApplicationCommands(session.State.User.ID, guildId)
		if err != nil {
			log.Fatal(err)
		}
		guildCommands[i] = commands
	}

	log.Println("Unregistering globally registered commands...")
	unregisterGlobalCommands(session)
	log.Println("Finished unregistering globally registered commands.")

	log.Println("Registering guild commands...")
	for _, commandConfig := range commandConfigs {
		log.Printf("%s... ", commandConfig.info.Name)
		commandConfig.handler(session, appCtx)

		for _, guildId := range guildIds {
			_, err := session.ApplicationCommandCreate(session.State.User.ID, guildId, &commandConfig.info)
			if err != nil {
				log.Fatalf("Failed to create command 'sync': %s\n", err)
			}
			log.Printf("Command '%s' registered to guild '%s'.\n", commandConfig.info.Name, guildId)
		}
		log.Printf("Command '%s' finished registering.\n", commandConfig.info.Name)
	}
	log.Println("Finished registering guild commands.")
}

func unregisterGlobalCommands(session *discordgo.Session) {
	// Get globally registered commands
	globallyRegisteredCommands, err := session.ApplicationCommands(session.State.User.ID, "")
	if err != nil {
		log.Fatal(err)
	}

	// Remove globally registered commands
	for _, command := range globallyRegisteredCommands {
		err := session.ApplicationCommandDelete(session.State.User.ID, "", command.ID)
		if err != nil {
			log.Printf("Failed to delete command '%v', %v\n", command.Name, err)
		}
		log.Printf("Removed global command '%v'\n", command.Name)
	}
}
