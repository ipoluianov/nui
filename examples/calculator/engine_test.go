package calculator

import "testing"

// press feeds keys to the engine: digits, ".", + - * /, "=", "C", "E" (CE),
// "<" (backspace), "n" (±), "%", "r" (√), "s" (x²), "i" (1/x)
func press(e *Engine, keys string) {
	for _, k := range keys {
		switch {
		case k >= '0' && k <= '9':
			e.Digit(int(k - '0'))
		case k == '.':
			e.Point()
		case k == '+':
			e.Operator(OpAdd)
		case k == '-':
			e.Operator(OpSub)
		case k == '*':
			e.Operator(OpMul)
		case k == '/':
			e.Operator(OpDiv)
		case k == '=':
			e.Equals()
		case k == 'C':
			e.Clear()
		case k == 'E':
			e.ClearEntry()
		case k == '<':
			e.Backspace()
		case k == 'n':
			e.Negate()
		case k == '%':
			e.Percent()
		case k == 'r':
			e.Sqrt()
		case k == 's':
			e.Square()
		case k == 'i':
			e.Reciprocal()
		}
	}
}

func TestEngine(t *testing.T) {
	tests := []struct {
		keys, display, expr string
	}{
		{"", "0", ""},
		{"12+7=", "19", "12 + 7 ="},
		{"2+3*4=", "20", "5 × 4 ="}, // left to right
		{"12+7", "7", "12 +"},
		{"12+7+", "19", "19 +"},
		{"2+-", "2", "2 −"}, // the operator is replaced
		{"2+3===", "11", "8 + 3 ="},
		{"10-4=1=", "-3", "1 − 4 ="}, // a new number repeats the last op
		{"5+=", "10", "5 + 5 ="},
		{"7/0=", "Cannot divide by zero", "7 ÷ 0 ="},
		{"7/0=3+4=", "7", "3 + 4 ="},
		{"0.1+0.2=", "0.3", "0.1 + 0.2 ="},
		{"1234567", "1,234,567", ""},
		{"1.5.5", "1.55", ""},
		{".5", "0.5", ""},
		{"123<<", "1", ""},
		{"123<<<<", "0", ""},
		{"5n", "-5", ""},
		{"5nn", "5", ""},
		{"9+5E3=", "12", "9 + 3 ="},
		{"9+5C", "0", ""},
		{"200+10%", "20", "200 +"},
		{"200+10%=", "220", "200 + 20 ="},
		{"50%", "0.5", ""},
		{"16r", "4", ""},
		{"4nr", "Invalid input", ""},
		{"3s", "9", ""},
		{"4i", "0.25", ""},
		{"0i", "Cannot divide by zero", ""},
		{"3s+1=", "10", "9 + 1 ="},
		{"99999999*99999999*99999999=", "9.9999997e+23", "9999999800000000 × 99999999 ="},
		{"1234567890123456789", "1,234,567,890,123,456", ""}, // 16 digits max
	}
	for _, tt := range tests {
		e := NewEngine()
		press(e, tt.keys)
		if e.Display() != tt.display || e.Expression() != tt.expr {
			t.Errorf("%q: got %q / %q, want %q / %q", tt.keys, e.Display(), e.Expression(), tt.display, tt.expr)
		}
	}
}

func TestMemoryAndHistory(t *testing.T) {
	e := NewEngine()
	press(e, "5")
	e.MemoryAdd()
	press(e, "3")
	e.MemoryAdd()
	press(e, "1")
	e.MemorySubtract()
	press(e, "C")
	if !e.HasMemory() {
		t.Fatal("C must keep the memory")
	}
	e.MemoryRecall()
	if e.Display() != "7" {
		t.Errorf("MR: got %q, want 7", e.Display())
	}
	press(e, "*2=")
	e.MemoryClear()
	if e.HasMemory() {
		t.Error("MC must clear the memory")
	}

	h := e.History()
	if len(h) != 1 || h[0].Expression != "7 × 2 =" || h[0].Result != "14" {
		t.Errorf("history: %+v", h)
	}
	e.ClearHistory()
	if len(e.History()) != 0 {
		t.Error("history not cleared")
	}
}

func TestSetText(t *testing.T) {
	tests := []struct {
		text  string
		value string
		ok    bool
	}{
		{"42", "42", true},
		{" 1,234.5 ", "1234.5", true},
		{"1 234,5", "1234.5", true},
		{"−3", "-3", true},
		{"2e3", "2000", true},
		{"abc", "0", false},
		{"", "0", false},
	}
	for _, tt := range tests {
		e := NewEngine()
		ok := e.SetText(tt.text)
		if ok != tt.ok || e.Text() != tt.value {
			t.Errorf("SetText(%q): got %q, %v; want %q, %v", tt.text, e.Text(), ok, tt.value, tt.ok)
		}
	}
	// A pasted number is the second operand
	e := NewEngine()
	press(e, "10+")
	e.SetText("5")
	press(e, "=")
	if e.Display() != "15" {
		t.Errorf("10 + paste 5 =: got %q", e.Display())
	}
}
