package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

var registeredCommands []*discordgo.ApplicationCommand

var commandInitializers = []func(*discordgo.Session) *discordgo.ApplicationCommand{
	addWriteCommand,
}

func AddAllCommands(session *discordgo.Session) {
	for _, commandInitilizer := range commandInitializers {
		var command = commandInitilizer(session)
		registeredCommands = append(registeredCommands, command)
		log.Printf("Added command '%v'\n", command.Name)
	}
}

func RemoveAllCommands(session *discordgo.Session) {
	for _, command := range registeredCommands {
		err := session.ApplicationCommandDelete(session.State.User.ID, "", command.ID)
		if err != nil {
			log.Printf("Failed to delete command '%v', %v\n", command.Name, err)
		}
		log.Printf("Removed command '%v'\n", command.Name)
	}
}
