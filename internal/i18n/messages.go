package i18n

import "lab042.ru/doomsdaycalc/internal/domain"

// Messages: the full set of user-facing strings for one language.
// A struct beats map[string]string: a typo in a field name is a compile error. Go
// does not require every field to be filled, so TestCatalogComplete checks each
// catalog entry for missing strings.
// Fields are grouped by the feature that shows them, in the order a user meets them.
// A new string goes into the matching group, not tacked onto the end.
type Messages struct {
	// Buttons: the main keyboard and the dormant tier picker.
	BtnAdd      string // button: add money to savings
	BtnWithdraw string // button: withdraw money from savings
	BtnStatus   string // button: show progress
	BtnOther    string // button: placeholder, no action yet
	// TODO: i need that? tier buttons are dormant, maybe for goal editing.
	BtnTierMin string // inline button: minimal tier
	BtnTierOK  string // inline button: acceptable tier
	BtnTierMax string // inline button: optimal tier

	// Common replies, shown by several features.
	Greeting     string // /start welcome
	Fallback     string // reply when nothing matched
	ScenarioDone string // confirmation after any scenario finishes
	BadAnswer    string // reply when an answer does not fit the question

	// Privacy and data removal.
	Privacy     string // /privacy text
	DataRemoved string // confirmation after data remove all

	// Onboarding: building the first goal.
	OnboardingIntro string // onboarding: intro node message
	BtnStart        string // button: advance the intro node
	AskTitle        string // onboarding: goal title
	AskDeadline     string // onboarding: deadline date
	AskTargetMin    string // onboarding: minimal amount
	AskTargetOK     string // onboarding: acceptable amount
	AskTargetMax    string // onboarding: optimal amount
	TierOrder       string // onboarding: a target is below the tier before it

	// Money dialogue: deposit and withdraw.
	AskDepositAmount  string // deposit: amount to put aside
	AskWithdrawAmount string // withdraw: amount to take back
	NoGoal            string // action needs a goal, but the user has none
	ChooseGoal        string // prompt above the inline goal picker
	AmountCancelled   string // money dialogue: zero amount, nothing was stored
	Overdraw          string // withdraw: amount is over the saved total, the question repeats

	// Status screen.
	GoalDone            string // status: goal has every tier reached
	StatusSavedLabel    string // status line label: saved
	StatusSavedOf       string // status: word between the saved total and the target triple
	StatusLeftLabel     string // status line label: remaining
	StatusPerDayLabel   string // status line label: per day
	StatusPerMonthLabel string // status line label: per month
	StatusDeadlineLabel string // status line label: deadline
	StatusPaceLabel     string // status line label: pace
	StatusDaysLeft      string // status suffix, takes the day count
	StatusTiers         string // tier names for the status header, minimal first

	// Admin commands.
	SendUsage           string // admin: send command help
	SendUnknownScenario string // admin: scenario name is not registered
	SendQueued          string // admin: dispatch accepted
	BroadcastUsage      string // admin: broadcast command help
}

// catalog: every supported language mapped to its strings. "rofl" is a joke variant
// of Russian, reachable only through an explicit per-user override.
var catalog = map[domain.Lang]Messages{
	domain.LangRU: {
		// buttons
		BtnAdd:      "➕ Отложил",
		BtnWithdraw: "➖ Снял",
		BtnStatus:   "📊 Статус",
		BtnOther:    "⚙️ Другое",
		BtnTierMin:  "Минимум",
		BtnTierOK:   "Приемлемо",
		BtnTierMax:  "Оптимально",

		// common
		Greeting:     "Привет! Я помогу откладывать к дедлайну.",
		Fallback:     "Не понял. Нажми кнопку ниже или напиши status.",
		ScenarioDone: "Готово.",
		BadAnswer:    "Не понял ответ. Напиши ещё раз.",

		// privacy
		Privacy:     "Я храню твой Telegram ID, имя, а также цели и взносы. Третьим лицам ничего не передаю. Чтобы удалить всё — напиши «data remove all».",
		DataRemoved: "Готово. Все твои данные удалены.",

		// onboarding
		OnboardingIntro: "Давай заведём первую цель. Спрошу название, срок и три суммы: минимум, приемлемо и оптимум.",
		BtnStart:        "Поехали",
		AskTitle:        "Как назовём цель?",
		AskDeadline:     "К какому сроку? Дата в формате ГГГГ-ММ-ДД.",
		AskTargetMin:    "Минимальная сумма?",
		AskTargetOK:     "Приемлемая сумма?",
		AskTargetMax:    "Оптимальная сумма?",
		TierOrder:       "Каждая следующая сумма не может быть меньше предыдущей.",

		// money
		AskDepositAmount:  "Сколько отложить? Число.",
		AskWithdrawAmount: "Сколько снять? Число.",
		NoGoal:            "Сначала заведи цель.",
		ChooseGoal:        "Выбери цель:",
		AmountCancelled:   "Ок, отменил.",
		Overdraw:          "Больше, чем накоплено, снять нельзя.",

		// status
		GoalDone:            "Цель пройдена!",
		StatusSavedLabel:    "Отложено",
		StatusSavedOf:       "из",
		StatusLeftLabel:     "Осталось",
		StatusPerDayLabel:   "В день",
		StatusPerMonthLabel: "В месяц",
		StatusDeadlineLabel: "Дедлайн",
		StatusPaceLabel:     "Темп",
		StatusDaysLeft:      "осталось %d дн.",
		StatusTiers:         "минимум/приемлемо/оптимум",

		// admin
		SendUsage:           "Формат: send <сценарий> <id|all>",
		SendUnknownScenario: "Нет такого сценария.",
		SendQueued:          "Разослал.",
		BroadcastUsage:      "Формат: broadcast <текст>",
	},
	domain.LangEN: {
		// buttons
		BtnAdd:      "➕ Saved",
		BtnWithdraw: "➖ Withdrew",
		BtnStatus:   "📊 Status",
		BtnOther:    "⚙️ Other",
		BtnTierMin:  "Minimum",
		BtnTierOK:   "Acceptable",
		BtnTierMax:  "Optimum",

		// common
		Greeting:     "Hi! I will help you save towards a deadline.",
		Fallback:     "I did not get that. Tap a button below or type status.",
		ScenarioDone: "Done.",
		BadAnswer:    "I did not get that. Send it again.",

		// privacy
		Privacy:     "I store your Telegram ID, your name, and your goals and deposits. I pass nothing to third parties. To erase everything, send «data remove all».",
		DataRemoved: "Done. All your data has been deleted.",

		// onboarding
		OnboardingIntro: "Let us set up your first goal. I will ask for a title, a deadline and three amounts: minimum, acceptable and optimum.",
		BtnStart:        "Lets go",
		AskTitle:        "What should we call the goal?",
		AskDeadline:     "By when? Date as YYYY-MM-DD.",
		AskTargetMin:    "Minimum amount?",
		AskTargetOK:     "Acceptable amount?",
		AskTargetMax:    "Optimum amount?",
		TierOrder:       "Each amount cannot be below the previous one.",

		// money
		AskDepositAmount:  "How much to set aside? A number.",
		AskWithdrawAmount: "How much to take back? A number.",
		NoGoal:            "Set up a goal first.",
		ChooseGoal:        "Pick a goal:",
		AmountCancelled:   "Okay, cancelled.",
		Overdraw:          "You cannot take back more than is saved.",

		// status
		GoalDone:            "Goal reached!",
		StatusSavedLabel:    "Saved",
		StatusSavedOf:       "of",
		StatusLeftLabel:     "Left",
		StatusPerDayLabel:   "Per day",
		StatusPerMonthLabel: "Per month",
		StatusDeadlineLabel: "Deadline",
		StatusPaceLabel:     "Pace",
		StatusDaysLeft:      "%d days left",
		StatusTiers:         "minimal/acceptable/optimal",

		// admin
		SendUsage:           "Format: send <scenario> <id|all>",
		SendUnknownScenario: "No such scenario.",
		SendQueued:          "Sent.",
		BroadcastUsage:      "Format: broadcast <text>",
	},
	domain.LangRofl: {
		// buttons
		BtnAdd:      "➕ В Европу🚀🚀🚀",
		BtnWithdraw: "➖ НА СВО🥀🥀🥀",
		BtnStatus:   "📊 Стата",
		BtnOther:    "⚙️ штуки",
		BtnTierMin:  "ЖИЗНЬ В НИЩИТЕ",
		BtnTierOK:   "норм",
		BtnTierMax:  "АХУЕННО ШИКУЕМ",

		// common
		Greeting:     "Ээээ, вообщем, ЙО. Этот бот разработан special for u, чтобы мотивировать тя откладывать бабло ии удобно отслеживать прогресс, шоб по кайфу. Пока в бете, обновы буду выкатывать по настроению",
		Fallback:     "ты чет написал чет что я не продумал. в последствии это будет переслано мне в личку, интернет не анонимен",
		ScenarioDone: "готово",
		BadAnswer:    "ты ты че еблан у тебя че спросили а ты че пишешь?",

		// privacy
		Privacy:     "Я храню ваще всё: ID, имя, цели и бабло. Третьим лицам ничего не солью. Потом сделаю на сайте страничку с privacy policy. Стереть все данные — напиши «data remove all».",
		DataRemoved: "ВСЁ СТЁР. Удален из базы.",

		// onboarding
		OnboardingIntro: "погнали заведем цель. спрошу название, срок и три суммы: минимум, норм и шик",
		BtnStart:        "ГАЗ",
		AskTitle:        "как назовем?",
		AskDeadline:     "када нада? дату пиши типа 2027-06-25 это год-месяц-день если ты балбес",
		AskTargetMin:    "минимум скока?",
		AskTargetOK:     "норм скока?",
		AskTargetMax:    "шик скока?",
		TierOrder:       "сумма не может быть меньше предыдущей, ёпта",

		// money
		AskDepositAmount:  "скока закинем? циферкой",
		AskWithdrawAmount: "скока снимем? циферкой",
		NoGoal:            "цели-то нет. заведи сначала /start если бота сломал",
		ChooseGoal:        "выбирай цель:",
		AmountCancelled:   "чел...🥀🥀🥀",
		Overdraw:          "больше чем накопил не снять 🥀",

		// status
		GoalDone:            "цель закрыта, красава 🎉",
		StatusSavedLabel:    "накопил",
		StatusSavedOf:       "из",
		StatusLeftLabel:     "осталось",
		StatusPerDayLabel:   "в день",
		StatusPerMonthLabel: "в месяц",
		StatusDeadlineLabel: "дедлайн",
		StatusPaceLabel:     "отсталость от графика",
		StatusDaysLeft:      "осталось %d дн.",
		StatusTiers:         "минимум/норм/шик",

		// admin
		SendUsage:           "формат: send <сценарий> <id|all>",
		SendUnknownScenario: "нет такого",
		SendQueued:          "разослал",
		BroadcastUsage:      "формат: broadcast <текст>",
	},
}

// Get returns the catalog for lang, falling back to Default when the language is
// unknown.
func Get(lang domain.Lang) Messages {
	if m, ok := catalog[lang]; ok {
		return m
	}
	return catalog[Default]
}
