package ui

import (
	"math"
	"strings"
	"testing"
	"time"
)

// measure7 measures a label as 7 pixels a character
func measure7(s string) float64 { return float64(len([]rune(s))) * 7 }

func majorLabels(ticks []ChartTick) []string {
	var res []string
	for _, t := range ticks {
		if !t.Minor {
			res = append(res, t.Label)
		}
	}
	return res
}

func TestLinearScaleTicks(t *testing.T) {
	s := NewLinearScale(0, 100)
	got := strings.Join(majorLabels(s.Ticks(500, measure7)), " ")
	if got != "0 10 20 30 40 50 60 70 80 90 100" {
		t.Fatalf("ticks %q", got)
	}
	// A short axis takes a coarser step
	got = strings.Join(majorLabels(s.Ticks(100, measure7)), " ")
	if got != "0 50 100" {
		t.Fatalf("short axis ticks %q", got)
	}
	// Decimals as the step needs
	s.SetDomain(0, 1)
	got = strings.Join(majorLabels(s.Ticks(400, measure7)), " ")
	if got != "0.0 0.1 0.2 0.3 0.4 0.5 0.6 0.7 0.8 0.9 1.0" {
		t.Fatalf("decimal ticks %q", got)
	}
	// Negative values, and no -0
	s.SetDomain(-3, 3)
	got = strings.Join(majorLabels(s.Ticks(300, measure7)), " ")
	if got != "-3 -2 -1 0 1 2 3" {
		t.Fatalf("negative ticks %q", got)
	}
}

// The labels never overlap, whatever the length of the axis
func TestLinearScaleTicksDoNotOverlap(t *testing.T) {
	s := NewLinearScale(-1234.5, 98765.4)
	for length := 30.0; length < 2000; length *= 1.3 {
		var prevEnd = math.Inf(-1)
		for _, tk := range s.Ticks(length, measure7) {
			if tk.Minor {
				continue
			}
			p := s.ToUnit(tk.Value) * length
			w := measure7(tk.Label)
			if p-w/2 < prevEnd {
				t.Fatalf("length %.0f: %q overlaps the previous label", length, tk.Label)
			}
			prevEnd = p + w/2
		}
	}
}

func TestLinearScaleMinorTicks(t *testing.T) {
	s := NewLinearScale(0, 10)
	minor := 0
	for _, tk := range s.Ticks(1000, measure7) {
		if tk.Minor {
			minor++
			if tk.Label != "" {
				t.Fatal("a minor tick with a label")
			}
		}
	}
	if minor == 0 {
		t.Fatal("no minor ticks on a long axis")
	}
}

func TestLinearScaleIntegerAndUnit(t *testing.T) {
	s := NewLinearScale(0, 3)
	s.Integer = true
	s.Unit = " pcs"
	got := strings.Join(majorLabels(s.Ticks(1000, measure7)), ",")
	if got != "0 pcs,1 pcs,2 pcs,3 pcs" {
		t.Fatalf("integer ticks %q", got)
	}
}

func TestLinearScaleDegenerateDomain(t *testing.T) {
	s := NewLinearScale(5, 5)
	if a, b := s.Domain(); !(a < 5 && b > 5) {
		t.Fatalf("a single value is not widened: %v %v", a, b)
	}
	s.SetDomain(math.NaN(), 3)
	if a, b := s.Domain(); a != 0 || b != 1 {
		t.Fatalf("NaN domain: %v %v", a, b)
	}
	if len(s.Ticks(0, measure7)) != 0 {
		t.Fatal("ticks on an axis of no length")
	}
}

func TestFormatChartNumber(t *testing.T) {
	cases := []struct {
		v, step float64
		si      bool
		want    string
	}{
		{0.25, 0.05, false, "0.25"},
		{1200, 100, false, "1200"},
		{1200, 100, true, "1.2k"},
		{2_500_000, 500_000, true, "2.5M"},
		{0.0015, 0.0005, true, "1.5m"},
		{-0.0, 1, false, "0"},
		{1e20, 1e19, false, "1e+20"},
		{math.Inf(1), 1, false, "∞"},
	}
	for _, c := range cases {
		if got := FormatChartNumber(c.v, c.step, c.si); got != c.want {
			t.Errorf("FormatChartNumber(%v, %v, %v) = %q, want %q", c.v, c.step, c.si, got, c.want)
		}
	}
}

func TestLogScale(t *testing.T) {
	s := NewLogScale(1, 1000)
	if u := s.ToUnit(10); math.Abs(u-1.0/3) > 1e-9 {
		t.Fatalf("10 at %v, want 1/3", u)
	}
	if !math.IsNaN(s.ToUnit(0)) || !math.IsNaN(s.ToUnit(-5)) {
		t.Fatal("0 and below are shown")
	}
	if v := s.FromUnit(s.ToUnit(42)); math.Abs(v-42) > 1e-9 {
		t.Fatalf("round trip %v", v)
	}
	got := strings.Join(majorLabels(s.Ticks(300, measure7)), " ")
	if got != "1 10 100 1000" {
		t.Fatalf("decades %q", got)
	}
	// Long decades get 2 and 5 too
	got = strings.Join(majorLabels(NewLogScale(1, 100).Ticks(500, measure7)), " ")
	if got != "1 2 5 10 20 50 100" {
		t.Fatalf("long decades %q", got)
	}
	// Many short decades: a label every few
	many := majorLabels(NewLogScale(1e-10, 1e10).Ticks(200, measure7))
	if len(many) < 2 || len(many) > 8 {
		t.Fatalf("20 decades on 200 px: %q", many)
	}
	// A domain at or below zero is kept above it
	s.SetDomain(-5, 0)
	if a, b := s.Domain(); !(a > 0 && b > a) {
		t.Fatalf("domain %v %v", a, b)
	}
}

func TestSymlogScale(t *testing.T) {
	s := NewSymlogScale(-1000, 1000)
	if u := s.ToUnit(0); math.Abs(u-0.5) > 1e-9 {
		t.Fatalf("0 at %v", u)
	}
	if v := s.FromUnit(s.ToUnit(-37)); math.Abs(v+37) > 1e-9 {
		t.Fatalf("round trip %v", v)
	}
	labels := majorLabels(s.Ticks(600, measure7))
	if strings.Join(labels, " ") != "-1000 -100 -10 -1 0 1 10 100 1000" {
		t.Fatalf("ticks %q", labels)
	}
}

func TestTimeScaleTicks(t *testing.T) {
	loc := time.UTC
	from := time.Date(2026, 3, 1, 10, 0, 0, 0, loc)
	s := NewTimeScale(from, from.Add(time.Hour))
	s.Location = loc
	ticks := s.Ticks(600, measure7)
	if len(ticks) < 4 {
		t.Fatalf("%d ticks for an hour on 600 px", len(ticks))
	}
	for _, tk := range ticks {
		tm := ChartTimeToTime(tk.Value, loc)
		if tm.Second() != 0 || tm.Minute()%5 != 0 {
			t.Fatalf("tick %v not on a round minute", tm)
		}
	}
	if ticks[0].Label != "10:00" {
		t.Fatalf("first label %q", ticks[0].Label)
	}
	// Months land on the first day of the month (a year on 300 px)
	s = NewTimeScale(time.Date(2025, 1, 15, 0, 0, 0, 0, loc), time.Date(2026, 1, 15, 0, 0, 0, 0, loc))
	s.Location = loc
	monthTicks := s.Ticks(300, measure7)
	if len(monthTicks) < 3 || monthTicks[0].Label != "Apr" {
		t.Fatalf("month ticks %+v", monthTicks)
	}
	for _, tk := range monthTicks {
		if d := ChartTimeToTime(tk.Value, loc).Day(); d != 1 {
			t.Fatalf("a month tick on day %d", d)
		}
	}
}

func TestTimeScaleDateRibbon(t *testing.T) {
	loc := time.UTC
	// Across midnight: two dates
	s := NewTimeScale(time.Date(2026, 3, 1, 22, 0, 0, 0, loc), time.Date(2026, 3, 2, 2, 0, 0, 0, loc))
	s.Location = loc
	spans := s.DateRibbon(800, measure7)
	if len(spans) != 2 || spans[0].Label != "2026-03-01" || spans[1].Label != "2026-03-02" {
		t.Fatalf("ribbon %+v", spans)
	}
	if lo, hi := s.Domain(); spans[0].Start != lo || spans[1].End != hi {
		t.Fatal("the ribbon is not clipped to the domain")
	}
}

func TestChartTimeRoundTrip(t *testing.T) {
	tm := time.Date(2026, 10, 29, 12, 34, 56, 789_000_000, time.UTC)
	back := ChartTimeToTime(ChartTime(tm), time.UTC)
	if d := back.Sub(tm); d < -time.Microsecond || d > time.Microsecond {
		t.Fatalf("round trip off by %v", d)
	}
}

func TestCategoryScale(t *testing.T) {
	s := NewCategoryScale("Jan", "Feb", "Mar", "Apr")
	if u := s.ToUnit(0); math.Abs(u-0.125) > 1e-9 {
		t.Fatalf("first band center at %v", u)
	}
	if s.BandWidth() != 0.25 {
		t.Fatalf("band %v", s.BandWidth())
	}
	if got := strings.Join(majorLabels(s.Ticks(400, measure7)), " "); got != "Jan Feb Mar Apr" {
		t.Fatalf("labels %q", got)
	}
	// A short axis labels every other category
	if got := strings.Join(majorLabels(s.Ticks(100, measure7)), " "); got != "Jan Mar" {
		t.Fatalf("short axis labels %q", got)
	}
	// New categories follow a domain that showed all of them
	s.SetCategories([]string{"a", "b"})
	if lo, hi := s.Domain(); lo != -0.5 || hi != 1.5 {
		t.Fatalf("domain after SetCategories %v %v", lo, hi)
	}
	if s.Format(1.2) != "b" || s.Index("b") != 1 || s.Index("z") != -1 {
		t.Fatal("Format or Index")
	}
}

// The pyramid gives the buckets of the merging from all the samples: the
// same extremes, a sample per bucket, the gaps kept
func TestChartMemorySourcePyramid(t *testing.T) {
	n := 300_000
	samples := make([]ChartSample, n)
	for i := range samples {
		samples[i] = ChartValue(float64(i), math.Sin(float64(i)/1000)+float64(i%7)/10)
	}
	samples[150_000] = ChartGap(150_000)
	src := NewChartMemorySource()
	// Appended in pieces: the levels grow with them
	for i := 0; i < n; i += 70_001 {
		src.Append(samples[i:min(i+70_001, n)]...)
		src.GetData(0, float64(n), 300)
	}
	got := src.GetData(0, float64(n), 300)
	want := ChartDownsample(samples, 0, float64(n), 300)
	if len(got) < len(want)-5 || len(got) > len(want)+5 {
		t.Fatalf("%d samples, the full merge gives %d", len(got), len(want))
	}
	hi, lo := math.Inf(-1), math.Inf(1)
	gap := false
	for _, s := range got {
		if s.IsGap() {
			gap = true
		}
		if s.HasValue() {
			hi, lo = math.Max(hi, s.High), math.Min(lo, s.Low)
		}
	}
	wh, wl := math.Inf(-1), math.Inf(1)
	for _, s := range want {
		if s.HasValue() {
			wh, wl = math.Max(wh, s.High), math.Min(wl, s.Low)
		}
	}
	if !gap || hi != wh || lo != wl {
		t.Fatalf("gap %v, range %v..%v, want %v..%v", gap, lo, hi, wl, wh)
	}
	// The cache: the same call returns the same slice
	a, b := src.GetData(1000, 2000, 1), src.GetData(1000, 2000, 1)
	if &a[0] != &b[0] {
		t.Fatal("not cached")
	}
	src.AppendValue(float64(n+1), 1)
	if c := src.GetData(1000, 2000, 1); &c[0] == &a[0] {
		t.Fatal("the cache outlived a change")
	}
}
