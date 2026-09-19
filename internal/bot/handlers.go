package bot

import (
	"context"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/i18n"
)

// req: per-update data every handler gets: stored user (language override applied),
// catalog for that user language, inbound message.
type req struct {
	user domain.User
	m    i18n.Messages
	msg  *models.Message
}

// intent: user intent. A typed command and a reply-keyboard button both map to one.
type intent int

const (
	intentNone intent = iota
	intentStart
	intentAdd
	intentWithdraw
	intentStatus
	intentPrivacy
	intentDataRemoveAll
	intentBroadcast
)

// String: intent label for logs.
func (i intent) String() string {
	switch i {
	case intentStart:
		return "start"
	case intentAdd:
		return "add"
	case intentWithdraw:
		return "withdraw"
	case intentStatus:
		return "status"
	case intentPrivacy:
		return "privacy"
	case intentDataRemoveAll:
		return "data_remove_all"
	case intentBroadcast:
		return "broadcast"
	default:
		return "none"
	}
}

// resolveIntent maps message text to an intent. Accepts typed commands and localized
// button labels, so it takes the user catalog: the same button reads differently per
// language. Free function, testable without a Bot.
func resolveIntent(m i18n.Messages, text string) intent {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "/start":
		return intentStart
	case "add", strings.ToLower(m.BtnAdd):
		return intentAdd
	case "withdraw", strings.ToLower(m.BtnWithdraw):
		return intentWithdraw
	case "status", strings.ToLower(m.BtnStatus):
		return intentStatus
	case "privacy":
		return intentPrivacy
	case "data remove all":
		return intentDataRemoveAll
	case "broadcast":
		return intentBroadcast
	default:
		return intentNone
	}
}

// route: single entry point for every update. First a pending scenario dialogue,
// then a command or button, then an inline callback. Every update records the
// Telegram profile and resolves the user language.
func (b *Bot) route(ctx context.Context, api *tgbot.Bot, u *models.Update) {
	from := senderOf(u)
	if from == nil {
		return
	}
	user, m, err := b.resolveUser(ctx, from)
	if err != nil {
		b.deps.Log.Error("resolve user", "err", err)
		return
	}

	// scenario dialogue takes the message first

	// command or button
	if u.Message != nil {
		in := resolveIntent(m, u.Message.Text)
		if in == intentBroadcast && !b.deps.Cfg.IsAdmin(user.ID) {
			return // silently ignore admin commands from non-admins
		}
		r := req{user: user, m: m, msg: u.Message}
		if err := b.handlerFor(in)(ctx, api, r); err != nil {
			b.deps.Log.Error("handler failed", "intent", in.String(), "err", err)
		}
		return
	}

	// inline-button press
	if u.CallbackQuery != nil {
		if err := b.onCallback(ctx, api, u.CallbackQuery); err != nil {
			b.deps.Log.Error("callback failed", "err", err)
		}
	}
}

// handlerFor: handler for an intent. intentNone falls to onFallback.
func (b *Bot) handlerFor(in intent) func(context.Context, *tgbot.Bot, req) error {
	switch in {
	case intentStart:
		return b.onStart
	case intentAdd:
		return b.cmdAdd
	case intentWithdraw:
		return b.cmdWithdraw
	case intentStatus:
		return b.cmdStatus
	case intentPrivacy:
		return b.cmdPrivacy
	case intentDataRemoveAll:
		return b.cmdDataRemoveAll
	case intentBroadcast:
		return b.cmdBroadcast
	default:
		return b.onFallback
	}
}

// senderOf: Telegram user out of whichever update arrived.
func senderOf(u *models.Update) *models.User {
	switch {
	case u.Message != nil:
		return u.Message.From
	case u.CallbackQuery != nil:
		return &u.CallbackQuery.From
	default:
		return nil
	}
}

// resolveUser records the incoming Telegram profile and returns the stored user with
// the catalog for the user language. The ui_language override wins over the
// language_code hint.
func (b *Bot) resolveUser(ctx context.Context, from *models.User) (domain.User, i18n.Messages, error) {
	u, err := b.deps.Users.Touch(ctx, domain.User{
		ID:           from.ID,
		Username:     from.Username,
		FirstName:    from.FirstName,
		LanguageCode: from.LanguageCode,
	})
	if err != nil {
		return domain.User{}, i18n.Messages{}, err
	}
	return u, i18n.Get(i18n.Resolve(u.UILanguage, u.LanguageCode)), nil
}

// reply sends a text message to the user chat. withMenu attaches the main keyboard.
func (b *Bot) reply(ctx context.Context, api *tgbot.Bot, r req, text string, withMenu bool) error {
	params := &tgbot.SendMessageParams{
		ChatID: r.msg.Chat.ID,
		Text:   text,
	}
	if withMenu {
		params.ReplyMarkup = mainKeyboard(r.m)
	}
	_, err := api.SendMessage(ctx, params)
	return err
}

// --- handlers ---

// onStart greets the user and shows the main keyboard.
func (b *Bot) onStart(ctx context.Context, api *tgbot.Bot, r req) error {
	return b.reply(ctx, api, r, r.m.Greeting, true)
}

// onFallback handles a message that matched nothing above.
func (b *Bot) onFallback(ctx context.Context, api *tgbot.Bot, r req) error {
	return b.reply(ctx, api, r, r.m.Fallback, true)
}

// onCallback handles inline-button presses (callback data carrying the "dep:" prefix).
func (b *Bot) onCallback(ctx context.Context, api *tgbot.Bot, cq *models.CallbackQuery) error {
	return nil
}

// cmdAdd records money put aside. Filled in with the scenario flow.
func (b *Bot) cmdAdd(ctx context.Context, api *tgbot.Bot, r req) error {
	return b.reply(ctx, api, r, r.m.AddStub, true)
}

// cmdWithdraw records money taken back. Filled in with the scenario flow.
func (b *Bot) cmdWithdraw(ctx context.Context, api *tgbot.Bot, r req) error {
	return b.reply(ctx, api, r, r.m.WithdrawStub, true)
}

// cmdStatus reports progress towards the goal.
func (b *Bot) cmdStatus(ctx context.Context, api *tgbot.Bot, r req) error {
	return b.reply(ctx, api, r, r.m.StatusEmpty, false)
}

// cmdPrivacy explains what data the bot stores and why.
func (b *Bot) cmdPrivacy(ctx context.Context, api *tgbot.Bot, r req) error {
	return b.reply(ctx, api, r, r.m.Privacy, false)
}

// cmdDataRemoveAll wipes all data for the user (hard delete).
func (b *Bot) cmdDataRemoveAll(ctx context.Context, api *tgbot.Bot, r req) error {
	if err := b.deps.Users.Delete(ctx, r.user.ID); err != nil {
		return err
	}
	return b.reply(ctx, api, r, r.m.DataRemoved, true)
}

// cmdBroadcast sends a message to every known user. Admin-only.
func (b *Bot) cmdBroadcast(ctx context.Context, api *tgbot.Bot, r req) error { return nil }
