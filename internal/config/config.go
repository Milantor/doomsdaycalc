// Package config: loads and validates process configuration from environment
// variables. No config files and no global variables; everything is built once in
// Load and passed down explicitly.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config: application configuration. One instance per process, created by Load and
// passed down by pointer.
type Config struct {
	// BotToken is the token from @BotFather. Required.
	BotToken string

	// DatabaseURL is the PostgreSQL DSN, e.g.
	// postgres://user:pass@localhost:5432/doomsdaycalc?sslmode=disable. Required.
	DatabaseURL string

	// AdminIDs are the Telegram user IDs allowed to run admin commands.
	// Set as a comma-separated list in the ADMIN_IDS env var.
	AdminIDs []int64

	// LogLevel is the minimum slog level: debug|info|warn|error.
	LogLevel string

	// Debug enables dumping of incoming updates (handy while developing handlers).
	Debug bool
}

// Load reads the environment, applies defaults and returns an error if anything
// required is missing. Called once at startup.
// A .env file in the working directory is read first; real environment variables
// win over it.
func Load() (*Config, error) {
	// .env is optional, so a missing file is not an error. godotenv leaves existing
	// environment variables untouched.
	_ = godotenv.Load()

	c := &Config{
		BotToken:    os.Getenv("BOT_TOKEN"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		LogLevel:    envOr("LOG_LEVEL", "info"),
		Debug:       os.Getenv("DEBUG") == "1",
	}

	var missing []string
	if c.BotToken == "" {
		missing = append(missing, "BOT_TOKEN")
	}
	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}

	ids, err := parseIDList(os.Getenv("ADMIN_IDS"))
	if err != nil {
		return nil, fmt.Errorf("ADMIN_IDS: %w", err)
	}
	c.AdminIDs = ids
	return c, nil
}

// IsAdmin reports whether the user is listed in ADMIN_IDS. Router uses it to gate
// admin-only commands.
func (c *Config) IsAdmin(userID int64) bool {
	for _, id := range c.AdminIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// envOr returns the env var value, or def when the variable is empty.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// parseIDList parses a comma-separated list of int64 IDs, skipping blank entries.
func parseIDList(raw string) ([]int64, error) {
	var out []int64
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid id %q", part)
		}
		out = append(out, id)
	}
	return out, nil
}
