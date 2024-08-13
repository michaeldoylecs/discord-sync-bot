package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"

	"github.com/bwmarrin/discordgo"
	"github.com/michaeldoylecs/discord-sync-bot/config"
	"github.com/michaeldoylecs/discord-sync-bot/db"
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

			options := interaction.ApplicationCommandData().Options
			channelId := options[0].Value.(string)

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

			// Get file contents
			fileToSync, err := appCtx.DB.GetGuildChannelSync(context.Background(), db.GetGuildChannelSyncParams{
				GuildID:   interaction.GuildID,
				ChannelID: channelId,
			})
			if err != nil {
				log.Println(err)
				return
			}

			oldFileContents := fileToSync.FileContents
			fileUri := fileToSync.FileToSyncUri
			fileContentsResponse, err := http.Get(fileUri)
			if err != nil {
				log.Println(err)
				return
			}
			if fileContentsResponse.StatusCode != http.StatusOK {
				log.Printf("Failed to GET '%s'\n", fileUri)
				return
			}
			fileBytes, err := io.ReadAll(fileContentsResponse.Body)
			if err != nil {
				log.Println(err)
				return
			}
			fileContents := string(fileBytes)

			// Compare current file contents with previously synced contents.
			if oldFileContents == fileContents {
				// Respond that messages are already in-sync
				log.Println("Files already match. No need to sync")
				return
			}

			// Chunk the file contents to fit within discord message limits.
			messageCharacterLimit := 1950
			contentChunks := make([]string, 0, len(fileContents)/messageCharacterLimit+1)
			remainder := fileContents
			for len(remainder) > messageCharacterLimit {
				cursor := 1950
				for remainder[cursor] != '\n' && cursor >= 0 {
					cursor--
				}
				contentChunks = append(contentChunks, remainder[:cursor])
				remainder = remainder[cursor+1:]
			}
			if len(remainder) > 0 {
				contentChunks = append(contentChunks, remainder)
			}

			// Get current content chunks if they exist in db
			existingMessageChunkRows, err := appCtx.DB.GetFileContentChunks(context.Background(), channelId)
			if err != nil {
				log.Println(err)
				return
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
			log.Println("Attempting to send channel messages...")
			for i, chunk := range contentChunks {
				// Update existing message
				if msg_ids[i] != "" {
					_, err := session.ChannelMessageEdit(channelId, msg_ids[i], chunk)
					if err != nil {
						log.Println(err)
						return
					}
					continue
				}

				// Send new message
				msg, err := session.ChannelMessageSend(channelId, chunk)
				if err != nil {
					log.Println(err)
					return
				}
				msg_ids[i] = msg.ID
			}

			// Remove excess pre-existing messages
			for i := range existingMessageChunkRows[len(msg_ids):] {
				msg_id := existingMessageChunkRows[i].DiscordMessageID
				err := session.ChannelMessageDelete(channelId, msg_id)
				if err != nil {
					log.Println(err)
				}
			}

			// Update database with content chunk info
			_, err = appCtx.DB.AddFileContentChunks(context.Background(), db.AddFileContentChunksParams{
				FilesToSyncFk:     fileToSync.ID,
				ChunkNumbers:      makeInt32Range(1, int32(len(msg_ids))),
				DiscordMessageIds: msg_ids,
			})
			if err != nil {
				log.Println(err)
				return
			}

			// Update file contents in db
			err = appCtx.DB.SetFileSyncContents(context.Background(), db.SetFileSyncContentsParams{
				FileContents: fileContents,
				ChannelID:    channelId,
			})
			if err != nil {
				log.Println(err)
				return
			}

			// Respond to command
			msg := fmt.Sprintf("Synced file to <#%s>", channelId)
			session.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: msg,
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
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
