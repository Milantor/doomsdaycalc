package bot

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"testing"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"lab042.ru/doomsdaycalc/internal/config"
	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/i18n"
	"lab042.ru/doomsdaycalc/internal/scenario"
	"lab042.ru/doomsdaycalc/internal/service"
)

// fakeSender: stands in for the Telegram client. Keeps every message the handlers
// send, so a test can read back what a user would see.
type fakeSender struct {
	sent     []sentMessage
	answered []string
	err      error
}

// sentMessage: one captured SendMessage call.
type sentMessage struct {
	chatID any
	text   string
	markup models.ReplyMarkup
}

// newFakeSender: an empty recording sender.
func newFakeSender() *fakeSender { return &fakeSender{} }

// SendMessage records the call. A set err short-circuits, so a test can exercise the
// error path.
func (f *fakeSender) SendMessage(_ context.Context, params *tgbot.SendMessageParams) (*models.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.sent = append(f.sent, sentMessage{chatID: params.ChatID, text: params.Text, markup: params.ReplyMarkup})
	return &models.Message{}, nil
}

// AnswerCallbackQuery records the acknowledged callback id, so a test can check the
// spinner was stopped.
func (f *fakeSender) AnswerCallbackQuery(_ context.Context, params *tgbot.AnswerCallbackQueryParams) (bool, error) {
	f.answered = append(f.answered, params.CallbackQueryID)
	return true, nil
}

// texts: the text of every captured message, in order.
func (f *fakeSender) texts() []string {
	out := make([]string, 0, len(f.sent))
	for _, s := range f.sent {
		out = append(out, s.text)
	}
	return out
}

// fakeStateRepo: in-memory dialogue positions.
type fakeStateRepo struct {
	states map[int64]domain.ScenarioState
}

// newFakeStateRepo: an empty state repository.
func newFakeStateRepo() *fakeStateRepo {
	return &fakeStateRepo{states: map[int64]domain.ScenarioState{}}
}

// Get returns the saved position, or ErrNotFound when none is set.
func (f *fakeStateRepo) Get(_ context.Context, userID int64) (domain.ScenarioState, error) {
	st, ok := f.states[userID]
	if !ok {
		return domain.ScenarioState{}, domain.ErrNotFound
	}
	return st, nil
}

// Save stores the position, keyed by user.
func (f *fakeStateRepo) Save(_ context.Context, s domain.ScenarioState) error {
	f.states[s.UserID] = s
	return nil
}

// Delete drops the position of a user. Silent when none is set.
func (f *fakeStateRepo) Delete(_ context.Context, userID int64) error {
	delete(f.states, userID)
	return nil
}

// mustSave: seeds a position and fails the test on error.
func (f *fakeStateRepo) mustSave(t *testing.T, s domain.ScenarioState) {
	t.Helper()
	if err := f.Save(context.Background(), s); err != nil {
		t.Fatalf("save state: %v", err)
	}
}

// fakeGoalRepo: in-memory goals.
type fakeGoalRepo struct {
	created []domain.Goal
	nextID  int64
}

// newFakeGoalRepo: an empty goal repository.
func newFakeGoalRepo() *fakeGoalRepo { return &fakeGoalRepo{} }

// Create assigns an id and keeps the goal.
func (f *fakeGoalRepo) Create(_ context.Context, g domain.Goal) (domain.Goal, error) {
	f.nextID++
	g.ID = f.nextID
	f.created = append(f.created, g)
	return g, nil
}

// Get returns the goal with the given id, or ErrNotFound.
func (f *fakeGoalRepo) Get(_ context.Context, id int64) (domain.Goal, error) {
	for _, g := range f.created {
		if g.ID == id {
			return g, nil
		}
	}
	return domain.Goal{}, domain.ErrNotFound
}

// ListByUser returns the goals of a user, oldest first.
func (f *fakeGoalRepo) ListByUser(_ context.Context, userID int64) ([]domain.Goal, error) {
	var out []domain.Goal
	for _, g := range f.created {
		if g.UserID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}

// SetActive is a no-op here: the dialogue tests never flip activity.
func (f *fakeGoalRepo) SetActive(context.Context, int64, bool) error { return nil }

// fakeDepositRepo: in-memory deposits.
type fakeDepositRepo struct {
	added  []domain.Deposit
	nextID int64
}

// newFakeDepositRepo: an empty deposit repository.
func newFakeDepositRepo() *fakeDepositRepo { return &fakeDepositRepo{} }

// Add assigns an id and keeps the deposit.
func (f *fakeDepositRepo) Add(_ context.Context, d domain.Deposit) (domain.Deposit, error) {
	f.nextID++
	d.ID = f.nextID
	f.added = append(f.added, d)
	return d, nil
}

// ListByGoal returns deposits of a goal, oldest first.
func (f *fakeDepositRepo) ListByGoal(_ context.Context, goalID int64) ([]domain.Deposit, error) {
	var out []domain.Deposit
	for _, d := range f.added {
		if d.GoalID == goalID {
			out = append(out, d)
		}
	}
	return out, nil
}

// fakeUserRepo: in-memory users.
type fakeUserRepo struct {
	users map[int64]domain.User
	order []int64
}

// newFakeUserRepo: an empty user repository.
func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[int64]domain.User{}}
}

// Upsert inserts or refreshes the profile and keeps the ui_language override.
func (f *fakeUserRepo) Upsert(_ context.Context, u domain.User) (domain.User, error) {
	if old, ok := f.users[u.ID]; ok {
		u.UILanguage = old.UILanguage
	} else {
		f.order = append(f.order, u.ID)
	}
	f.users[u.ID] = u
	return u, nil
}

// Get returns the user with the given id, or ErrNotFound.
func (f *fakeUserRepo) Get(_ context.Context, id int64) (domain.User, error) {
	u, ok := f.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

// Delete drops the user.
func (f *fakeUserRepo) Delete(_ context.Context, id int64) error {
	delete(f.users, id)
	return nil
}

// ListIDs returns every known id, oldest first.
func (f *fakeUserRepo) ListIDs(_ context.Context) ([]int64, error) {
	return append([]int64(nil), f.order...), nil
}

// testEnv: a Bot wired with the in-memory fakes plus the recording sender.
type testEnv struct {
	bot      *Bot
	sender   *fakeSender
	states   *fakeStateRepo
	goals    *fakeGoalRepo
	deposits *fakeDepositRepo
	users    *fakeUserRepo
}

// newTestEnv: builds the Bot over the fakes, so the dialogue loop runs with no
// Telegram and no SQL.
func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	states := newFakeStateRepo()
	goals := newFakeGoalRepo()
	deposits := newFakeDepositRepo()
	users := newFakeUserRepo()

	savings := service.NewSavingsService(goals, deposits)

	b := &Bot{deps: &Deps{
		Cfg:       &config.Config{},
		Log:       slog.New(slog.NewTextHandler(io.Discard, nil)),
		Users:     service.NewUserService(users),
		Savings:   savings,
		Scenarios: service.NewScenarioService(states, savings),
	}}
	return &testEnv{bot: b, sender: newFakeSender(), states: states, goals: goals, deposits: deposits, users: users}
}

// testReq: a request for one user in one chat.
func testReq(m i18n.Messages, userID, chatID int64, text string) req {
	return req{
		user: domain.User{ID: userID},
		m:    m,
		msg:  &models.Message{Text: text, Chat: models.Chat{ID: chatID}},
	}
}

// feed: pushes one text into the dialogue and fails when it is not consumed.
func feed(t *testing.T, env *testEnv, ctx context.Context, r req, text string) {
	t.Helper()
	r.msg.Text = text
	if !env.bot.dialogue(ctx, env.sender, r) {
		t.Fatalf("dialogue did not consume %q", text)
	}
}

// TestDialogueEscape covers which messages leave a running dialogue for the router.
func TestDialogueEscape(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		admin bool
		leave bool
		abort bool
	}{
		{"slash start", "/start", false, true, true},
		{"start uppercase and padded", "  /START  ", false, true, true},
		{"data remove all", "data remove all", false, true, true},
		{"admin send", "send add_goal 5", true, true, false},
		{"admin send uppercase", "SEND onboarding all", true, true, false},
		{"admin broadcast", "broadcast maintenance tonight", true, true, false},
		{"non-admin send is an answer", "send add_goal 5", false, false, false},
		{"plain answer", "vacation", false, false, false},
		{"sender word", "sender", false, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			leave, abort := dialogueEscape(c.text, c.admin)
			if leave != c.leave || abort != c.abort {
				t.Fatalf("dialogueEscape(%q, %v) = (%v, %v), want (%v, %v)", c.text, c.admin, leave, abort, c.leave, c.abort)
			}
		})
	}
}

// TestDialogueNoScenario: with nothing running the message is left alone.
func TestDialogueNoScenario(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "hello")

	if env.bot.dialogue(ctx, env.sender, r) {
		t.Fatalf("dialogue consumed a message while no scenario was running")
	}
	if len(env.sender.sent) != 0 {
		t.Fatalf("dialogue sent %d messages, want 0", len(env.sender.sent))
	}
}

// TestDialogueAbortsOnStart: /start drops the position and leaves the message.
func TestDialogueAbortsOnStart(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "/start")

	env.states.mustSave(t, domain.ScenarioState{
		UserID:       100,
		ScenarioName: "add_goal",
		NodeID:       "ask_title",
		Vars:         map[string]string{},
	})

	if env.bot.dialogue(ctx, env.sender, r) {
		t.Fatalf("dialogue consumed /start")
	}
	if len(env.states.states) != 0 {
		t.Fatalf("position not dropped")
	}
	if len(env.sender.sent) != 0 {
		t.Fatalf("dialogue sent %d messages, want 0", len(env.sender.sent))
	}
}

// TestDialogueAdminSendKeepsPosition: the admin send command passes through and the
// running dialogue stays where it was.
func TestDialogueAdminSendKeepsPosition(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "send add_goal 5")

	env.bot.deps.Cfg.AdminIDs = []int64{100}
	env.states.mustSave(t, domain.ScenarioState{
		UserID:       100,
		ScenarioName: "add_goal",
		NodeID:       "ask_title",
		Vars:         map[string]string{},
	})

	if env.bot.dialogue(ctx, env.sender, r) {
		t.Fatalf("dialogue consumed the admin send command")
	}
	if _, ok := env.states.states[100]; !ok {
		t.Fatalf("position dropped")
	}
}

// TestDialogueReasksBadAnswer: an answer the node refuses asks the same question again.
func TestDialogueReasksBadAnswer(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "")

	env.states.mustSave(t, domain.ScenarioState{
		UserID:       100,
		ScenarioName: "add_goal",
		NodeID:       "ask_deadline",
		Vars:         map[string]string{},
	})

	feed(t, env, ctx, r, "not-a-date")

	if got, want := env.sender.texts(), []string{m.BadAnswer, m.AskDeadline}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if st := env.states.states[100]; st.NodeID != "ask_deadline" {
		t.Fatalf("node = %q, want ask_deadline", st.NodeID)
	}
}

// TestDialogueCancelsZeroAmount: a zero amount ends the money dialogue, stores nothing
// and shows the cancel reply with the menu back.
func TestDialogueCancelsZeroAmount(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "")

	env.states.mustSave(t, domain.ScenarioState{
		UserID:       100,
		ScenarioName: "add_deposit",
		NodeID:       "ask_amount",
		Vars:         map[string]string{scenario.VarGoal: "7"},
	})

	feed(t, env, ctx, r, "0")

	if got, want := env.sender.texts(), []string{m.AmountCancelled}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if len(env.deposits.added) != 0 {
		t.Errorf("stored %d deposits, want 0", len(env.deposits.added))
	}
	if len(env.states.states) != 0 {
		t.Fatalf("position not dropped after a cancel")
	}
}

// TestDialogueWithdrawOverdraw: a withdrawal over the saved total shows the overdraw note
// and asks the amount again, keeping the position and storing nothing.
func TestDialogueWithdrawOverdraw(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "")

	g, err := env.goals.Create(ctx, domain.Goal{UserID: 100, Title: "Trip"})
	if err != nil {
		t.Fatalf("seed goal: %v", err)
	}
	if _, err := env.deposits.Add(ctx, domain.Deposit{GoalID: g.ID, Amount: 10000}); err != nil {
		t.Fatalf("seed deposit: %v", err)
	}
	env.states.mustSave(t, domain.ScenarioState{
		UserID:       100,
		ScenarioName: "withdraw",
		NodeID:       "ask_amount",
		Vars:         map[string]string{scenario.VarGoal: strconv.FormatInt(g.ID, 10)},
	})

	feed(t, env, ctx, r, "500")

	if got, want := env.sender.texts(), []string{m.Overdraw, m.AskWithdrawAmount}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if st := env.states.states[100]; st.NodeID != "ask_amount" {
		t.Fatalf("node = %q, want ask_amount", st.NodeID)
	}
	if len(env.deposits.added) != 1 {
		t.Errorf("stored %d deposits, want only the seeded one", len(env.deposits.added))
	}
}

// TestDialogueTierOrderReasks: a target below the previous tier shows the order note and
// asks the amount again.
func TestDialogueTierOrderReasks(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "")

	env.states.mustSave(t, domain.ScenarioState{
		UserID:       100,
		ScenarioName: "add_goal",
		NodeID:       "ask_target_max",
		Vars: map[string]string{
			scenario.VarTitle:     "Trip",
			scenario.VarDeadline:  "2030-01-01",
			scenario.VarTargetMin: "35",
			scenario.VarTargetOK:  "42",
		},
	})

	feed(t, env, ctx, r, "35")

	if got, want := env.sender.texts(), []string{m.TierOrder, m.AskTargetMax}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if st := env.states.states[100]; st.NodeID != "ask_target_max" {
		t.Fatalf("node = %q, want ask_target_max", st.NodeID)
	}
}

// TestDialogueWithdrawExactBalance: withdrawing exactly the saved total finishes and
// stores the negative row.
func TestDialogueWithdrawExactBalance(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "")

	g, err := env.goals.Create(ctx, domain.Goal{UserID: 100, Title: "Trip"})
	if err != nil {
		t.Fatalf("seed goal: %v", err)
	}
	if _, err := env.deposits.Add(ctx, domain.Deposit{GoalID: g.ID, Amount: 10000}); err != nil {
		t.Fatalf("seed deposit: %v", err)
	}
	env.states.mustSave(t, domain.ScenarioState{
		UserID:       100,
		ScenarioName: "withdraw",
		NodeID:       "ask_amount",
		Vars:         map[string]string{scenario.VarGoal: strconv.FormatInt(g.ID, 10)},
	})

	feed(t, env, ctx, r, "100")

	if got, want := env.sender.texts(), []string{m.ScenarioDone}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if len(env.deposits.added) != 2 {
		t.Fatalf("stored %d deposits, want 2", len(env.deposits.added))
	}
	if got := env.deposits.added[1].Amount; got != -10000 {
		t.Errorf("withdraw amount = %d, want -10000", got)
	}
	if len(env.states.states) != 0 {
		t.Fatalf("position not cleared after finish")
	}
}

// TestDialogueDepositIgnoresBalance: the overdraw rule guards withdrawals only, so a
// deposit over the saved total still finishes.
func TestDialogueDepositIgnoresBalance(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "")

	g, err := env.goals.Create(ctx, domain.Goal{UserID: 100, Title: "Trip"})
	if err != nil {
		t.Fatalf("seed goal: %v", err)
	}
	env.states.mustSave(t, domain.ScenarioState{
		UserID:       100,
		ScenarioName: "add_deposit",
		NodeID:       "ask_amount",
		Vars:         map[string]string{scenario.VarGoal: strconv.FormatInt(g.ID, 10)},
	})

	feed(t, env, ctx, r, "500")

	if got, want := env.sender.texts(), []string{m.ScenarioDone}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if len(env.deposits.added) != 1 || env.deposits.added[0].Amount != 50000 {
		t.Fatalf("deposits = %+v, want one 50000", env.deposits.added)
	}
}

// TestDialogueWalksOnboarding: a full onboarding walk stores one goal and clears the
// position.
func TestDialogueWalksOnboarding(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "")

	if err := env.bot.startScenario(ctx, env.sender, r, "onboarding"); err != nil {
		t.Fatalf("startScenario: %v", err)
	}

	answers := []string{m.BtnStart, "Vacation", "2030-01-01", "100", "200", "300"}
	for _, text := range answers {
		feed(t, env, ctx, r, text)
	}

	if len(env.goals.created) != 1 {
		t.Fatalf("goals = %d, want 1", len(env.goals.created))
	}
	g := env.goals.created[0]
	if g.Title != "Vacation" {
		t.Fatalf("title = %q, want Vacation", g.Title)
	}
	wantTargets := map[domain.Tier]domain.Money{
		domain.TierMinimal:    10000,
		domain.TierAcceptable: 20000,
		domain.TierOptimal:    30000,
	}
	for tier, want := range wantTargets {
		if got := g.Targets[tier]; got != want {
			t.Fatalf("target %s = %d, want %d", tier, got, want)
		}
	}
	if len(env.states.states) != 0 {
		t.Fatalf("position not cleared after finish")
	}
	if last := env.sender.texts(); last[len(last)-1] != m.ScenarioDone {
		t.Fatalf("last text = %q, want %q", last[len(last)-1], m.ScenarioDone)
	}
}

// TestDialogueOnboardingIntroTakesAnyAnswer: a typed word at the intro moves on.
func TestDialogueOnboardingIntroTakesAnyAnswer(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "")

	if err := env.bot.startScenario(ctx, env.sender, r, "onboarding"); err != nil {
		t.Fatalf("startScenario: %v", err)
	}
	env.sender.sent = nil

	feed(t, env, ctx, r, "привет, чё это")

	if got, want := env.sender.texts(), []string{m.AskTitle}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if st := env.states.states[100]; st.NodeID != "ask_title" {
		t.Fatalf("node = %q, want ask_title", st.NodeID)
	}
}

// TestStartScenarioSendsFirstNode: opening a scenario sends its entry node and saves
// the position there.
func TestStartScenarioSendsFirstNode(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)
	r := testReq(m, 100, 100, "")

	if err := env.bot.startScenario(ctx, env.sender, r, "add_goal"); err != nil {
		t.Fatalf("startScenario: %v", err)
	}
	if got, want := env.sender.texts(), []string{m.AskTitle}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if st := env.states.states[100]; st.NodeID != "ask_title" {
		t.Fatalf("node = %q, want ask_title", st.NodeID)
	}
}

// TestCmdSendToUser: the admin send command opens a scenario in the target chat and
// confirms in the admin chat.
func TestCmdSendToUser(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)

	// the target must be known, so pushScenario can read the language
	if _, err := env.bot.deps.Users.Touch(ctx, domain.User{ID: 42, LanguageCode: "en"}); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	r := testReq(m, 100, 100, "send add_goal 42")
	if err := env.bot.cmdSend(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdSend: %v", err)
	}

	if got, want := env.sender.texts(), []string{m.AskTitle, m.SendQueued}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if env.sender.sent[0].chatID != int64(42) {
		t.Fatalf("first chat = %v, want 42", env.sender.sent[0].chatID)
	}
}

// TestCmdSendToAll: the all target opens the scenario for every known user.
func TestCmdSendToAll(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)

	for _, id := range []int64{1, 2} {
		if _, err := env.bot.deps.Users.Touch(ctx, domain.User{ID: id, LanguageCode: "en"}); err != nil {
			t.Fatalf("seed user %d: %v", id, err)
		}
	}

	r := testReq(m, 99, 99, "send add_goal all")
	if err := env.bot.cmdSend(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdSend: %v", err)
	}

	if got, want := env.sender.texts(), []string{m.AskTitle, m.AskTitle, m.SendQueued}; !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
}

// TestCmdSendUsage: bad arguments come back as the usage hint.
func TestCmdSendUsage(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)

	bad := []string{"send add_goal", "send add_goal notanumber", "send add_goal 1 2"}
	for _, text := range bad {
		env.sender.sent = nil
		r := testReq(m, 100, 100, text)
		if err := env.bot.cmdSend(ctx, env.sender, r); err != nil {
			t.Fatalf("cmdSend(%q): %v", text, err)
		}
		if got := env.sender.texts(); !slices.Equal(got, []string{m.SendUsage}) {
			t.Fatalf("cmdSend(%q) texts = %q, want [%q]", text, got, m.SendUsage)
		}
	}
}

// TestCmdSendUnknownScenario: an unknown name comes back as the unknown hint.
func TestCmdSendUnknownScenario(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)

	r := testReq(m, 100, 100, "send nope 42")
	if err := env.bot.cmdSend(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdSend: %v", err)
	}
	if got := env.sender.texts(); !slices.Equal(got, []string{m.SendUnknownScenario}) {
		t.Fatalf("texts = %q, want [%q]", got, m.SendUnknownScenario)
	}
}

// TestCmdBroadcast: the admin broadcast sends the text to every known user and confirms.
func TestCmdBroadcast(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)

	for _, id := range []int64{1, 2} {
		if _, err := env.bot.deps.Users.Touch(ctx, domain.User{ID: id, LanguageCode: "en"}); err != nil {
			t.Fatalf("seed user %d: %v", id, err)
		}
	}

	r := testReq(m, 99, 99, "broadcast maintenance at 22:00")
	if err := env.bot.cmdBroadcast(ctx, env.sender, r); err != nil {
		t.Fatalf("cmdBroadcast: %v", err)
	}

	want := []string{"maintenance at 22:00", "maintenance at 22:00", m.SendQueued}
	if got := env.sender.texts(); !slices.Equal(got, want) {
		t.Fatalf("texts = %q, want %q", got, want)
	}
	if env.sender.sent[0].chatID != int64(1) || env.sender.sent[1].chatID != int64(2) {
		t.Fatalf("broadcast chats = %v, %v, want 1 and 2", env.sender.sent[0].chatID, env.sender.sent[1].chatID)
	}
}

// TestCmdBroadcastUsage: a broadcast without text comes back as the usage hint.
func TestCmdBroadcastUsage(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()
	m := i18n.Get(domain.LangEN)

	for _, text := range []string{"broadcast", "broadcast   "} {
		env.sender.sent = nil
		r := testReq(m, 99, 99, text)
		if err := env.bot.cmdBroadcast(ctx, env.sender, r); err != nil {
			t.Fatalf("cmdBroadcast(%q): %v", text, err)
		}
		if got := env.sender.texts(); !slices.Equal(got, []string{m.BroadcastUsage}) {
			t.Fatalf("cmdBroadcast(%q) texts = %q, want [%q]", text, got, m.BroadcastUsage)
		}
	}
}

// TestNodeMarkup: a free-text node drops the keyboard, a button node becomes a reply
// keyboard.
func TestNodeMarkup(t *testing.T) {
	m := i18n.Get(domain.LangEN)

	free := scenario.Node{}
	if _, ok := nodeMarkup(free, m).(models.ReplyKeyboardRemove); !ok {
		t.Fatalf("free-text node did not remove the keyboard")
	}

	btn := scenario.Node{Buttons: []scenario.Button{
		{Label: func(i18n.Messages) string { return "Go" }, Data: "go"},
	}}
	kb, ok := nodeMarkup(btn, m).(models.ReplyKeyboardMarkup)
	if !ok {
		t.Fatalf("button node did not build a reply keyboard")
	}
	if len(kb.Keyboard) != 1 || len(kb.Keyboard[0]) != 1 || kb.Keyboard[0][0].Text != "Go" {
		t.Fatalf("keyboard = %+v, want one Go button", kb.Keyboard)
	}
}
