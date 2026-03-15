package handlers

import (
	"fmt"
	"math"
	"strconv"
)

type Expr func(x float64) float64

func Parse(s string) (Expr, error) {
	p := &parser{s: s}
	fn, err := p.expr()
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	if p.pos < len(p.s) {
		return nil, fmt.Errorf("unexpected character %q at position %d", string(p.s[p.pos]), p.pos)
	}
	return fn, nil
}

type parser struct {
	s   string
	pos int
}

func (p *parser) skipSpace() {
	for p.pos < len(p.s) && p.s[p.pos] == ' ' {
		p.pos++
	}
}

func (p *parser) peek() byte {
	p.skipSpace()
	if p.pos >= len(p.s) {
		return 0
	}
	return p.s[p.pos]
}

func (p *parser) consume(ch byte) bool {
	if p.peek() == ch {
		p.pos++
		return true
	}
	return false
}

func (p *parser) expr() (Expr, error) {
	left, err := p.term()
	if err != nil {
		return nil, err
	}
	for {
		if p.consume('+') {
			right, err := p.term()
			if err != nil {
				return nil, err
			}
			l, r := left, right
			left = func(x float64) float64 { return l(x) + r(x) }
		} else if p.consume('-') {
			right, err := p.term()
			if err != nil {
				return nil, err
			}
			l, r := left, right
			left = func(x float64) float64 { return l(x) - r(x) }
		} else {
			break
		}
	}
	return left, nil
}

func (p *parser) term() (Expr, error) {
	left, err := p.unary()
	if err != nil {
		return nil, err
	}
	for {
		if p.consume('*') {
			right, err := p.unary()
			if err != nil {
				return nil, err
			}
			l, r := left, right
			left = func(x float64) float64 { return l(x) * r(x) }
		} else if p.consume('/') {
			right, err := p.unary()
			if err != nil {
				return nil, err
			}
			l, r := left, right
			left = func(x float64) float64 { return l(x) / r(x) }
		} else {
			break
		}
	}
	return left, nil
}

func (p *parser) unary() (Expr, error) {
	if p.consume('-') {
		operand, err := p.unary()
		if err != nil {
			return nil, err
		}
		return func(x float64) float64 { return -operand(x) }, nil
	}
	if p.consume('+') {
		return p.unary()
	}
	return p.power()
}

func (p *parser) power() (Expr, error) {
	base, err := p.call()
	if err != nil {
		return nil, err
	}
	if p.consume('^') {
		exp, err := p.unary()
		if err != nil {
			return nil, err
		}
		b := base
		return func(x float64) float64 { return math.Pow(b(x), exp(x)) }, nil
	}
	return base, nil
}

func (p *parser) call() (Expr, error) {
	p.skipSpace()
	if p.pos >= len(p.s) || !(p.s[p.pos] >= 'a' && p.s[p.pos] <= 'z' || p.s[p.pos] >= 'A' && p.s[p.pos] <= 'Z') {
		return p.primary()
	}
	start := p.pos
	for p.pos < len(p.s) && (p.s[p.pos] >= 'a' && p.s[p.pos] <= 'z' || p.s[p.pos] >= 'A' && p.s[p.pos] <= 'Z' || p.s[p.pos] >= '0' && p.s[p.pos] <= '9') {
		p.pos++
	}
	name := p.s[start:p.pos]

	if p.consume('(') {
		arg, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.consume(')') {
			return nil, fmt.Errorf("expected ')' after function argument")
		}
		fn, ok := builtins[name]
		if !ok {
			return nil, fmt.Errorf("unknown function %q", name)
		}
		return func(x float64) float64 { return fn(arg(x)) }, nil
	}

	switch name {
	case "x":
		return func(x float64) float64 { return x }, nil
	case "pi":
		return func(x float64) float64 { return math.Pi }, nil
	case "e":
		return func(x float64) float64 { return math.E }, nil
	default:
		return nil, fmt.Errorf("unknown variable %q", name)
	}
}

func (p *parser) primary() (Expr, error) {
	if p.consume('(') {
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		if !p.consume(')') {
			return nil, fmt.Errorf("expected ')'")
		}
		return e, nil
	}
	return p.number()
}

func (p *parser) number() (Expr, error) {
	p.skipSpace()
	start := p.pos
	for p.pos < len(p.s) && (p.s[p.pos] >= '0' && p.s[p.pos] <= '9' || p.s[p.pos] == '.') {
		p.pos++
	}
	if p.pos == start {
		return nil, fmt.Errorf("expected number at position %d", p.pos)
	}
	val, err := strconv.ParseFloat(p.s[start:p.pos], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q: %v", p.s[start:p.pos], err)
	}
	return func(x float64) float64 { return val }, nil
}

var builtins = map[string]func(float64) float64{
	"sin":   math.Sin,
	"cos":   math.Cos,
	"tan":   math.Tan,
	"asin":  math.Asin,
	"acos":  math.Acos,
	"atan":  math.Atan,
	"exp":   math.Exp,
	"log":   math.Log,
	"ln":    math.Log,
	"log2":  math.Log2,
	"log10": math.Log10,
	"sqrt":  math.Sqrt,
	"abs":   math.Abs,
	"floor": math.Floor,
	"ceil":  math.Ceil,
}
