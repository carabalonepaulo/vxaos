package script

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Compile parses source into a Program.
func Compile(name, source string) (*Program, error) {
	r := newReader(name, source)
	var forms []any
	for !r.eof() {
		form, err := r.readForm()
		if err != nil {
			return nil, err
		}
		forms = append(forms, form)
	}
	c := &compiler{name: name, source: source}
	if err := c.compileBody(forms); err != nil {
		return nil, err
	}
	c.emit(OpReturn, 0)
	return &Program{Proto: c.proto()}, nil
}

type reader struct {
	name string
	src  string
	pos  int
	line int
	col  int
}

func newReader(name, src string) *reader {
	return &reader{name: name, src: src, line: 1, col: 1}
}

func (r *reader) eof() bool { return r.pos >= len(r.src) }

func (r *reader) errf(format string, args ...any) error {
	return fmt.Errorf("script: %s:%d:%d: %s", r.name, r.line, r.col, fmt.Sprintf(format, args...))
}

func (r *reader) peek() rune {
	if r.eof() {
		return 0
	}
	ru, _ := utf8.DecodeRuneInString(r.src[r.pos:])
	return ru
}

func (r *reader) advance() rune {
	if r.eof() {
		return 0
	}
	ru, size := utf8.DecodeRuneInString(r.src[r.pos:])
	r.pos += size
	if ru == '\n' {
		r.line++
		r.col = 1
	} else {
		r.col++
	}
	return ru
}

func (r *reader) skipSpace() {
	for !r.eof() {
		ru := r.peek()
		if unicode.IsSpace(ru) {
			r.advance()
			continue
		}
		if ru == ';' {
			for !r.eof() && r.peek() != '\n' {
				r.advance()
			}
			continue
		}
		if ru == '#' && r.pos+1 < len(r.src) && r.src[r.pos+1] == '|' {
			r.advance()
			r.advance()
			for !r.eof() {
				if r.peek() == '|' && r.pos+1 < len(r.src) && r.src[r.pos+1] == '#' {
					r.advance()
					r.advance()
					break
				}
				r.advance()
			}
			continue
		}
		break
	}
}

func (r *reader) readForm() (any, error) {
	r.skipSpace()
	if r.eof() {
		return nil, r.errf("unexpected end of input")
	}
	ru := r.peek()
	switch ru {
	case '(':
		r.advance()
		list := &List{}
		for {
			r.skipSpace()
			if r.eof() {
				return nil, r.errf("unterminated list")
			}
			if r.peek() == ')' {
				r.advance()
				return list, nil
			}
			form, err := r.readForm()
			if err != nil {
				return nil, err
			}
			list.Elems = append(list.Elems, form)
		}
	case ')':
		return nil, r.errf("unexpected ')'")
	case '\'':
		r.advance()
		form, err := r.readForm()
		if err != nil {
			return nil, err
		}
		return &List{Elems: []any{Symbol("quote"), form}}, nil
	case '"':
		return r.readString()
	default:
		return r.readAtom()
	}
}

func (r *reader) readString() (any, error) {
	r.advance() // opening quote
	var b strings.Builder
	for {
		if r.eof() {
			return nil, r.errf("unterminated string")
		}
		ru := r.advance()
		switch ru {
		case '"':
			return b.String(), nil
		case '\\':
			if r.eof() {
				return nil, r.errf("unterminated escape")
			}
			esc := r.advance()
			switch esc {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '0':
				b.WriteByte(0)
			case '\\':
				b.WriteByte('\\')
			case '"':
				b.WriteByte('"')
			default:
				b.WriteRune(esc)
			}
		default:
			b.WriteRune(ru)
		}
	}
}

func (r *reader) readAtom() (any, error) {
	start := r.pos
	for !r.eof() {
		ru := r.peek()
		if unicode.IsSpace(ru) || ru == '(' || ru == ')' || ru == '\'' || ru == '"' || ru == ';' {
			break
		}
		r.advance()
	}
	text := r.src[start:r.pos]
	if text == "" {
		return nil, r.errf("empty atom")
	}
	switch text {
	case "nil":
		return nil, nil
	case "true", "#t":
		return true, nil
	case "false", "#f":
		return false, nil
	}
	if i, err := strconv.ParseInt(text, 10, 64); err == nil {
		return i, nil
	}
	if f, err := strconv.ParseFloat(text, 64); err == nil && (strings.ContainsAny(text, ".eE") || text == "inf" || text == "-inf" || text == "nan") {
		return f, nil
	}
	return Symbol(text), nil
}
