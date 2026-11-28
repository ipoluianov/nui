package ui

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

func auditTry(t *testing.T, name string, f func()) {
	t.Helper()
	done := make(chan string, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Sprintf("PANIC %v\n%s", r, debug.Stack())
				return
			}
			done <- ""
		}()
		f()
	}()
	select {
	case s := <-done:
		if s != "" {
			t.Errorf("%s: %s", name, s)
		}
	case <-time.After(10 * time.Second):
		t.Errorf("%s: HANG (>10s)", name)
	}
}

func meas(s string) float64 { return float64(len(s)) * 7 }

// runCase runs f inline, reporting panics and slow runs
func runCase(t *testing.T, name string, f func()) {
	t.Helper()
	for _, sk := range strings.Split(os.Getenv("AUDIT_SKIP"), ",") {
		if sk != "" && strings.HasPrefix(name, sk) {
			return
		}
	}
	fmt.Fprintln(os.Stderr, "CASE", name)
	start := time.Now()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				msg := fmt.Sprintf("%s: PANIC %v\n%s", name, r, debug.Stack())
				fmt.Fprintln(os.Stderr, msg)
				t.Error(msg)
			}
		}()
		f()
	}()
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		buf := make([]byte, 1<<20)
		n := runtime.Stack(buf, true)
		fmt.Fprintf(os.Stderr, "HANG %s\n%s\n", name, buf[:n])
		os.Exit(3)
	}
	if d := time.Since(start); d > 3*time.Second {
		fmt.Fprintf(os.Stderr, "SLOW %s %v\n", name, d)
	}
}

func TestAuditScales(t *testing.T) {
	vals := []float64{0, 1, -1, 5e-324, 1e-300, 1e300, -math.MaxFloat64, math.MaxFloat64, math.NaN(), math.Inf(1), math.Inf(-1), 1e16, 9.2e18, -9.2e18, 1e20}
	for _, a := range vals {
		for _, b := range vals {
			a, b := a, b
			for _, L := range []float64{0, 1, 300, 1e6, math.Inf(1), math.NaN()} {
				L := L
				runCase(t, fmt.Sprintf("linear %v %v L=%v", a, b, L), func() {
					s := NewLinearScale(a, b)
					s.Ticks(L, meas)
					s.Format(a)
					s.Integer = true
					s.Ticks(L, meas)
					s.SI = true
					s.Ticks(L, meas)
				})
				runCase(t, fmt.Sprintf("log %v %v L=%v", a, b, L), func() {
					s := NewLogScale(a, b)
					s.Ticks(L, meas)
					s.Format(a)
					s.Base = 2
					s.Ticks(L, meas)
				})
				if math.Abs(a) > 1e308 && a == b { continue }
				runCase(t, fmt.Sprintf("symlog %v %v L=%v", a, b, L), func() {
					s := NewSymlogScale(a, b)
					s.Ticks(L, meas)
					s.Format(a)
				})
				runCase(t, fmt.Sprintf("time %v %v L=%v", a, b, L), func() {
					s := &TimeScale{Location: time.UTC}
					s.SetDomain(a, b)
					s.Ticks(L, meas)
					s.DateRibbon(L, meas)
					s.Format(a)
				})
				runCase(t, fmt.Sprintf("cat %v %v L=%v", a, b, L), func() {
					s := NewCategoryScale("a", "b", "c")
					s.SetDomain(a, b)
					s.Ticks(L, meas)
					s.Format(a)
				})
			}
		}
	}
}
