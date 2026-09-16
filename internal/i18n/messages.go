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
	},
	domain.LangEN: {
		BtnAdd:      "➕ Saved",
		BtnWithdraw: "➖ Withdrew",
		BtnStatus:   "📊 Status",
		BtnTierMin:  "Minimum",
		BtnTierOK:   "Acceptable",
		BtnTierMax:  "Optimum",
	},
	domain.LangRofl: {
		BtnAdd:      "➕ В Европу🚀🚀🚀",
		BtnWithdraw: "➖ НА СВО🥀🥀🥀",
		BtnStatus:   "📊 Стата",
		BtnTierMin:  "ЖИЗНЬ В НИЩИТЕ",
		BtnTierOK:   "норм",
		BtnTierMax:  "АХУЕННО ШИКУЕМ",
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
