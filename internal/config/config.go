package config

import (
	"log"
	"os"
)

// Config holds application configuration
type Config struct {
	// Server
	Port string
	Env  string

	// AWS
	AWSRegion string

	// DynamoDB
	DynamoDBEndpoint string // For local development
	ItemsTable       string
	UsersTable       string
	SRSTable         string
	AnswersTable     string

	// Auth
	GoogleClientID     string
	GoogleClientSecret string
	JWTSecret          string

	// AI Generation
	AnthropicAPIKey string

	// Frontend
	FrontendURL string
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),

		AWSRegion:        getEnv("AWS_REGION", "ap-northeast-1"),
		DynamoDBEndpoint: getEnv("DYNAMODB_ENDPOINT", ""), // Empty for production

		ItemsTable:   getEnv("DYNAMODB_ITEMS_TABLE", "syun-eng-items"),
		UsersTable:   getEnv("DYNAMODB_USERS_TABLE", "syun-eng-users"),
		SRSTable:     getEnv("DYNAMODB_SRS_TABLE", "syun-eng-srs"),
		AnswersTable: getEnv("DYNAMODB_ANSWERS_TABLE", "syun-eng-answers"),

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		JWTSecret:          getEnv("JWT_SECRET", ""),

		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),

		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
	}

	// Security warnings
	if cfg.Env == "production" {
		if cfg.JWTSecret == "" {
			log.Fatal("FATAL: JWT_SECRET must be set in production")
		}
		if len(cfg.JWTSecret) < 32 {
			log.Fatal("FATAL: JWT_SECRET must be at least 32 characters in production")
		}
		if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
			log.Fatal("FATAL: Google OAuth credentials must be set in production")
		}
	} else {
		// Development defaults
		if cfg.JWTSecret == "" {
			cfg.JWTSecret = "development-secret-do-not-use-in-production"
			log.Println("WARNING: Using default JWT_SECRET. Set JWT_SECRET environment variable for production.")
		}
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
