package commands

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/michaeldoylecs/discord-sync-bot/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

type CommandHandler func(*discordgo.Session, *config.AppCtx)

type CommandConfig struct {
	info    *discordgo.ApplicationCommand
	handler CommandHandler
}

// Keep sorted alphanumerically for readability.
var commandConfigs = []CommandConfig{
	commandConfigAddSync,
	commandConfigSync,
}

func RegisterAllCommands(session *discordgo.Session, appCtx *config.AppCtx) {
	// Handle globally registered commands
	log.Info().Msg("Unregistering globally registered commands...")
	unregisterGlobalCommands(session)
	log.Info().Msg("Finished unregistering globally registered commands.")

	// Handle guild registered commands
	guildsCommandsMap := getCommandsForAllGuilds(session)
	botCommandsMap := getAllBotCommands(commandConfigs)

	// Add/Update guild commands
	changedGuildsCommandsMap := filterCommandsToAdd(guildsCommandsMap, botCommandsMap)
	log.Info().Msg("Started Registering all guild commands.")
	logRegisterAllCommandsTime := logExecutionTime(log.Logger, "Finished registering all guild commands.")
	for guildId, guildCommands := range changedGuildsCommandsMap {
		log.Info().Str("guild_id", guildId).Msg("Started registering individual guild commands.")
		logRegisterGuildCommands := logExecutionTime(log.Logger.With().Str("guild_id", guildId).Logger(),
			"Finished registering individual guild commands.",
		)
		for _, cmd := range guildCommands {
			log.Info().Str("guild_id", guildId).Interface("application_command", cmd).
				Msg("Attemping to register command.")
			regCmd, err := session.ApplicationCommandCreate(session.State.User.ID, guildId, cmd)
			if err != nil {
				log.Fatal().Err(err).Str("guild_id", guildId).Interface("application_command", cmd).
					Msg("Failed to create command")
			} else {
				log.Info().Str("guild_id", guildId).Interface("application_command", regCmd).
					Msg("Successfully registered command.")
			}
		}
		logRegisterGuildCommands()
	}
	logRegisterAllCommandsTime()

	// Remove guild commands
	removeGuildsCommandsMap := filterCommandsToRemove(guildsCommandsMap, botCommandsMap)
	log.Info().Msg("Started removing commands for all guilds.")
	logRemoveAllCommandsTime := logExecutionTime(log.Logger, "Finished removing commands for all guilds.")
	for guildId, guildCommands := range removeGuildsCommandsMap {
		log.Info().Str("guild_id", guildId).Msg("Started removing commands for specific guild.")
		logRemoveGuildCommandsTime := logExecutionTime(log.Logger.With().Str("guild_id", guildId).Logger(),
			"Finished removing commands for specific guild.",
		)
		for _, cmd := range guildCommands {
			err := session.ApplicationCommandDelete(session.State.User.ID, guildId, cmd.ID)
			if err != nil {
				log.Fatal().Err(err).Interface("command", cmd).Msg("Failed to remove command.")
			} else {
				log.Info().Str("guild_id", guildId).Interface("command", cmd).Msg("Successfully removed command.")
			}
		}
		logRemoveGuildCommandsTime()
	}
	logRemoveAllCommandsTime()

	// Initialize command handlers
	initializeCommandHandlers(session, appCtx)
}

func initializeCommandHandlers(session *discordgo.Session, appCtx *config.AppCtx) {
	for _, config := range commandConfigs {
		config.handler(session, appCtx)
		log.Info().Str("command_name", config.info.Name).Msg("Command handler initialized.")
	}
}

func unregisterGlobalCommands(session *discordgo.Session) {
	// Get globally registered commands
	globallyRegisteredCommands, err := session.ApplicationCommands(session.State.User.ID, "")
	if err != nil {
		log.Fatal().Err(err)
	}

	// Remove globally registered commands
	for _, command := range globallyRegisteredCommands {
		err := session.ApplicationCommandDelete(session.State.User.ID, "", command.ID)
		if err != nil {
			log.Fatal().Err(err).Str("command_name", command.Name).Msg("Failed to remove command.")
		}
		log.Info().Str("command_name", command.Name).Msg("Successfully removed command.")
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
			log.Fatal().Err(err)
		}

		commandMap := make(map[string]*discordgo.ApplicationCommand)
		for _, cmd := range commands {
			commandMap[cmd.Name] = cmd
		}
		guildCommandsMap[guildId] = commandMap
	}

	return guildCommandsMap
}

func filterCommandsToAdd(guildsCommands map[string]map[string]*discordgo.ApplicationCommand, botCommands map[string]*discordgo.ApplicationCommand) map[string]map[string]*discordgo.ApplicationCommand {
	newGuildsCommands := make(map[string]map[string]*discordgo.ApplicationCommand)
	for guildId, guildCommands := range guildsCommands {
		// Get commands guild does not have registered
		unregisteredCommands := make(map[string]*discordgo.ApplicationCommand)
		for botCmdName, botCmd := range botCommands {
			if _, found := guildCommands[botCmdName]; !found {
				log.Info().Str("guild_id", guildId).Str("command_name", botCmdName).
					Msg("Command not found in guild, adding.")
				unregisteredCommands[botCmdName] = botCmd
			}
		}

		// Get commands that have changed
		changedCommands := make(map[string]*discordgo.ApplicationCommand)
		for _, cmd := range guildCommands {
			// Do not try to update guild command if not is bot's command list.
			// It will be handled in command removal.
			if botCommands[cmd.Name] == nil {
				continue
			}
			if !botCommandAndRegisteredCommandAreEqual(botCommands[cmd.Name], cmd) {
				log.Debug().
					Str("guild_id", guildId).
					Interface("bot_command", botCommands[cmd.Name]).
					Interface("registered_command", cmd).
					Msg("Bot command differs from registered guild command.")
				changedCommands[cmd.Name] = botCommands[cmd.Name]
			} else {
				log.Info().Str("guild_id", guildId).Str("command_name", cmd.Name).
					Msg("Bot command matches registered guild command, ignoring.")
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

func filterCommandsToRemove(guildsCommands map[string]map[string]*discordgo.ApplicationCommand, botCommands map[string]*discordgo.ApplicationCommand) map[string]map[string]*discordgo.ApplicationCommand {
	guildsCommandsToRemove := make(map[string]map[string]*discordgo.ApplicationCommand)
	for guildId, guildCommands := range guildsCommands {
		commandsToRemove := make(map[string]*discordgo.ApplicationCommand)
		for cmdName, cmd := range guildCommands {
			if _, found := botCommands[cmdName]; !found {
				commandsToRemove[cmdName] = cmd
			}
		}
		guildsCommandsToRemove[guildId] = commandsToRemove
	}
	return guildsCommandsToRemove
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

func newInteractionLogger(interaction *discordgo.Interaction) zerolog.Logger {
	var userId string
	if interaction.User != nil {
		userId = interaction.User.ID
	} else {
		userId = interaction.Member.User.ID
	}

	isDM := interaction.Member == nil

	return log.With().
		Str("trace_id", uuid.New().String()).
		Str("interaction_command_name", interaction.ApplicationCommandData().Name).
		Str("interaction_guild_id", interaction.GuildID).
		Str("interaction_channel_id", interaction.ChannelID).
		Str("interaction_user_id", userId).
		Bool("interaction_is_dm", isDM).
		Logger()
}

// Must call with defer and trailing (), or store return value in a variable and call again later.
// Ex:
//
//		Deferred:
//			  defer logExecutionTime(log.Logger, "Finished executing")()
//	      // Do stuff here...
//
//		Re-called:
//			  logTime := logExecutionTime(log.Logger, "Finished executing")
//		    // Do stuff here...
//			  logTime()
func logExecutionTime(logger zerolog.Logger, msg string) func() {
	start := time.Now()
	return func() {
		logger.Info().Str("execution_time", time.Since(start).String()).Msg(msg)
	}
}

func sendEphemeralResponse(session *discordgo.Session, interaction *discordgo.Interaction, msg string) {
	session.InteractionRespond(interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

func sendErrorResponse(session *discordgo.Session, interaction *discordgo.Interaction) {
	sendEphemeralResponse(session, interaction, "Something went wrong. Please try again or contact bot owner.")
}
