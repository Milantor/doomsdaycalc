package i18n

import "lab042.ru/doomsdaycalc/internal/domain"

// Messages: the full set of user-facing strings for one language.
// A struct beats map[string]string: a typo in a field name is a compile error. Go
// does not require every field to be filled, so TestCatalogComplete checks each
// catalog entry for missing strings.
type Messages struct {
	BtnAdd      string // button: add money to savings
	BtnWithdraw string // button: withdraw money from savings
	BtnStatus   string // button: show progress
	BtnTierMin  string // inline button: minimal tier
	BtnTierOK   string // inline button: acceptable tier
	BtnTierMax  string // inline button: optimal tier

	Greeting     string // /start welcome
	Fallback     string // reply when nothing matched
	AddStub      string // reply to add button, until scenario flow lands
	WithdrawStub string // reply to withdraw button, until scenario flow lands
	StatusEmpty  string // /status with no goals yet
	Privacy      string // /privacy text
	DataRemoved  string // confirmation after data remove all
}

// catalog: every supported language mapped to its strings. "rofl" is a joke variant
// of Russian, reachable only through an explicit per-user override.
var catalog = map[domain.Lang]Messages{
	domain.LangRU: {
		BtnAdd:      "➕ Отложил",
		BtnWithdraw: "➖ Снял",
		BtnStatus:   "📊 Статус",
		BtnTierMin:  "Минимум",
		BtnTierOK:   "Приемлемо",
		BtnTierMax:  "Оптимально",

		Greeting:     "Привет! Я помогу откладывать к дедлайну.",
		Fallback:     "Не понял. Нажми кнопку ниже или напиши status.",
		AddStub:      "Запись взносов недоступна.",
		WithdrawStub: "Снятие средств недоступно.",
		StatusEmpty:  "Целей пока нет.",
		Privacy:      "Я храню твой Telegram ID, имя, а также цели и взносы. Третьим лицам ничего не передаю. Чтобы удалить всё — напиши «data remove all».",
		DataRemoved:  "Готово. Все твои данные удалены.",
	},
	domain.LangEN: {
		BtnAdd:      "➕ Saved",
		BtnWithdraw: "➖ Withdrew",
		BtnStatus:   "📊 Status",
		BtnTierMin:  "Minimum",
		BtnTierOK:   "Acceptable",
		BtnTierMax:  "Optimum",

		Greeting:     "Hi! I will help you save towards a deadline.",
		Fallback:     "I did not get that. Tap a button below or type status.",
		AddStub:      "Recording deposits is not available.",
		WithdrawStub: "Withdrawals are not available.",
		StatusEmpty:  "No goals yet.",
		Privacy:      "I store your Telegram ID, your name, and your goals and deposits. I pass nothing to third parties. To erase everything, send «data remove all».",
		DataRemoved:  "Done. All your data has been deleted.",
	},
	domain.LangRofl: {
		BtnAdd:      "➕ В Европу🚀🚀🚀",
		BtnWithdraw: "➖ НА СВО🥀🥀🥀",
		BtnStatus:   "📊 Стата",
		BtnTierMin:  "ЖИЗНЬ В НИЩИТЕ",
		BtnTierOK:   "норм",
		BtnTierMax:  "АХУЕННО ШИКУЕМ",

		Greeting:     "Ээээ, вообщем, ЙО. Этот бот разработан special for u, чтобы мотивировать тя откладывать бабло ии удобно отслеживать прогресс, шоб по кайфу. Пока в бете, обновы буду выкатывать по настроению",
		Fallback:     "ты чет написал чет что я не продумал. в последствии это будет переслано мне в личку, интернет не анонимен",
		AddStub:      "не готово",
		WithdrawStub: "не готово",
		StatusEmpty:  "не готово",
		Privacy:      "Я храню ваще всё: ID, имя, цели и бабло. Третьим лицам ничего не солью. Потом сделаю на сайте страничку с privacy policy. Стереть все данные — напиши «data remove all».",
		DataRemoved:  "ВСЁ СТЁР. Удален из базы.",
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
