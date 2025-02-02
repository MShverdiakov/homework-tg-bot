package main

import (
	"context"
	"dashka-homework-bot/handlers"
	"dashka-homework-bot/logger"
	"dashka-homework-bot/storage/mongo"
	"dashka-homework-bot/updater"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize logger
	logDir := filepath.Join(".", "logs")
	if err := logger.InitLogger(logDir); err != nil {
		logger.Fatal("Failed to initialize logger: %v", err)
	}

	err := godotenv.Load()
	if err != nil {
		logger.Fatal("Error loading.env file: %v", err)
	}

	tgToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if tgToken == "" {
		logger.Fatal("please set TELEGRAM_BOT_TOKEN environment variable")
	}

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		logger.Fatal("Please set MONGO_URI environment variable")
	}

	ctx := context.Background()
	homeworkDB, err := mongo.NewHomeworkDatabase(ctx, mongoURI)
	if err != nil {
		logger.Fatal("Failed to initialize MongoDB: %v", err)
	}
	defer func() {
		if err := homeworkDB.Close(ctx); err != nil {
			logger.Error("Error closing MongoDB connection: %v", err)
		}
	}()

	// Start periodic eraser as a background goroutine
	go homeworkDB.StartEraseAtMidnight()

	bot, err := tgbotapi.NewBotAPI(tgToken)
	if err != nil {
		logger.Fatal("Failed to create bot: %v", err)
	}

	h := handlers.NewHandler(bot, homeworkDB)

	// Set bot commands
	if err := h.SetBotCommands(); err != nil {
		logger.Error("Failed to set bot commands: %v", err)
	}

	// Start the daily summaries scheduler
	homeworkDB.StartDailySummaries(bot)

	// Start periodic eraser as a background goroutine
	go homeworkDB.StartEraseAtMidnight()

	bot.Debug = true
	logger.Info("Authorized on account %s", bot.Self.UserName)

	upd := updater.NewUpdater(bot, h)

	// Gracefully handle shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		logger.Info("Shutting down bot...")
		os.Exit(0)
	}()

	logger.Info("Bot is running...")
	upd.PollUpdates()
}
