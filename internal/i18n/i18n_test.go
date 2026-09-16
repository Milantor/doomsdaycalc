package i18n

import (
	"reflect"
	"testing"

	"lab042.ru/doomsdaycalc/internal/domain"
)

func TestParseLang(t *testing.T) {
	cases := []struct {
		name string
		code string
		want domain.Lang
	}{
		{"plain ru", "ru", domain.LangRU},
		{"ru with region", "ru-RU", domain.LangRU},
		{"uppercase ru", "RU", domain.LangRU},
		{"underscore variant", "ru_RU", domain.LangRU},
		{"plain en", "en", domain.LangEN},
		{"en with region", "en-US", domain.LangEN},
		{"empty falls back", "", Default},
		{"unsupported falls back", "de", Default},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ParseLang(c.code); got != c.want {
				t.Fatalf("ParseLang(%q) = %q, want %q", c.code, got, c.want)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name     string
		override domain.Lang
		tgCode   string
		want     domain.Lang
	}{
		{"override wins over telegram", domain.LangRofl, "en-US", domain.LangRofl},
		{"override wins over empty", domain.LangEN, "", domain.LangEN},
		{"no override uses telegram", "", "en", domain.LangEN},
		{"no override, unknown telegram", "", "de", Default},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Resolve(c.override, c.tgCode); got != c.want {
				t.Fatalf("Resolve(%q, %q) = %q, want %q", c.override, c.tgCode, got, c.want)
			}
		})
	}
}

// TestCatalogComplete fails when a catalog entry has an empty field, catching a
// forgotten translation before it becomes a blank button at runtime. Also checks
// that Default is present.
func TestCatalogComplete(t *testing.T) {
	if _, ok := catalog[Default]; !ok {
		t.Fatalf("catalog has no entry for Default (%q)", Default)
	}
	for lang, m := range catalog {
		v := reflect.ValueOf(m)
		typ := v.Type()
		for i := 0; i < typ.NumField(); i++ {
			if v.Field(i).String() == "" {
				t.Errorf("language %q: field %s is empty", lang, typ.Field(i).Name)
			}
		}
	}
}

func TestGetFallback(t *testing.T) {
	if got := Get("klingon"); got != catalog[Default] {
		t.Fatalf("Get(unknown) = %+v, want Default", got)
	}
}
