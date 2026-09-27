package i18n

import (
	"reflect"
	"testing"
)

type testStrings struct {
	Save   string
	Cancel string
	Count  func(n int) string
	Dialog struct {
		Title string
	}
}

func testCatalog() *Catalog[testStrings] {
	en := testStrings{Save: "Save", Cancel: "Cancel", Count: func(n int) string { return "en" }}
	en.Dialog.Title = "Title"
	ru := testStrings{Save: "Сохранить"}
	ru.Dialog.Title = "Заголовок"
	zh := testStrings{Save: "保存", Cancel: "取消", Count: func(n int) string { return "zh" }}
	zh.Dialog.Title = "标题"
	zhTW := testStrings{Save: "儲存"}
	return NewCatalog(en, map[string]testStrings{"ru": ru, "zh": zh, "zh_TW": zhTW})
}

func TestCatalogGet(t *testing.T) {
	c := testCatalog()
	cases := []struct{ lang, save string }{
		{"ru", "Сохранить"},
		{"ru-RU", "Сохранить"},
		{"zh-Hans-CN", "保存"},
		{"zh-TW", "儲存"},
		{"zh-Hant", "儲存"},
		{"de", "Save"},
		{"", "Save"},
	}
	for _, tc := range cases {
		if got := c.Get(tc.lang).Save; got != tc.save {
			t.Errorf("Get(%q).Save = %q, want %q", tc.lang, got, tc.save)
		}
	}
}

func TestCatalogFillsMissing(t *testing.T) {
	c := testCatalog()
	ru := c.Get("ru")
	if ru.Cancel != "Cancel" || ru.Count == nil || ru.Count(1) != "en" {
		t.Errorf("missing Russian strings aren't taken from the base: %+v", ru)
	}
	if ru.Dialog.Title != "Заголовок" {
		t.Errorf("nested translated string lost: %q", ru.Dialog.Title)
	}
	// A regional translation falls back to its base language's one
	zhTW := c.Get("zh-TW")
	if zhTW.Cancel != "取消" || zhTW.Dialog.Title != "标题" || zhTW.Count(1) != "zh" {
		t.Errorf("zh-TW doesn't fall back to zh: %+v", zhTW)
	}

	// Missing are the strings left in English
	want := map[string][]string{
		"ru": {"Cancel", "Count"},
	}
	if got := c.Missing(); !reflect.DeepEqual(got, want) {
		t.Errorf("Missing() = %v, want %v", got, want)
	}
}

func TestNormalize(t *testing.T) {
	for tag, want := range map[string]string{
		"ru":          "ru",
		"ru_RU.UTF-8": "ru-RU",
		"en-us":       "en-US",
		"zh-Hans-CN":  "zh-CN",
		"zh-Hans":     "zh",
		"zh-Hant":     "zh-TW",
		"zh-Hant-HK":  "zh-HK",
		"sr_RS@latin": "sr-RS",
		"":            "",
	} {
		if got := Normalize(tag); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestPlural(t *testing.T) {
	ru := []string{"файл", "файла", "файлов"}
	for n, want := range map[int]string{0: "файлов", 1: "файл", 2: "файла", 4: "файла", 5: "файлов", 11: "файлов", 12: "файлов", 21: "файл", 22: "файла", 111: "файлов", 1001: "файл"} {
		if got := Plural("ru", n, ru...); got != want {
			t.Errorf("ru %d: %q, want %q", n, got, want)
		}
	}
	for n, want := range map[int]string{0: "files", 1: "file", 2: "files"} {
		if got := Plural("en-US", n, "file", "files"); got != want {
			t.Errorf("en %d: %q, want %q", n, got, want)
		}
	}
	if got := Plural("zh", 5, "个文件"); got != "个文件" {
		t.Errorf("zh: %q", got)
	}
	if got := Plural("ru", 5, "файл", "файла"); got != "файла" {
		t.Errorf("missing form: %q", got)
	}
}

// A translation equal to the base string ("OK" is "OK" in Russian too) is
// translated, not missing
func TestSameAsBaseIsNotMissing(t *testing.T) {
	type okStrings struct{ OK string }
	c := NewCatalog(okStrings{OK: "OK"}, map[string]okStrings{"ru": {OK: "OK"}})
	if missing := c.Missing(); len(missing) != 0 {
		t.Errorf("Missing() = %v", missing)
	}
}
