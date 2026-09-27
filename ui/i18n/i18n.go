// Package i18n holds the translations of an application as Go code: the
// strings are the fields of a struct, and each language is a value of it.
//
//	type Strings struct {
//		SaveAs     string
//		ItemsFound func(n int) string
//	}
//
//	var en = Strings{SaveAs: "Save As...", ItemsFound: func(n int) string {...}}
//	var ru = Strings{SaveAs: "Сохранить как...", ...}
//
//	var catalog = i18n.NewCatalog(en, map[string]Strings{"ru": ru})
//
//	func T() *Strings { return catalog.Get(ui.Language()) }
//
// A misspelled string is a compile error, the translations are built into
// the executable, and a string missing in a language falls back to the base
// one (see Catalog.Missing to list them in a test).
package i18n

import (
	"reflect"
	"sort"
	"strings"
)

// Catalog holds a struct of strings in the base language and its translations.
type Catalog[T any] struct {
	base         T
	translations map[string]*T
	missing      map[string][]string
}

// NewCatalog makes a catalog from the strings in the base language and the
// translations by language tag ("ru", "zh", "zh-TW"...). Fields a translation
// leaves empty (an empty string, a nil func) are taken from the translation
// of its base language ("zh" for "zh-TW"), if there is one, else from base.
func NewCatalog[T any](base T, translations map[string]T) *Catalog[T] {
	c := &Catalog[T]{
		base:         base,
		translations: make(map[string]*T, len(translations)),
		missing:      make(map[string][]string),
	}

	normalized := make(map[string]T, len(translations))
	for lang, t := range translations {
		normalized[Normalize(lang)] = t
	}
	// Base languages first: the regional translations fall back to them
	langs := make([]string, 0, len(normalized))
	for lang := range normalized {
		langs = append(langs, lang)
	}
	sort.Slice(langs, func(i, j int) bool {
		iRegional, jRegional := strings.Contains(langs[i], "-"), strings.Contains(langs[j], "-")
		if iRegional != jRegional {
			return !iRegional
		}
		return langs[i] < langs[j]
	})

	for _, lang := range langs {
		filled := normalized[lang]
		missing := emptyFields(reflect.ValueOf(filled), "")
		fallback := &c.base
		if parent, ok := c.translations[Base(lang)]; ok && Base(lang) != lang {
			fallback = parent
			// Missing only if the base language translation lacks it too
			missing = intersect(missing, c.missing[Base(lang)])
		}
		fillFromBase(reflect.ValueOf(&filled).Elem(), reflect.ValueOf(*fallback))
		c.translations[lang] = &filled
		if len(missing) > 0 {
			c.missing[lang] = missing
		}
	}
	return c
}

// Get returns the strings for the language: the translation for the tag,
// else for its base language ("zh" for "zh-TW"), else the base strings.
func (c *Catalog[T]) Get(lang string) *T {
	lang = Normalize(lang)
	if t, ok := c.translations[lang]; ok {
		return t
	}
	if t, ok := c.translations[Base(lang)]; ok {
		return t
	}
	return &c.base
}

// Languages returns the tags of the translations, sorted.
func (c *Catalog[T]) Languages() []string {
	langs := make([]string, 0, len(c.translations))
	for lang := range c.translations {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

// Missing returns, by language, the fields left in the base language (not
// translated there nor in the base language translation), e.g. for a test
// that all the strings are translated.
func (c *Catalog[T]) Missing() map[string][]string {
	return c.missing
}

// fillFromBase copies the empty fields of v from base, recursing into
// nested structs.
func fillFromBase(v, base reflect.Value) {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}
		if field.Kind() == reflect.Struct {
			fillFromBase(field, base.Field(i))
			continue
		}
		if field.IsZero() {
			field.Set(base.Field(i))
		}
	}
}

// emptyFields returns the names of the empty (zero) exported fields of v,
// nested ones as "Outer.Inner".
func emptyFields(v reflect.Value, prefix string) []string {
	var empty []string
	for i := 0; i < v.NumField(); i++ {
		if !v.Type().Field(i).IsExported() {
			continue
		}
		field := v.Field(i)
		name := prefix + v.Type().Field(i).Name
		if field.Kind() == reflect.Struct {
			empty = append(empty, emptyFields(field, name+".")...)
		} else if field.IsZero() {
			empty = append(empty, name)
		}
	}
	return empty
}

func intersect(a, b []string) []string {
	var both []string
	for _, x := range a {
		for _, y := range b {
			if x == y {
				both = append(both, x)
				break
			}
		}
	}
	return both
}

// Normalize brings a language tag to the form "ll" or "ll-RR": "ru_RU.UTF-8"
// becomes "ru-RU", "zh-Hans-CN" becomes "zh-CN", "zh-Hant" becomes "zh-TW".
func Normalize(tag string) string {
	tag, _, _ = strings.Cut(tag, ".") // encoding: en_US.UTF-8
	tag, _, _ = strings.Cut(tag, "@") // modifier: sr_RS@latin
	parts := strings.FieldsFunc(tag, func(r rune) bool { return r == '-' || r == '_' })
	if len(parts) == 0 {
		return ""
	}

	lang := strings.ToLower(parts[0])
	region, script := "", ""
	for _, part := range parts[1:] {
		switch {
		case len(part) == 4:
			script = strings.ToLower(part)
		case len(part) == 2 || len(part) == 3:
			region = strings.ToUpper(part)
		}
	}
	// Chinese is told apart by script: Traditional is Taiwan's unless the
	// region says otherwise
	if lang == "zh" && region == "" && script == "hant" {
		region = "TW"
	}
	if region == "" {
		return lang
	}
	return lang + "-" + region
}

// Base returns the language of the tag without the region: "zh" for "zh-TW".
func Base(tag string) string {
	base, _, _ := strings.Cut(Normalize(tag), "-")
	return base
}

// Plural returns the form of a word for the number n in the language:
//
//	i18n.Plural("en", n, "file", "files")
//	i18n.Plural("ru", n, "файл", "файла", "файлов")
//
// Russian, Ukrainian and Belarusian take three forms (1, 2-4, 5+), Chinese,
// Japanese and Korean one, other languages two (1, the rest). A missing form
// is replaced by the last one given.
func Plural(lang string, n int, forms ...string) string {
	if len(forms) == 0 {
		return ""
	}
	if n < 0 {
		n = -n
	}
	index := 1
	switch Base(lang) {
	case "ru", "uk", "be":
		switch {
		case n%10 == 1 && n%100 != 11:
			index = 0
		case n%10 >= 2 && n%10 <= 4 && (n%100 < 12 || n%100 > 14):
			index = 1
		default:
			index = 2
		}
	case "zh", "ja", "ko":
		index = 0
	default:
		if n == 1 {
			index = 0
		}
	}
	return forms[min(index, len(forms)-1)]
}
