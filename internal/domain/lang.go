package domain

// Lang: a message language. Explicit per-user override: empty means "derive from
// the Telegram profile". Telegram suggests "ru"/"en"/"sr"; "rofl" exists only as a
// manual operator override.
type Lang string

const (
	LangRU   Lang = "ru"
	LangEN   Lang = "en"
	LangRofl Lang = "rofl"
	LangSR   Lang = "sr"
)
