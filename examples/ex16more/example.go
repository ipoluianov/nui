package ex16more

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// NewExampleForm shows Expander and Accordion, PropertyGrid, drag and drop
// (between widgets and files from the system), toasts, the file dialogs and
// the tray icon.
func NewExampleForm() *ui.Form {
	form := ui.NewForm()
	form.SetTitle("More controls")
	form.SetSize(820, 620)

	tabs := ui.NewTabWidget()
	form.Panel().AddWidget(0, 0, tabs)
	tabs.AddPage("Expander", expanderPage())
	tabs.AddPage("PropertyGrid", propertyGridPage(form))
	tabs.AddPage("Drag & Drop", dragDropPage(form))
	tabs.AddPage("Toasts & Tray", toastsPage(form))
	return form
}

func expanderPage() ui.Widgeter {
	page := ui.NewPanel()

	options := ui.NewExpander("Options")
	options.Content().AddWidget(0, 0, ui.NewCheckbox("Word wrap"))
	options.Content().AddWidget(1, 0, ui.NewCheckbox("Show line numbers"))
	page.AddWidget(0, 0, options)

	details := ui.NewExpander("Details (expanded)")
	details.SetExpanded(true)
	details.Content().AddLabel(0, 0, "Created")
	details.Content().AddLabel(0, 1, "2026-09-30")
	details.Content().AddLabel(1, 0, "Size")
	details.Content().AddLabel(1, 1, "12 KB")
	details.Content().AddHSpacer(0, 2)
	page.AddWidget(1, 0, details)

	page.AddLabel(2, 0, "Accordion: one section open at a time")
	acc := ui.NewAccordion()
	general := acc.AddSection("General")
	general.Content().AddLabel(0, 0, "Name")
	name := ui.NewTextBox()
	name.SetText("Server 1")
	general.Content().AddWidget(0, 1, name)
	network := acc.AddSection("Network")
	network.Content().AddLabel(0, 0, "Port")
	port := ui.NewNumBox()
	port.SetDecimals(0)
	port.SetValue(8080)
	network.Content().AddWidget(0, 1, port)
	notes := acc.AddSection("Notes (stretches)")
	text := ui.NewTextBox()
	text.SetMultiline(true)
	text.SetYExpandable(true)
	text.SetText("An expanded section with a stretching\nwidget takes the free space.")
	notes.Content().AddWidget(0, 0, text)
	// The accordion takes the free space; a section with a stretching
	// widget fills it
	acc.SetYExpandable(true)
	page.AddWidget(3, 0, acc)
	return page
}

// serverSettings is edited in the PropertyGrid through SetObject
type serverSettings struct {
	Name     string     `category:"General" desc:"The name shown in the list"`
	Mode     string     `category:"General" options:"Development, Staging, Production"`
	Enabled  bool       `category:"General"`
	Host     string     `category:"Network"`
	Port     int        `category:"Network" min:"1" max:"65535"`
	TLS      bool       `category:"Network" prop:"Use TLS"`
	Timeout  float64    `category:"Network" prop:"Timeout, s" decimals:"1" min:"0" max:"600"`
	Color    color.RGBA `category:"Look" prop:"Label color"`
	Created  time.Time  `category:"Look" readonly:"true"`
	password string
}

func propertyGridPage(form *ui.Form) ui.Widgeter {
	page := ui.NewPanel()
	settings := &serverSettings{
		Name: "Server 1", Mode: "Staging", Enabled: true,
		Host: "example.com", Port: 8443, TLS: true, Timeout: 30,
		Color: color.RGBA{0x1D, 0x6F, 0xB5, 255}, Created: time.Now(),
	}
	status := page.AddLabel(0, 0, "Edit a value: the struct changes")
	grid := ui.NewPropertyGrid()
	grid.SetObject(settings)
	grid.SetOnChanged(func(p *ui.Property) {
		status.SetText(fmt.Sprintf("%s = %v   (struct: %s:%d, TLS %v, %s)",
			p.Name(), p.Value(), settings.Host, settings.Port, settings.TLS, settings.Mode))
	})
	page.AddWidget(1, 0, grid)
	page.AddButton(2, 0, "Reset port to 443 (from code)", func() {
		settings.Port = 443
		grid.Refresh()
		form.ShowToast("Port reset to 443", ui.ToastInfo)
	})
	return page
}

func dragDropPage(form *ui.Form) ui.Widgeter {
	page := ui.NewPanel()
	page.AddLabel(0, 0, "Drag the fruits into the basket, or files from the file manager onto the box below")

	columns := page.AddPanel(1, 0)
	fruits := ui.NewGroupBox("Fruits")
	columns.AddWidget(0, 0, fruits)
	for i, name := range []string{"Apple", "Banana", "Cherry", "Grape"} {
		lbl := fruits.AddLabel(i, 0, "  "+name)
		lbl.SetMouseCursor(ui.MouseCursorPointer)
		lbl.SetDragSource(func(x, y int) *ui.DragData {
			return &ui.DragData{Text: name, Value: name}
		})
	}
	fruits.AddVSpacer(10, 0)

	basket := ui.NewGroupBox("Basket (takes fruits)")
	columns.AddWidget(0, 1, basket)
	contents := basket.AddLabel(0, 0, "(empty)")
	basket.AddVSpacer(1, 0)
	var inBasket []string
	basket.SetDropTarget(
		func(d *ui.DragData, x, y int) bool { _, ok := d.Value.(string); return ok },
		func(d *ui.DragData, x, y int) {
			inBasket = append(inBasket, d.Value.(string))
			contents.SetText(strings.Join(inBasket, ", "))
		})

	files := ui.NewGroupBox("Files (drop from the file manager)")
	page.AddWidget(2, 0, files)
	fileList := files.AddLabel(0, 0, "(no files yet)")
	files.SetDropTarget(
		func(d *ui.DragData, x, y int) bool { return len(d.Files) > 0 },
		func(d *ui.DragData, x, y int) {
			fileList.SetText(strings.Join(d.Files, ";  "))
			form.ShowToast(fmt.Sprintf("%d file(s) dropped", len(d.Files)), ui.ToastSuccess)
		})
	form.SetOnFilesDropped(func(paths []string, x, y int) {
		form.ShowToast("Drop files on the Files box", ui.ToastWarning)
	})
	page.AddVSpacer(3, 0)
	return page
}

func toastsPage(form *ui.Form) ui.Widgeter {
	page := ui.NewPanel()
	status := page.AddLabel(0, 0, "")

	toasts := ui.NewGroupBox("Toasts")
	page.AddWidget(1, 0, toasts)
	toasts.AddButton(0, 0, "Info", func() { form.ShowToast("Something happened", ui.ToastInfo) })
	toasts.AddButton(0, 1, "Success", func() { form.ShowToast("Saved", ui.ToastSuccess) })
	toasts.AddButton(0, 2, "Warning", func() { form.ShowToast("Disk almost full", ui.ToastWarning) })
	toasts.AddButton(0, 3, "Error", func() {
		form.ShowToastFor("Connection failed: the server did not answer within 30 seconds. Check the address and try again.", ui.ToastError, 6*time.Second)
	})

	dialogs := ui.NewGroupBox("File dialogs")
	page.AddWidget(2, 0, dialogs)
	dialogs.AddButton(0, 0, "Open files...", func() {
		form.ShowOpenFileDialog(ui.OpenFileDialogOptions{Title: "Open", AllowMultiple: true}, func(paths []string, err error) {
			status.SetText(fmt.Sprintf("Open: %v %v", paths, errText(err)))
		})
	})
	dialogs.AddButton(0, 1, "Save as...", func() {
		form.ShowSaveFileDialog(ui.SaveFileDialogOptions{Title: "Save", DefaultFileName: "report.txt"}, func(path string, err error) {
			status.SetText(fmt.Sprintf("Save: %q %v", path, errText(err)))
		})
	})
	dialogs.AddButton(0, 2, "Select folder...", func() {
		form.ShowSelectDirectoryDialog(ui.SelectDirectoryDialogOptions{Title: "Folder"}, func(path string, err error) {
			status.SetText(fmt.Sprintf("Folder: %q %v", path, errText(err)))
		})
	})

	trayBox := ui.NewGroupBox("Tray icon")
	page.AddWidget(3, 0, trayBox)
	var tray *ui.TrayIcon
	muted := false
	var updateMenu func()
	updateMenu = func() {
		tray.SetMenu(
			ui.TrayMenuItem{Text: "Show window", OnClick: func() { form.Show() }},
			ui.TrayMenuItem{Text: "Mute", Checkable: true, Checked: muted, OnClick: func() {
				muted = !muted
				updateMenu()
				status.SetText(fmt.Sprintf("Muted: %v", muted))
			}},
			ui.TrayMenuItem{Separator: true},
			ui.TrayMenuItem{Text: "Say hello", OnClick: func() {
				if err := tray.ShowNotification("Hello", "A notification from the tray icon"); err != nil {
					status.SetText(err.Error())
				}
			}},
		)
	}
	trayBox.AddButton(0, 0, "Add tray icon", func() {
		if tray != nil {
			return
		}
		var err error
		tray, err = ui.NewTrayIcon()
		if err != nil {
			status.SetText("Tray: " + err.Error())
			return
		}
		tray.SetIcon(trayImage())
		tray.SetTooltip("nui example")
		tray.SetOnClick(func() { form.Show(); status.SetText("Tray icon clicked") })
		updateMenu()
		status.SetText("The icon is in the tray: click it, or right-click for the menu")
	})
	trayBox.AddButton(0, 1, "Hide window to tray", func() {
		if tray == nil {
			status.SetText("Add the tray icon first")
			return
		}
		form.Hide()
	})
	trayBox.AddButton(0, 2, "Remove tray icon", func() {
		if tray != nil {
			tray.Close()
			tray = nil
		}
	})
	form.OnClose = func() bool {
		if tray != nil {
			tray.Close()
		}
		return true
	}
	page.AddVSpacer(4, 0)
	return page
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return "(" + err.Error() + ")"
}

// trayImage draws a round blue icon
func trayImage() image.Image {
	const size = 32
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx, dy := x-size/2, y-size/2
			if dx*dx+dy*dy <= (size/2-1)*(size/2-1) {
				img.Set(x, y, color.RGBA{0x1D, 0x6F, 0xB5, 255})
			}
		}
	}
	return img
}
