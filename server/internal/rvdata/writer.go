package rvdata

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"reflect"
	"strconv"
	"unicode/utf8"
)

// Encode serializes v as a Ruby Marshal 4.8 document.
//
// Object identity is not preserved: shared, acyclic values are written as
// independent copies. Cyclic graphs are rejected with an error. RPG Maker data
// files are acyclic, so this keeps the writer simple and deterministic.
func Encode(v any) ([]byte, error) {
	e := &encoder{stack: make(map[uintptr]int)}
	e.buf = append(e.buf, MarshalMajor, MarshalMinor)
	if err := e.writeValue(v); err != nil {
		return nil, err
	}
	return e.buf, nil
}

// SaveFile encodes v and writes it to path.
func SaveFile(path string, v any) error {
	data, err := Encode(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

type encoder struct {
	buf   []byte
	stack map[uintptr]int
}

func (e *encoder) errf(format string, args ...any) error {
	return fmt.Errorf("rvdata: %s", fmt.Sprintf(format, args...))
}

func (e *encoder) byte(b byte) {
	e.buf = append(e.buf, b)
}

func (e *encoder) bytes(b []byte) {
	e.buf = append(e.buf, b...)
}

func (e *encoder) enter(v any) (uintptr, error) {
	ptr := reflect.ValueOf(v).Pointer()
	if ptr == 0 {
		return 0, e.errf("nil pointer in value tree")
	}
	if e.stack[ptr] > 0 {
		return 0, e.errf("cyclic object graph is not supported")
	}
	e.stack[ptr]++
	return ptr, nil
}

func (e *encoder) leave(ptr uintptr) {
	e.stack[ptr]--
	if e.stack[ptr] == 0 {
		delete(e.stack, ptr)
	}
}

func (e *encoder) writeLong(v int64) {
	switch {
	case v == 0:
		e.byte(0)
	case v >= 1 && v <= 122:
		e.byte(byte(v + 5))
	case v >= -122 && v <= -1:
		e.byte(byte(v - 5))
	default:
		if v < math.MinInt32 || v > math.MaxInt32 {
			e.writeBignumInt(v)
			return
		}
		var tmp [8]byte
		n := 0
		value := v
		for {
			tmp[n] = byte(value & 0xff)
			value >>= 8
			n++
			if value == 0 && v > 0 {
				break
			}
			if value == -1 && v < 0 {
				break
			}
		}
		if v > 0 {
			e.byte(byte(n))
		} else {
			e.byte(byte(-n))
		}
		e.bytes(tmp[:n])
	}
}

func (e *encoder) writeBignumInt(v int64) {
	var value *big.Int
	if v < 0 {
		value = big.NewInt(v)
	} else {
		value = big.NewInt(v)
	}
	e.writeBignumValue(value)
}

func (e *encoder) writeBignumValue(v *big.Int) {
	if v == nil {
		v = new(big.Int)
	}
	sign := byte('+')
	mag := new(big.Int).Set(v)
	if mag.Sign() < 0 {
		sign = '-'
		mag.Neg(mag)
	}
	be := mag.Bytes() // big endian, no leading zeros
	// Marshal stores the magnitude as little-endian 16-bit words.
	words := (len(be) + 1) / 2
	raw := make([]byte, words*2)
	for i := range be {
		raw[len(be)-1-i] = be[i]
	}
	e.byte('l')
	e.byte(sign)
	e.writeLong(int64(words))
	e.bytes(raw)
}

func (e *encoder) writeRawStringBytes(b []byte) {
	e.writeLong(int64(len(b)))
	e.bytes(b)
}

func (e *encoder) writeSymbol(name string) {
	isASCII := true
	for i := 0; i < len(name); i++ {
		if name[i] >= 0x80 {
			isASCII = false
			break
		}
	}
	if isASCII {
		e.byte(':')
		e.writeRawStringBytes([]byte(name))
		return
	}
	// Ruby 3.x wraps non-ASCII symbols in an ivar wrapper with :E true.
	e.byte('I')
	e.byte(':')
	e.writeRawStringBytes([]byte(name))
	e.writeLong(1)
	e.writeSymbol("E")
	e.byte('T')
}

func (e *encoder) writeString(s string) {
	if utf8.ValidString(s) {
		e.byte('I')
		e.byte('"')
		e.writeRawStringBytes([]byte(s))
		e.writeLong(1)
		e.writeSymbol("E")
		e.byte('T')
		return
	}
	e.byte('"')
	e.writeRawStringBytes([]byte(s))
}

func (e *encoder) writeFixnum(v int64) {
	if v < math.MinInt32 || v > math.MaxInt32 {
		e.writeBignumInt(v)
		return
	}
	e.byte('i')
	e.writeLong(v)
}

func (e *encoder) writeValue(v any) error {
	switch x := v.(type) {
	case nil:
		e.byte('0')
	case bool:
		if x {
			e.byte('T')
		} else {
			e.byte('F')
		}
	case int:
		e.writeFixnum(int64(x))
	case int8:
		e.writeFixnum(int64(x))
	case int16:
		e.writeFixnum(int64(x))
	case int32:
		e.writeFixnum(int64(x))
	case int64:
		e.writeFixnum(x)
	case uint:
		e.writeFixnum(int64(x))
	case uint8:
		e.writeFixnum(int64(x))
	case uint16:
		e.writeFixnum(int64(x))
	case uint32:
		e.writeFixnum(int64(x))
	case uint64:
		if x > math.MaxInt64 {
			e.writeBignumValue(new(big.Int).SetUint64(x))
		} else {
			e.writeFixnum(int64(x))
		}
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return e.errf("cannot encode non-finite float %v", x)
		}
		s := strconv.FormatFloat(x, 'g', -1, 64)
		e.byte('f')
		e.writeRawStringBytes([]byte(s))
	case float32:
		return e.writeValue(float64(x))
	case string:
		e.writeString(x)
	case Bytes:
		e.byte('"')
		e.writeRawStringBytes(x)
	case Symbol:
		e.writeSymbol(string(x))
	case *Array:
		ptr, err := e.enter(x)
		if err != nil {
			return err
		}
		defer e.leave(ptr)
		e.byte('[')
		e.writeLong(int64(len(x.Elems)))
		for _, el := range x.Elems {
			if err := e.writeValue(el); err != nil {
				return err
			}
		}
	case *Hash:
		ptr, err := e.enter(x)
		if err != nil {
			return err
		}
		defer e.leave(ptr)
		if x.HasDefault {
			e.byte('}')
		} else {
			e.byte('{')
		}
		e.writeLong(int64(len(x.Pairs)))
		for _, p := range x.Pairs {
			if err := e.writeValue(p.Key); err != nil {
				return err
			}
			if err := e.writeValue(p.Value); err != nil {
				return err
			}
		}
		if x.HasDefault {
			if err := e.writeValue(x.Default); err != nil {
				return err
			}
		}
	case *Object:
		ptr, err := e.enter(x)
		if err != nil {
			return err
		}
		defer e.leave(ptr)
		if x.Struct {
			e.byte('S')
		} else {
			e.byte('o')
		}
		e.writeSymbol(x.Class)
		e.writeLong(int64(len(x.Ivars)))
		for _, iv := range x.Ivars {
			e.writeSymbol(iv.Name)
			if err := e.writeValue(iv.Value); err != nil {
				return err
			}
		}
	case *Table:
		raw := x.Bytes()
		e.byte('u')
		e.writeSymbol("Table")
		e.writeRawStringBytes(raw)
	case *Color:
		raw := x.Bytes()
		e.byte('u')
		e.writeSymbol("Color")
		e.writeRawStringBytes(raw)
	case *Tone:
		raw := x.Bytes()
		e.byte('u')
		e.writeSymbol("Tone")
		e.writeRawStringBytes(raw)
	case *UserDefined:
		e.byte('u')
		e.writeSymbol(x.Class)
		e.writeRawStringBytes(x.Data)
	case *UserMarshal:
		ptr, err := e.enter(x)
		if err != nil {
			return err
		}
		defer e.leave(ptr)
		e.byte('U')
		e.writeSymbol(x.Class)
		if err := e.writeValue(x.Value); err != nil {
			return err
		}
	case *UserClass:
		ptr, err := e.enter(x)
		if err != nil {
			return err
		}
		defer e.leave(ptr)
		e.byte('C')
		e.writeSymbol(x.Class)
		if err := e.writeValue(x.Value); err != nil {
			return err
		}
	case *Bignum:
		e.writeBignumValue(x.Value)
	case *Regexp:
		e.byte('/')
		e.writeRawStringBytes([]byte(x.Source))
		e.byte(x.Options)
	case ClassRef:
		if x.Module {
			e.byte('m')
		} else {
			e.byte('c')
		}
		e.writeSymbol(x.Name)
	case *big.Int:
		e.writeBignumValue(x)
	default:
		return e.errf("unsupported value type %T", v)
	}
	return nil
}
