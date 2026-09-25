package bot

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/i18n"
	"lab042.ru/doomsdaycalc/internal/scenario"
	"lab042.ru/doomsdaycalc/internal/service"
)

// sender: the Telegram calls the handlers make. *tgbot.Bot has the methods, so the
// Telegram client is passed in; tests pass a fake and read back what was sent.
// Handlers take this, so the dialogue loop runs without a live bot.
type sender interface {
	SendMessage(ctx context.Context, params *tgbot.SendMessageParams) (*models.Message, error)
	AnswerCallbackQuery(ctx context.Context, params *tgbot.AnswerCallbackQueryParams) (bool, error)
}

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
	intentSend
	intentLang
	intentHelp
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
	case intentSend:
		return "send"
	case intentLang:
		return "lang"
	case intentHelp:
		return "help"
	default:
		return "none"
	}
}

// resolveIntent maps message text to an intent. Accepts typed commands and localized
// button labels, so it takes the user catalog: the same button reads differently per
// language. Free function, testable without a Bot.
func resolveIntent(m i18n.Messages, text string) intent {
	text = strings.ToLower(strings.TrimSpace(text))
	switch text {
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
	case "send":
		return intentSend
	case "lang":
		return intentLang
	case "/help", "help", strings.ToLower(m.BtnOther):
		return intentHelp
	}

	// "send <scenario> <target>", "broadcast <text>" and "lang <arg>" carry arguments,
	// so they are matched by prefix. The handlers split the arguments themselves.
	if strings.HasPrefix(text, "send ") {
		return intentSend
	}
	if strings.HasPrefix(text, "broadcast ") {
		return intentBroadcast
	}
	if strings.HasPrefix(text, "lang ") {
		return intentLang
	}
	return intentNone
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

	if u.Message != nil {
		r := req{user: user, m: m, msg: u.Message}

		// a running dialogue takes the message first
		if b.dialogue(ctx, api, r) {
			return
		}

		// command or button
		in := resolveIntent(m, u.Message.Text)
		if isAdminIntent(in) && !b.deps.Cfg.IsAdmin(user.ID) {
			return // silently ignore admin commands from non-admins
		}
		if err := b.handlerFor(in)(ctx, api, r); err != nil {
			b.deps.Log.Error("handler failed", "intent", in.String(), "err", err)
		}
		return
	}

	// inline-button press
	if u.CallbackQuery != nil {
		if err := b.onCallback(ctx, api, u.CallbackQuery, user, m); err != nil {
			b.deps.Log.Error("callback failed", "err", err)
		}
	}
}

// handlerFor: handler for an intent. intentNone falls to onFallback.
func (b *Bot) handlerFor(in intent) func(context.Context, sender, req) error {
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
	case intentSend:
		return b.cmdSend
	case intentLang:
		return b.cmdLang
	case intentHelp:
		return b.cmdHelp
	default:
		return b.onFallback
	}
}

// isAdminIntent: intents only an operator may run.
func isAdminIntent(in intent) bool {
	return in == intentBroadcast || in == intentSend
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

// reply sends text to the user chat. withMenu attaches the main keyboard.
func (b *Bot) reply(ctx context.Context, api sender, r req, text string, withMenu bool) error {
	var markup models.ReplyMarkup
	if withMenu {
		markup = mainKeyboard(r.m)
	}
	return b.replyWith(ctx, api, r, text, markup)
}

// replyWith sends text with the given reply markup. nil leaves the current keyboard
// alone.
func (b *Bot) replyWith(ctx context.Context, api sender, r req, text string, markup models.ReplyMarkup) error {
	params := &tgbot.SendMessageParams{
		ChatID: r.msg.Chat.ID,
		Text:   text,
	}
	if markup != nil {
		params.ReplyMarkup = markup
	}
	_, err := api.SendMessage(ctx, params)
	return err
}

// --- handlers ---

// onStart greets the user and shows the main keyboard. A user without goals lands in
// onboarding instead, so the first thing they see is the first question.
func (b *Bot) onStart(ctx context.Context, api sender, r req) error {
	goals, err := b.deps.Savings.ListGoals(ctx, r.user.ID)
	if err != nil {
		return err
	}
	if len(goals) == 0 {
		return b.startScenario(ctx, api, r, "onboarding")
	}
	return b.reply(ctx, api, r, r.m.Greeting, true)
}

// onFallback handles a message that matched nothing above.
func (b *Bot) onFallback(ctx context.Context, api sender, r req) error {
	return b.reply(ctx, api, r, r.m.Fallback, true)
}

// inline-button callback actions. Button data is "<action>:<goal id>", written by
// goalsKeyboard and read back here.
const (
	callbackDeposit  = "dep"
	callbackWithdraw = "wit"
	callbackStatus   = "st"
)

// onCallback handles inline-button presses. The action picks what happens next: open a
// money dialogue on the picked goal, or render its status. user and m are the profile and
// catalog the router already resolved. The pressed message may be gone, so the reply goes
// to the callback chat when it is known.
func (b *Bot) onCallback(ctx context.Context, api sender, cq *models.CallbackQuery, user domain.User, m i18n.Messages) error {
	if _, err := api.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: cq.ID}); err != nil {
		b.deps.Log.Error("answer callback", "err", err)
	}

	action, arg, ok := parseCallback(cq.Data)
	if !ok {
		return nil
	}
	goalID, err := strconv.ParseInt(arg, 10, 64)
	if err != nil {
		return nil
	}

	r := req{user: user, m: m, msg: &models.Message{Chat: models.Chat{ID: callbackChat(cq)}}}

	switch action {
	case callbackDeposit:
		return b.startScenarioVars(ctx, api, r, "add_deposit", map[string]string{scenario.VarGoal: arg})
	case callbackWithdraw:
		return b.startScenarioVars(ctx, api, r, "withdraw", map[string]string{scenario.VarGoal: arg})
	case callbackStatus:
		return b.renderStatus(ctx, api, r, goalID)
	default:
		return nil
	}
}

// parseCallback: splits inline-button data into an action and its argument. Missing
// colon means data this bot did not write, so it is ignored.
func parseCallback(data string) (action, arg string, ok bool) {
	i := strings.IndexByte(data, ':')
	if i < 0 {
		return "", "", false
	}
	return data[:i], data[i+1:], true
}

// callbackChat: chat of the message the button sits under, or 0 when the message is
// inaccessible.
func callbackChat(cq *models.CallbackQuery) int64 {
	if cq.Message.Message != nil {
		return cq.Message.Message.Chat.ID
	}
	return 0
}

// cmdAdd records money put aside. With one goal it jumps straight to the amount
// question; with several it first asks which goal.
func (b *Bot) cmdAdd(ctx context.Context, api sender, r req) error {
	return b.startDepositFlow(ctx, api, r, "add_deposit", callbackDeposit)
}

// cmdWithdraw records money taken back. Same flow as cmdAdd, the amount goes out.
func (b *Bot) cmdWithdraw(ctx context.Context, api sender, r req) error {
	return b.startDepositFlow(ctx, api, r, "withdraw", callbackWithdraw)
}

// cmdStatus reports progress towards a goal. With one goal it renders right away; with
// several it first asks which goal.
// TODO: post-MVP cleanup. Goal pick repeats in startDepositFlow; fold both into a
// pickGoal helper.
func (b *Bot) cmdStatus(ctx context.Context, api sender, r req) error {
	goals, err := b.deps.Savings.ListGoals(ctx, r.user.ID)
	if err != nil {
		return err
	}
	switch len(goals) {
	case 0:
		return b.reply(ctx, api, r, r.m.NoGoal, true)
	case 1:
		return b.renderStatus(ctx, api, r, goals[0].ID)
	default:
		return b.replyWith(ctx, api, r, r.m.ChooseGoal, goalsKeyboard(callbackStatus, goals))
	}
}

// startDepositFlow: the shared start of a money dialogue. No goal means there is
// nothing to put money into; one goal seeds VarGoal and opens the amount question;
// several send an inline picker, and onCallback continues once a goal is tapped.
func (b *Bot) startDepositFlow(ctx context.Context, api sender, r req, name, action string) error {
	goals, err := b.deps.Savings.ListGoals(ctx, r.user.ID)
	if err != nil {
		return err
	}
	switch len(goals) {
	case 0:
		return b.reply(ctx, api, r, r.m.NoGoal, true)
	case 1:
		return b.startScenarioVars(ctx, api, r, name, map[string]string{
			scenario.VarGoal: strconv.FormatInt(goals[0].ID, 10),
		})
	default:
		return b.replyWith(ctx, api, r, r.m.ChooseGoal, goalsKeyboard(action, goals))
	}
}

// renderStatus: computes and sends the status of one goal of the user.
func (b *Bot) renderStatus(ctx context.Context, api sender, r req, goalID int64) error {
	g, st, err := b.deps.Savings.GoalStatus(ctx, r.user.ID, goalID, time.Now().UTC())
	if err != nil {
		return err
	}
	return b.reply(ctx, api, r, formatStatus(g, st, r.m), true)
}

// cmdHelp lists the commands the bot understands.
func (b *Bot) cmdHelp(ctx context.Context, api sender, r req) error {
	return b.reply(ctx, api, r, r.m.Help, true)
}

// cmdPrivacy explains what data the bot stores and why.
// TODO: link to PP on website
func (b *Bot) cmdPrivacy(ctx context.Context, api sender, r req) error {
	return b.reply(ctx, api, r, r.m.Privacy, false)
}

// cmdDataRemoveAll wipes all data for the user (hard delete).
// TODO: also wipe archive/<user_id>/ once the archive is built.
func (b *Bot) cmdDataRemoveAll(ctx context.Context, api sender, r req) error {
	if err := b.deps.Users.Delete(ctx, r.user.ID); err != nil {
		return err
	}
	return b.reply(ctx, api, r, r.m.DataRemoved, true)
}

// cmdLang switches the interface language, or explains the command without an argument.
// With one it stores the override and answers in the new language, menu included.
func (b *Bot) cmdLang(ctx context.Context, api sender, r req) error {
	trimmed := strings.TrimSpace(r.msg.Text)
	i := strings.IndexByte(trimmed, ' ')
	if i < 0 {
		return b.reply(ctx, api, r, r.m.LangUsage, true)
	}

	lang, ok := i18n.ParseExplicit(strings.TrimSpace(trimmed[i+1:]))
	if !ok {
		return b.reply(ctx, api, r, r.m.LangUsage, true)
	}
	if err := b.deps.Users.SetLanguage(ctx, r.user.ID, lang); err != nil {
		return err
	}

	m := i18n.Get(lang)
	r.m = m
	return b.reply(ctx, api, r, fmt.Sprintf(m.LangSet, lang), true)
}

// cmdBroadcast sends one text to every known user. Admin-only: "broadcast <text>".
func (b *Bot) cmdBroadcast(ctx context.Context, api sender, r req) error {
	// The command word has no spaces, so the first space splits it from the text.
	trimmed := strings.TrimSpace(r.msg.Text)
	i := strings.IndexByte(trimmed, ' ')
	if i < 0 {
		return b.reply(ctx, api, r, r.m.BroadcastUsage, false)
	}
	msg := strings.TrimSpace(trimmed[i+1:])
	if msg == "" {
		return b.reply(ctx, api, r, r.m.BroadcastUsage, false)
	}

	ids, err := b.deps.Users.ListIDs(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		// A blocked chat must not stop the rest of the dispatch.
		if _, serr := api.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: id, Text: msg}); serr != nil {
			b.deps.Log.Error("broadcast", "user", id, "err", serr)
		}
	}
	return b.reply(ctx, api, r, r.m.SendQueued, false)
}

// cmdSend starts a scenario for one user or for everyone. Admin-only:
// "send <scenario> <id|all>".
func (b *Bot) cmdSend(ctx context.Context, api sender, r req) error {
	fields := strings.Fields(r.msg.Text)
	if len(fields) != 3 {
		return b.reply(ctx, api, r, r.m.SendUsage, false)
	}

	name := strings.ToLower(fields[1])
	if _, ok := scenario.ByName(name); !ok {
		return b.reply(ctx, api, r, r.m.SendUnknownScenario, false)
	}

	if target := strings.ToLower(fields[2]); target == "all" {
		ids, err := b.deps.Users.ListIDs(ctx)
		if err != nil {
			return err
		}
		for _, id := range ids {
			// A blocked chat must not stop the rest of the dispatch.
			if err := b.pushScenario(ctx, api, id, name); err != nil {
				b.deps.Log.Error("push scenario", "user", id, "err", err)
			}
		}
		return b.reply(ctx, api, r, r.m.SendQueued, false)
	}

	id, err := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
	if err != nil {
		return b.reply(ctx, api, r, r.m.SendUsage, false)
	}
	if err := b.pushScenario(ctx, api, id, name); err != nil {
		return err
	}
	return b.reply(ctx, api, r, r.m.SendQueued, false)
}

// --- dialogue ---

// dialogueEscape: whether a message leaves the running dialogue for the router.
// /start and "data remove all" also abort the dialogue; the admin send/broadcast
// commands pass through and the position stays. Pure, so it is testable without a Bot.
func dialogueEscape(text string, isAdmin bool) (leave, abort bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	switch {
	case lower == "/start", lower == "data remove all":
		return true, true
	case isAdmin && (strings.HasPrefix(lower, "send ") || strings.HasPrefix(lower, "broadcast")):
		return true, false
	default:
		return false, false
	}
}

// dialogue hands the message to a running dialogue. Returns true when it was consumed.
// /start and "data remove all" leave the dialogue; so do the admin send/broadcast
// commands, which pass through without touching the position.
func (b *Bot) dialogue(ctx context.Context, api sender, r req) bool {
	st, node, running, err := b.deps.Scenarios.Current(ctx, r.user.ID)
	if err != nil {
		b.deps.Log.Error("current scenario", "err", err)
		return false
	}
	if !running {
		return false
	}

	text := strings.TrimSpace(r.msg.Text)

	// Commands that leave the dialogue never reach Step. /start and "data remove all"
	// abort the dialogue; the admin send/broadcast commands pass through and the
	// position stays.
	if leave, abort := dialogueEscape(text, b.deps.Cfg.IsAdmin(r.user.ID)); leave {
		if abort {
			if aerr := b.deps.Scenarios.Abort(ctx, r.user.ID); aerr != nil {
				b.deps.Log.Error("abort scenario", "err", aerr)
			}
		}
		return false
	}

	// A reply-keyboard button sends its label; the node knows which Data that means.
	answer := node.ButtonData(r.m, text)

	out, err := b.deps.Scenarios.Answer(ctx, st, answer, time.Now().UTC())
	switch {
	case errors.Is(err, service.ErrCancelled):
		// Zero amount cancels the dialogue; the position is dropped already, so hand
		// the menu back.
		if rerr := b.reply(ctx, api, r, r.m.AmountCancelled, true); rerr != nil {
			b.deps.Log.Error("reply", "err", rerr)
		}
	case errors.Is(err, service.ErrOverdraw):
		// A withdrawal over the saved total keeps the position, so ask the amount again.
		if rerr := b.reply(ctx, api, r, r.m.Overdraw, false); rerr != nil {
			b.deps.Log.Error("reply", "err", rerr)
		}
		if serr := b.sendNode(ctx, api, r, node); serr != nil {
			b.deps.Log.Error("send node", "err", serr)
		}
	case errors.Is(err, service.ErrTierOrder):
		// A target is below the tier before it; the position stays, so ask the question
		// again.
		if rerr := b.reply(ctx, api, r, r.m.TierOrder, false); rerr != nil {
			b.deps.Log.Error("reply", "err", rerr)
		}
		if serr := b.sendNode(ctx, api, r, node); serr != nil {
			b.deps.Log.Error("send node", "err", serr)
		}
	case err != nil:
		// The answer did not fit, or the goal could not be built. Re-ask while the
		// dialogue is still on, otherwise hand the menu back.
		if rerr := b.reply(ctx, api, r, r.m.BadAnswer, !out.Running); rerr != nil {
			b.deps.Log.Error("reply", "err", rerr)
		}
		if out.Running {
			if serr := b.sendNode(ctx, api, r, node); serr != nil {
				b.deps.Log.Error("send node", "err", serr)
			}
		}
	case out.Running:
		if serr := b.sendNode(ctx, api, r, out.Node); serr != nil {
			b.deps.Log.Error("send node", "err", serr)
		}
	default:
		if rerr := b.reply(ctx, api, r, moodPhrase(r.m, out.Mood), true); rerr != nil {
			b.deps.Log.Error("reply", "err", rerr)
		}
	}
	return true
}

// startScenario opens a scenario for the person who asked and sends its first node
// into their chat.
func (b *Bot) startScenario(ctx context.Context, api sender, r req, name string) error {
	return b.startScenarioVars(ctx, api, r, name, nil)
}

// startScenarioVars: opens a scenario with some answers already filled in and sends its
// first node into the chat. Used when a value is picked before the dialogue opens, e.g.
// the goal of a deposit.
func (b *Bot) startScenarioVars(ctx context.Context, api sender, r req, name string, vars map[string]string) error {
	node, err := b.deps.Scenarios.BeginWithVars(ctx, r.user.ID, name, vars, time.Now().UTC())
	if err != nil {
		return err
	}
	return b.sendNode(ctx, api, r, node)
}

// pushScenario opens a scenario for another user and sends its first node into their
// chat. Language comes from their stored profile.
// TODO: post-MVP cleanup. Node send repeats in sendNode; fold both into a sendNodeTo
// helper.
func (b *Bot) pushScenario(ctx context.Context, api sender, userID int64, name string) error {
	u, err := b.deps.Users.Get(ctx, userID)
	if err != nil {
		return err
	}
	node, err := b.deps.Scenarios.Begin(ctx, userID, name, time.Now().UTC())
	if err != nil {
		return err
	}

	m := i18n.Get(i18n.Resolve(u.UILanguage, u.LanguageCode))
	_, err = api.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:      userID,
		Text:        node.Prompt(m),
		ReplyMarkup: nodeMarkup(node, m),
	})
	return err
}

// sendNode renders one dialogue node into the chat of the update.
func (b *Bot) sendNode(ctx context.Context, api sender, r req, node scenario.Node) error {
	return b.replyWith(ctx, api, r, node.Prompt(r.m), nodeMarkup(node, r.m))
}

// nodeMarkup: reply markup for a dialogue node. Buttons become a reply keyboard;
// without buttons the keyboard is removed, so the main menu does not sit under a
// question.
func nodeMarkup(node scenario.Node, m i18n.Messages) models.ReplyMarkup {
	if len(node.Buttons) == 0 {
		return models.ReplyKeyboardRemove{RemoveKeyboard: true}
	}
	rows := make([][]models.KeyboardButton, 0, len(node.Buttons))
	for _, btn := range node.Buttons {
		rows = append(rows, []models.KeyboardButton{{Text: btn.Label(m)}})
	}
	return models.ReplyKeyboardMarkup{Keyboard: rows, ResizeKeyboard: true}
}

// moodPhrase: the reply after a finished dialogue. A money mood gives one random phrase
// from its pool; MoodNone gives the plain done line.
func moodPhrase(m i18n.Messages, mood domain.Mood) string {
	pool := m.MoodPhrases(mood)
	if len(pool) == 0 {
		return m.ScenarioDone
	}
	return pool[rand.IntN(len(pool))]
}
