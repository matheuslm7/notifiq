//internal/config/config.go

package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr             string
	RedisAddr            string
	EventQueue           string
	EmailJobQueue        string
	EmailDeadLetterQueue string
	MySQLAddr            string
	MySQLDatabase        string
	MySQLUser            string
	MySQLPassword        string
	MigrationPath        string
	JWTSecret            string
	EmailEncryptionKey   string
	WebhookSecret        string
	PasswordResetURL     string
	InviteURL            string
}

func Load() Config {
	return Config{
		HTTPAddr:             envOrDefault("HTTP_ADDR", ":8080"),
		RedisAddr:            envOrDefault("REDIS_ADDR", "localhost:4000"),
		EventQueue:           envOrDefault("EVENT_QUEUE", "notifiq:events"),
		EmailJobQueue:        envOrDefault("EMAIL_JOB_QUEUE", "notifiq:email_jobs"),
		EmailDeadLetterQueue: envOrDefault("EMAIL_DEAD_LETTER_QUEUE", "notifiq:email_jobs:dead_letters"),
		MySQLAddr:            envOrDefault("MYSQL_ADDR", "localhost:3306"),
		MySQLDatabase:        envOrDefault("MYSQL_DATABASE", "notifiq"),
		MySQLUser:            envOrDefault("MYSQL_USER", "notifiq"),
		MySQLPassword:        envOrDefault("MYSQL_PASSWORD", "notifiq"),
		MigrationPath:        envOrDefault("MIGRATION_PATH", "db/migrations"),
		JWTSecret:            envOrDefault("JWT_SECRET", "notifiq-development-secret-change-me"),
		EmailEncryptionKey:   envOrDefault("EMAIL_ENCRYPTION_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY"),
		WebhookSecret:        envOrDefault("WEBHOOK_SECRET", "notifiq-development-webhook-secret"),
		PasswordResetURL:     envOrDefault("PASSWORD_RESET_URL", "http://localhost:5173/reset-password"),
		InviteURL:            envOrDefault("INVITE_URL", "http://localhost:5173/invite"),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	if path := os.Getenv(key + "_FILE"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return ""
		}
		if value := strings.TrimSpace(string(data)); value != "" {
			return value
		}
		return ""
	}
	return fallback
}
