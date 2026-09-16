// Package i18n: every user-facing string, grouped by language. Handlers and
// keyboards read text from here, so adding a language is one new catalog entry.
// User language is resolved in one place: an explicit override (set by the operator,
// e.g. "rofl" for a friend) wins, otherwise the Telegram language_code hint,
// otherwise Default.
package i18n

import (
	"strings"

	"lab042.ru/doomsdaycalc/internal/domain"
)

// Default is used when the user language is unknown or unsupported.
const Default = domain.LangRU

// ParseLang maps a Telegram/BCP-47 tag ("ru", "en-US", "ru_RU") to a supported
// language. Only the primary subtag matters; anything unrecognized falls back to
// Default.
func ParseLang(code string) domain.Lang {
	base := strings.ToLower(code)
	if i := strings.IndexAny(base, "-_"); i >= 0 {
		base = base[:i]
	}
	switch base {
	case "ru":
		return domain.LangRU
	case "en":
		return domain.LangEN
	default:
		return Default
	}
}

// Resolve picks the message language for a user: an explicit override wins,
// otherwise the Telegram hint, otherwise Default. Pure, so it is easy to test.
func Resolve(override domain.Lang, tgCode string) domain.Lang {
	if override != "" {
		return override
	}
	return ParseLang(tgCode)
}
