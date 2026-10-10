package plotter

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ipoluianov/nui/ui"
)

// valueRows is the number of rows in the Values table.
const valueRows = 11

// updateValues fills the Values table with f at evenly spaced x.
func (c *Plotter) updateValues(f *Formula, vars Vars, x0, x1 float64) {
	c.values.SetRowCount(valueRows)
	for i := range valueRows {
		vars.X = x0 + (x1-x0)*float64(i)/(valueRows-1)
		c.values.SetCellText2(i, 0, formatNumber(vars.X))
		c.values.SetCellText2(i, 1, formatNumber(f.Eval(vars)))
		c.values.SetCellHAlign(i, 0, ui.HAlignRight)
		c.values.SetCellHAlign(i, 1, ui.HAlignRight)
	}
}

func formatNumber(v float64) string {
	return strconv.FormatFloat(v, 'g', 6, 64)
}

// copyTable copies the Values table as tab-separated text, which pastes
// into a spreadsheet as cells.
func (c *Plotter) copyTable() {
	var sb strings.Builder
	sb.WriteString("x\tf(x)\n")
	for row := range c.values.RowCount() {
		fmt.Fprintf(&sb, "%s\t%s\n", c.values.GetCellText2(row, 0), c.values.GetCellText2(row, 1))
	}
	ui.ClipboardSetText(sb.String())
}

func (c *Plotter) exportCSV() {
	opts := ui.SaveFileDialogOptions{
		Title:           "Export CSV",
		DefaultFileName: "plot.csv",
		Filters:         []ui.FileDialogFilter{{DisplayName: "CSV files", Patterns: []string{"*.csv"}}},
	}
	// The points are taken now: the curve may move while the dialog is open
	points := c.points
	formula := strings.TrimSpace(c.formula.Text())
	c.Form().ShowSaveFileDialog(opts, func(path string, err error) {
		if errors.Is(err, ui.ErrNoFileDialog) {
			ui.ShowMessageBox(c, "Export CSV", "No file dialog is available (install zenity or kdialog).")
			return
		}
		if err == nil && path != "" {
			err = writeCSV(path, formula, points)
		}
		if err != nil {
			ui.ShowMessageBox(c, "Export CSV", "Can't save the file: "+err.Error())
		}
	})
}

// writeCSV writes the points as "x,y" lines with a header.
func writeCSV(path, formula string, points []ui.ChartPoint) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	w := bufio.NewWriter(file)
	fmt.Fprintf(w, "x,%q\n", "f(x) = "+formula)
	for _, p := range points {
		fmt.Fprintf(w, "%s,%s\n", strconv.FormatFloat(p.X, 'g', -1, 64), strconv.FormatFloat(p.Y, 'g', -1, 64))
	}
	if err := w.Flush(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
