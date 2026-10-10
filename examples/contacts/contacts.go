package contacts

import (
	"encoding/csv"
	"fmt"
	"image"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/fogleman/gg"
	"github.com/ipoluianov/nui/examples/icons"
	"github.com/ipoluianov/nui/ui"
)

// Table columns
const (
	colFavorite = iota
	colName
	colCompany
	colPhone
	colEmail
)

var columnNames = []string{"", "Name", "Company", "Phone", "Email"}

// addressBook is the state of the application.
type addressBook struct {
	form     *ui.Form
	contacts []*Contact
	shown    []*Contact // the filtered and sorted contacts, one per table row
	sortCol  int
	sortDesc bool

	search  *ui.TextBox
	group   *ui.ComboBox
	table   *ui.Table
	details *detailsPanel
	status  *ui.Label
}

// NewForm is an address book: a sortable, filterable contact list with a
// details panel, and a dialog to add and edit contacts. It shows a toolbar of
// flat ToolButtons with tooltips, a search TextBox with a hint, a ComboBox
// filter, a Table with row multiselection, header sorting and a context menu,
// a Splitter, Links, an Accordion of Expanders, a DialogContent with a
// DatePicker and inline validation, question message boxes, the clipboard,
// the save file dialog (CSV export), toasts and keyboard shortcuts.
func NewForm() *ui.Form {
	b := &addressBook{
		form:     ui.NewForm(),
		contacts: sampleContacts(),
		sortCol:  colName,
	}
	b.form.SetTitle("Contacts")
	b.form.SetSize(1000, 640)

	panel := b.form.Panel()
	panel.AddWidget(0, 0, b.toolbar())

	b.table = b.newTable()
	b.details = newDetailsPanel()
	splitter := ui.NewHSplitter()
	splitter.SetWidgets(b.table, b.details)
	splitter.SetSecondSize(340)
	panel.AddWidget(1, 0, splitter)

	b.status = panel.AddLabel(2, 0, "")

	b.form.AddShortcut("Mod+N", b.add)
	b.form.AddShortcut("Delete", b.delete)
	b.form.AddShortcut("Mod+F", func() {
		b.search.Focus()
		b.search.SelectAllText()
	})

	b.refresh(nil)
	if len(b.shown) > 0 {
		b.table.SetCurrentCell2(0, colName)
	}
	return b.form
}

func (b *addressBook) toolbar() ui.Widgeter {
	bar := ui.NewPanel()
	toolButton := func(col int, img image.Image, tooltip string, onClick func()) {
		btn := ui.NewToolButton(img, tooltip, onClick)
		btn.SetFlat(true)
		btn.SetButtonSize(36, 36)
		bar.AddWidget(0, col, btn)
	}
	toolButton(0, icons.Plus(20), "Add contact (Ctrl+N)", b.add)
	toolButton(1, icons.Pencil(20), "Edit contact (Enter)", b.edit)
	toolButton(2, icons.Cross(20), "Delete contact (Delete)", b.delete)

	b.search = ui.NewTextBox()
	b.search.SetHint("Search…")
	b.search.SetMinWidth(240)
	b.search.SetMaxWidth(240)
	b.search.SetOnTextChanged(func() { b.refresh(b.selected()) })
	bar.AddWidget(0, 3, b.search)

	b.group = ui.NewComboBox()
	b.group.AddItem("All groups", "")
	for _, g := range groups {
		b.group.AddItem(g, g)
	}
	b.group.SetSelectedIndex(0)
	b.group.SetOnSelectedIndexChanged(func() { b.refresh(b.selected()) })
	bar.AddWidget(0, 4, b.group)

	bar.AddHSpacer(0, 5)
	return bar
}

func (b *addressBook) newTable() *ui.Table {
	t := ui.NewTable()
	t.SetColumnCount(len(columnNames))
	for col, width := range []int{28, 160, 140, 130, 180} {
		t.SetColumnWidth(col, width)
	}
	t.SetColumnImage(colFavorite, starImage(), 16)
	t.SetStretchLastColumn(true)
	t.SetMultiselect(true)
	t.SetOnSelectionChanged(func(row, col int) { b.selectionChanged() })
	t.SetOnColumnClick(func(col int) {
		if col == b.sortCol {
			b.sortDesc = !b.sortDesc
		} else {
			b.sortCol, b.sortDesc = col, false
		}
		b.refresh(b.selected())
	})
	t.SetOnCellMouseDblClick(func() {
		ui.CurrentEvent().Parameter.(*ui.EventTableCellMouseDblClick).Processed = true
		b.edit()
	})
	t.SetOnKeyDown(func(key ui.Key, mods ui.KeyModifiers) bool {
		if key == ui.KeyEnter {
			b.edit()
			return true
		}
		return false
	})

	menu := ui.NewContextMenu(t)
	menu.AddItem("Edit…", b.edit).SetImage(icons.Pencil(icons.Size))
	menu.AddItem("Delete", b.delete).SetImage(icons.Cross(icons.Size)).SetShortcut("Delete")
	menu.AddItem("Copy email", b.copyEmail)
	menu.AddItem("Toggle favorite", b.toggleFavorite).SetImage(starImage())
	menu.AddSeparator()
	menu.AddItem("Export CSV…", b.exportCSV).SetImage(icons.Save(icons.Size))
	t.SetContextMenu(menu)
	return t
}

// refresh filters and sorts the contacts into the table, then selects the
// given contacts again if they are still shown.
func (b *addressBook) refresh(keep []*Contact) {
	text := strings.ToLower(strings.TrimSpace(b.search.Text()))
	group, _ := b.group.SelectedItemData().(string)
	b.shown = b.shown[:0]
	for _, c := range b.contacts {
		if (group == "" || c.Group == group) && (text == "" || c.matches(text)) {
			b.shown = append(b.shown, c)
		}
	}
	slices.SortStableFunc(b.shown, func(x, y *Contact) int {
		r := strings.Compare(strings.ToLower(sortKey(x, b.sortCol)), strings.ToLower(sortKey(y, b.sortCol)))
		if b.sortDesc {
			r = -r
		}
		return r
	})

	// The sorted column shows a triangle; the favorites column keeps its star
	for col, name := range columnNames {
		b.table.SetColumnName(col, name)
		if col == colFavorite {
			continue
		}
		if col == b.sortCol {
			b.table.SetColumnImage(col, sortImage(b.sortDesc), 12)
		} else {
			b.table.SetColumnImage(col, nil, 0)
		}
	}

	b.table.ClearRows()
	var rows []int
	for row, c := range b.shown {
		if c.Favorite {
			b.table.SetCellImage(row, colFavorite, starImage(), 16)
		}
		b.table.SetCellText2(row, colName, c.Name)
		b.table.SetCellText2(row, colCompany, c.Company)
		b.table.SetCellText2(row, colPhone, c.Phone)
		b.table.SetCellText2(row, colEmail, c.Email)
		if slices.Contains(keep, c) {
			rows = append(rows, row)
		}
	}
	b.table.SetRowCount(len(b.shown))
	if len(rows) > 0 {
		b.table.SetSelectedRows(rows)
	}
	b.selectionChanged()
}

func sortKey(c *Contact, col int) string {
	switch col {
	case colFavorite:
		return map[bool]string{true: "0", false: "1"}[c.Favorite] + c.Name
	case colCompany:
		return c.Company
	case colPhone:
		return c.Phone
	case colEmail:
		return c.Email
	}
	return c.Name
}

func (b *addressBook) selectionChanged() {
	b.details.show(b.current())
	text := fmt.Sprintf("%d contacts, %d shown", len(b.contacts), len(b.shown))
	if n := len(b.selected()); n > 1 {
		text += fmt.Sprintf(", %d selected", n)
	}
	b.status.SetText(text)
}

// current is the contact in the current row, or nil.
func (b *addressBook) current() *Contact {
	row := b.table.CurrentRow()
	if row < 0 || row >= len(b.shown) {
		return nil
	}
	return b.shown[row]
}

func (b *addressBook) selected() []*Contact {
	var list []*Contact
	for _, row := range b.table.SelectedRows() {
		if row < len(b.shown) {
			list = append(list, b.shown[row])
		}
	}
	return list
}

func (b *addressBook) add() {
	dialog := NewDialogContact("New Contact", Contact{Group: "Friends", Birthday: date(1990, time.January, 1)})
	dialog.OnOK = func(contact Contact) {
		c := &contact
		b.contacts = append(b.contacts, c)
		// Show the new contact even if the filters would hide it
		b.search.SetText("")
		b.group.SetSelectedIndex(0)
		b.refresh([]*Contact{c})
		b.form.ShowToast("Added "+c.Name, ui.ToastSuccess)
	}
	b.form.Panel().ShowDialog(dialog)
}

func (b *addressBook) edit() {
	c := b.current()
	if c == nil {
		return
	}
	dialog := NewDialogContact("Edit Contact", *c)
	dialog.OnOK = func(edited Contact) {
		*c = edited
		b.refresh([]*Contact{c})
	}
	b.form.Panel().ShowDialog(dialog)
}

func (b *addressBook) delete() {
	list := b.selected()
	if len(list) == 0 {
		return
	}
	question := fmt.Sprintf("Delete %d contacts?", len(list))
	if len(list) == 1 {
		question = fmt.Sprintf("Delete \"%s\"?", list[0].Name)
	}
	ui.ShowQuestionMessageBoxYesNo(b.form.Panel(), "Delete", question, func() {
		b.contacts = slices.DeleteFunc(b.contacts, func(c *Contact) bool { return slices.Contains(list, c) })
		b.refresh(nil)
		if len(list) == 1 {
			b.form.ShowToast("Deleted "+list[0].Name, ui.ToastInfo)
		} else {
			b.form.ShowToast(fmt.Sprintf("Deleted %d contacts", len(list)), ui.ToastInfo)
		}
	}, nil)
}

func (b *addressBook) copyEmail() {
	var emails []string
	for _, c := range b.selected() {
		if c.Email != "" {
			emails = append(emails, c.Email)
		}
	}
	if len(emails) == 0 {
		return
	}
	ui.ClipboardSetText(strings.Join(emails, ", "))
	b.form.ShowToast("Copied: "+strings.Join(emails, ", "), ui.ToastInfo)
}

func (b *addressBook) toggleFavorite() {
	list := b.selected()
	for _, c := range list {
		c.Favorite = !c.Favorite
	}
	b.refresh(list)
}

// exportCSV saves the shown contacts to a CSV file.
func (b *addressBook) exportCSV() {
	opts := ui.SaveFileDialogOptions{
		Title:           "Export Contacts",
		DefaultFileName: "contacts.csv",
		Filters:         []ui.FileDialogFilter{{DisplayName: "CSV files", Patterns: []string{"*.csv"}}},
	}
	list := slices.Clone(b.shown)
	b.form.ShowSaveFileDialog(opts, func(path string, err error) {
		if err != nil {
			ui.ShowMessageBox(b.form.Panel(), "Export", err.Error())
			return
		}
		if path == "" {
			return
		}
		if err := writeCSV(path, list); err != nil {
			ui.ShowMessageBox(b.form.Panel(), "Export", "Cannot export: "+err.Error())
			return
		}
		b.form.ShowToast(fmt.Sprintf("Exported %d contacts", len(list)), ui.ToastSuccess)
	})
}

func writeCSV(path string, list []*Contact) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	w := csv.NewWriter(f)
	w.Write([]string{"Name", "Company", "Group", "Phone", "Email", "Birthday", "Favorite", "Notes"})
	for _, c := range list {
		w.Write([]string{c.Name, c.Company, c.Group, c.Phone, c.Email,
			c.Birthday.Format(time.DateOnly), fmt.Sprint(c.Favorite), c.Notes})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

var star image.Image

// starImage is the gold star of the favorites; it's drawn once, the table
// shows it in many rows.
func starImage() image.Image {
	if star == nil {
		star = icons.Draw(icons.Size, func(dc *gg.Context) {
			for i := range 10 {
				r := 7.0
				if i%2 == 1 {
					r = 3
				}
				a := gg.Radians(float64(i)*36 - 90)
				dc.LineTo(8+r*math.Cos(a), 8.5+r*math.Sin(a))
			}
			dc.ClosePath()
			dc.SetHexColor("#F5B301")
			dc.Fill()
		})
	}
	return star
}

var sortImages = map[bool]image.Image{}

// sortImage is a triangle pointing up for the ascending order, down for the
// descending one.
func sortImage(desc bool) image.Image {
	if img, ok := sortImages[desc]; ok {
		return img
	}
	img := icons.Draw(icons.Size, func(dc *gg.Context) {
		if desc {
			dc.RotateAbout(gg.Radians(180), 8, 8)
		}
		dc.MoveTo(8, 4)
		dc.LineTo(13, 11)
		dc.LineTo(3, 11)
		dc.ClosePath()
		dc.SetHexColor("#8C8C8C")
		dc.Fill()
	})
	sortImages[desc] = img
	return img
}
