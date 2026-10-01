package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/ipoluianov/nui/internal/platforms"
)

// showPrintDialog is nui's print dialog, for the systems without one of their
// own (Linux, macOS): the CUPS printers and "Save as PDF", copies, pages,
// paper and orientation. done is called once: nil when printed, or
// ErrPrintCanceled, or the error.
func showPrintDialog(parent *Form, job *PrintJob, done func(err error)) {
	t := UIText()
	printers, defaultPrinter, _ := platforms.Printers()

	form := NewForm()
	form.SetTitle(t.Print)
	form.SetSize(460, 250)
	form.SetAllowMinimize(false)
	form.SetAllowMaximize(false)
	p := form.Panel()
	p.SetPanelPadding(12)
	p.SetCellPadding(8)

	finished := false
	finish := func(err error) {
		if finished {
			return
		}
		finished = true
		done(err)
	}

	// The fields: names on the left, the editors take the rest of the width
	fields := p.AddPanel(0, 0)
	fields.SetPanelPadding(0)
	fields.SetCellPadding(8)
	addField := func(row int, name string, editor Widgeter) {
		fields.AddLabel(row, 0, name)
		editor.SetXExpandable(true)
		fields.AddWidget(row, 1, editor)
	}

	printer := NewComboBox()
	for _, name := range printers {
		printer.AddItem(name, name)
	}
	printer.AddItem(t.SaveAsPDF, "")
	for i, name := range printers {
		if name == defaultPrinter {
			printer.SetSelectedIndex(i)
		}
	}
	addField(0, t.Printer, printer)

	copies := NewNumBox()
	copies.SetDecimals(0)
	copies.SetMin(1)
	copies.SetMax(999)
	copies.SetValue(1)
	addField(1, t.Copies, copies)

	pages := NewTextBox()
	pages.SetHint(t.PagesHint)
	addField(2, t.Pages, pages)

	paperRow := NewPanel()
	paperRow.SetPanelPadding(0)
	paper := NewComboBox()
	paper.SetMinWidth(110)
	paper.SetXExpandable(true)
	current := job.paper()
	for i, size := range PaperSizes {
		paper.AddItem(size.Name, size)
		if size.Name == current.Name {
			paper.SetSelectedIndex(i)
		}
	}
	paperRow.AddWidget(0, 0, paper)
	orientation := NewComboBox()
	orientation.SetMinWidth(110)
	orientation.SetXExpandable(true)
	orientation.AddItem(t.Portrait, false)
	orientation.AddItem(t.Landscape, true)
	if job.Landscape {
		orientation.SetSelectedIndex(1)
	}
	paperRow.AddWidget(0, 1, orientation)
	addField(3, t.Paper, paperRow)

	status := p.AddLabel(1, 0, "")
	status.SetForegroundColor(CurrentPalette().Error)
	p.AddVSpacer(2, 0)

	buttons := p.AddPanel(3, 0)
	buttons.SetPanelPadding(0)
	buttons.AddHSpacer(0, 0)
	printButton := buttons.AddButton(0, 1, t.Print, nil)
	printButton.SetRole("primary")
	cancelButton := buttons.AddButton(0, 2, t.Cancel, func() {
		form.Close()
		finish(ErrPrintCanceled)
	})
	form.SetAcceptButton(printButton)
	form.SetCancelButton(cancelButton)
	form.OnClose = func() bool {
		finish(ErrPrintCanceled)
		return true
	}

	printButton.SetOnClick(func() {
		// The job with the chosen page; the application's job stays as it is
		j := *job
		if size, ok := paper.SelectedItemData().(PaperSize); ok {
			j.Paper = size
		}
		j.Landscape, _ = orientation.SelectedItemData().(bool)
		width, height := j.pageSize()
		count := j.pageCount(width, height, j.dpi())
		var pageList []int
		if text := strings.TrimSpace(pages.Text()); text != "" {
			var err error
			if pageList, err = parsePageRange(text, count); err != nil {
				status.SetText(err.Error())
				return
			}
		}
		printerName, _ := printer.SelectedItemData().(string)
		form.Close()

		if printerName == "" {
			savePrintPDF(parent, &j, pageList, finish)
			return
		}
		finish(sendToPrinter(&j, pageList, printerName, int(copies.Value())))
	})

	form.ShowModal(parent)
}

// savePrintPDF asks for a file and saves the pages there
func savePrintPDF(parent *Form, job *PrintJob, pages []int, done func(err error)) {
	name := job.Title
	if name == "" {
		name = "document"
	}
	parent.ShowSaveFileDialog(SaveFileDialogOptions{
		Title:           UIText().SaveAsPDF,
		DefaultFileName: name + ".pdf",
		Filters:         []FileDialogFilter{{DisplayName: "PDF", Patterns: []string{"*.pdf"}}},
	}, func(path string, err error) {
		if err != nil {
			done(err)
			return
		}
		if path == "" {
			done(ErrPrintCanceled)
			return
		}
		f, err := os.Create(path)
		if err == nil {
			err = job.WritePDF(f, pages)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
		done(err)
	})
}

// sendToPrinter makes a PDF of the pages and gives it to CUPS
func sendToPrinter(job *PrintJob, pages []int, printer string, copies int) error {
	f, err := os.CreateTemp("", "nui-print-*.pdf")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	err = job.WritePDF(f, pages)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	// lp copies the file to the spool before it returns
	if err := platforms.PrintFile(printer, f.Name(), copies, job.Title, map[string]string{"media": job.paper().Name}); err != nil {
		return fmt.Errorf("%s: %w", printer, err)
	}
	return nil
}
