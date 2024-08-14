package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/michaeldoylecs/discord-sync-bot/commands"
	"github.com/michaeldoylecs/discord-sync-bot/config"
	"github.com/michaeldoylecs/discord-sync-bot/db"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("%s\n", "No .env file found.")
	}

	// Create logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	logWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339Nano,
	}
	log.Logger = zerolog.New(logWriter).With().Timestamp().Logger()

	// Initialize database connection pool
	dbUser := os.Getenv("POSTGRES_USER")
	dbPass := os.Getenv("POSTGRES_PASSWORD")
	dbName := os.Getenv("POSTGRES_DB")
	dbAddress := os.Getenv("POSTGRES_ADDRESS")
	dbPort := os.Getenv("POSTGRES_PORT")
	dbConnString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbAddress, dbPort, dbName)

	log.Info().Str("db-connection-string", dbConnString).Msg("Attempting to connect to database")
	conn, err := pgxpool.New(context.Background(), dbConnString)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer conn.Close()

	appCtx := &config.AppCtx{
		DB: db.New(conn),
	}

	discordPrivateToken := os.Getenv("DISCORD_PRIVATE_TOKEN")

	discord, err := discordgo.New("Bot " + discordPrivateToken)
	if err != nil {
		log.Fatal().Err(err)
	}

	discord.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		// Ignore message if it was sent by the bot
		if m.Author.ID == s.State.User.ID {
			return
		}

		if m.Content == "hello" {
			_, err := s.ChannelMessageSend(m.ChannelID, "world!")
			if err != nil {
				log.Fatal().Err(err)
			}
		}
	})

	discord.Identify.Intents = discordgo.IntentsGuildMessages

	err = discord.Open()
	if err != nil {
		log.Fatal().Err(err)
	}
	defer discord.Close()

	commands.RegisterAllCommands(discord, appCtx)

	// Wait for Ctrl+c interrupt
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}
