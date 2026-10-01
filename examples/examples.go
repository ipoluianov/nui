package examples

import (
	"github.com/ipoluianov/nui/examples/ex00gallery"
	"github.com/ipoluianov/nui/examples/ex01base"
	"github.com/ipoluianov/nui/examples/ex02messagebox"
	"github.com/ipoluianov/nui/examples/ex04dialog"
	"github.com/ipoluianov/nui/examples/ex05chart"
	"github.com/ipoluianov/nui/examples/ex06timechart"
	"github.com/ipoluianov/nui/examples/ex07tooltip"
	"github.com/ipoluianov/nui/examples/ex08contextmenu"
	"github.com/ipoluianov/nui/examples/ex09custompopup"
	"github.com/ipoluianov/nui/examples/ex10languages"
	"github.com/ipoluianov/nui/examples/ex11i18n"
	"github.com/ipoluianov/nui/examples/ex12menubar"
	"github.com/ipoluianov/nui/examples/ex13treeview"
	"github.com/ipoluianov/nui/examples/ex14colorpicker"
	"github.com/ipoluianov/nui/examples/ex15controls"
	"github.com/ipoluianov/nui/examples/ex16more"
	"github.com/ipoluianov/nui/ui"
)

func Run() {
	{
		form := ui.NewForm()
		form.SetTitle("Examples")
		form.SetSize(1000, 800)

		addButton := func(text string, newFormFunc func() *ui.Form) {
			btn := ui.NewButton(text)
			btn.SetOnClick(func() {
				newForm := newFormFunc()
				newForm.ShowModal(form)
			})
			form.Panel().AddWidget(form.Panel().NextGridRow(), 0, btn)
		}

		addButton("Example 00 - Gallery", ex00gallery.NewExampleForm)
		addButton("Example 01 - Base Form", ex01base.NewExampleForm)
		addButton("Example 02 - MessageBox", ex02messagebox.NewExampleForm)
		addButton("Example 04 - Dialog", ex04dialog.NewExampleForm)
		addButton("Example 05 - Chart", ex05chart.NewExampleForm)
		addButton("Example 06 - TimeChart", ex06timechart.NewExampleForm)
		addButton("Example 07 - Tooltips", ex07tooltip.NewExampleForm)
		addButton("Example 08 - Context Menus", ex08contextmenu.NewExampleForm)
		addButton("Example 09 - Custom Popup", ex09custompopup.NewExampleForm)
		addButton("Example 10 - Languages", ex10languages.NewExampleForm)
		addButton("Example 11 - Translations", ex11i18n.NewExampleForm)
		addButton("Example 12 - Main Menu", ex12menubar.NewExampleForm)
		addButton("Example 13 - TreeView", ex13treeview.NewExampleForm)
		addButton("Example 14 - Color Picker", ex14colorpicker.NewExampleForm)
		addButton("Example 15 - Controls", ex15controls.NewExampleForm)
		addButton("Example 16 - Expander, PropertyGrid, Drag & Drop, Toasts, Tray", ex16more.NewExampleForm)

		form.Panel().AddWidget(form.Panel().NextGridRow(), 0, ui.NewVSpacer())
		form.Panel().AddButton(form.Panel().NextGridRow(), 0, "Light Theme", func() {
			ui.ApplyLightTheme()
		})
		form.Panel().AddButton(form.Panel().NextGridRow(), 0, "Dark Theme", func() {
			ui.ApplyDarkTheme()
		})
		systemFont := ui.SystemUIFontName()
		form.Panel().AddButton(form.Panel().NextGridRow(), 0, "System Font ("+systemFont+")", func() {
			if err := ui.UseSystemFont(systemFont); err != nil {
				form.ShowToast(err.Error(), ui.ToastError)
			}
		})
		form.Panel().AddButton(form.Panel().NextGridRow(), 0, "Built-in Font (Noto Sans)", func() {
			ui.UseBuiltinFont()
		})
		form.Show()
		form.Exec()
	}
}
