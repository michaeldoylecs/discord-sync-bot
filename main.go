package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/michaeldoylecs/discord-sync-bot/commands"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("%s\n", "No .env file found.")
	}

	discordPrivateToken := os.Getenv("DISCORD_PRIVATE_TOKEN")

	discord, err := discordgo.New("Bot " + discordPrivateToken)
	if err != nil {
		log.Fatal(err)
	}

	discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		fmt.Println(m.Content)

		// Ignore message if it was sent by the bot
		if m.Author.ID == s.State.User.ID {
			fmt.Printf("Ignoring message sent by bot: %s\n", m.Content)
			return
		}

		if m.Content == "hello" {
			_, err := s.ChannelMessageSend(m.ChannelID, "world!")
			if err != nil {
				fmt.Println(err)
			}
		}
	})

	discord.Identify.Intents = discordgo.IntentsGuildMessages

	err = discord.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer discord.Close()

	fmt.Println("Bot running...")

	commands.AddAllCommands(discord)

	// Wait for Ctrl+c interrupt
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanup
	commands.RemoveAllCommands(discord)
}
