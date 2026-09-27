package ex11i18n

import (
	"fmt"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/i18n"
)

// Strings are all the texts of the example. A misspelled field is a compile
// error; a field a language leaves empty falls back to English.
type Strings struct {
	Title       string
	Greeting    string
	NameHint    string
	Remember    string
	SaveTooltip string
	Save        string
	TabGeneral  string
	TabAbout    string
	About       string
	Theme       string
	ThemeLight  string
	ThemeDark   string
	ThemeSystem string

	// Column names of the table, updated in SetOnLanguageChanged
	ColumnName string
	ColumnSize string

	// Texts with parameters and numbers are funcs
	FilesSelected func(n int) string
	ConfirmTitle  string
	ConfirmDelete func(name string) string

	// Groups of texts are nested structs
	Menu MenuStrings
}

type MenuStrings struct {
	Copy  string
	Paste string
	More  string
	Help  string
}

var en = Strings{
	Title:       "Languages - live switching",
	Greeting:    "Hello! Switch the language with the buttons above.",
	NameHint:    "Your name",
	Remember:    "Remember me",
	SaveTooltip: "Saves the settings",
	Save:        "Save",
	TabGeneral:  "General",
	TabAbout:    "About",
	About:       "Texts are Go code: a struct per application, a value per language.",
	Theme:       "Theme",
	ThemeLight:  "Light",
	ThemeDark:   "Dark",
	ThemeSystem: "As in the system",
	ColumnName:  "Name",
	ColumnSize:  "Size",
	FilesSelected: func(n int) string {
		return fmt.Sprintf("%d %s selected", n, i18n.Plural("en", n, "file", "files"))
	},
	ConfirmTitle:  "Confirmation",
	ConfirmDelete: func(name string) string { return "Delete \"" + name + "\"?" },
	Menu:          MenuStrings{Copy: "Copy", Paste: "Paste", More: "More", Help: "Help"},
}

var ru = Strings{
	Title:       "Языки - переключение на лету",
	Greeting:    "Привет! Язык переключается кнопками выше.",
	NameHint:    "Ваше имя",
	Remember:    "Запомнить меня",
	SaveTooltip: "Сохраняет настройки",
	Save:        "Сохранить",
	TabGeneral:  "Общие",
	TabAbout:    "О программе",
	About:       "Тексты - это Go-код: структура на приложение, значение на язык.",
	Theme:       "Тема",
	ThemeLight:  "Светлая",
	ThemeDark:   "Тёмная",
	ThemeSystem: "Как в системе",
	ColumnName:  "Имя",
	ColumnSize:  "Размер",
	FilesSelected: func(n int) string {
		return fmt.Sprintf("%s %d %s",
			i18n.Plural("ru", n, "Выбран", "Выбрано", "Выбрано"), n,
			i18n.Plural("ru", n, "файл", "файла", "файлов"))
	},
	ConfirmTitle:  "Подтверждение",
	ConfirmDelete: func(name string) string { return "Удалить «" + name + "»?" },
	Menu:          MenuStrings{Copy: "Копировать", Paste: "Вставить", More: "Ещё", Help: "Справка"},
}

var zh = Strings{
	Title:       "语言 - 实时切换",
	Greeting:    "你好！使用上方的按钮切换语言。",
	NameHint:    "您的姓名",
	Remember:    "记住我",
	SaveTooltip: "保存设置",
	Save:        "保存",
	TabGeneral:  "常规",
	TabAbout:    "关于",
	About:       "文本就是 Go 代码：每个应用一个结构体，每种语言一个值。",
	Theme:       "主题",
	ThemeLight:  "浅色",
	ThemeDark:   "深色",
	ThemeSystem: "跟随系统",
	ColumnName:  "名称",
	ColumnSize:  "大小",
	FilesSelected: func(n int) string {
		return fmt.Sprintf("已选择 %d 个文件", n)
	},
	ConfirmTitle:  "确认",
	ConfirmDelete: func(name string) string { return "删除“" + name + "”？" },
	Menu:          MenuStrings{Copy: "复制", Paste: "粘贴", More: "更多", Help: "帮助"},
}

var catalog = i18n.NewCatalog(en, map[string]Strings{"ru": ru, "zh": zh})

// T returns the texts in the language of the application.
func T() *Strings {
	return catalog.Get(ui.Language())
}
