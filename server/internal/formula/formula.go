// Package formula converts RPG Maker's Ruby damage formulas to the Lisp form
// used by the server. The converted text is stored in the .rvdata2 formula
// field itself; the runtime never parses Ruby.
package formula

import "strings"

// ToLisp translates the arithmetic subset used by RPG Maker formulas into an
// S-expression. Unsupported constructs return false so the caller can report
// them instead of silently producing wrong damage.
func ToLisp(source string) (string, bool) {
	p := &emitter{text: source}
	value, ok := p.expr()
	if !ok {
		return "", false
	}
	p.skip()
	if p.pos != len(p.text) {
		return "", false
	}
	return value, true
}

// IsLisp reports whether source already looks like an S-expression or a plain
// number, so migrations are idempotent.
func IsLisp(source string) bool {
	text := strings.TrimSpace(source)
	if text == "" {
		return true
	}
	if text[0] == '(' || text[0] == '-' || text[0] == '+' {
		return true
	}
	c := text[0]
	return c >= '0' && c <= '9' || c == '.'
}

type emitter struct {
	text string
	pos  int
}

func (p *emitter) skip() {
	for p.pos < len(p.text) {
		switch p.text[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *emitter) peek() byte {
	if p.pos >= len(p.text) {
		return 0
	}
	return p.text[p.pos]
}

func (p *emitter) expr() (string, bool) {
	value, ok := p.term()
	if !ok {
		return "", false
	}
	for {
		p.skip()
		op := p.peek()
		if op != '+' && op != '-' {
			return value, true
		}
		p.pos++
		rhs, ok := p.term()
		if !ok {
			return "", false
		}
		value = "(" + string(op) + " " + value + " " + rhs + ")"
	}
}

func (p *emitter) term() (string, bool) {
	value, ok := p.factor()
	if !ok {
		return "", false
	}
	for {
		p.skip()
		op := p.peek()
		if op != '*' && op != '/' && op != '%' {
			return value, true
		}
		p.pos++
		rhs, ok := p.factor()
		if !ok {
			return "", false
		}
		value = "(" + string(op) + " " + value + " " + rhs + ")"
	}
}

func (p *emitter) factor() (string, bool) {
	p.skip()
	if p.pos >= len(p.text) {
		return "", false
	}
	switch p.peek() {
	case '-':
		p.pos++
		value, ok := p.factor()
		if !ok {
			return "", false
		}
		return "(- " + value + ")", true
	case '+':
		p.pos++
		return p.factor()
	case '(':
		p.pos++
		value, ok := p.expr()
		if !ok {
			return "", false
		}
		p.skip()
		if p.peek() != ')' {
			return "", false
		}
		p.pos++
		return value, true
	}
	if p.peek() >= '0' && p.peek() <= '9' || p.peek() == '.' {
		start := p.pos
		for p.pos < len(p.text) && ((p.text[p.pos] >= '0' && p.text[p.pos] <= '9') || p.text[p.pos] == '.') {
			p.pos++
		}
		return p.text[start:p.pos], true
	}
	if !isIdentStart(p.peek()) {
		return "", false
	}
	name := p.ident()
	p.skip()
	if p.peek() == '.' {
		p.pos++
		field := p.ident()
		if name == "Math" {
			p.skip()
			if p.peek() != '(' {
				return "", false
			}
			p.pos++
			arg, ok := p.expr()
			if !ok {
				return "", false
			}
			p.skip()
			if p.peek() != ')' {
				return "", false
			}
			p.pos++
			switch field {
			case "sqrt", "abs", "floor", "ceil":
				return "(" + field + " " + arg + ")", true
			default:
				return "", false
			}
		}
		return "(get " + name + " \"" + field + "\")", true
	}
	if p.peek() == '[' {
		p.pos++
		index, ok := p.expr()
		if !ok {
			return "", false
		}
		p.skip()
		if p.peek() != ']' {
			return "", false
		}
		p.pos++
		return "(get " + name + " " + index + ")", true
	}
	if p.peek() == '(' {
		p.pos++
		arg, ok := p.expr()
		if !ok {
			return "", false
		}
		p.skip()
		if p.peek() != ')' {
			return "", false
		}
		p.pos++
		switch name {
		case "sqrt", "abs", "floor", "ceil":
			return "(" + name + " " + arg + ")", true
		}
		return "", false
	}
	return "0", true
}

func (p *emitter) ident() string {
	start := p.pos
	for p.pos < len(p.text) && isIdentPart(p.text[p.pos]) {
		p.pos++
	}
	return p.text[start:p.pos]
}

func isIdentStart(c byte) bool {
	return c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || c >= '0' && c <= '9'
}
