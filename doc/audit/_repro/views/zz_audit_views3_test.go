package ui

import (
	"fmt"
	"image"
	"sync"
	"testing"
	"time"

	"github.com/ipoluianov/nui/internal/platforms"
)

func TestAuditTableColumnShrinkNavigation(t *testing.T) {
	table := NewTable()
	newTestForm(t, table)
	m := &auditRowsModel{rows: 5, cols: 10}
	table.SetModel(m)
	table.SetCurrentCell2(2, 8)
	m.cols = 2
	m.Changed()
	for i := 0; i < 3; i++ {
		table.ProcessKeyDown(KeyArrowLeft, KeyModifiers{})
	}
	t.Logf("after 3x Left: CurrentColumn=%d currentCellX=%d", table.CurrentColumn(), table.currentCellX)
	if table.CurrentColumn() < 0 {
		t.Errorf("Left arrow can't bring the current column back after the model lost columns (currentCellX=%d)", table.currentCellX)
	}
}

func TestAuditTableNegativeColumnWidthPaint(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	table.SetColumnCount(3)
	table.SetRowCount(3)
	table.SetColumnWidth(0, -500)
	table.SetColumnWidth(1, 0)
	table.SetStretchLastColumn(true)
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	auditPaint(form)
	table.onMouseDown(MouseButtonLeft, 10, 40, KeyModifiers{})
	table.onMouseMove(10, 40, KeyModifiers{})
}

func TestAuditTableHeaderSpanHuge(t *testing.T) {
	table := NewTable()
	form := newTestForm(t, table)
	table.SetColumnCount(2)
	table.SetRowCount(1)
	table.SetHeaderCellSpan2(0, 0, 1, 50_000_000)
	start := time.Now()
	auditPaint(form)
	t.Logf("paint with a header span of 5e7 over 2 columns: %v", time.Since(start))
}

func TestAuditListViewTinyRowIcon(t *testing.T) {
	for _, h := range []int{1, 2, 3, 4, 5} {
		list := NewListView()
		form := newTestForm(t, list)
		m := &auditIconModel{rows: 3, icon: image.NewRGBA(image.Rect(0, 0, 16, 16))}
		list.SetModel(m)
		list.SetRowHeight(h)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("row height %d: panic %v", h, r)
				}
			}()
			auditPaint(form)
		}()
	}
}

func TestAuditLogViewConcurrentWrite(t *testing.T) {
	logs := NewLogView()
	newTestForm(t, logs)
	logs.SetMaxLines(50)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				fmt.Fprintf(logs, "g%d line %d\n", g, i)
			}
		}(g)
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
loop:
	for {
		select {
		case <-done:
			break loop
		default:
			platforms.RunHeadlessPosted()
		}
	}
	platforms.RunHeadlessPosted()
	if n := len(logs.lines); n > 50 {
		t.Errorf("%d lines kept, max 50", n)
	}
}
