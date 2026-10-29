package ui

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// ChartTick is a mark on an axis: a value of the scale, its label, and
// whether it is a minor tick (a finer grid line, without a label)
type ChartTick struct {
	Value float64
	Label string
	Minor bool
}

// ChartScale turns the values of an axis into positions along it and picks
// the ticks: a LinearScale, a LogScale, a SymlogScale, a TimeScale or a
// CategoryScale. The visible range of the axis is the scale's domain.
type ChartScale interface {
	// ToUnit returns where the value is in the domain: 0 at its start, 1 at
	// its end, beyond them outside it; NaN for a value the scale cannot
	// show (0 or below on a LogScale)
	ToUnit(v float64) float64
	// FromUnit is the inverse of ToUnit
	FromUnit(u float64) float64
	// Domain returns the visible range
	Domain() (min, max float64)
	// SetDomain sets the visible range; a scale may widen a range of no
	// length or limit it (a LogScale keeps it above 0)
	SetDomain(min, max float64)
	// Ticks returns the ticks of an axis length pixels long whose labels,
	// measured along the axis by measure, do not overlap
	Ticks(length float64, measure func(label string) float64) []ChartTick
	// Format returns the value as text precise enough for the current
	// domain: for a tooltip or a crosshair label
	Format(v float64) string
}

// chartTickGap is the least gap between two labels along an axis, in
// logical pixels
const chartTickGap = 12

// ---------------------------------------------------------------------------
// Linear

// LinearScale is the ordinary scale: equal steps of the value are equal
// lengths along the axis. Its ticks are at round steps of 1, 2 and 5 times a
// power of ten.
type LinearScale struct {
	min, max float64
	// Format of the labels and the tooltips; nil - by the step, see
	// FormatChartNumber
	Formatter func(v float64) string
	// Unit is added to the labels and the tooltips: "%", " ms", " °C"
	Unit string
	// SI writes big and small numbers with the prefixes k, M, G, m, µ:
	// 1.5k, 20M
	SI bool
	// Integer keeps the ticks on whole numbers: for counts
	Integer bool
}

// NewLinearScale makes a linear scale over the domain
func NewLinearScale(min, max float64) *LinearScale {
	s := &LinearScale{}
	s.SetDomain(min, max)
	return s
}

func (s *LinearScale) Domain() (float64, float64) { return s.min, s.max }

func (s *LinearScale) SetDomain(min, max float64) {
	s.min, s.max = widenDomain(min, max, 1)
}

func (s *LinearScale) ToUnit(v float64) float64 {
	return (v - s.min) / (s.max - s.min)
}

func (s *LinearScale) FromUnit(u float64) float64 {
	return s.min + u*(s.max-s.min)
}

func (s *LinearScale) Ticks(length float64, measure func(string) float64) []ChartTick {
	return linearTicks(s.min, s.max, length, measure, s.Integer, func(v, step float64) string {
		return s.label(v, step)
	})
}

func (s *LinearScale) label(v, step float64) string {
	if s.Formatter != nil {
		return s.Formatter(v)
	}
	return FormatChartNumber(v, step, s.SI) + s.Unit
}

func (s *LinearScale) Format(v float64) string {
	if s.Formatter != nil {
		return s.Formatter(v)
	}
	// As precise as a thousandth of the range, a pixel or so of the axis,
	// without the zeros at the end: 4, 0.598, 10.04
	return trimChartZeros(FormatChartNumber(v, niceStepAtLeast((s.max-s.min)/1000), s.SI)) + s.Unit
}

// trimChartZeros drops the zeros at the end of the decimals of a number,
// before its SI prefix: "4.00" is "4", "1.50k" is "1.5k"
func trimChartZeros(s string) string {
	end := len(s)
	for end > 0 && (s[end-1] < '0' || s[end-1] > '9') {
		end-- // the prefix
	}
	num, suffix := s[:end], s[end:]
	if !strings.Contains(num, ".") || strings.ContainsAny(num, "eE") {
		return s
	}
	num = strings.TrimRight(num, "0")
	num = strings.TrimSuffix(num, ".")
	return num + suffix
}

// widenDomain returns the range in order and of some length: a range of a
// single value is widened around it by pad (or by a part of it)
func widenDomain(min, max, pad float64) (float64, float64) {
	if math.IsNaN(min) || math.IsInf(min, 0) || math.IsNaN(max) || math.IsInf(max, 0) {
		return 0, 1
	}
	if min > max {
		min, max = max, min
	}
	if max-min < 1e-300 || (max-min) < math.Abs(min)*1e-12 {
		d := math.Max(math.Abs(min)*0.1, pad)
		min, max = min-d, max+d
	}
	return min, max
}

// niceSteps are the mantissas of the tick steps
var niceSteps = [...]float64{1, 2, 5}

// niceStepAtLeast returns the least step of 1, 2, 5 times a power of ten
// not below raw
func niceStepAtLeast(raw float64) float64 {
	if !(raw > 0) || math.IsInf(raw, 0) {
		return 1
	}
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range niceSteps {
		if m*mag >= raw*(1-1e-9) {
			return m * mag
		}
	}
	return 10 * mag
}

// nextNiceStep returns the next step of 1, 2, 5 times a power of ten
func nextNiceStep(step float64) float64 {
	return niceStepAtLeast(step * 1.0001)
}

// linearTicks picks the least round step whose labels fit along the axis,
// and the minor ticks between the major ones when there is room for them
func linearTicks(min, max, length float64, measure func(string) float64, integer bool, label func(v, step float64) string) []ChartTick {
	span := max - min
	if !(span > 0) || !(length > 0) {
		return nil
	}
	// Start from steps of about 20 pixels; at most 40 tries
	step := niceStepAtLeast(span / math.Max(1, length/20))
	if integer {
		step = math.Max(step, 1)
	}
	for try := 0; try < 40; try++ {
		px := step / span * length
		first := math.Ceil(min/step - 1e-9)
		last := math.Floor(max/step + 1e-9)
		if last-first > 1000 {
			step = nextNiceStep(step)
			continue
		}
		widest := 0.0
		for i := first; i <= last; i++ {
			widest = math.Max(widest, measure(label(i*step, step)))
		}
		if px >= widest+chartTickGap || last-first < 1 {
			ticks := make([]ChartTick, 0, int(last-first)+1)
			// The minor ticks: steps of 1/5 or 1/4 of the major one, when
			// they are at least 6 pixels apart
			minorDiv := 5.0
			if m := step / math.Pow(10, math.Floor(math.Log10(step)+1e-9)); math.Abs(m-2) < 1e-6 {
				minorDiv = 4
			}
			minor := px/minorDiv >= 6 && !(integer && step/minorDiv < 1)
			for i := first - 1; i <= last; i++ {
				v := i * step
				if i >= first {
					ticks = append(ticks, ChartTick{Value: cleanZero(v), Label: label(cleanZero(v), step)})
				}
				if minor {
					for k := 1.0; k < minorDiv; k++ {
						mv := v + step*k/minorDiv
						if mv >= min && mv <= max {
							ticks = append(ticks, ChartTick{Value: mv, Minor: true})
						}
					}
				}
			}
			return ticks
		}
		step = nextNiceStep(step)
	}
	return nil
}

// cleanZero turns -0 and the rounding dust around zero into 0
func cleanZero(v float64) float64 {
	if math.Abs(v) < 1e-300 {
		return 0
	}
	return v
}

// FormatChartNumber formats the value with as many decimals as the step of
// the ticks needs: 0.25 with a step of 0.05, 1200 with a step of 100. With si
// big and small numbers get the prefixes k, M, G, T, m, µ, n: 1.2k, 35M.
// Numbers too big or too small for that are written with an exponent.
func FormatChartNumber(v, step float64, si bool) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	if math.IsInf(v, 0) {
		if v > 0 {
			return "∞"
		}
		return "-∞"
	}
	v = cleanZero(v)
	step = math.Abs(step)
	if !(step > 0) {
		step = math.Abs(v) / 1000
		if step == 0 {
			step = 1
		}
	}
	if si {
		if prefix, mul, ok := siPrefix(math.Max(math.Abs(v), step)); ok && mul != 1 {
			return strconv.FormatFloat(v/mul, 'f', stepDecimals(step/mul), 64) + prefix
		}
	}
	a := math.Abs(v)
	if a >= 1e15 || (step < 1e-9 && a != 0) {
		return strconv.FormatFloat(v, 'g', 6, 64)
	}
	return strconv.FormatFloat(v, 'f', stepDecimals(step), 64)
}

// stepDecimals returns how many decimals the multiples of the step need:
// 0 for 5, 1 for 0.5, 2 for 0.25
func stepDecimals(step float64) int {
	for d := 0; d < 12; d++ {
		x := step * math.Pow(10, float64(d))
		if math.Abs(x-math.Round(x)) < 1e-6*math.Max(1, x) {
			return d
		}
	}
	return 12
}

// siPrefix returns the SI prefix and its multiplier for a number of the
// magnitude a
func siPrefix(a float64) (string, float64, bool) {
	switch {
	case a >= 1e12:
		return "T", 1e12, true
	case a >= 1e9:
		return "G", 1e9, true
	case a >= 1e6:
		return "M", 1e6, true
	case a >= 1e3:
		return "k", 1e3, true
	case a >= 1 || a == 0:
		return "", 1, true
	case a >= 1e-3:
		return "m", 1e-3, true
	case a >= 1e-6:
		return "µ", 1e-6, true
	case a >= 1e-9:
		return "n", 1e-9, true
	}
	return "", 1, false
}

// ---------------------------------------------------------------------------
// Log

// LogScale is a logarithmic scale: each power of the base is the same
// length along the axis. It shows values above 0 only; the others are not
// drawn. For quantities over many orders of magnitude: frequencies, sizes,
// latencies.
type LogScale struct {
	min, max float64
	// Base of the logarithm; 0 is 10
	Base      float64
	Formatter func(v float64) string
	Unit      string
	SI        bool
}

// NewLogScale makes a logarithmic scale over the domain
func NewLogScale(min, max float64) *LogScale {
	s := &LogScale{}
	s.SetDomain(min, max)
	return s
}

func (s *LogScale) base() float64 {
	if s.Base > 1 {
		return s.Base
	}
	return 10
}

func (s *LogScale) Domain() (float64, float64) { return s.min, s.max }

func (s *LogScale) SetDomain(min, max float64) {
	if min > max {
		min, max = max, min
	}
	if !(max > 0) || math.IsInf(max, 0) {
		max = 10
	}
	if !(min > 0) || math.IsInf(min, 0) {
		min = math.Min(1, max/10)
	}
	if max/min < 1+1e-9 {
		min, max = min/s.base(), max*s.base()
	}
	s.min, s.max = min, max
}

func (s *LogScale) ToUnit(v float64) float64 {
	if !(v > 0) {
		return math.NaN()
	}
	return math.Log(v/s.min) / math.Log(s.max/s.min)
}

func (s *LogScale) FromUnit(u float64) float64 {
	return s.min * math.Pow(s.max/s.min, u)
}

func (s *LogScale) label(v float64) string {
	if s.Formatter != nil {
		return s.Formatter(v)
	}
	// The step of a tick at v is about its own size: 2, 3... show as many
	// decimals as v has, at most 3 significant digits
	return FormatChartNumber(v, logLabelStep(v), s.SI) + s.Unit
}

// logLabelStep returns the step the label of a tick at v on a log scale is
// formatted with: its own least significant digit
func logLabelStep(v float64) float64 {
	if v <= 0 {
		return 1
	}
	p := math.Pow(10, math.Floor(math.Log10(v)+1e-9))
	for _, m := range []float64{1, 0.1, 0.01} {
		if math.Abs(v/(p*m)-math.Round(v/(p*m))) < 1e-6 {
			return p * m
		}
	}
	return p * 0.001
}

func (s *LogScale) Format(v float64) string {
	if s.Formatter != nil {
		return s.Formatter(v)
	}
	if v <= 0 {
		return FormatChartNumber(v, 1, s.SI) + s.Unit
	}
	// Three significant digits
	return FormatChartNumber(v, math.Pow(10, math.Floor(math.Log10(v))-2), s.SI) + s.Unit
}

func (s *LogScale) Ticks(length float64, measure func(string) float64) []ChartTick {
	if !(length > 0) {
		return nil
	}
	b := s.base()
	lo := math.Floor(math.Log(s.min)/math.Log(b) + 1e-9)
	hi := math.Ceil(math.Log(s.max)/math.Log(b) - 1e-9)
	decades := math.Log(s.max/s.min) / math.Log(b)
	pxPerDecade := length / decades
	var ticks []ChartTick
	in := func(v float64) bool { return v >= s.min*(1-1e-9) && v <= s.max*(1+1e-9) }
	// The multiples within a decade: labeled 2 and 5 when a decade is long,
	// all of them as minor ticks when there is room
	var labeled []float64
	if b == 10 {
		switch {
		case pxPerDecade >= 300:
			labeled = []float64{2, 3, 5}
		case pxPerDecade >= 120:
			labeled = []float64{2, 5}
		}
	}
	every := 1.0 // a label for every n-th power when the decades are short
	for hi-lo > 0 && every < 100 {
		widest := math.Max(measure(s.label(math.Pow(b, lo))), measure(s.label(math.Pow(b, hi))))
		if pxPerDecade*every >= widest+chartTickGap {
			break
		}
		every++
	}
	for e := lo; e <= hi && e-lo < 400; e++ {
		p := math.Pow(b, e)
		major := math.Mod(e, every) == 0
		if in(p) {
			t := ChartTick{Value: p, Minor: !major}
			if major {
				t.Label = s.label(p)
			}
			ticks = append(ticks, t)
		}
		if b == 10 && pxPerDecade >= 40 {
			for m := 2.0; m < 10; m++ {
				v := m * p
				if !in(v) {
					continue
				}
				t := ChartTick{Value: v, Minor: true}
				for _, l := range labeled {
					if l == m {
						t.Minor, t.Label = false, s.label(v)
					}
				}
				ticks = append(ticks, t)
			}
		}
	}
	return dropOverlappingLabels(ticks, func(v float64) float64 { return s.ToUnit(v) * length }, measure)
}

// dropOverlappingLabels makes the labeled ticks whose labels would overlap
// the previous label minor, keeping the order
func dropOverlappingLabels(ticks []ChartTick, pos func(float64) float64, measure func(string) float64) []ChartTick {
	lastEnd := math.Inf(-1)
	for i := range ticks {
		if ticks[i].Minor {
			continue
		}
		p := pos(ticks[i].Value)
		half := measure(ticks[i].Label) / 2
		if p-half < lastEnd+chartTickGap {
			ticks[i].Minor, ticks[i].Label = true, ""
			continue
		}
		lastEnd = p + half
	}
	return ticks
}

// ---------------------------------------------------------------------------
// Symlog

// SymlogScale is logarithmic away from zero and linear near it, so it shows
// values of both signs and zero over many orders of magnitude: a flow in
// and out, a profit and a loss. Within Constant of zero (1 when 0) it is
// almost linear.
type SymlogScale struct {
	min, max  float64
	Constant  float64
	Formatter func(v float64) string
	Unit      string
	SI        bool
}

// NewSymlogScale makes a symmetric log scale over the domain
func NewSymlogScale(min, max float64) *SymlogScale {
	s := &SymlogScale{}
	s.SetDomain(min, max)
	return s
}

func (s *SymlogScale) c() float64 {
	if s.Constant > 0 {
		return s.Constant
	}
	return 1
}

func (s *SymlogScale) f(v float64) float64 {
	return math.Copysign(math.Log1p(math.Abs(v)/s.c()), v)
}

func (s *SymlogScale) inv(y float64) float64 {
	return math.Copysign(math.Expm1(math.Abs(y))*s.c(), y)
}

func (s *SymlogScale) Domain() (float64, float64) { return s.min, s.max }

func (s *SymlogScale) SetDomain(min, max float64) {
	s.min, s.max = widenDomain(min, max, s.c())
}

func (s *SymlogScale) ToUnit(v float64) float64 {
	a, b := s.f(s.min), s.f(s.max)
	return (s.f(v) - a) / (b - a)
}

func (s *SymlogScale) FromUnit(u float64) float64 {
	a, b := s.f(s.min), s.f(s.max)
	return s.inv(a + u*(b-a))
}

func (s *SymlogScale) label(v float64) string {
	if s.Formatter != nil {
		return s.Formatter(v)
	}
	return FormatChartNumber(v, logLabelStep(math.Abs(v)), s.SI) + s.Unit
}

func (s *SymlogScale) Format(v float64) string {
	if s.Formatter != nil {
		return s.Formatter(v)
	}
	a := math.Max(math.Abs(v), s.c())
	return FormatChartNumber(v, math.Pow(10, math.Floor(math.Log10(a))-2), s.SI) + s.Unit
}

func (s *SymlogScale) Ticks(length float64, measure func(string) float64) []ChartTick {
	if !(length > 0) {
		return nil
	}
	c := s.c()
	var vals []float64
	// 0, and ±c·10^k on each side within the domain
	add := func(v float64) {
		if v >= s.min-1e-12*math.Abs(s.min) && v <= s.max+1e-12*math.Abs(s.max) {
			vals = append(vals, v)
		}
	}
	top := math.Max(math.Abs(s.min), math.Abs(s.max))
	kMax := math.Ceil(math.Log10(math.Max(top/c, 1)))
	for k := kMax; k >= 0; k-- {
		add(-c * math.Pow(10, k))
	}
	add(0)
	for k := 0.0; k <= kMax; k++ {
		add(c * math.Pow(10, k))
	}
	ticks := make([]ChartTick, 0, len(vals))
	for _, v := range vals {
		ticks = append(ticks, ChartTick{Value: v, Label: s.label(v)})
	}
	return dropOverlappingLabels(ticks, func(v float64) float64 { return s.ToUnit(v) * length }, measure)
}

// ---------------------------------------------------------------------------
// Time

// ChartTime returns the value of the time on a TimeScale: seconds since
// the Unix epoch, precise to a fraction of a microsecond
func ChartTime(t time.Time) float64 {
	return float64(t.Unix()) + float64(t.Nanosecond())/1e9
}

// ChartTimeToTime returns the time of a value of a TimeScale, in the location
func ChartTimeToTime(v float64, loc *time.Location) time.Time {
	sec := math.Floor(v)
	t := time.Unix(int64(sec), int64(math.Round((v-sec)*1e9)))
	if loc != nil {
		t = t.In(loc)
	}
	return t
}

// TimeScale is a scale of time: its values are seconds since the Unix epoch
// (ChartTime). Its ticks are on the boundaries of the calendar in its
// Location: whole seconds, minutes, hours, days, months, years; the date
// under them is given by DateRibbon.
type TimeScale struct {
	min, max float64
	// Location is the time zone of the ticks; nil - time.Local
	Location *time.Location
	// Formatter formats a tooltip; nil - the date and the time as precise
	// as the domain needs
	Formatter func(t time.Time) string
}

// NewTimeScale makes a time scale from one time to the other
func NewTimeScale(from, to time.Time) *TimeScale {
	s := &TimeScale{}
	s.SetDomain(ChartTime(from), ChartTime(to))
	return s
}

func (s *TimeScale) loc() *time.Location {
	if s.Location != nil {
		return s.Location
	}
	return time.Local
}

func (s *TimeScale) Domain() (float64, float64) { return s.min, s.max }

func (s *TimeScale) SetDomain(min, max float64) {
	s.min, s.max = widenDomain(min, max, 1)
}

func (s *TimeScale) ToUnit(v float64) float64 {
	return (v - s.min) / (s.max - s.min)
}

func (s *TimeScale) FromUnit(u float64) float64 {
	return s.min + u*(s.max-s.min)
}

// TimeRange returns the domain as times
func (s *TimeScale) TimeRange() (from, to time.Time) {
	return ChartTimeToTime(s.min, s.loc()), ChartTimeToTime(s.max, s.loc())
}

func (s *TimeScale) Format(v float64) string {
	t := ChartTimeToTime(v, s.loc())
	if s.Formatter != nil {
		return s.Formatter(t)
	}
	span := s.max - s.min
	switch {
	case span < 2:
		return t.Format("2006-01-02 15:04:05.000")
	case span < 3*86400:
		return t.Format("2006-01-02 15:04:05")
	case span < 400*86400:
		return t.Format("2006-01-02 15:04")
	}
	return t.Format("2006-01-02")
}

// timeStep is a step of the ticks of a TimeScale: a duration, or a number
// of months
type timeStep struct {
	d      time.Duration
	months int
	format string
}

var timeSteps = []timeStep{
	{time.Millisecond, 0, "15:04:05.000"},
	{2 * time.Millisecond, 0, "15:04:05.000"},
	{5 * time.Millisecond, 0, "15:04:05.000"},
	{10 * time.Millisecond, 0, "15:04:05.000"},
	{20 * time.Millisecond, 0, "15:04:05.000"},
	{50 * time.Millisecond, 0, "15:04:05.000"},
	{100 * time.Millisecond, 0, "15:04:05.000"},
	{200 * time.Millisecond, 0, "15:04:05.000"},
	{500 * time.Millisecond, 0, "15:04:05.000"},
	{time.Second, 0, "15:04:05"},
	{2 * time.Second, 0, "15:04:05"},
	{5 * time.Second, 0, "15:04:05"},
	{10 * time.Second, 0, "15:04:05"},
	{15 * time.Second, 0, "15:04:05"},
	{30 * time.Second, 0, "15:04:05"},
	{time.Minute, 0, "15:04"},
	{2 * time.Minute, 0, "15:04"},
	{5 * time.Minute, 0, "15:04"},
	{10 * time.Minute, 0, "15:04"},
	{15 * time.Minute, 0, "15:04"},
	{30 * time.Minute, 0, "15:04"},
	{time.Hour, 0, "15:04"},
	{2 * time.Hour, 0, "15:04"},
	{3 * time.Hour, 0, "15:04"},
	{6 * time.Hour, 0, "15:04"},
	{12 * time.Hour, 0, "15:04"},
	{24 * time.Hour, 0, "02"},
	{2 * 24 * time.Hour, 0, "02"},
	{7 * 24 * time.Hour, 0, "02"},
	{14 * 24 * time.Hour, 0, "02"},
	{0, 1, "Jan"},
	{0, 3, "Jan"},
	{0, 6, "Jan"},
	{0, 12, "2006"},
	{0, 24, "2006"},
	{0, 60, "2006"},
	{0, 120, "2006"},
	{0, 600, "2006"},
}

func (st timeStep) approx() float64 {
	if st.months > 0 {
		return float64(st.months) * 30 * 86400
	}
	return st.d.Seconds()
}

// pickStep returns the least step whose labels fit along an axis of the
// length
func (s *TimeScale) pickStep(length float64, measure func(string) float64) timeStep {
	sample := time.Date(2000, 12, 28, 23, 59, 59, 999000000, time.UTC)
	for _, st := range timeSteps {
		px := st.approx() / (s.max - s.min) * length
		if px >= measure(sample.Format(st.format))+chartTickGap {
			return st
		}
	}
	return timeSteps[len(timeSteps)-1]
}

func (s *TimeScale) Ticks(length float64, measure func(string) float64) []ChartTick {
	if !(length > 0) {
		return nil
	}
	st := s.pickStep(length, measure)
	var ticks []ChartTick
	for _, t := range timeTicksOf(st, ChartTimeToTime(s.min, s.loc()), ChartTimeToTime(s.max, s.loc())) {
		ticks = append(ticks, ChartTick{Value: ChartTime(t), Label: t.Format(st.format)})
	}
	return ticks
}

// timeTicksOf returns the times of the step from from to to, on the
// boundaries of the calendar in from's location
func timeTicksOf(st timeStep, from, to time.Time) []time.Time {
	var res []time.Time
	add := func(t time.Time) bool {
		if !t.Before(from) {
			res = append(res, t)
		}
		return len(res) < 1000
	}
	loc := from.Location()
	switch {
	case st.months > 0:
		y, m, _ := from.Date()
		idx := y*12 + int(m) - 1
		t := time.Date(y, m, 1, 0, 0, 0, 0, loc).AddDate(0, -(idx % st.months), 0)
		for ; !t.After(to); t = t.AddDate(0, st.months, 0) {
			if !add(t) {
				break
			}
		}
	case st.d >= 24*time.Hour:
		days := int(st.d / (24 * time.Hour))
		y, m, d := from.Date()
		dayNum := int(time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400)
		t := time.Date(y, m, d, 0, 0, 0, 0, loc).AddDate(0, 0, -(((dayNum % days) + days) % days))
		for ; !t.After(to); t = t.AddDate(0, 0, days) {
			if !add(t) {
				break
			}
		}
	default:
		_, off := from.Zone()
		offD := time.Duration(off) * time.Second
		t := from.Add(offD).Truncate(st.d).Add(-offD)
		for ; !t.After(to); t = t.Add(st.d) {
			if !add(t) {
				break
			}
		}
	}
	return res
}

// ChartDateSpan is a part of the date ribbon under a time axis: the day, the
// month or the year from Start to End (values of the scale), and its label
type ChartDateSpan struct {
	Start, End float64
	Label      string
}

// DateRibbon returns the dates under the ticks of an axis length pixels
// long: one unit of the calendar coarser than the step of the ticks, so the
// ticks and the ribbon give the whole date and time. Empty when the ticks
// are years.
func (s *TimeScale) DateRibbon(length float64, measure func(string) float64) []ChartDateSpan {
	st := s.pickStep(length, measure)
	var unitStart func(time.Time) time.Time
	var next func(time.Time) time.Time
	var format string
	switch {
	case st.months >= 12:
		return nil
	case st.months > 0:
		unitStart = func(t time.Time) time.Time { return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location()) }
		next = func(t time.Time) time.Time { return t.AddDate(1, 0, 0) }
		format = "2006"
	case st.d >= 24*time.Hour:
		unitStart = func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location()) }
		next = func(t time.Time) time.Time { return t.AddDate(0, 1, 0) }
		format = "January 2006"
	default:
		unitStart = func(t time.Time) time.Time {
			y, m, d := t.Date()
			return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
		}
		next = func(t time.Time) time.Time { return t.AddDate(0, 0, 1) }
		format = "2006-01-02"
	}
	from, to := s.TimeRange()
	var spans []ChartDateSpan
	for t := unitStart(from); t.Before(to) && len(spans) < 1000; t = next(t) {
		spans = append(spans, ChartDateSpan{
			Start: math.Max(ChartTime(t), s.min),
			End:   math.Min(ChartTime(next(t)), s.max),
			Label: t.Format(format),
		})
	}
	return spans
}

// ---------------------------------------------------------------------------
// Category

// CategoryScale is a scale of named categories: the category i is at the
// value i, in the middle of its band from i-0.5 to i+0.5. The domain is
// from -0.5 to n-0.5 for all of them; a narrower one zooms in on some.
// Columns and bars, box plots, heat maps use it.
type CategoryScale struct {
	min, max   float64
	categories []string
	// fitted: the domain follows the number of categories
	fitted bool
}

// NewCategoryScale makes a scale of the categories
func NewCategoryScale(categories ...string) *CategoryScale {
	s := &CategoryScale{fitted: true}
	s.SetCategories(categories)
	return s
}

// SetCategories replaces the categories; a domain that showed all of them
// shows all the new ones
func (s *CategoryScale) SetCategories(categories []string) {
	s.categories = append(s.categories[:0], categories...)
	if s.fitted || s.max <= s.min {
		s.min, s.max = -0.5, math.Max(float64(len(categories))-0.5, 0.5)
		s.fitted = true
	}
}

// Categories returns the names of the categories
func (s *CategoryScale) Categories() []string { return s.categories }

// Index returns the value of the category of the name, -1 if there is none
func (s *CategoryScale) Index(name string) int {
	for i, c := range s.categories {
		if c == name {
			return i
		}
	}
	return -1
}

func (s *CategoryScale) Domain() (float64, float64) { return s.min, s.max }

func (s *CategoryScale) SetDomain(min, max float64) {
	s.min, s.max = widenDomain(min, max, 0.5)
	s.fitted = math.Abs(s.min+0.5) < 1e-9 && math.Abs(s.max-(float64(len(s.categories))-0.5)) < 1e-9
}

func (s *CategoryScale) ToUnit(v float64) float64 {
	return (v - s.min) / (s.max - s.min)
}

func (s *CategoryScale) FromUnit(u float64) float64 {
	return s.min + u*(s.max-s.min)
}

// BandWidth returns the width of the band of a category as a part of the
// domain: the column of a bar chart is a part of it
func (s *CategoryScale) BandWidth() float64 {
	return 1 / (s.max - s.min)
}

func (s *CategoryScale) name(i int) string {
	if i >= 0 && i < len(s.categories) {
		return s.categories[i]
	}
	return ""
}

func (s *CategoryScale) Format(v float64) string {
	return s.name(int(math.Round(v)))
}

func (s *CategoryScale) Ticks(length float64, measure func(string) float64) []ChartTick {
	if !(length > 0) || len(s.categories) == 0 {
		return nil
	}
	first := max(0, int(math.Ceil(s.min-1e-9)))
	last := min(len(s.categories)-1, int(math.Floor(s.max+1e-9)))
	if last < first {
		return nil
	}
	// Every n-th label when they do not fit, the bands' edges as minor ticks
	pxPerBand := length / (s.max - s.min)
	widest := 0.0
	for i := first; i <= last && i-first < 2000; i++ {
		widest = math.Max(widest, measure(s.categories[i]))
	}
	every := 1
	if pxPerBand > 0 {
		every = max(1, int(math.Ceil((widest+chartTickGap)/pxPerBand)))
	}
	var ticks []ChartTick
	for i := first; i <= last; i++ {
		t := ChartTick{Value: float64(i), Minor: true}
		if (i-first)%every == 0 {
			t.Minor, t.Label = false, s.categories[i]
		}
		ticks = append(ticks, t)
	}
	return ticks
}
