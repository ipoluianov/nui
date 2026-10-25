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

	// The print dialog (Form.Print on Linux and macOS)
	Print, Printer, Copies, Pages, PagesHint, Paper string
	Portrait, Landscape, SaveAsPDF                  string

	// nui's file dialog (UseBuiltinFileDialogs). FileNotFound and
	// FileExistsReplace have a %s for the name of the file.
	Open, Save, SelectFolder, Home, Up               string
	FileName, FileSize, FileModified, FileType       string
	ShowHiddenFiles, FileNotFound, FileExistsReplace string
}

var (
	uiStringsMu     sync.RWMutex
	uiStringsByLang = map[string]UIStrings{
		"ru": {OK: "OK", Cancel: "Отмена", Yes: "Да", No: "Нет",
			Today: "Сегодня",
			MonthNames: [12]string{"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
				"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"},
			WeekdaysShort: [7]string{"Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"},
			Print:         "Печать", Printer: "Принтер", Copies: "Копии", Pages: "Страницы",
			PagesHint: "Все, или например 1-3, 5", Paper: "Бумага",
			Portrait: "Книжная", Landscape: "Альбомная", SaveAsPDF: "Сохранить в PDF",
			Open: "Открыть", Save: "Сохранить", SelectFolder: "Выбрать папку", Home: "Домой", Up: "Вверх",
			FileName: "Имя", FileSize: "Размер", FileModified: "Изменён", FileType: "Тип файлов",
			ShowHiddenFiles: "Показывать скрытые файлы", FileNotFound: "Файл «%s» не найден.",
			FileExistsReplace: "Файл «%s» уже существует. Заменить его?"},
		"zh": {OK: "确定", Cancel: "取消", Yes: "是", No: "否",
			Today: "今天",
			MonthNames: [12]string{"一月", "二月", "三月", "四月", "五月", "六月",
				"七月", "八月", "九月", "十月", "十一月", "十二月"},
			WeekdaysShort: [7]string{"日", "一", "二", "三", "四", "五", "六"},
			Print:         "打印", Printer: "打印机", Copies: "份数", Pages: "页面",
			PagesHint: "全部，或例如 1-3, 5", Paper: "纸张",
			Portrait: "纵向", Landscape: "横向", SaveAsPDF: "另存为 PDF",
			Open: "打开", Save: "保存", SelectFolder: "选择文件夹", Home: "主目录", Up: "上一级",
			FileName: "名称", FileSize: "大小", FileModified: "修改时间", FileType: "文件类型",
			ShowHiddenFiles: "显示隐藏文件", FileNotFound: "找不到文件“%s”。",
			FileExistsReplace: "文件“%s”已存在。要替换它吗？"},
	}
	uiStringsEnglish = UIStrings{OK: "OK", Cancel: "Cancel", Yes: "Yes", No: "No",
		Today: "Today",
		MonthNames: [12]string{"January", "February", "March", "April", "May", "June",
			"July", "August", "September", "October", "November", "December"},
		WeekdaysShort: [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"},
		Print:         "Print", Printer: "Printer", Copies: "Copies", Pages: "Pages",
		PagesHint: "All, or e.g. 1-3, 5", Paper: "Paper",
		Portrait: "Portrait", Landscape: "Landscape", SaveAsPDF: "Save as PDF",
		Open: "Open", Save: "Save", SelectFolder: "Select Folder", Home: "Home", Up: "Up",
		FileName: "Name", FileSize: "Size", FileModified: "Modified", FileType: "Files of type",
		ShowHiddenFiles: "Show hidden files", FileNotFound: "The file \"%s\" is not found.",
		FileExistsReplace: "The file \"%s\" already exists. Replace it?"}
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
