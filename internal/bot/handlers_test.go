package bot

import (
	"testing"

	"lab042.ru/doomsdaycalc/internal/domain"
	"lab042.ru/doomsdaycalc/internal/i18n"
)

// TestResolveIntent checks typed commands and localized button labels. The same
// physical button reads differently per language, so every catalog is covered.
func TestResolveIntent(t *testing.T) {
	ru := i18n.Get(domain.LangRU)
	en := i18n.Get(domain.LangEN)
	rofl := i18n.Get(domain.LangRofl)

	cases := []struct {
		name string
		m    i18n.Messages
		text string
		want intent
	}{
		{"slash start", ru, "/start", intentStart},
		{"ru add button", ru, ru.BtnAdd, intentAdd},
		{"ru withdraw button", ru, ru.BtnWithdraw, intentWithdraw},
		{"ru status button", ru, ru.BtnStatus, intentStatus},
		{"en status button", en, en.BtnStatus, intentStatus},
		{"rofl status button", rofl, rofl.BtnStatus, intentStatus},
		{"typed add", ru, "add", intentAdd},
		{"typed withdraw", ru, "withdraw", intentWithdraw},
		{"typed status", en, "status", intentStatus},
		{"typed privacy", ru, "privacy", intentPrivacy},
		{"typed data remove all", ru, "data remove all", intentDataRemoveAll},
		{"typed broadcast", ru, "broadcast", intentBroadcast},
		{"broadcast with text", ru, "broadcast hi there", intentBroadcast},
		{"typed send alone", ru, "send", intentSend},
		{"send with args", ru, "send add_goal 123", intentSend},
		{"send trimmed and uppercase", ru, "  SEND onboarding all  ", intentSend},
		{"sender is not send", ru, "sender", intentNone},
		{"send with no space", ru, "sendoff", intentNone},
		{"uppercase status", ru, "STATUS", intentStatus},
		{"whitespace trimmed", ru, "  status  ", intentStatus},
		{"foreign label does not match", en, ru.BtnStatus, intentNone},
		{"unknown text", ru, "hello", intentNone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := resolveIntent(c.m, c.text); got != c.want {
				t.Fatalf("resolveIntent(%q) = %v, want %v", c.text, got, c.want)
			}
		})
	}
}
