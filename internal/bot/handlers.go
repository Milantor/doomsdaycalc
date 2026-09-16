package bot

import (
	"context"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// textCommand: a plain-text chat command, matched against the exact whole message
// text, e.g. "data remove all".
type textCommand struct {
	text    string // exact, case-insensitive full message text
	admin   bool   // admin-only: silently ignored for regular users
	handler func(ctx context.Context, api *tgbot.Bot, msg *models.Message) error
}

// commands returns the command table. Adding a command is a one-line change.
func (b *Bot) commands() []textCommand {
	return []textCommand{
		{text: "status", handler: b.cmdStatus},
		{text: "privacy", handler: b.cmdPrivacy},
		{text: "data remove all", handler: b.cmdDataRemoveAll},
		{text: "broadcast", admin: true, handler: b.cmdBroadcast},
	}
}

// route is the single entry point for every update. Dispatch order:
//  1. an active scenario dialogue waiting for input
//  2. a plain-text command
//  3. an inline-button callback
//  4. the fallback reply
func (b *Bot) route(ctx context.Context, api *tgbot.Bot, u *models.Update) {
	// 1) user is inside a scenario dialogue and we expect an answer -> pass it to
	//    the FSM

	// 2) plain-text command
	if u.Message != nil {
		if cmd, ok := b.matchCommand(u.Message.Text); ok {
			if cmd.admin && !b.deps.Cfg.IsAdmin(u.Message.From.ID) {
				return // silently ignore admin commands from non-admins
			}
			if err := cmd.handler(ctx, api, u.Message); err != nil {
				b.deps.Log.Error("command failed", "cmd", cmd.text, "err", err)
			}
			return
		}
	}

	// 3) inline-button press
	if u.CallbackQuery != nil {
		if err := b.onCallback(ctx, api, u.CallbackQuery); err != nil {
			b.deps.Log.Error("callback failed", "err", err)
		}
		return
	}

	// 4) fallback
	b.onFallback(ctx, api, u)
}

// matchCommand looks up a text command by an exact, case-insensitive match.
func (b *Bot) matchCommand(text string) (textCommand, bool) {
	text = strings.ToLower(strings.TrimSpace(text))
	for _, c := range b.commands() {
		if c.text == text {
			return c, true
		}
	}
	return textCommand{}, false
}

// --- handlers -------------------------------------------------------------
// Bodies are stubs; logic lands with the matching feature.

// onStart greets the user and shows the main keyboard.
func (b *Bot) onStart(ctx context.Context, api *tgbot.Bot, u *models.Update) { /* stub */ }

// onFallback handles any message that matched nothing else above.
func (b *Bot) onFallback(ctx context.Context, api *tgbot.Bot, u *models.Update) { /* stub */ }

// onCallback handles inline-button presses (callback data carrying the "dep:" prefix).
func (b *Bot) onCallback(ctx context.Context, api *tgbot.Bot, cq *models.CallbackQuery) error {
	return nil
}

// cmdStatus reports progress towards the goal.
func (b *Bot) cmdStatus(ctx context.Context, api *tgbot.Bot, m *models.Message) error { return nil }

// cmdPrivacy explains what data the bot stores and why.
func (b *Bot) cmdPrivacy(ctx context.Context, api *tgbot.Bot, m *models.Message) error { return nil }

// cmdDataRemoveAll wipes all data for the user (hard delete) and restarts the bot.
func (b *Bot) cmdDataRemoveAll(ctx context.Context, api *tgbot.Bot, m *models.Message) error {
	return nil
}

// cmdBroadcast sends a message to every known user. Admin-only.
func (b *Bot) cmdBroadcast(ctx context.Context, api *tgbot.Bot, m *models.Message) error { return nil }
