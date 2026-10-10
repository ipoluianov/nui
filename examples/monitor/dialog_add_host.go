package monitor

import (
	"strings"

	"github.com/ipoluianov/nui/ui"
)

// dialogAddHost asks for a new host: its name, the expected latency and the
// kind of link.
type dialogAddHost struct {
	ui.DialogContent

	txtName    *ui.TextBox
	numLatency *ui.NumBox
	cbProfile  *ui.ComboBox

	OnOK func(h *host)
}

func newDialogAddHost() *dialogAddHost {
	var c dialogAddHost
	c.InitWidget()

	fields := c.AddPanel(0, 0)
	fields.AddLabel(0, 0, "Host name:")
	c.txtName = ui.NewTextBox()
	c.txtName.SetHint("server.example.com")
	fields.AddWidget(0, 1, c.txtName)

	fields.AddLabel(1, 0, "Link:")
	c.cbProfile = ui.NewComboBox()
	for _, p := range profiles {
		c.cbProfile.AddItem(p.name, p)
	}
	c.cbProfile.SetSelectedIndex(1)
	// A new profile suggests its typical latency.
	c.cbProfile.SetOnSelectedIndexChanged(func() {
		c.numLatency.SetValue(c.profile().base)
	})
	fields.AddWidget(1, 1, c.cbProfile)

	fields.AddLabel(2, 0, "Base latency, ms:")
	c.numLatency = ui.NewNumBox()
	c.numLatency.SetDecimals(1)
	c.numLatency.SetMin(0.1)
	c.numLatency.SetMax(2000)
	c.numLatency.SetStep(1)
	c.numLatency.SetValue(c.profile().base)
	fields.AddWidget(2, 1, c.numLatency)

	c.AddVSpacer(1, 0)

	buttons := c.AddPanel(2, 0)
	buttons.AddHSpacer(0, 0)
	btnOK := buttons.AddButton(0, 1, "Add", c.accept)
	btnCancel := buttons.AddButton(0, 2, "Cancel", func() { c.Form().Close() })

	c.OnDialogShow = func() {
		f := c.Form()
		f.SetTitle("Add host")
		f.SetSize(380, 180)
		f.MoveToCenterOfParent()
		f.SetAcceptButton(btnOK)
		f.SetCancelButton(btnCancel)
		c.txtName.Focus()
	}
	return &c
}

func (c *dialogAddHost) profile() profile {
	return c.cbProfile.SelectedItemData().(profile)
}

func (c *dialogAddHost) accept() {
	name := strings.TrimSpace(c.txtName.Text())
	if name == "" {
		ui.ShowMessageBox(c, "Add host", "Enter the name or the address of the host.")
		return
	}
	h := newHost(name, c.numLatency.Value(), c.profile())
	c.Form().Close()
	c.RunInParent(func() {
		if c.OnOK != nil {
			c.OnOK(h)
		}
	})
}

func (c *monitor) addHost() {
	dialog := newDialogAddHost()
	dialog.OnOK = func(h *host) {
		// Starts without history, from the next sample.
		c.hosts = append(c.hosts, h)
		c.events.add(c.lastSample, h.name, severityInfo, "Host added")
		c.updateHosts()
		c.tblHosts.SetCurrentCell2(len(c.hosts)-1, 1)
		c.showSelected()
		c.updateStatus()
	}
	c.ShowDialog(dialog)
}

func (c *monitor) removeHosts() {
	selected := c.selectedHosts()
	if len(selected) == 0 {
		return
	}
	text := "Stop monitoring " + selected[0].name + "?"
	if len(selected) > 1 {
		names := make([]string, len(selected))
		for i, h := range selected {
			names[i] = h.name
		}
		text = "Stop monitoring these hosts?\n" + strings.Join(names, "\n")
	}
	ui.ShowQuestionMessageBoxYesNo(c, "Remove hosts", text, func() {
		remove := map[*host]bool{}
		for _, h := range selected {
			remove[h] = true
			c.events.add(c.lastSample, h.name, severityInfo, "Host removed")
		}
		var rest []*host
		for _, h := range c.hosts {
			if !remove[h] {
				rest = append(rest, h)
			}
		}
		c.hosts = rest
		c.tblHosts.ClearRows()
		c.updateHosts()
		if len(c.hosts) > 0 {
			c.tblHosts.SetCurrentCell2(0, 1)
		}
		c.showSelected()
		c.updateStatus()
	}, nil)
}
