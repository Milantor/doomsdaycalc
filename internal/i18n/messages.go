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

	// Motivation: the reply after a finished money move. One pool per mood, picked at
	// random, so the same amount does not read the same twice. A deposit mood comes from
	// fixed amounts, a withdrawal mood from the share of the saved total.
	MoodDepositSmall  []string // deposit: under the mid amount
	MoodDepositMid    []string // deposit: between the two amounts
	MoodDepositBig    []string // deposit: at or over the big amount
	MoodWithdrawSmall []string // withdraw: small share of the savings
	MoodWithdrawMid   []string // withdraw: noticeable share of the savings
	MoodWithdrawBig   []string // withdraw: large share of the savings

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

		// motivation
		MoodDepositSmall: []string{
			"начало положено",
			"копейка рубль бережёт",
			"немного, но счёт идёт",
		},
		MoodDepositMid: []string{
			"хороший взнос, темп держишь",
			"вот это уже заметно",
			"так и до цели недалеко",
		},
		MoodDepositBig: []string{
			"ого, вот это рывок. уважение",
			"такими темпами цель закроется раньше срока",
			"мощно. так и надо",
		},
		MoodWithdrawSmall: []string{
			"ладно, бывает",
			"небольшая прореха, не страшно",
			"вернём на место",
		},
		MoodWithdrawMid: []string{
			"а вот это уже зря",
			"так до цели не дойти",
			"сбавь обороты, а то не накопишь",
		},
		MoodWithdrawBig: []string{
			"так ты точно не накопишь",
			"это большой откат назад",
			"ты почти обнулил всё, к чему шёл",
		},

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

		// motivation
		MoodDepositSmall: []string{
			"a start is a start",
			"every coin counts",
			"small, but it counts",
		},
		MoodDepositMid: []string{
			"nice one, keep it up",
			"now thats more like it",
			"solid, respect",
		},
		MoodDepositBig: []string{
			"whoa, what a push. respect",
			"at this rate the goal lands early",
			"huge. keep going like this",
		},
		MoodWithdrawSmall: []string{
			"fine, it happens",
			"small dent, nothing fatal",
			"we will put it back",
		},
		MoodWithdrawMid: []string{
			"thats a shame",
			"this way the goal stays out of reach",
			"slow down or you will never get there",
		},
		MoodWithdrawBig: []string{
			"you will never save anything like this",
			"thats a big step back",
			"you just wiped most of your progress",
		},

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

		// motivation
		MoodDepositSmall: []string{
			"ну хоть что-то",
			"нищета detected, но похвально",
			"ок жи есть",
		},
		MoodDepositMid: []string{
			"о, уже похоже на человека",
			"норм бабки занёс респект уважене",
			"живём нах",
		},
		MoodDepositBig: []string{
			"ого нихуя ты крутой ебать чел хорош ахуителен ваще легенда",
			"ты че банк ограбил?? вызываю мусоров но уважаю",
			"такими темпами ты купишь себе планету или хватит на первый платеж по ипотеке в мск",
		},
		MoodWithdrawSmall: []string{
			"ладно, хуй с ним",
			"по мелочи, живём",
			"42 брат",
		},
		MoodWithdrawMid: []string{
			"а вот это зря, чел",
			"ой ой ой, куда собрался",
			"так ты не накопишь даже на пиво",
		},
		MoodWithdrawBig: []string{
			"ТЫ УМРЕШЬ В НИЩИТЕ.",
			"поздравляю, ты официально бомж",
			"ну ты и лох, блять. пиздос",
		},

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

// MoodPhrases: the phrase pool for one mood, nil for MoodNone. Picking one out of the
// pool is up to the caller, so the random source stays out of the catalog.
func (m Messages) MoodPhrases(mood domain.Mood) []string {
	switch mood {
	case domain.MoodDepositSmall:
		return m.MoodDepositSmall
	case domain.MoodDepositMid:
		return m.MoodDepositMid
	case domain.MoodDepositBig:
		return m.MoodDepositBig
	case domain.MoodWithdrawSmall:
		return m.MoodWithdrawSmall
	case domain.MoodWithdrawMid:
		return m.MoodWithdrawMid
	case domain.MoodWithdrawBig:
		return m.MoodWithdrawBig
	default:
		return nil
	}
}
