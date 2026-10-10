package calculator

import (
	"math"
	"strconv"
	"strings"
)

// Engine is the calculator logic without any UI: a standard calculator that
// evaluates left to right (2 + 3 × 4 = 20), repeats the last operation on
// a repeated "=", and has a memory register and a history of results.
type Engine struct {
	display string // the number shown, unformatted ("-12.5")
	typing  bool   // digits are appended to display
	operand bool   // display holds a value entered after the last operator
	expr    string // the pending expression shown above the number
	err     string // error message; the engine waits for C, CE or a digit

	acc float64 // left operand of op
	op  rune    // pending operator: 0, '+', '-', '×', '÷'

	lastOp      rune // for a repeated "="
	lastOperand float64

	memory    float64
	hasMemory bool

	history []HistoryEntry
}

// HistoryEntry is one finished calculation.
type HistoryEntry struct {
	Expression string // "12 + 7 ="
	Result     string // "19"
}

// maxDigits limits the length of a typed number.
const maxDigits = 16

// Operators accepted by Operator.
const (
	OpAdd = '+'
	OpSub = '−'
	OpMul = '×'
	OpDiv = '÷'
)

const errDivByZero = "Cannot divide by zero"

// NewEngine returns a calculator showing 0.
func NewEngine() *Engine {
	return &Engine{display: "0"}
}

// Display is the current number with thousands separators, or the error.
func (e *Engine) Display() string {
	if e.err != "" {
		return e.err
	}
	return groupThousands(e.display)
}

// Text is the current number without separators, for the clipboard.
func (e *Engine) Text() string {
	if e.err != "" {
		return ""
	}
	return e.display
}

// Expression is the pending expression: "12 + 7 =".
func (e *Engine) Expression() string { return e.expr }

// HasError reports whether an error is shown.
func (e *Engine) HasError() bool { return e.err != "" }

// HasMemory reports whether the memory register holds a value.
func (e *Engine) HasMemory() bool { return e.hasMemory }

// History returns the finished calculations, the oldest first.
func (e *Engine) History() []HistoryEntry { return e.history }

// ClearHistory forgets the finished calculations.
func (e *Engine) ClearHistory() { e.history = nil }

func (e *Engine) value() float64 {
	v, _ := strconv.ParseFloat(e.display, 64)
	return v
}

// setValue shows a computed value; it is an operand, but typing replaces it.
func (e *Engine) setValue(v float64) bool {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		e.fail("Overflow")
		return false
	}
	e.display = formatNumber(v)
	e.typing = false
	e.operand = true
	return true
}

func (e *Engine) fail(msg string) {
	e.err = msg
	e.op = 0
	e.lastOp = 0
	e.typing = false
}

// Digit types a digit 0-9.
func (e *Engine) Digit(d int) {
	if e.err != "" {
		e.Clear()
	}
	if !e.typing {
		if e.op == 0 {
			e.expr = "" // a new calculation after "="
		}
		e.display = "0"
		e.typing = true
	}
	if countDigits(e.display) >= maxDigits {
		return
	}
	if e.display == "0" {
		e.display = strconv.Itoa(d)
	} else if e.display == "-0" {
		e.display = "-" + strconv.Itoa(d)
	} else {
		e.display += strconv.Itoa(d)
	}
	e.operand = true
}

// Point types the decimal point.
func (e *Engine) Point() {
	if e.err != "" {
		e.Clear()
	}
	if !e.typing {
		e.Digit(0)
	}
	if !strings.Contains(e.display, ".") {
		e.display += "."
	}
}

// Operator sets the pending operator, first evaluating the one before it.
func (e *Engine) Operator(op rune) {
	if e.err != "" {
		return
	}
	v := e.value()
	switch {
	case e.op != 0 && e.operand:
		r, ok := e.apply(e.acc, e.op, v)
		if !ok {
			return
		}
		e.acc = r
	case e.op == 0:
		e.acc = v
	}
	e.op = op
	e.display = formatNumber(e.acc)
	e.typing = false
	e.operand = false
	e.expr = formatNumber(e.acc) + " " + string(op)
}

// Equals evaluates the pending operation; without one it repeats the last.
func (e *Engine) Equals() {
	if e.err != "" {
		return
	}
	a, b, op := e.acc, e.value(), e.op
	if op == 0 {
		if e.lastOp == 0 {
			e.expr = formatNumber(b) + " ="
			e.typing = false
			return
		}
		a, b, op = b, e.lastOperand, e.lastOp
	}
	expr := formatNumber(a) + " " + string(op) + " " + formatNumber(b) + " ="
	r, ok := e.apply(a, op, b)
	e.expr = expr
	if !ok {
		return
	}
	e.lastOp, e.lastOperand = op, b
	e.op = 0
	e.acc = r
	if e.setValue(r) {
		e.operand = false
		e.history = append(e.history, HistoryEntry{expr, e.display})
	}
}

func (e *Engine) apply(a float64, op rune, b float64) (float64, bool) {
	var r float64
	switch op {
	case OpAdd:
		r = a + b
	case OpSub:
		r = a - b
	case OpMul:
		r = a * b
	case OpDiv:
		if b == 0 {
			e.fail(errDivByZero)
			return 0, false
		}
		r = a / b
	}
	if math.IsInf(r, 0) || math.IsNaN(r) {
		e.fail("Overflow")
		return 0, false
	}
	return r, true
}

// Clear (C) resets the calculation; memory and history are kept.
func (e *Engine) Clear() {
	*e = Engine{display: "0", memory: e.memory, hasMemory: e.hasMemory, history: e.history}
}

// ClearEntry (CE) clears the current number, keeping the pending operation.
func (e *Engine) ClearEntry() {
	if e.err != "" {
		e.Clear()
		return
	}
	if e.op == 0 {
		e.expr = ""
	}
	e.display = "0"
	e.typing = true
	e.operand = true
}

// Backspace erases the last typed character.
func (e *Engine) Backspace() {
	if e.err != "" {
		e.Clear()
		return
	}
	if !e.typing {
		if e.op == 0 {
			e.expr = "" // after "=" it only clears the expression
		}
		return
	}
	e.display = e.display[:len(e.display)-1]
	if e.display == "" || e.display == "-" {
		e.display = "0"
	}
}

// Negate (±) changes the sign of the current number.
func (e *Engine) Negate() {
	if e.err != "" {
		return
	}
	if e.typing {
		if strings.HasPrefix(e.display, "-") {
			e.display = e.display[1:]
		} else if e.display != "0" {
			e.display = "-" + e.display
		}
		return
	}
	e.setValue(-e.value())
}

// Percent: in "a + b %" and "a − b %" b becomes b percent of a, otherwise
// b / 100.
func (e *Engine) Percent() {
	if e.err != "" {
		return
	}
	v := e.value()
	if e.op == OpAdd || e.op == OpSub {
		e.setValue(e.acc * v / 100)
	} else {
		e.setValue(v / 100)
	}
}

// Sqrt replaces the current number with its square root.
func (e *Engine) Sqrt() {
	if e.err != "" {
		return
	}
	if v := e.value(); v < 0 {
		e.fail("Invalid input")
	} else {
		e.setValue(math.Sqrt(v))
	}
}

// Square replaces the current number with its square.
func (e *Engine) Square() {
	if e.err == "" {
		v := e.value()
		e.setValue(v * v)
	}
}

// Reciprocal replaces the current number with 1/x.
func (e *Engine) Reciprocal() {
	if e.err != "" {
		return
	}
	if v := e.value(); v == 0 {
		e.fail(errDivByZero)
	} else {
		e.setValue(1 / v)
	}
}

// SetText enters a number given as text (pasted, recalled from the
// history): "1,234.5", "1 234,5" and "-2e3" are accepted. It reports
// whether the text was a number.
func (e *Engine) SetText(text string) bool {
	v, ok := ParseNumber(text)
	if !ok {
		return false
	}
	if e.err != "" {
		e.Clear()
	}
	if e.op == 0 {
		e.expr = ""
	}
	return e.setValue(v)
}

// MemoryClear (MC) empties the memory.
func (e *Engine) MemoryClear() { e.memory, e.hasMemory = 0, false }

// MemoryRecall (MR) enters the number in the memory.
func (e *Engine) MemoryRecall() {
	if e.hasMemory {
		e.SetText(formatNumber(e.memory))
	}
}

// MemoryAdd (M+) adds the current number to the memory.
func (e *Engine) MemoryAdd() { e.memoryChange(1) }

// MemorySubtract (M−) subtracts the current number from the memory.
func (e *Engine) MemorySubtract() { e.memoryChange(-1) }

func (e *Engine) memoryChange(sign float64) {
	if e.err != "" {
		return
	}
	e.memory += sign * e.value()
	e.hasMemory = true
	e.typing = false // the next digit starts a new number
}

// ParseNumber parses a number the way people write it: with spaces,
// thousands separators or a decimal comma.
func ParseNumber(text string) (float64, bool) {
	s := strings.NewReplacer(" ", "", " ", "", "_", "", "'", "").Replace(strings.TrimSpace(text))
	s = strings.Replace(s, "−", "-", 1)
	if strings.Contains(s, ".") {
		s = strings.ReplaceAll(s, ",", "")
	} else if strings.Count(s, ",") == 1 {
		s = strings.Replace(s, ",", ".", 1)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, false
	}
	return v, true
}

// formatNumber rounds to 15 significant digits, hiding the binary
// fractions (0.1 + 0.2 shows 0.3), and switches to the exponent form for
// very large and very small numbers.
func formatNumber(v float64) string {
	r, _ := strconv.ParseFloat(strconv.FormatFloat(v, 'g', 15, 64), 64)
	if r == 0 {
		return "0"
	}
	if a := math.Abs(r); a < 1e16 && a >= 1e-9 {
		return strconv.FormatFloat(r, 'f', -1, 64)
	}
	return strconv.FormatFloat(r, 'g', 15, 64)
}

// groupThousands inserts commas into the integer part: "1234567.5" ->
// "1,234,567.5". Numbers in the exponent form are left as they are.
func groupThousands(s string) string {
	if strings.ContainsAny(s, "eE") {
		return s
	}
	sign := ""
	if strings.HasPrefix(s, "-") {
		sign, s = "-", s[1:]
	}
	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return sign + b.String() + frac
}

func countDigits(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n++
		}
	}
	return n
}
