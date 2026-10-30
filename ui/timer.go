package ui

import (
	"slices"
	"time"
)

// Timer runs a function on the UI thread at an interval, while its widget
// is in an open form (the form's event loop ticks the timers every few
// milliseconds). Widget.AddTimer makes a running timer, Widget.NewTimer a
// stopped one, Widget.AfterFunc one that runs once:
//
//	refresh := panel.AddTimer(1000, reload) // every second
//	refresh.Stop()                          // until Start
//
//	// A search that waits for the typing to pause (debounce): each Start
//	// counts the interval anew
//	search := box.NewTimer(300*time.Millisecond, runSearch)
//	search.SetSingleShot(true)
//	box.OnTextChanged.Connect(func(string) { search.Start() })
type Timer struct {
	owner      *Widget
	interval   time.Duration
	f          func()
	running    bool
	singleShot bool
	// last is when the function ran last, or the timer was started
	last time.Time
	// immediate runs the function at the next tick (AddTimer's first run)
	immediate bool
}

// NewTimer makes a stopped timer of the widget; Start starts it. After the
// function the widget is repainted: what the function changes elsewhere
// asks for its own repaint (the setters do, Update for what a widget paints
// of its own data). A timer of the form's panel repaints all of the form.
func (c *Widget) NewTimer(interval time.Duration, f func()) *Timer {
	return &Timer{owner: c, interval: interval, f: f}
}

// AddTimer starts a timer running the function every intervalMs
// milliseconds, the first time at the next tick; it returns the timer, to
// stop it
func (c *Widget) AddTimer(intervalMs int, f func()) *Timer {
	t := c.NewTimer(time.Duration(intervalMs)*time.Millisecond, f)
	t.Start()
	t.immediate = true
	return t
}

// AfterFunc runs the function once, after d; the timer returned can stop it
// before, or Start it again
func (c *Widget) AfterFunc(d time.Duration, f func()) *Timer {
	t := c.NewTimer(d, f)
	t.SetSingleShot(true)
	t.Start()
	return t
}

// Start starts the timer, or restarts it: the function runs a full interval
// from now
func (t *Timer) Start() {
	t.last = time.Now()
	t.immediate = false
	if !t.running {
		t.running = true
		t.owner.timers = append(t.owner.timers, t)
	}
}

// Stop stops the timer: the function doesn't run until Start
func (t *Timer) Stop() {
	if !t.running {
		return
	}
	t.running = false
	t.owner.timers = slices.DeleteFunc(slices.Clone(t.owner.timers), func(o *Timer) bool { return o == t })
}

// Running reports whether the timer runs
func (t *Timer) Running() bool {
	return t.running
}

// SetInterval changes the interval; the current one counts from its start
func (t *Timer) SetInterval(d time.Duration) {
	t.interval = d
}

func (t *Timer) Interval() time.Duration {
	return t.interval
}

// SetSingleShot makes the timer run its function once a Start
func (t *Timer) SetSingleShot(singleShot bool) {
	t.singleShot = singleShot
}

func (t *Timer) SingleShot() bool {
	return t.singleShot
}

func (t *Timer) tick() {
	// A zero interval: a single shot runs at the next tick, a repeating
	// timer never (it would run at every tick)
	if !t.running || t.interval <= 0 && !t.singleShot {
		return
	}
	if !t.immediate && time.Since(t.last) < t.interval {
		return
	}
	t.immediate = false
	t.last = time.Now()
	if t.singleShot {
		t.Stop()
	}
	if t.f != nil {
		t.f()
	}
	// What the function changed is painted: the widget of the timer, all
	// of the form for a timer of its panel (see Widget.AddTimer)
	t.owner.Update()
}
