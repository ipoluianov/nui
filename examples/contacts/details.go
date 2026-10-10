package contacts

import (
	"fmt"
	"strings"
	"time"

	"github.com/ipoluianov/nui/ui"
)

// detailsPanel shows the current contact on the right of the list.
type detailsPanel struct {
	*ui.Panel

	lblName     *ui.Label
	lblCompany  *ui.Label
	linkEmail   *ui.Link
	linkPhone   *ui.Link
	sections    *ui.Accordion
	lblGroup    *ui.Label
	lblBirthday *ui.Label
	lblFavorite *ui.Label
	txtNotes    *ui.TextBox
}

func newDetailsPanel() *detailsPanel {
	d := &detailsPanel{Panel: ui.NewPanel()}
	d.SetMinWidth(260)

	d.lblName = d.AddLabel(0, 0, "")
	d.lblName.SetFontSize(20)
	d.lblCompany = d.AddLabel(1, 0, "")

	d.linkEmail = ui.NewLink("", nil)
	d.AddWidget(2, 0, d.linkEmail)
	d.linkPhone = ui.NewLink("", nil)
	d.AddWidget(3, 0, d.linkPhone)

	d.sections = ui.NewAccordion()
	d.sections.SetExclusive(false)
	info := d.sections.AddSection("Details")
	info.SetExpanded(true)
	row := func(r int, name string) *ui.Label {
		info.Content().AddLabel(r, 0, name)
		return info.Content().AddLabel(r, 1, "")
	}
	d.lblGroup = row(0, "Group:")
	d.lblBirthday = row(1, "Birthday:")
	d.lblFavorite = row(2, "Favorite:")
	info.Content().AddHSpacer(0, 2)

	notes := d.sections.AddSection("Notes")
	notes.SetExpanded(true)
	d.txtNotes = ui.NewTextBox()
	d.txtNotes.SetMultiline(true)
	d.txtNotes.SetReadOnly(true)
	d.txtNotes.SetYExpandable(true)
	notes.Content().AddWidget(0, 0, d.txtNotes)
	d.sections.SetYExpandable(true)
	d.AddWidget(4, 0, d.sections)

	d.show(nil)
	return d
}

// show fills the panel with the contact; nil clears it.
func (d *detailsPanel) show(c *Contact) {
	empty := c == nil
	d.lblCompany.SetVisible(!empty)
	d.linkEmail.SetVisible(!empty && c.Email != "")
	d.linkPhone.SetVisible(!empty && c.Phone != "")
	d.sections.SetVisible(!empty)
	if empty {
		d.lblName.SetText("No contact selected")
		return
	}

	d.lblName.SetText(c.Name)
	company := c.Company
	if company == "" {
		company = "(no company)"
	}
	d.lblCompany.SetText(company)
	d.linkEmail.SetText(c.Email)
	d.linkEmail.SetURL("mailto:" + c.Email)
	d.linkPhone.SetText(c.Phone)
	d.linkPhone.SetURL("tel:" + strings.ReplaceAll(c.Phone, " ", ""))

	d.lblGroup.SetText(c.Group)
	d.lblBirthday.SetText(fmt.Sprintf("%s (age %d)", c.Birthday.Format("January 2, 2006"), c.age(time.Now())))
	d.lblFavorite.SetText(map[bool]string{true: "Yes", false: "No"}[c.Favorite])
	d.txtNotes.SetText(c.Notes)
}
