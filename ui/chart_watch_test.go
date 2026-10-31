package ui

import "testing"

// A memory source repaints a chart while the chart is shown: one watcher
// for a series, moved to the new source by SetSource, gone with the series
// or the closed form
func TestChartSourceWatchers(t *testing.T) {
	watchers := func(s *ChartMemorySource) int {
		s.mtx.RLock()
		defer s.mtx.RUnlock()
		return len(s.watchers)
	}
	a, b := NewChartMemorySource(), NewChartMemorySource()
	form := NewForm()
	chart := NewLineChart()
	form.Panel().AddWidget(0, 0, chart)
	line := chart.Series("s")
	line.SetSource(a)
	line.SetSource(a)
	if watchers(a) != 1 {
		t.Errorf("the same source twice: %d watchers", watchers(a))
	}
	line.SetSource(b)
	if watchers(a) != 0 || watchers(b) != 1 {
		t.Errorf("another source: old %d, new %d", watchers(a), watchers(b))
	}
	chart.Pane(0).RemoveSeries(line)
	if watchers(b) != 0 {
		t.Errorf("removed series: %d", watchers(b))
	}

	second := chart.Pane(0).AddLine("t")
	second.SetSource(a)
	form.Panel().RemoveWidget(chart)
	if watchers(a) != 0 {
		t.Errorf("chart out of the form: %d", watchers(a))
	}
	form.Panel().AddWidget(0, 0, chart)
	if watchers(a) != 1 {
		t.Errorf("chart back in the form: %d", watchers(a))
	}
}
