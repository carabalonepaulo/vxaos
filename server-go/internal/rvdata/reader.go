package rvdata

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"strconv"
	"strings"
)

// MarshalMajor and MarshalMinor are the Ruby Marshal version this package
// understands.
const (
	MarshalMajor = 4
	MarshalMinor = 8
)

// Decode parses one Ruby Marshal 4.8 document.
func Decode(data []byte) (any, error) {
	d := &decoder{data: data}
	if err := d.readHeader(); err != nil {
		return nil, err
	}
	v, err := d.readValue()
	if err != nil {
		return nil, err
	}
	if d.pos != len(d.data) {
		return nil, d.errf("trailing %d bytes after document", len(d.data)-d.pos)
	}
	return v, nil
}

// LoadFile reads and decodes path.
func LoadFile(path string) (any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Decode(data)
}

type decoder struct {
	data    []byte
	pos     int
	objects []any
	symbols []string
}

func (d *decoder) errf(format string, args ...any) error {
	return fmt.Errorf("rvdata: %s at offset %d", fmt.Sprintf(format, args...), d.pos)
}

func (d *decoder) readHeader() error {
	if len(d.data) < 2 {
		return d.errf("short header")
	}
	major, minor := d.data[0], d.data[1]
	if major != MarshalMajor || minor != MarshalMinor {
		return d.errf("unsupported marshal version %d.%d", major, minor)
	}
	d.pos = 2
	return nil
}

func (d *decoder) readByte() (byte, error) {
	if d.pos >= len(d.data) {
		return 0, d.errf("unexpected end of input")
	}
	b := d.data[d.pos]
	d.pos++
	return b, nil
}

func (d *decoder) readN(n int) ([]byte, error) {
	if n < 0 {
		return nil, d.errf("negative length %d", n)
	}
	if n > len(d.data)-d.pos {
		return nil, d.errf("length %d exceeds remaining input", n)
	}
	b := d.data[d.pos : d.pos+n]
	d.pos += n
	return b, nil
}

// readLong decodes Ruby Marshal's packed integer. The encoding is:
//
//	0            -> 0
//	1..4         -> that many little-endian bytes (positive)
//	-1..-4       -> that many little-endian bytes, sign-extended
//	5..127       -> lead-5
//	-128..-5     -> lead+5
func (d *decoder) readLong() (int64, error) {
	b, err := d.readByte()
	if err != nil {
		return 0, err
	}
	lead := int8(b)
	switch {
	case lead == 0:
		return 0, nil
	case lead >= 1 && lead <= 4:
		n := int(lead)
		buf, err := d.readN(n)
		if err != nil {
			return 0, err
		}
		var v uint64
		for i := 0; i < n; i++ {
			v |= uint64(buf[i]) << (8 * i)
		}
		return int64(v), nil
	case lead <= -1 && lead >= -4:
		n := int(-lead)
		buf, err := d.readN(n)
		if err != nil {
			return 0, err
		}
		var v uint64
		for i := 0; i < n; i++ {
			v |= uint64(buf[i]) << (8 * i)
		}
		// Sign-extend from n bytes.
		shift := uint(64 - 8*n)
		return int64(v<<shift) >> shift, nil
	case lead > 0:
		return int64(lead) - 5, nil
	default:
		return int64(lead) + 5, nil
	}
}

func (d *decoder) readRawString() ([]byte, error) {
	n, err := d.readLong()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		return nil, d.errf("negative string length %d", n)
	}
	if n > int64(len(d.data)-d.pos) {
		return nil, d.errf("string length %d exceeds remaining input", n)
	}
	b := d.data[d.pos : d.pos+int(n)]
	d.pos += int(n)
	return b, nil
}

func (d *decoder) readSymbolFromTag(tag byte) (Symbol, error) {
	switch tag {
	case ':':
		raw, err := d.readRawString()
		if err != nil {
			return "", err
		}
		name := string(raw)
		d.symbols = append(d.symbols, name)
		return Symbol(name), nil
	case ';':
		idx, err := d.readLong()
		if err != nil {
			return "", err
		}
		if idx < 0 || idx >= int64(len(d.symbols)) {
			return "", d.errf("invalid symbol link %d", idx)
		}
		return Symbol(d.symbols[idx]), nil
	default:
		return "", d.errf("expected symbol, got tag %q", tag)
	}
}

func (d *decoder) readSymbol() (Symbol, error) {
	tag, err := d.readByte()
	if err != nil {
		return "", err
	}
	return d.readSymbolFromTag(tag)
}

func (d *decoder) register(v any) int {
	idx := len(d.objects)
	d.objects = append(d.objects, v)
	return idx
}

func (d *decoder) readValue() (any, error) {
	tag, err := d.readByte()
	if err != nil {
		return nil, err
	}
	switch tag {
	case '0':
		return nil, nil
	case 'T':
		return true, nil
	case 'F':
		return false, nil
	case 'i':
		return d.readLong()
	case 'l':
		return d.readBignum()
	case 'f':
		raw, err := d.readRawString()
		if err != nil {
			return nil, err
		}
		// RGSS-era Marshal floats carry a short binary tail after a NUL byte
		// (e.g. "0.94999999999999996\x00ff"). Ruby's strtod stops at the NUL;
		// do the same instead of rejecting the trailing bytes.
		if i := bytes.IndexByte(raw, 0); i >= 0 {
			raw = raw[:i]
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(string(raw)), 64)
		if err != nil {
			return nil, d.errf("invalid float %q", string(raw))
		}
		d.register(f)
		return f, nil
	case ':', ';':
		return d.readSymbolFromTag(tag)
	case '"':
		raw, err := d.readRawString()
		if err != nil {
			return nil, err
		}
		s := string(raw)
		d.register(s)
		return s, nil
	case '[':
		return d.readArray()
	case '{', '}':
		return d.readHash(tag == '}')
	case 'o', 'S':
		return d.readObject(tag == 'S')
	case 'u':
		return d.readUserDefined()
	case 'U':
		return d.readUserMarshal()
	case 'C':
		return d.readUserClass()
	case 'I':
		return d.readIVarWrapper()
	case 'e':
		return d.readExtended()
	case '@':
		idx, err := d.readLong()
		if err != nil {
			return nil, err
		}
		if idx < 0 || idx >= int64(len(d.objects)) {
			return nil, d.errf("invalid object link %d", idx)
		}
		return d.objects[idx], nil
	case 'c':
		name, err := d.readSymbol()
		if err != nil {
			return nil, err
		}
		return ClassRef{Name: string(name)}, nil
	case 'm', 'M':
		name, err := d.readSymbol()
		if err != nil {
			return nil, err
		}
		return ClassRef{Name: string(name), Module: true}, nil
	case '/':
		raw, err := d.readRawString()
		if err != nil {
			return nil, err
		}
		opts, err := d.readByte()
		if err != nil {
			return nil, err
		}
		re := &Regexp{Source: string(raw), Options: opts}
		d.register(re)
		return re, nil
	default:
		return nil, d.errf("unsupported tag %q", tag)
	}
}

func (d *decoder) readBignum() (any, error) {
	sign, err := d.readByte()
	if err != nil {
		return nil, err
	}
	if sign != '+' && sign != '-' {
		return nil, d.errf("invalid bignum sign %q", sign)
	}
	words, err := d.readLong()
	if err != nil {
		return nil, err
	}
	if words < 0 || words > int64(len(d.data)-d.pos)/2 {
		return nil, d.errf("invalid bignum word count %d", words)
	}
	raw, err := d.readN(int(words) * 2)
	if err != nil {
		return nil, err
	}
	// Marshal stores the magnitude little endian; math/big wants big endian.
	be := make([]byte, len(raw))
	for i := range raw {
		be[len(raw)-1-i] = raw[i]
	}
	v := new(big.Int).SetBytes(be)
	if sign == '-' && v.Sign() != 0 {
		v.Neg(v)
	}
	b := &Bignum{Value: v}
	d.register(b)
	return b, nil
}

func (d *decoder) readArray() (any, error) {
	n, err := d.readLong()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		n = 0
	}
	if n > int64(len(d.data)-d.pos) {
		return nil, d.errf("array length %d exceeds remaining input", n)
	}
	arr := &Array{Elems: make([]any, 0, int(n))}
	d.register(arr)
	for i := int64(0); i < n; i++ {
		v, err := d.readValue()
		if err != nil {
			return nil, err
		}
		arr.Elems = append(arr.Elems, v)
	}
	return arr, nil
}

func (d *decoder) readHash(withDefault bool) (any, error) {
	n, err := d.readLong()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		n = 0
	}
	if n > int64(len(d.data)-d.pos) {
		return nil, d.errf("hash length %d exceeds remaining input", n)
	}
	h := &Hash{Pairs: make([]HashPair, 0, int(n)), HasDefault: withDefault}
	d.register(h)
	for i := int64(0); i < n; i++ {
		k, err := d.readValue()
		if err != nil {
			return nil, err
		}
		v, err := d.readValue()
		if err != nil {
			return nil, err
		}
		h.Pairs = append(h.Pairs, HashPair{Key: k, Value: v})
	}
	if withDefault {
		def, err := d.readValue()
		if err != nil {
			return nil, err
		}
		h.Default = def
	}
	return h, nil
}

func (d *decoder) readObject(isStruct bool) (any, error) {
	class, err := d.readSymbol()
	if err != nil {
		return nil, err
	}
	n, err := d.readLong()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		n = 0
	}
	if n > int64(len(d.data)-d.pos) {
		return nil, d.errf("object ivar count %d exceeds remaining input", n)
	}
	obj := &Object{Class: string(class), Ivars: make([]Ivar, 0, int(n)), Struct: isStruct}
	d.register(obj)
	for i := int64(0); i < n; i++ {
		name, err := d.readSymbol()
		if err != nil {
			return nil, err
		}
		value, err := d.readValue()
		if err != nil {
			return nil, err
		}
		obj.Ivars = append(obj.Ivars, Ivar{Name: string(name), Value: value})
	}
	return obj, nil
}

func (d *decoder) readUserDefined() (any, error) {
	class, err := d.readSymbol()
	if err != nil {
		return nil, err
	}
	raw, err := d.readRawString()
	if err != nil {
		return nil, err
	}
	var v any
	switch string(class) {
	case "Table":
		t, err := ParseTable(raw)
		if err != nil {
			return nil, d.errf("invalid Table: %v", err)
		}
		v = t
	case "Color":
		c, err := ParseColor(raw)
		if err != nil {
			return nil, d.errf("invalid Color: %v", err)
		}
		v = c
	case "Tone":
		t, err := ParseTone(raw)
		if err != nil {
			return nil, d.errf("invalid Tone: %v", err)
		}
		v = t
	default:
		data := make([]byte, len(raw))
		copy(data, raw)
		v = &UserDefined{Class: string(class), Data: data}
	}
	d.register(v)
	return v, nil
}

func (d *decoder) readUserMarshal() (any, error) {
	class, err := d.readSymbol()
	if err != nil {
		return nil, err
	}
	v := &UserMarshal{Class: string(class)}
	d.register(v)
	value, err := d.readValue()
	if err != nil {
		return nil, err
	}
	v.Value = value
	return v, nil
}

func (d *decoder) readUserClass() (any, error) {
	class, err := d.readSymbol()
	if err != nil {
		return nil, err
	}
	v := &UserClass{Class: string(class)}
	d.register(v)
	value, err := d.readValue()
	if err != nil {
		return nil, err
	}
	v.Value = value
	return v, nil
}

func (d *decoder) readIVarWrapper() (any, error) {
	inner, err := d.readValue()
	if err != nil {
		return nil, err
	}
	n, err := d.readLong()
	if err != nil {
		return nil, err
	}
	if n < 0 {
		n = 0
	}
	if n > int64(len(d.data)-d.pos) {
		return nil, d.errf("ivar count %d exceeds remaining input", n)
	}
	for i := int64(0); i < n; i++ {
		name, err := d.readSymbol()
		if err != nil {
			return nil, err
		}
		value, err := d.readValue()
		if err != nil {
			return nil, err
		}
		// Ruby wraps strings and symbols with :E (UTF-8) / :encoding; the Go
		// representation already stores text as Go strings. For objects, the
		// wrapper carries real instance variables and must be merged.
		if obj, ok := inner.(*Object); ok {
			obj.Set(string(name), value)
		}
	}
	return inner, nil
}

func (d *decoder) readExtended() (any, error) {
	module, err := d.readSymbol()
	if err != nil {
		return nil, err
	}
	inner, err := d.readValue()
	if err != nil {
		return nil, err
	}
	if obj, ok := inner.(*Object); ok {
		obj.Extensions = append(obj.Extensions, string(module))
	}
	return inner, nil
}

// ErrTrailingData is returned when a document has bytes after its root value.
var ErrTrailingData = errors.New("rvdata: trailing data")

// FloatBits is a helper for tests.
func FloatBits(f float64) uint64 { return math.Float64bits(f) }
