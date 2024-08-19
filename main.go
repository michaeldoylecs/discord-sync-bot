package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

	log.Info().Msg("Attempting to connect to database")
	conn, err := pgxpool.New(context.Background(), dbConnString)
	if err != nil {
		log.Fatal().Err(err)
	}
	defer conn.Close()

	// Initialize discord session
	discordPrivateToken := os.Getenv("DISCORD_PRIVATE_TOKEN")
	discord, err := discordgo.New("Bot " + discordPrivateToken)
	if err != nil {
		log.Fatal().Err(err)
	}
	discord.Identify.Intents = discordgo.IntentsGuildMessages

	err = discord.Open()
	if err != nil {
		log.Fatal().Err(err)
	}
	defer discord.Close()

	// Initial application context
	appCtx := &config.AppCtx{
		DB:             db.New(conn),
		DiscordSession: discord,
	}

	// Register discord slash commands
	commands.RegisterAllCommands(discord, appCtx)

	// Initialize webhook listener
	go func() {
		log.Info().
			Int("http_port", 8080).
			Msg("Webhook listener started.")
		http.HandleFunc("/github", githubWebhookHandler(*appCtx))
		log.Fatal().Err(http.ListenAndServe(":8080", nil))
	}()

	// Wait for Ctrl+c interrupt
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

type GithubRepository struct {
	Url string `json:"url"`
}
type GithubEventPush struct {
	Repository GithubRepository `json:"repository"`
}

func githubWebhookHandler(appCtx config.AppCtx) func(w http.ResponseWriter, r *http.Request) {
	logger := commands.NewTraceLogger()

	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info().Msg("Connection received.")
		var pushEvent GithubEventPush
		jsonDecoder := json.NewDecoder(r.Body)
		err := jsonDecoder.Decode(&pushEvent)
		if err != nil {
			logger.Error().Err(err).Msg("")
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		repo_url := pushEvent.Repository.Url

		// Get files associated with github repo
		files, err := appCtx.DB.GetGithubRepoSyncFiles(context.Background(), repo_url)
		if err != nil {
			logger.Error().Err(err).Msg("")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Sync each file
		for _, file := range files {
			ctx := logger.WithContext(context.Background())
			err := commands.SyncFileToDiscordMessages(ctx, appCtx, file.GuildID, file.ChannelID, file.Url, file.FileContents)
			if err != nil {
				logger.Error().Err(err).Msg("")
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

		log.Info().Interface("request_body", pushEvent).Msg("Connection processed.")
	}
}
