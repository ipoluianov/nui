package plotter

import (
	"fmt"
	"math"
	"strconv"
	"unicode"
)

// Vars are the values a formula is evaluated with.
type Vars struct {
	X, A, B float64
}

// Formula is a parsed expression of x and the parameters a and b.
type Formula struct {
	eval func(v *Vars) float64
}

// Eval computes the formula; the result is NaN or ±Inf where the function is
// not defined (sqrt(-1), 1/0).
func (f *Formula) Eval(v Vars) float64 {
	return f.eval(&v)
}

var functions = map[string]func(float64) float64{
	"sin": math.Sin, "cos": math.Cos, "tan": math.Tan,
	"sqrt": math.Sqrt, "abs": math.Abs, "exp": math.Exp,
	"ln": math.Log, "log": math.Log10, "floor": math.Floor,
}

var constants = map[string]float64{"pi": math.Pi, "e": math.E}

// Parse parses a formula such as "a*sin(b*x) + x^2/10". It knows
// + - * / ^, unary minus, parentheses, the variables x, a, b, the constants
// pi and e, and the functions sin cos tan sqrt abs exp ln log floor.
func Parse(src string) (*Formula, error) {
	p := parser{src: []rune(src)}
	p.next()
	if p.tok == tokEnd {
		return nil, fmt.Errorf("the formula is empty")
	}
	e, err := p.expr()
	if err != nil {
		return nil, err
	}
	if p.tok != tokEnd {
		return nil, p.unexpected()
	}
	return &Formula{eval: e}, nil
}

type token int

const (
	tokEnd token = iota
	tokNum
	tokIdent
	tokOp // one of + - * / ^ ( )
)

type node = func(v *Vars) float64

// parser is a recursive descent parser that compiles the formula into
// closures, so evaluating it for thousands of points is cheap.
type parser struct {
	src []rune
	pos int // position after the current token

	tok   token
	start int // position of the current token
	text  string
	num   float64
	err   error
}

func (p *parser) next() {
	for p.pos < len(p.src) && unicode.IsSpace(p.src[p.pos]) {
		p.pos++
	}
	p.start = p.pos
	if p.pos >= len(p.src) {
		p.tok, p.text = tokEnd, ""
		return
	}
	ch := p.src[p.pos]
	switch {
	case unicode.IsDigit(ch) || ch == '.':
		for p.pos < len(p.src) && (unicode.IsDigit(p.src[p.pos]) || p.src[p.pos] == '.') {
			p.pos++
		}
		// exponent: 1e-3
		if p.pos < len(p.src) && (p.src[p.pos] == 'e' || p.src[p.pos] == 'E') {
			end := p.pos + 1
			if end < len(p.src) && (p.src[end] == '+' || p.src[end] == '-') {
				end++
			}
			if end < len(p.src) && unicode.IsDigit(p.src[end]) {
				for end < len(p.src) && unicode.IsDigit(p.src[end]) {
					end++
				}
				p.pos = end
			}
		}
		p.tok, p.text = tokNum, string(p.src[p.start:p.pos])
		n, err := strconv.ParseFloat(p.text, 64)
		if err != nil {
			p.err = fmt.Errorf("bad number %q at %d", p.text, p.start+1)
		}
		p.num = n
	case unicode.IsLetter(ch) || ch == '_':
		for p.pos < len(p.src) && (unicode.IsLetter(p.src[p.pos]) || unicode.IsDigit(p.src[p.pos]) || p.src[p.pos] == '_') {
			p.pos++
		}
		p.tok, p.text = tokIdent, string(p.src[p.start:p.pos])
	default:
		p.pos++
		p.tok, p.text = tokOp, string(ch)
	}
}

func (p *parser) unexpected() error {
	if p.err != nil {
		return p.err
	}
	if p.tok == tokEnd {
		return fmt.Errorf("unexpected end of the formula")
	}
	return fmt.Errorf("unexpected %q at %d", p.text, p.start+1)
}

func (p *parser) isOp(op string) bool {
	return p.tok == tokOp && p.text == op
}

// expr := term {("+" | "-") term}
func (p *parser) expr() (node, error) {
	left, err := p.term()
	if err != nil {
		return nil, err
	}
	for p.isOp("+") || p.isOp("-") {
		op := p.text
		p.next()
		right, err := p.term()
		if err != nil {
			return nil, err
		}
		l := left
		if op == "+" {
			left = func(v *Vars) float64 { return l(v) + right(v) }
		} else {
			left = func(v *Vars) float64 { return l(v) - right(v) }
		}
	}
	return left, nil
}

// term := unary {("*" | "/") unary}
func (p *parser) term() (node, error) {
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for p.isOp("*") || p.isOp("/") {
		op := p.text
		p.next()
		right, err := p.unary()
		if err != nil {
			return nil, err
		}
		l := left
		if op == "*" {
			left = func(v *Vars) float64 { return l(v) * right(v) }
		} else {
			left = func(v *Vars) float64 { return l(v) / right(v) }
		}
	}
	return left, nil
}

// unary := ("-" | "+") unary | power
func (p *parser) unary() (node, error) {
	if p.isOp("-") || p.isOp("+") {
		neg := p.text == "-"
		p.next()
		operand, err := p.unary()
		if err != nil || !neg {
			return operand, err
		}
		return func(v *Vars) float64 { return -operand(v) }, nil
	}
	return p.power()
}

// power := primary ["^" unary]; right-associative, and -x^2 is -(x^2)
func (p *parser) power() (node, error) {
	base, err := p.primary()
	if err != nil {
		return nil, err
	}
	if !p.isOp("^") {
		return base, nil
	}
	p.next()
	exp, err := p.unary()
	if err != nil {
		return nil, err
	}
	return func(v *Vars) float64 { return math.Pow(base(v), exp(v)) }, nil
}

// primary := number | variable | constant | function "(" expr ")" | "(" expr ")"
func (p *parser) primary() (node, error) {
	switch {
	case p.err != nil:
		return nil, p.err
	case p.tok == tokNum:
		n := p.num
		p.next()
		return func(*Vars) float64 { return n }, nil
	case p.isOp("("):
		p.next()
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.isOp(")") {
			return nil, fmt.Errorf("missing ')' at %d", p.start+1)
		}
		p.next()
		return e, nil
	case p.tok == tokIdent:
		name, at := p.text, p.start+1
		p.next()
		switch name {
		case "x":
			return func(v *Vars) float64 { return v.X }, nil
		case "a":
			return func(v *Vars) float64 { return v.A }, nil
		case "b":
			return func(v *Vars) float64 { return v.B }, nil
		}
		if c, ok := constants[name]; ok {
			return func(*Vars) float64 { return c }, nil
		}
		f, ok := functions[name]
		if !ok {
			return nil, fmt.Errorf("unknown name %q at %d", name, at)
		}
		if !p.isOp("(") {
			return nil, fmt.Errorf("%s needs an argument: %s(...)", name, name)
		}
		arg, err := p.primary()
		if err != nil {
			return nil, err
		}
		return func(v *Vars) float64 { return f(arg(v)) }, nil
	}
	return nil, p.unexpected()
}
