package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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
	// Create logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	logWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339Nano,
	}
	log.Logger = zerolog.New(logWriter).With().Timestamp().Logger()

	// Load ENV file
	err := godotenv.Load()
	if err != nil {
		log.Info().Msg("No .env file found.")
	}

	// Checks if environment variables are set
	if os.Getenv("DATABASE_USER") == "" {
		log.Fatal().Msg("DATABASE_USER environment variable not set.")
	}

	if os.Getenv("DATABASE_PASSWORD") == "" {
		log.Fatal().Msg("DATABASE_PASSWORD environment variable not set.")
	}

	if os.Getenv("DATABASE_DB") == "" {
		log.Fatal().Msg("DATABASE_DB environment variable not set.")
	}

	if os.Getenv("DATABASE_ADDRESS") == "" {
		log.Fatal().Msg("DATABASE_ADDRESS environment variable not set.")
	}

	if os.Getenv("DATABASE_PORT") == "" {
		log.Fatal().Msg("DATABASE_PORT environment variable not set.")
	}

	if os.Getenv("DATABASE_URL") == "" {
		log.Fatal().Msg("DATABASE_URL environment variable not set.")
	}

	if os.Getenv("DISCORD_APP_ID") == "" {
		log.Fatal().Msg("DISCORD_APP_ID environment variable not set.")
	}

	if os.Getenv("DISCORD_PUBLIC_KEY") == "" {
		log.Fatal().Msg("DISCORD_PUBLIC_KEY environment variable not set.")
	}

	if os.Getenv("DISCORD_PRIVATE_KEY") == "" {
		log.Fatal().Msg("DISCORD_PRIVATE_KEY environment variable not set.")
	}

	// Read in DEBUG env variable, defaulting to False.
	isDebug, err := strconv.ParseBool(os.Getenv("DEBUG"))
	if err != nil {
		isDebug = false
	}

	// Initialize database connection pool
	dbUser := os.Getenv("DATABASE_USER")
	dbPass := os.Getenv("DATABASE_PASSWORD")
	dbName := os.Getenv("DATABASE_DB")
	dbAddress := os.Getenv("DATABASE_ADDRESS")
	dbPort := os.Getenv("DATABASE_PORT")
	dbSslMode := "require"
	if isDebug {
		dbSslMode = "disable"
	}
	dbConnString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", dbUser, dbPass, dbAddress, dbPort, dbName, dbSslMode)

	log.Info().Msg("Attempting to connect to database")
	conn, err := pgxpool.New(context.Background(), dbConnString)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database.")
	}
	defer conn.Close()

	// Initialize discord session
	log.Info().Msg("Attempting authenticate with discord")
	discordPrivateToken := os.Getenv("DISCORD_PRIVATE_TOKEN")
	discord, err := discordgo.New("Bot " + discordPrivateToken)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create new discord session.")
	}
	discord.Identify.Intents = discordgo.IntentsGuildMessages

	err = discord.Open()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create websocket connection with discord.")
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
