package plotter

import (
	"math"
	"os"
	"strings"
	"testing"
)

func TestEval(t *testing.T) {
	vars := Vars{X: 2, A: 3, B: 0.5}
	tests := []struct {
		src  string
		want float64
	}{
		{"1 + 2 * 3", 7},
		{"(1 + 2) * 3", 9},
		{"10 - 4 - 3", 3},
		{"12 / 4 / 3", 1},
		{"2 ^ 3 ^ 2", 512},
		{"-x^2", -4},
		{"2^-1", 0.5},
		{"--x", 2},
		{"x * a + b", 6.5},
		{"a*sin(b*x)", 3 * math.Sin(1)},
		{"sqrt(16) + abs(-2)", 6},
		{"exp(0) + ln(e) + log(100)", 4},
		{"floor(2.7) + cos(0) + tan(0)", 3},
		{"2*pi", 2 * math.Pi},
		{"1.5e2 + .5", 150.5},
		{"sin(x)*exp(-x/10)", math.Sin(2) * math.Exp(-0.2)},
	}
	for _, tt := range tests {
		f, err := Parse(tt.src)
		if err != nil {
			t.Errorf("Parse(%q): %v", tt.src, err)
			continue
		}
		if got := f.Eval(vars); math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("%q = %v, want %v", tt.src, got, tt.want)
		}
	}
}

func TestUndefined(t *testing.T) {
	f, _ := Parse("sin(x)/x")
	if v := f.Eval(Vars{}); !math.IsNaN(v) {
		t.Errorf("sin(0)/0 = %v, want NaN", v)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		src, msg string
	}{
		{"", "empty"},
		{"   ", "empty"},
		{"1 +", "end of the formula"},
		{"(x + 1", "missing ')'"},
		{"x + )", `unexpected ")" at 5`},
		{"y * 2", `unknown name "y"`},
		{"sin x", "sin needs an argument"},
		{"x 2", `unexpected "2"`},
		{"1.2.3", "bad number"},
		{"x $ 2", `unexpected "$"`},
	}
	for _, tt := range tests {
		_, err := Parse(tt.src)
		if err == nil {
			t.Errorf("Parse(%q) succeeded, want an error", tt.src)
			continue
		}
		if !strings.Contains(err.Error(), tt.msg) {
			t.Errorf("Parse(%q) error %q, want it to contain %q", tt.src, err, tt.msg)
		}
	}
}

func TestSampleAndCSV(t *testing.T) {
	f, _ := Parse("sqrt(x)")
	points := sample(f, Vars{}, -1, 4, 6) // x = -1 is skipped
	if len(points) != 5 || points[4].Y != 2 {
		t.Fatalf("sample: %v", points)
	}
	path := t.TempDir() + "/plot.csv"
	if err := writeCSV(path, "sqrt(x)", points); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	want := "x,\"f(x) = sqrt(x)\"\n0,0\n1,1\n2,1.4142135623730951\n3,1.7320508075688772\n4,2\n"
	if string(data) != want {
		t.Errorf("CSV:\n%s\nwant:\n%s", data, want)
	}
}
