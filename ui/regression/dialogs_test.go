package regression

import (
	"context"
	"errors"
	"image/color"
	"testing"
	"time"

	"github.com/ipoluianov/nui/ui"
	"github.com/ipoluianov/nui/ui/uitest"
)

func TestInputDialog(t *testing.T) {
	form := newForm(600, 400)
	w := uitest.Show(t, form)
	got := ""
	ui.ShowInputDialog(form.Panel(), ui.InputDialog{
		Title: "Rename", Prompt: "New name:", Value: "old.txt",
		Validate: func(text string) error {
			if text == "" {
				return errors.New("The name is empty")
			}
			return nil
		},
	}, func(text string) { got = text })
	d := w.Dialog()
	if d.Title() != "Rename" {
		t.Fatalf("title %q", d.Title())
	}
	d.Key(ui.KeyBackspace, ui.KeyModifiers{}) // the value is selected: deleted
	d.Key(ui.KeyEnter, ui.KeyModifiers{})
	if d.Closed() || d.Find("The name is empty") == nil {
		t.Fatal("an empty name passed the validation")
	}
	d.MatchGolden("input-dialog-error")
	d.Type("new.txt\n")
	if !d.Closed() || got != "new.txt" {
		t.Errorf("closed %v, got %q", d.Closed(), got)
	}
}

func TestRunAsync(t *testing.T) {
	form := newForm(300, 200)
	w := uitest.Show(t, form)
	var result int
	var reports []float64
	task := ui.RunAsync(form.Panel(), func(ctx context.Context, p *ui.Progress) (int, error) {
		for i := 1; i <= 100; i++ {
			p.Report(float64(i)/100, "")
		}
		return 42, nil
	}, func(r int, err error) { result = r })
	task.OnProgress.Connect(func(r ui.ProgressReport) { reports = append(reports, r.Fraction) })
	for !task.Finished() {
		w.Wait(5 * time.Millisecond)
	}
	if result != 42 {
		t.Errorf("result %d", result)
	}
	// Joined: far fewer than 100, the last one among them
	if len(reports) == 0 || len(reports) > 20 {
		t.Errorf("%d progress reports", len(reports))
	}

	// The form closes: the work is cancelled, done isn't called
	other := newForm(300, 200)
	uitest.Show(t, other)
	called, cancelled := false, make(chan bool, 1)
	ui.RunAsync(other.Panel(), func(ctx context.Context, p *ui.Progress) (int, error) {
		select {
		case <-ctx.Done():
			cancelled <- true
		case <-time.After(3 * time.Second):
			cancelled <- false
		}
		return 0, ctx.Err()
	}, func(int, error) { called = true })
	other.Close()
	if !<-cancelled {
		t.Error("the work isn't cancelled when its form closes")
	}
	w.Wait(20 * time.Millisecond)
	if called {
		t.Error("done called after the form closed")
	}
}

func TestRunWithProgress(t *testing.T) {
	form := newForm(600, 400)
	w := uitest.Show(t, form)
	var gotErr error
	finished := false
	task := ui.RunWithProgress(form.Panel(), "Copying", "Copying the files...", func(ctx context.Context, p *ui.Progress) (int, error) {
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return i, ctx.Err()
			case <-time.After(5 * time.Millisecond):
				p.Report(0.5, "file 1 of 2")
			}
		}
	}, func(n int, err error) { gotErr, finished = err, true })
	w.Wait(30 * time.Millisecond)
	d := w.Dialog()
	if d.Find("file 1 of 2") == nil {
		t.Error("the report isn't shown")
	}
	d.Key(ui.KeyEsc, ui.KeyModifiers{}) // Cancel
	for !task.Finished() {
		w.Wait(5 * time.Millisecond)
	}
	if !finished || !errors.Is(gotErr, context.Canceled) || !d.Closed() {
		t.Errorf("after Cancel: finished %v, err %v, dialog closed %v", finished, gotErr, d.Closed())
	}
}

func TestAboutDialog(t *testing.T) {
	form := newForm(600, 400)
	w := uitest.Show(t, form)
	ui.ShowAboutDialog(form.Panel(), ui.AboutDialog{
		Name: "Notepad", Version: "Version 1.0", Description: "A primitive text editor.",
		Copyright: "© 2026", Website: "https://example.com",
		Icon: squareIcon(color.RGBA{0, 120, 215, 255}),
	})
	d := w.Dialog()
	d.MatchGolden("about-dialog")
	d.Key(ui.KeyEsc, ui.KeyModifiers{})
	if !d.Closed() {
		t.Error("Esc doesn't close the about box")
	}
}
