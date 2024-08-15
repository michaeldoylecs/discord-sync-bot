package commands

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/michaeldoylecs/discord-sync-bot/config"
	"github.com/michaeldoylecs/discord-sync-bot/db"
	"github.com/rs/zerolog"
)

var commandConfigSync = CommandConfig{
	info: &discordgo.ApplicationCommand{
		Name:        "sync",
		Description: "Update syncs",
		Options: []*discordgo.ApplicationCommandOption{
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

			// Create logger with command relevant info
			logger := newInteractionLogger(interaction.Interaction)
			defer logExecutionTime(logger, "Command finished executing.")()
			logger.Info().Msg("Command started.")

			// Parse arguments
			options := interaction.ApplicationCommandData().Options
			channelId := options[0].Value.(string)
			logger.Info().Interface("arguments", options).Msg("Command arguments parsed.")

			// Handle channel not existing within current guild.
			channels, err := session.GuildChannels(interaction.GuildID)
			if err != nil {
				logger.Error().Err(err).Msg("")
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

			// Get file contents
			fileToSync, err := appCtx.DB.GetGuildChannelSync(context.Background(), db.GetGuildChannelSyncParams{
				GuildID:   interaction.GuildID,
				ChannelID: channelId,
			})
			if err != nil {
				logger.Error().Err(err).Msg("")
				sendErrorResponse(session, interaction.Interaction)
				return
			}

			oldFileContents := fileToSync.FileContents
			fileUri := fileToSync.FileToSyncUri

			err = SyncFileToDiscordMessages(logger.WithContext(context.Background()), *appCtx, interaction.GuildID, channelId, fileUri, oldFileContents)
			if err != nil {
				logger.Error().Err(err).Msg("")
				sendErrorResponse(session, interaction.Interaction)
			}

			// Respond to command
			msg := fmt.Sprintf("Synced file to <#%s>", channelId)
			sendEphemeralResponse(session, interaction.Interaction, msg)
		})
	},
}

func makeInt32Range(min int32, max int32) []int32 {
	l := make([]int32, max-min+1)
	for i := range l {
		l[i] = min + int32(i)
	}
	return l
}

func chunkContents(contents string, maxChunkSize int) []string {
	chunks := make([]string, 0, len(contents)/maxChunkSize+1)
	remainder := contents
	for len(remainder) > maxChunkSize {
		cursor := 1950
		for remainder[cursor] != '\n' && cursor >= 0 {
			cursor--
		}
		chunks = append(chunks, remainder[:cursor])
		remainder = remainder[cursor+1:]
	}
	if len(remainder) > 0 {
		chunks = append(chunks, remainder)
	}
	return chunks
}

func SyncFileToDiscordMessages(ctx context.Context, appCtx config.AppCtx, guildId string, channelId string, fileUrl string, prevFileContents string) error {
	logger := zerolog.Ctx(ctx)
	session := appCtx.DiscordSession

	fileContentsResponse, err := http.Get(fileUrl)
	if err != nil {
		logger.Error().Err(err)
		return err
	}
	if fileContentsResponse.StatusCode != http.StatusOK {
		logger.Warn().Str("file_uri", fileUrl).Msg("Failed to GET file.")
		return err
	}
	fileBytes, err := io.ReadAll(fileContentsResponse.Body)
	if err != nil {
		logger.Error().Err(err)
		return err
	}
	fileContents := string(fileBytes)

	// Compare current file contents with previously synced contents.
	if prevFileContents == fileContents {
		// Respond that messages are already in-sync
		logger.Info().Msg("Files already match.")
		return nil
	}

	// Chunk the file contents to fit within discord message limits.
	contentChunks := chunkContents(fileContents, 1950)

	// Get current content chunks if they exist in db
	existingMessageChunkRows, err := appCtx.DB.GetFileContentChunks(context.Background(), channelId)
	if err != nil {
		logger.Error().Err(err)
		return err
	}

	// Associate existing message chunk ids with new chunks to update instead of making new mesages
	msg_ids := make([]string, len(contentChunks))
	for i := range existingMessageChunkRows {
		if i > len(msg_ids) {
			break
		}
		chunk_index := existingMessageChunkRows[i].ChunkNumber - 1
		chunk_message_id := existingMessageChunkRows[i].DiscordMessageID
		msg_ids[chunk_index] = chunk_message_id
	}

	// Send discord messages with the chunks.
	logger.Info().Msg("Attempting to send channel messages...")
	for i, chunk := range contentChunks {
		// Update existing message
		if msg_ids[i] != "" {
			msg, err := session.ChannelMessageEdit(channelId, msg_ids[i], chunk)
			if err != nil {
				logger.Error().Err(err)
				return err
			} else {
				logger.Info().
					Str("message_channel_id", msg.ChannelID).
					Str("message_id", msg_ids[i]).
					Int("message_chunk_num", i+1).
					Msg("Updated message chunk.")
			}
			continue
		}

		// Send new message
		msg, err := session.ChannelMessageSend(channelId, chunk)
		if err != nil {
			logger.Error().Err(err)
			return err
		} else {
			logger.Info().
				Str("message_channel_id", msg.ChannelID).
				Str("message_id", msg.ID).
				Int("message_chunk_num", i+1).
				Msg("Sent new message chunk.")
		}
		msg_ids[i] = msg.ID
	}

	// Remove excess pre-existing messages
	if len(existingMessageChunkRows) > len(msg_ids) {
		for i := range existingMessageChunkRows[len(msg_ids):] {
			msg_id := existingMessageChunkRows[i].DiscordMessageID
			err := session.ChannelMessageDelete(channelId, msg_id)
			if err != nil {
				logger.Warn().Err(err).Msg("")
			}
		}
	}

	// Get File db foreign key
	fileToSync, err := appCtx.DB.GetChannelSync(context.Background(), fileUrl)
	if err != nil {
		logger.Error().Err(err).Msg("")
		return err
	}
	fileFK := fileToSync.ID

	// Update database with content chunk info
	_, err = appCtx.DB.AddFileContentChunks(context.Background(), db.AddFileContentChunksParams{
		FilesToSyncFk:     fileFK,
		ChunkNumbers:      makeInt32Range(1, int32(len(msg_ids))),
		DiscordMessageIds: msg_ids,
	})
	if err != nil {
		logger.Error().Err(err)
		return err
	}

	// Update file contents in db
	err = appCtx.DB.SetFileSyncContents(context.Background(), db.SetFileSyncContentsParams{
		FileContents: fileContents,
		ChannelID:    channelId,
	})
	if err != nil {
		logger.Error().Err(err)
		return err
	}

	return nil
}
