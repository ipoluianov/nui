package ui

import "time"

type timer struct {
	intervalMs        int
	lastExecutionTime time.Time
	callback          func()
}

func (t *timer) tick() {
	if t.intervalMs <= 0 {
		return
	}

	if time.Since(t.lastExecutionTime) < time.Duration(t.intervalMs)*time.Millisecond {
		return
	}

	if t.callback != nil {
		t.callback()
	}
	t.lastExecutionTime = time.Now()
}
