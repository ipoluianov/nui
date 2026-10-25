package ui

import (
	"context"
	"sync"
)

// Work in the background. RunAsync runs a function in a goroutine and gives
// its result to the UI thread; the work is cancelled (its context) by
// Cancel, or when the owner's form closes - then the result is dropped, the
// widgets are gone.
//
//	task := ui.RunAsync(form.Panel(), func(ctx context.Context, p *ui.Progress) ([]File, error) {
//		return scan(ctx, dir, func(done, all int) { p.Report(float64(done)/float64(all), "") })
//	}, func(files []File, err error) {
//		if err != nil { showError(err); return }
//		list.SetModel(filesModel(files))
//	})
//	task.OnProgress.Connect(func(p ui.ProgressReport) { bar.SetValue(p.Fraction) })
//
// RunWithProgress does the same with a progress dialog.

// ProgressReport is where a work is, see Progress
type ProgressReport struct {
	Fraction float64 // the part done, 0..1; negative if unknown
	Text     string
}

// Progress is what a work reports its progress through, from its goroutine.
// Reports faster than the UI shows them are joined: the last one is shown.
type Progress struct {
	task *Task

	mu      sync.Mutex
	last    ProgressReport
	pending bool
}

// Report tells how far the work is; safe from any goroutine
func (p *Progress) Report(fraction float64, text string) {
	p.mu.Lock()
	p.last = ProgressReport{fraction, text}
	pending := p.pending
	p.pending = true
	p.mu.Unlock()
	if pending {
		return
	}
	Invoke(func() {
		p.mu.Lock()
		report := p.last
		p.pending = false
		p.mu.Unlock()
		if !p.task.finished {
			p.task.OnProgress.Emit(report)
		}
	})
}

// Task is a work started by RunAsync
type Task struct {
	// OnProgress: the work reported its progress (on the UI thread)
	OnProgress Signal[ProgressReport]

	cancel   context.CancelFunc
	finished bool // on the UI thread: the result is given, or dropped
}

// Cancel cancels the work: its context is done. The result is still given
// to done (mostly with the context's error) unless the owner's form closed.
func (t *Task) Cancel() {
	t.cancel()
}

// Finished reports whether the result was given (or dropped); UI thread
func (t *Task) Finished() bool {
	return t.finished
}

// RunAsync runs work in a goroutine and done with its result on the UI
// thread. owner is a widget of the form the work is for: when the form
// closes, the work is cancelled and done isn't called.
func RunAsync[T any](owner Widgeter, work func(ctx context.Context, p *Progress) (T, error), done func(result T, err error)) *Task {
	ctx, cancel := context.WithCancel(context.Background())
	task := &Task{cancel: cancel}
	p := &Progress{task: task}
	removeClosing := func() {}
	if form := owner.Form(); form != nil {
		removeClosing = form.onClosing(func() {
			task.finished = true
			cancel()
		})
	}
	go func() {
		result, err := work(ctx, p)
		Invoke(func() {
			if task.finished {
				return // the form closed
			}
			task.finished = true
			cancel()
			removeClosing()
			if done != nil {
				done(result, err)
			}
		})
	}()
	return task
}

// RunWithProgress runs the work as RunAsync, with a progress dialog over the
// owner's form showing its reports; Cancel in the dialog cancels it. The
// dialog closes when the work ends, before done.
func RunWithProgress[T any](owner Widgeter, title, text string, work func(ctx context.Context, p *Progress) (T, error), done func(result T, err error)) *Task {
	var dialog *ProgressDialog
	task := RunAsync(owner, work, func(result T, err error) {
		dialog.Close()
		if done != nil {
			done(result, err)
		}
	})
	dialog = ShowProgressDialog(owner, title, text, task.Cancel)
	task.OnProgress.Connect(func(r ProgressReport) {
		dialog.SetProgress(r.Fraction)
		if r.Text != "" {
			dialog.SetText(r.Text)
		}
	})
	return task
}
