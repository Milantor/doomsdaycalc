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
// forgotten translation before it becomes a blank button at runtime. A slice field is a
// phrase pool, so it needs at least one non-empty phrase. Also checks that Default is
// present.
func TestCatalogComplete(t *testing.T) {
	if _, ok := catalog[Default]; !ok {
		t.Fatalf("catalog has no entry for Default (%q)", Default)
	}
	for lang, m := range catalog {
		v := reflect.ValueOf(m)
		typ := v.Type()
		for i := 0; i < typ.NumField(); i++ {
			name := typ.Field(i).Name
			f := v.Field(i)
			switch f.Kind() {
			case reflect.String:
				if f.String() == "" {
					t.Errorf("language %q: field %s is empty", lang, name)
				}
			case reflect.Slice:
				if f.Len() == 0 {
					t.Errorf("language %q: pool %s is empty", lang, name)
					continue
				}
				for j := 0; j < f.Len(); j++ {
					if f.Index(j).String() == "" {
						t.Errorf("language %q: pool %s has an empty phrase at %d", lang, name, j)
					}
				}
			}
		}
	}
}

// TestMoodPools: every mood that carries phrases has a pool in the default catalog, so a
// mood added to domain without phrases shows up here. MoodNone carries no phrases.
func TestMoodPools(t *testing.T) {
	for _, mood := range domain.Moods {
		if pool := catalog[Default].MoodPhrases(mood); len(pool) == 0 {
			t.Errorf("no phrases for mood %s", mood)
		}
	}
	if pool := catalog[Default].MoodPhrases(domain.MoodNone); pool != nil {
		t.Errorf("MoodNone has a pool of %d phrases", len(pool))
	}
}

func TestGetFallback(t *testing.T) {
	// Messages holds phrase pools, so it can no longer be compared with ==.
	if got := Get("klingon"); !reflect.DeepEqual(got, catalog[Default]) {
		t.Fatalf("Get(unknown) = %+v, want Default", got)
	}
}
