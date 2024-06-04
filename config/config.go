package config

import (
	"database/sql"
	"fmt"
	"log"
	"tgbot/models"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq"
)

var db *sql.DB
var bot *tgbotapi.BotAPI

// InitializeDatabase sets up a connection to PostgreSQL database
func InitializeDatabase(dbConfig models.DB) (*sql.DB, error) {
	dbInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.Name)

	dbConn, err := sql.Open("postgres", dbInfo)
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %v", err)
	}

	// Test the connection
	if err = dbConn.Ping(); err != nil {
		dbConn.Close()
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	return dbConn, nil
}

// InitializeBot creates a new Telegram bot instance
func InitializeBot(botToken string) (*tgbotapi.BotAPI, error) {
	botInstance, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return nil, fmt.Errorf("error creating new bot instance: %v", err)
	}

	return botInstance, nil
}

// GetDB returns the initialized database connection
func GetDB() *sql.DB {
	return db
}

// GetBot returns the initialized Telegram bot instance
func GetBot() *tgbotapi.BotAPI {
	return bot
}

// Setup initializes database and bot instances
func Setup(dbConfig models.DB, botToken string) error {
	// Initialize database
	dbConn, err := InitializeDatabase(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	db = dbConn

	// Initialize bot
	botInstance, err := InitializeBot(botToken)
	if err != nil {
		return fmt.Errorf("failed to initialize bot: %v", err)
	}
	bot = botInstance

	log.Println("Database and bot initialized successfully")
	return nil
}
