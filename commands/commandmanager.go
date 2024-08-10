package commands

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/google/go-cmp/cmp"
	"github.com/michaeldoylecs/discord-sync-bot/config"
	"github.com/samber/lo"
)

type CommandHandler func(*discordgo.Session, *config.AppCtx)

type CommandConfig struct {
	info    *discordgo.ApplicationCommand
	handler CommandHandler
}

var commandConfigs = []CommandConfig{
	commandConfigSync,
}

func RegisterAllCommands(session *discordgo.Session, appCtx *config.AppCtx) {
	// Handle globally registered commands
	log.Println("Unregistering globally registered commands...")
	unregisterGlobalCommands(session)
	log.Println("Finished unregistering globally registered commands.")

	// Handle guild registered commands
	guildsCommandsMap := getCommandsForAllGuilds(session)
	botCommandsMap := getAllBotCommands(commandConfigs)
	changedGuildsCommandsMap := filterOutUnchangedCommands(guildsCommandsMap, botCommandsMap)

	log.Println("Registering guild commands...")
	for guildId, guildCommands := range changedGuildsCommandsMap {
		log.Printf("Registering commands for guild '%s'...", guildId)
		for cmdName, cmd := range guildCommands {
			_, err := session.ApplicationCommandCreate(session.State.User.ID, guildId, cmd)
			if err != nil {
				log.Fatalf("Failed to create command '%s': %s\n", cmdName, err)
			} else {
				log.Printf("Guild %s: registered '%s' command.\n", guildId, cmdName)
			}
		}
		log.Printf("Finished registering commands for guild '%s'\n", guildId)
	}
	log.Println("Finished registering guild commands.")

	// Initialize command handlers
	initializeCommandHandlers(session, appCtx)
}

func initializeCommandHandlers(session *discordgo.Session, appCtx *config.AppCtx) {
	for _, config := range commandConfigs {
		config.handler(session, appCtx)
	}
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

func getAllBotCommands(commandConfigs []CommandConfig) map[string]*discordgo.ApplicationCommand {
	commandMap := make(map[string]*discordgo.ApplicationCommand)
	for _, config := range commandConfigs {
		commandMap[config.info.Name] = config.info
	}
	return commandMap
}

func getCommandsForAllGuilds(session *discordgo.Session) map[string]map[string]*discordgo.ApplicationCommand {
	guildIds := lo.Map(session.State.Guilds, func(guild *discordgo.Guild, _ int) string {
		return guild.ID
	})

	guildCommandsMap := make(map[string]map[string]*discordgo.ApplicationCommand)

	for _, guildId := range guildIds {
		commands, err := session.ApplicationCommands(session.State.User.ID, guildId)
		if err != nil {
			log.Fatal(err)
		}

		commandMap := make(map[string]*discordgo.ApplicationCommand)
		for _, cmd := range commands {
			commandMap[cmd.Name] = cmd
		}
		guildCommandsMap[guildId] = commandMap
	}

	return guildCommandsMap
}

func filterOutUnchangedCommands(guildsCommands map[string]map[string]*discordgo.ApplicationCommand, botCommands map[string]*discordgo.ApplicationCommand) map[string]map[string]*discordgo.ApplicationCommand {
	newGuildsCommands := make(map[string]map[string]*discordgo.ApplicationCommand)
	for guildId, guildCommands := range guildsCommands {
		// Get commands guild does not have registered
		unregisteredCommands := make(map[string]*discordgo.ApplicationCommand)
		for botCmdName, botCmd := range botCommands {
			if _, found := guildCommands[botCmdName]; !found {
				log.Printf("Command '%s' not found in guild '%s', adding.", botCmdName, guildId)
				unregisteredCommands[botCmdName] = botCmd
			}
		}

		changedCommands := make(map[string]*discordgo.ApplicationCommand)
		for _, cmd := range guildCommands {
			if !botCommandAndRegisteredCommandAreEqual(botCommands[cmd.Name], cmd) {
				log.Printf("Command '%s' differs from guild '%s' registered value.", cmd.Name, guildId)
				changedCommands[cmd.Name] = cmd
			} else {
				log.Printf("Command '%s' matches registered command in guild '%s', ignoring.", cmd.Name, guildId)
			}
		}

		allCommandsToRegister := make(map[string]*discordgo.ApplicationCommand)
		for cmdName, cmd := range unregisteredCommands {
			allCommandsToRegister[cmdName] = cmd
		}
		for cmdName, cmd := range changedCommands {
			allCommandsToRegister[cmdName] = cmd
		}

		newGuildsCommands[guildId] = allCommandsToRegister
	}
	return newGuildsCommands
}

// ORDER OF ARGUMENTS MAY MATTER!
// Keep commented out logging for future addition of debug logging.
func botCommandAndRegisteredCommandAreEqual(botCmd *discordgo.ApplicationCommand, regCmd *discordgo.ApplicationCommand) bool {
	// log.Printf("'%s' '%s'\n", botCmd.Name, regCmd.Name)
	if !cmp.Equal(botCmd.Name, regCmd.Name) {
		return false
	}

	// cmd1nl, _ := json.Marshal(botCmd.NameLocalizations)
	// cmd2nl, _ := json.Marshal(regCmd.NameLocalizations)
	// log.Printf("'%s' '%s'\n", string(cmd1nl), string(cmd2nl))
	if !cmp.Equal(botCmd.NameLocalizations, regCmd.NameLocalizations) {
		return false
	}

	// log.Printf("'%s' '%s'\n", botCmd.Description, regCmd.Description)
	if !cmp.Equal(botCmd.Description, regCmd.Description) {
		return false
	}

	// cmd1dl, _ := json.Marshal(botCmd.DescriptionLocalizations)
	// cmd2dl, _ := json.Marshal(regCmd.DescriptionLocalizations)
	// log.Printf("'%s' '%s'\n", string(cmd1dl), string(cmd2dl))
	if !cmp.Equal(botCmd.DescriptionLocalizations, regCmd.DescriptionLocalizations) {
		return false
	}

	dmpdefault := int64(0)
	if botCmd.DefaultMemberPermissions == nil {
		botCmd.DefaultMemberPermissions = &dmpdefault
	}
	if regCmd.DefaultMemberPermissions == nil {
		regCmd.DefaultMemberPermissions = &dmpdefault
	}
	// log.Printf("'%b' '%b'\n", *botCmd.DefaultMemberPermissions, *regCmd.DefaultMemberPermissions)
	if !cmp.Equal(botCmd.DefaultMemberPermissions, regCmd.DefaultMemberPermissions) {
		return false
	}

	// log.Printf("'%v' '%v'\n", botCmd.Options, regCmd.Options)
	if !cmp.Equal(botCmd.Options, regCmd.Options) {
		return false
	}

	nsfwdefault := false
	if botCmd.NSFW == nil {
		botCmd.NSFW = &nsfwdefault
	}
	if regCmd.NSFW == nil {
		regCmd.NSFW = &nsfwdefault
	}
	// log.Printf("'%t' '%t'\n", *botCmd.NSFW, *regCmd.NSFW)
	return (*botCmd.NSFW == *regCmd.NSFW)
}
