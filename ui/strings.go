package ui

import (
	"strings"
	"sync"
	"time"

	"github.com/ipoluianov/nui/ui/i18n"
)

// UIStrings are the texts of the library's own widgets, e.g. the buttons of
// message boxes. English, Russian and Chinese are built in; an application
// adds other languages with RegisterUIStrings.
type UIStrings struct {
	OK     string
	Cancel string
	Yes    string
	No     string

	// Calendar, DatePicker
	Today string
	// MonthNames in the nominative case: January...December
	MonthNames [12]string
	// WeekdaysShort are two- or three-letter names, Sunday first (as time.Weekday)
	WeekdaysShort [7]string
}

var (
	uiStringsMu     sync.RWMutex
	uiStringsByLang = map[string]UIStrings{
		"ru": {OK: "OK", Cancel: "Отмена", Yes: "Да", No: "Нет",
			Today: "Сегодня",
			MonthNames: [12]string{"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
				"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"},
			WeekdaysShort: [7]string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"}},
		"zh": {OK: "确定", Cancel: "取消", Yes: "是", No: "否",
			Today: "今天",
			MonthNames: [12]string{"一月", "二月", "三月", "四月", "五月", "六月",
				"七月", "八月", "九月", "十月", "十一月", "十二月"},
			WeekdaysShort: [7]string{"日", "一", "二", "三", "四", "五", "六"}},
	}
	uiStringsEnglish = UIStrings{OK: "OK", Cancel: "Cancel", Yes: "Yes", No: "No",
		Today: "Today",
		MonthNames: [12]string{"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December"},
		WeekdaysShort: [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}}
	uiStringsCatalog = i18n.NewCatalog(uiStringsEnglish, uiStringsByLang)
)

// UIText returns the library's texts in the language of the application.
func UIText() *UIStrings {
	uiStringsMu.RLock()
	defer uiStringsMu.RUnlock()
	return uiStringsCatalog.Get(Language())
}

// RegisterUIStrings adds or replaces the library's texts for a language.
func RegisterUIStrings(lang string, s UIStrings) {
	uiStringsMu.Lock()
	defer uiStringsMu.Unlock()
	uiStringsByLang[i18n.Normalize(lang)] = s
	uiStringsCatalog = i18n.NewCatalog(uiStringsEnglish, uiStringsByLang)
}

// FirstDayOfWeek returns the day weeks start with in the language of the
// application: Sunday in the US, Japan, Korea and a few other places,
// Monday elsewhere.
func FirstDayOfWeek() time.Weekday {
	switch lang := Language(); lang {
	case "en", "en-US", "en-CA", "ja", "ko", "he", "pt-BR", "zh-TW", "zh-HK", "hi", "th":
		return time.Sunday
	}
	return time.Monday
}

// DefaultDateLayout returns the time.Format layout of a date in the
// language of the application, e.g. "02.01.2006" for Russian.
func DefaultDateLayout() string {
	lang := Language()
	switch lang {
	case "en", "en-US":
		return "01/02/2006"
	case "en-GB", "en-AU", "en-NZ", "en-IE", "fr", "es", "it", "pt", "pt-BR", "el", "vi":
		return "02/01/2006"
	case "zh", "zh-TW", "zh-HK", "ja", "ko", "sv", "lt", "hu", "en-CA", "fr-CA":
		return "2006-01-02"
	case "nl":
		return "02-01-2006"
	}
	if strings.HasPrefix(lang, "en") {
		return "02/01/2006"
	}
	return "02.01.2006"
}
