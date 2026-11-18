package variables

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// num is a calculation value. Integers stay integers so {{$calc 2 + 3}} is 5, not 5.0.
type num struct {
	i     int64
	f     float64
	isInt bool
}

func intNum(i int64) num     { return num{i: i, isInt: true} }
func floatNum(f float64) num { return num{f: f} }

func (n num) float() float64 {
	if n.isInt {
		return float64(n.i)
	}
	return n.f
}

func (n num) String() string {
	if n.isInt {
		return strconv.FormatInt(n.i, 10)
	}
	// Rounding to 10 places hides binary noise: 0.1*3 is 0.3, not 0.30000000000000004.
	f := n.f
	if math.Abs(f) < 1e15 {
		f = math.Round(f*1e10) / 1e10
	}
	if f == math.Trunc(f) && math.Abs(f) < 1e15 {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// calc evaluates a fixed arithmetic grammar: numbers, variable names, + - * / %, parentheses and
// round, floor, ceil, abs, min, max. It cannot call anything else. lookup resolves variable
// names; nil means "validate only", where every name counts as 1.
type calc struct {
	src    string
	pos    int
	lookup func(string) (string, bool)
}

func evalCalc(src string, lookup func(string) (string, bool)) (string, error) {
	c := &calc{src: src, lookup: lookup}
	v, err := c.expr()
	if err != nil {
		return "", err
	}
	c.skip()
	if c.pos < len(c.src) {
		return "", fmt.Errorf("unexpected %q in %q", string(c.src[c.pos]), src)
	}
	if !v.isInt && (math.IsNaN(v.f) || math.IsInf(v.f, 0)) {
		return "", fmt.Errorf("the result is not a finite number")
	}
	return v.String(), nil
}

func (c *calc) skip() {
	for c.pos < len(c.src) && (c.src[c.pos] == ' ' || c.src[c.pos] == '\t') {
		c.pos++
	}
}

func (c *calc) peek() byte {
	c.skip()
	if c.pos >= len(c.src) {
		return 0
	}
	return c.src[c.pos]
}

func (c *calc) expr() (num, error) {
	left, err := c.term()
	if err != nil {
		return num{}, err
	}
	for {
		op := c.peek()
		if op != '+' && op != '-' {
			return left, nil
		}
		c.pos++
		right, err := c.term()
		if err != nil {
			return num{}, err
		}
		left = arith(op, left, right)
	}
}

func (c *calc) term() (num, error) {
	left, err := c.unary()
	if err != nil {
		return num{}, err
	}
	for {
		op := c.peek()
		if op != '*' && op != '/' && op != '%' {
			return left, nil
		}
		c.pos++
		right, err := c.unary()
		if err != nil {
			return num{}, err
		}
		if (op == '/' || op == '%') && right.float() == 0 {
			return num{}, fmt.Errorf("division by zero")
		}
		left = arith(op, left, right)
	}
}

func (c *calc) unary() (num, error) {
	switch c.peek() {
	case '-':
		c.pos++
		v, err := c.unary()
		if err != nil {
			return num{}, err
		}
		if v.isInt {
			return intNum(-v.i), nil
		}
		return floatNum(-v.f), nil
	case '+':
		c.pos++
		return c.unary()
	}
	return c.primary()
}

func (c *calc) primary() (num, error) {
	ch := c.peek()
	switch {
	case ch == 0:
		return num{}, fmt.Errorf("the expression ends where a value was expected")
	case ch == '(':
		c.pos++
		v, err := c.expr()
		if err != nil {
			return num{}, err
		}
		if c.peek() != ')' {
			return num{}, fmt.Errorf("missing )")
		}
		c.pos++
		return v, nil
	case ch >= '0' && ch <= '9' || ch == '.':
		start := c.pos
		for c.pos < len(c.src) && (c.src[c.pos] >= '0' && c.src[c.pos] <= '9' || c.src[c.pos] == '.') {
			c.pos++
		}
		n, ok := parseNum(c.src[start:c.pos])
		if !ok {
			return num{}, fmt.Errorf("%q is not a number", c.src[start:c.pos])
		}
		return n, nil
	case unicode.IsLetter(rune(ch)) || ch == '_':
		start := c.pos
		for c.pos < len(c.src) {
			r := rune(c.src[c.pos])
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.' {
				c.pos++
				continue
			}
			break
		}
		name := c.src[start:c.pos]
		if c.peek() == '(' {
			c.pos++
			return c.function(name)
		}
		return c.variable(name)
	}
	return num{}, fmt.Errorf("unexpected %q", string(ch))
}

func (c *calc) variable(name string) (num, error) {
	if c.lookup == nil {
		return intNum(1), nil
	}
	text, ok := c.lookup(name)
	if !ok {
		return num{}, fmt.Errorf("unknown variable %q", name)
	}
	n, ok := parseNum(strings.TrimSpace(text))
	if !ok {
		return num{}, fmt.Errorf("variable %q is %q, which is not a number", name, text)
	}
	return n, nil
}

func (c *calc) function(name string) (num, error) {
	var args []num
	if c.peek() == ')' {
		c.pos++
	} else {
		for {
			v, err := c.expr()
			if err != nil {
				return num{}, err
			}
			args = append(args, v)
			if c.peek() == ',' {
				c.pos++
				continue
			}
			if c.peek() != ')' {
				return num{}, fmt.Errorf("missing ) after the arguments of %s", name)
			}
			c.pos++
			break
		}
	}
	want := func(lo, hi int) error {
		if len(args) < lo || len(args) > hi {
			return fmt.Errorf("%s takes %d to %d arguments, got %d", name, lo, hi, len(args))
		}
		return nil
	}
	switch name {
	case "round":
		if err := want(1, 2); err != nil {
			return num{}, err
		}
		places := 0
		if len(args) == 2 {
			if !args[1].isInt || args[1].i < 0 || args[1].i > 10 {
				return num{}, fmt.Errorf("round: decimals must be a whole number from 0 to 10")
			}
			places = int(args[1].i)
		}
		if args[0].isInt {
			return args[0], nil
		}
		p := math.Pow(10, float64(places))
		r := math.Round(args[0].f*p) / p
		if places == 0 {
			return intNum(int64(r)), nil
		}
		return floatNum(r), nil
	case "floor", "ceil", "abs":
		if err := want(1, 1); err != nil {
			return num{}, err
		}
		a := args[0]
		if name == "abs" {
			if a.isInt {
				if a.i < 0 {
					return intNum(-a.i), nil
				}
				return a, nil
			}
			return floatNum(math.Abs(a.f)), nil
		}
		if a.isInt {
			return a, nil
		}
		if name == "floor" {
			return intNum(int64(math.Floor(a.f))), nil
		}
		return intNum(int64(math.Ceil(a.f))), nil
	case "min", "max":
		if len(args) < 1 {
			return num{}, fmt.Errorf("%s needs at least one argument", name)
		}
		best := args[0]
		for _, a := range args[1:] {
			if name == "min" && a.float() < best.float() || name == "max" && a.float() > best.float() {
				best = a
			}
		}
		return best, nil
	}
	return num{}, fmt.Errorf("unknown function %s (use round, floor, ceil, abs, min, max)", name)
}

func parseNum(s string) (num, bool) {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return intNum(i), true
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil && !math.IsNaN(f) && !math.IsInf(f, 0) {
		return floatNum(f), true
	}
	return num{}, false
}

func arith(op byte, a, b num) num {
	if a.isInt && b.isInt {
		switch op {
		case '+':
			if s := a.i + b.i; (s > a.i) == (b.i > 0) {
				return intNum(s)
			}
		case '-':
			if s := a.i - b.i; (s < a.i) == (b.i > 0) {
				return intNum(s)
			}
		case '*':
			if a.i == 0 || b.i == 0 {
				return intNum(0)
			}
			if p := a.i * b.i; p/b.i == a.i && !(a.i == -1 && b.i == math.MinInt64) && !(b.i == -1 && a.i == math.MinInt64) {
				return intNum(p)
			}
		case '/':
			if a.i%b.i == 0 {
				return intNum(a.i / b.i)
			}
		case '%':
			return intNum(a.i % b.i)
		}
	}
	x, y := a.float(), b.float()
	switch op {
	case '+':
		return floatNum(x + y)
	case '-':
		return floatNum(x - y)
	case '*':
		return floatNum(x * y)
	case '/':
		return floatNum(x / y)
	}
	return floatNum(math.Mod(x, y))
}
