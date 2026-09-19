// Package bot: Telegram integration layer.
// Knows the bot librarys models: routes updates, parses text commands, builds
// keyboards, formats replies.
// Layer is thin: no business logic/direct database access, it only calls into the
// domain. Domain knows nothing about Telegram. This keeps the calculator and
// scenarios unit-testable without a bot.
package bot

import (
	"context"
	"fmt"
	"log/slog"

	tgbot "github.com/go-telegram/bot"

	"lab042.ru/doomsdaycalc/internal/config"
	"lab042.ru/doomsdaycalc/internal/service"
)

// Deps holds everything the handlers need. Grows as features are added.
// A single struct beats many positional args: adding a dependency later does not
// force changes in every constructor call.
type Deps struct {
	Cfg   *config.Config
	Log   *slog.Logger
	Users *service.UserService
	// Here later: Savings *service.SavingsService.
}

// Bot wraps the Telegram API client together with the handler dependencies.
type Bot struct {
	api  *tgbot.Bot
	deps *Deps
}

// New builds the Bot and registers the default handler.
// Every update goes through the single default handler (b.route).
// b.route is a method value captured before b.api is assigned. Handlers only run on
// updates, long after New returns, so this is safe.
func New(deps *Deps) (*Bot, error) {
	b := &Bot{deps: deps}

	api, err := tgbot.New(deps.Cfg.BotToken,
		tgbot.WithDefaultHandler(b.route), // single entry point for all updates
		tgbot.WithErrorsHandler(func(err error) {
			deps.Log.Error("telegram error", "err", err)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("create bot: %w", err)
	}
	b.api = api

	return b, nil
}

// Start runs the long-polling loop and blocks until ctx is cancelled.
func (b *Bot) Start(ctx context.Context) {
	b.deps.Log.Info("bot started", "id", b.api.ID())
	b.api.Start(ctx) // blocks until ctx is done
	b.deps.Log.Info("bot stopped")
}
