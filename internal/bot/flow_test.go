package bot

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"

	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/i18n"
	"lab042.ru/doomsdaycalc/internal/scenario"
)

// seedGoal: puts one goal into the fake repo, so ListGoals returns it.
func seedGoal(env *testEnv, id, userID int64, title string) {
	env.goals.created = append(env.goals.created, domain.Goal{
		ID:              id,
		UserID:          userID,
		Title:           title,
		Currency:        domain.CurrencyRUB,
		SavingsCurrency: domain.CurrencyRUB,
	})
}

// TestParseCallback covers the inline-button data split.
func TestParseCallback(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		action string
		arg    string
		ok     bool
	}{
		{"deposit", "dep:7", "dep", "7", true},
		{"status", "st:42", "st", "42", true},
		{"no colon", "garbage", "", "", false},
		{"empty", "", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			action, arg, ok := parseCallback(c.in)
			if ok != c.ok || action != c.action || arg != c.arg {
				t.Fatalf("parseCallback(%q) = (%q, %q, %v)", c.in, action, arg, ok)
			}
		})
	}
}

// TestGoalsKeyboard: one inline button per goal, data carrying the prefix and id.
func TestGoalsKeyboard(t *testing.T) {
	goals := []domain.Goal{{ID: 1, Title: "A"}, {ID: 20, Title: "B"}}
	kb := goalsKeyboard(callbackDeposit, goals)

	if len(kb.InlineKeyboard) != 2 {
		t.Fatalf("rows = %d, want 2", len(kb.InlineKeyboard))
	}
	if got := kb.InlineKeyboard[0][0]; got.Text != "A" || got.CallbackData != "dep:1" {
		t.Errorf("first button = %+v, want A/dep:1", got)
	}
	if got := kb.InlineKeyboard[1][0]; got.Text != "B" || got.CallbackData != "dep:20" {
		t.Errorf("second button = %+v, want B/dep:20", got)
	}
}

// TestCmdAddSingleGoal: with one goal the amount question opens right away and the goal
// is seeded into the dialogue vars.
func TestCmdAddSingleGoal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	seedGoal(env, 7, 100, "Trip")

	r := testReq(m, 100, 100, "")
	if err := env.bot.cmdAdd(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}
	if got, want := env.sender.texts(), []string{m.AskDepositAmount}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if st := env.states.states[100]; st.ScenarioName != "add_deposit" || st.Vars[scenario.VarGoal] != "7" {
		t.Fatalf("state = %+v, want add_deposit on goal 7", st)
	}
}

// TestCmdAddMultipleGoals: with several goals an inline picker goes out first.
func TestCmdAddMultipleGoals(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	seedGoal(env, 1, 100, "A")
	seedGoal(env, 2, 100, "B")

	r := testReq(m, 100, 100, "")
	if err := env.bot.cmdAdd(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}
	if got, want := env.sender.texts(), []string{m.ChooseGoal}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	kb, ok := env.sender.sent[0].markup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("no inline keyboard sent")
	}
	if len(kb.InlineKeyboard) != 2 ||
		kb.InlineKeyboard[0][0].CallbackData != "dep:1" ||
		kb.InlineKeyboard[1][0].CallbackData != "dep:2" {
		t.Fatalf("keyboard = %+v, want dep:1 and dep:2", kb.InlineKeyboard)
	}
}

// TestCmdAddWithoutGoal: nothing to put money into.
func TestCmdAddWithoutGoal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)

	r := testReq(m, 100, 100, "")
	if err := env.bot.cmdAdd(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}
	if got, want := env.sender.texts(), []string{m.NoGoal}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
}

// TestCmdStatusNoGoals: no goals gives the no-goal prompt.
func TestCmdStatusNoGoals(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)

	r := testReq(m, 100, 100, "")
	if err := env.bot.cmdStatus(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdStatus: %v", err)
	}
	if got, want := env.sender.texts(), []string{m.NoGoal}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
}

// TestCmdStatusSingleGoal: the status of the only goal is rendered right away.
func TestCmdStatusSingleGoal(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	env.goals.created = append(env.goals.created, seededStatusGoal(7, 100))

	r := testReq(m, 100, 100, "")
	if err := env.bot.cmdStatus(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdStatus: %v", err)
	}
	if len(env.sender.sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(env.sender.sent))
	}
	if !strings.Contains(env.sender.sent[0].text, "Trip") {
		t.Fatalf("status text = %q, want it to name the goal", env.sender.sent[0].text)
	}
}

// TestCmdStatusMultipleGoals: with several goals a picker goes out, keyed "st:".
func TestCmdStatusMultipleGoals(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	seedGoal(env, 1, 100, "A")
	seedGoal(env, 2, 100, "B")

	r := testReq(m, 100, 100, "")
	if err := env.bot.cmdStatus(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdStatus: %v", err)
	}
	kb, ok := env.sender.sent[0].markup.(models.InlineKeyboardMarkup)
	if !ok {
		t.Fatalf("no inline keyboard sent")
	}
	if kb.InlineKeyboard[0][0].CallbackData != "st:1" {
		t.Fatalf("first data = %q, want st:1", kb.InlineKeyboard[0][0].CallbackData)
	}
}

// TestOnCallbackDeposit: tapping a goal button opens the deposit dialogue on it and
// stops the spinner. The language comes from the callback from-profile, so with no code
// it is the default.
func TestOnCallbackDeposit(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangRU)
	seedGoal(env, 7, 100, "Trip")

	cq := &models.CallbackQuery{
		ID:   "c1",
		From: models.User{ID: 100},
		Data: "dep:7",
		Message: models.MaybeInaccessibleMessage{
			Message: &models.Message{Chat: models.Chat{ID: 100}},
		},
	}
	if err := env.bot.onCallback(ctx, env.sender, cq, domain.User{ID: 100}, m); err != nil {
		t.Fatalf("onCallback: %v", err)
	}
	if !slices.Equal(env.sender.answered, []string{"c1"}) {
		t.Fatalf("answered = %v, want [c1]", env.sender.answered)
	}
	if got, want := env.sender.texts(), []string{m.AskDepositAmount}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if st := env.states.states[100]; st.ScenarioName != "add_deposit" || st.Vars[scenario.VarGoal] != "7" {
		t.Fatalf("state = %+v, want add_deposit on goal 7", st)
	}
}

// TestOnCallbackStatus: tapping a goal button under a status picker renders that goal.
func TestOnCallbackStatus(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	env.goals.created = append(env.goals.created, seededStatusGoal(7, 100))

	cq := &models.CallbackQuery{
		ID:   "c2",
		From: models.User{ID: 100},
		Data: "st:7",
		Message: models.MaybeInaccessibleMessage{
			Message: &models.Message{Chat: models.Chat{ID: 100}},
		},
	}
	if err := env.bot.onCallback(ctx, env.sender, cq, domain.User{ID: 100}, i18n.Get(domain.LangRU)); err != nil {
		t.Fatalf("onCallback: %v", err)
	}
	if len(env.sender.sent) != 1 || !strings.Contains(env.sender.sent[0].text, "Trip") {
		t.Fatalf("sent = %+v, want one status message naming the goal", env.sender.sent)
	}
}

// TestOnCallbackIgnoresJunk: data this bot did not write is acknowledged and dropped.
func TestOnCallbackIgnoresJunk(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	cq := &models.CallbackQuery{
		ID:   "c3",
		From: models.User{ID: 100},
		Data: "garbage",
		Message: models.MaybeInaccessibleMessage{
			Message: &models.Message{Chat: models.Chat{ID: 100}},
		},
	}
	if err := env.bot.onCallback(ctx, env.sender, cq, domain.User{ID: 100}, i18n.Get(domain.LangRU)); err != nil {
		t.Fatalf("onCallback: %v", err)
	}
	if len(env.sender.sent) != 0 {
		t.Fatalf("sent %d messages, want 0", len(env.sender.sent))
	}
	if len(env.states.states) != 0 {
		t.Fatalf("a dialogue was started from junk data")
	}
}
