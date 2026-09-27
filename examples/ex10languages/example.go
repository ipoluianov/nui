package ex10languages

import "github.com/ipoluianov/nui/ui"

// NewExampleForm shows text in different languages. Latin, Cyrillic, Greek
// and Vietnamese come from the embedded Noto Sans; Chinese, Japanese and
// Korean from a system font (see ui.FontFamilyCJK), looked up the first
// time such text is shown.
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("Languages")
	form.SetSize(960, 900)

	panel := form.Panel()
	row := 0

	addHeader(panel, &row, "Supported")
	for _, s := range supported {
		addSample(panel, &row, s)
	}

	addHeader(panel, &row, "Controls")
	textBox := ui.NewTextBox()
	textBox.SetText("Hello Привет Γειά 你好 こんにちは 안녕하세요")
	panel.AddWidget(row, 0, ui.NewLabel("Text box"))
	panel.AddWidget(row, 1, textBox)
	row++

	combo := ui.NewComboBox()
	for _, s := range supported {
		combo.AddItem(s.name, nil)
	}
	combo.SetSelectedIndex(9)
	panel.AddWidget(row, 0, ui.NewLabel("Combo box"))
	panel.AddWidget(row, 1, combo)
	row++

	buttons := ui.NewPanel()
	buttons.AddButton(0, 0, "OK", nil)
	buttons.AddButton(0, 1, "Отмена", nil)
	buttons.AddButton(0, 2, "确定", nil)
	buttons.AddButton(0, 3, "キャンセル", nil)
	buttons.AddButton(0, 4, "확인", nil)
	buttons.AddHSpacer(0, 5)
	panel.AddWidget(row, 0, ui.NewLabel("Buttons"))
	panel.AddWidget(row, 1, buttons)
	row++

	panel.AddWidget(row, 0, ui.NewLabel("Check box"))
	panel.AddWidget(row, 1, ui.NewCheckbox("Запомнить / 记住我 / 覚えておく"))
	row++

	// These need what the text renderer doesn't do yet: right-to-left
	// order (Arabic, Hebrew), joining and reordering the glyphs (Arabic,
	// Devanagari, Thai) and color glyphs (emoji); and a font, which isn't
	// looked up for them
	addHeader(panel, &row, "Not supported yet")
	for _, s := range unsupported {
		addSample(panel, &row, s)
	}

	panel.AddVSpacer(row, 0)
	return form
}

type sample struct {
	name string
	text string
}

var supported = []sample{
	{"English", "The quick brown fox jumps over the lazy dog"},
	{"Русский", "Съешь же ещё этих мягких французских булок"},
	{"Українська", "Чуєш їх, доцю, га? Кумедна ж ти, прощайся без ґольфів!"},
	{"Ελληνικά", "Ξεσκεπάζω την ψυχοφθόρα βδελυγμία"},
	{"Deutsch", "Falsches Üben von Xylophonmusik quält jeden größeren Zwerg"},
	{"Français", "Portez ce vieux whisky au juge blond qui fume"},
	{"Polski", "Zażółć gęślą jaźń"},
	{"Türkçe", "Pijamalı hasta yağız şoföre çabucak güvendi"},
	{"Tiếng Việt", "Trăm năm trong cõi người ta, chữ tài chữ mệnh khéo là ghét nhau"},
	{"简体中文", "我能吞下玻璃而不伤身体。"},
	{"繁體中文", "我能吞下玻璃而不傷身體。"},
	{"日本語", "私はガラスを食べられます。それは私を傷つけません。"},
	{"한국어", "나는 유리를 먹을 수 있어요. 그래도 아프지 않아요"},
}

// Named in English: the names in these scripts wouldn't show either
var unsupported = []sample{
	{"Arabic", "أنا قادر على أكل الزجاج و هذا لا يؤلمني"},
	{"Hebrew", "אני יכול לאכול זכוכית וזה לא מזיק לי"},
	{"Hindi", "मैं काँच खा सकता हूँ और मुझे उससे कोई चोट नहीं पहुंचती"},
	{"Thai", "ฉันกินกระจกได้ แต่มันไม่ทำให้ฉันเจ็บ"},
	{"Emoji", "😀 👍 🎉"},
}

func addHeader(panel *ui.Panel, row *int, text string) {
	header := panel.AddLabel(*row, 0, text)
	header.SetUnderline(true)
	*row++
}

func addSample(panel *ui.Panel, row *int, s sample) {
	panel.AddLabel(*row, 0, s.name)
	panel.AddLabel(*row, 1, s.text)
	*row++
}
