// Command bot runs the Doomsday Calculator Telegram bot: wires the config, the
// PostgreSQL pool and the bot together, then starts long polling.
// main only wires things and holds no logic. Everything else is under internal/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"lab042.ru/doomsdaycalc/internal/bot"
	"lab042.ru/doomsdaycalc/internal/config"
	"lab042.ru/doomsdaycalc/internal/service"
	"lab042.ru/doomsdaycalc/internal/storage/postgres"
)

// version is stamped at build time with -ldflags "-X main.version=<sha>". A
// local build keeps the default.
var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// run performs the startup sequence: config -> logger -> context -> database ->
// bot. Returning errors keeps the deferred cleanups below working.
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := newLogger(cfg)
	slog.SetDefault(log)
	log.Info("startup", "version", version)

	// ctx is cancelled on SIGINT/SIGTERM, which stops polling and any in-flight
	// request.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("db: %w", err)
	}
	defer pool.Close()

	// Apply pending migrations before serving traffic. goose records which
	// versions already ran, so this is idempotent and safe on every start.
	if err := postgres.Migrate(ctx, pool); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	users := service.NewUserService(postgres.NewUserRepo(pool))
	savings := service.NewSavingsService(postgres.NewGoalRepo(pool), postgres.NewDepositRepo(pool))
	scenarios := service.NewScenarioService(postgres.NewScenarioRepo(pool), savings)
	deps := &bot.Deps{Cfg: cfg, Log: log, Users: users, Savings: savings, Scenarios: scenarios}
	b, err := bot.New(deps)
	if err != nil {
		return fmt.Errorf("bot: %w", err)
	}

	b.Start(ctx) // blocks until ctx is cancelled
	return nil
}

// newLogger builds the structured logger. JSON because production output goes to
// journald, where it is easier to parse.
func newLogger(cfg *config.Config) *slog.Logger {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
