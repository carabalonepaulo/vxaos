// Package rvdata reads and writes the Ruby Marshal 4.8 documents used by
// RPG Maker VX Ace (.rvdata2).
//
// The package intentionally implements the wire format, not Ruby semantics:
// objects are represented as class name plus ordered instance variables, so a
// document can be inspected and re-encoded without knowing every RPG::* class.
// Typed adapters live in higher level packages.
package rvdata

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
	"unicode/utf8"
)

// Symbol is a Ruby symbol. Its string value is the symbol name.
type Symbol string

// Ivar is one ordered instance variable of an Object.
type Ivar struct {
	Name  string
	Value any
}

// Object is a Ruby object with a class name and ordered instance variables.
// Struct is true when the document used Marshal's TYPE_STRUCT for it.
type Object struct {
	Class      string
	Ivars      []Ivar
	Struct     bool
	Extensions []string
}

// Get returns the named instance variable.
func (o *Object) Get(name string) (any, bool) {
	if o == nil {
		return nil, false
	}
	for _, iv := range o.Ivars {
		if iv.Name == name {
			return iv.Value, true
		}
	}
	return nil, false
}

// MustGet returns the named instance variable or nil.
func (o *Object) MustGet(name string) any {
	v, _ := o.Get(name)
	return v
}

// Set replaces an instance variable, preserving its position, or appends it.
func (o *Object) Set(name string, value any) {
	for i := range o.Ivars {
		if o.Ivars[i].Name == name {
			o.Ivars[i].Value = value
			return
		}
	}
	o.Ivars = append(o.Ivars, Ivar{Name: name, Value: value})
}

// Delete removes an instance variable.
func (o *Object) Delete(name string) {
	for i := range o.Ivars {
		if o.Ivars[i].Name == name {
			o.Ivars = append(o.Ivars[:i], o.Ivars[i+1:]...)
			return
		}
	}
}

// String renders a compact debug representation.
func (o *Object) String() string {
	if o == nil {
		return "nil"
	}
	return o.Class
}

// Array is a Ruby array.
type Array struct {
	Elems []any
}

// Len returns the array length.
func (a *Array) Len() int {
	if a == nil {
		return 0
	}
	return len(a.Elems)
}

// At returns the element at i, or nil.
func (a *Array) At(i int) any {
	if a == nil || i < 0 || i >= len(a.Elems) {
		return nil
	}
	return a.Elems[i]
}

// Append adds an element.
func (a *Array) Append(v any) {
	a.Elems = append(a.Elems, v)
}

// HashPair is one ordered key/value entry of a Hash.
type HashPair struct {
	Key   any
	Value any
}

// Hash is a Ruby hash, preserving insertion order. HasDefault distinguishes
// Marshal's TYPE_HASH_DEF from a plain Hash whose default is nil.
type Hash struct {
	Pairs      []HashPair
	Default    any
	HasDefault bool
}

// Get finds a key by Ruby value equality for the scalar key types used by
// RPG Maker data (symbols, strings, integers, booleans and nil).
func (h *Hash) Get(key any) (any, bool) {
	if h == nil {
		return nil, false
	}
	for _, p := range h.Pairs {
		if equalKey(p.Key, key) {
			return p.Value, true
		}
	}
	return nil, false
}

// Set replaces a key or appends it.
func (h *Hash) Set(key, value any) {
	for i := range h.Pairs {
		if equalKey(h.Pairs[i].Key, key) {
			h.Pairs[i].Value = value
			return
		}
	}
	h.Pairs = append(h.Pairs, HashPair{Key: key, Value: value})
}

// Delete removes a key.
func (h *Hash) Delete(key any) {
	for i := range h.Pairs {
		if equalKey(h.Pairs[i].Key, key) {
			h.Pairs = append(h.Pairs[:i], h.Pairs[i+1:]...)
			return
		}
	}
}

// Has reports whether key exists.
func (h *Hash) Has(key any) bool {
	_, ok := h.Get(key)
	return ok
}

func equalKey(a, b any) bool {
	switch x := a.(type) {
	case Symbol:
		y, ok := b.(Symbol)
		return ok && x == y
	case string:
		switch y := b.(type) {
		case string:
			return x == y
		case Symbol:
			return x == string(y)
		}
	case int64:
		switch y := b.(type) {
		case int64:
			return x == y
		case int:
			return x == int64(y)
		}
	case int:
		return equalKey(int64(x), b)
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case nil:
		return b == nil
	}
	return false
}

// Table is the RGSS Table user-defined object. Dims is the first uint32 of the
// serialized payload: 1 for a one-dimensional table, 2 for two, 3 for three.
type Table struct {
	Dims  int32
	XSize int32
	YSize int32
	ZSize int32
	Data  []uint16
}

// NewTable creates a zeroed table.
func NewTable(xsize, ysize, zsize int32) *Table {
	if xsize < 0 {
		xsize = 0
	}
	if ysize < 0 {
		ysize = 0
	}
	if zsize < 0 {
		zsize = 0
	}
	dims := int32(1)
	if zsize > 1 {
		dims = 3
	} else if ysize > 1 {
		dims = 2
	}
	n := int64(xsize) * int64(ysize) * int64(zsize)
	if n < 0 {
		n = 0
	}
	return &Table{
		Dims:  dims,
		XSize: xsize,
		YSize: ysize,
		ZSize: zsize,
		Data:  make([]uint16, n),
	}
}

// Index returns the linear index of (x,y,z).
func (t *Table) Index(x, y, z int32) (int, bool) {
	if t == nil || x < 0 || y < 0 || z < 0 || x >= t.XSize || y >= t.YSize || z >= t.ZSize {
		return 0, false
	}
	i := int64(x) + int64(y)*int64(t.XSize) + int64(z)*int64(t.XSize)*int64(t.YSize)
	if i < 0 || i >= int64(len(t.Data)) {
		return 0, false
	}
	return int(i), true
}

// At returns the cell value or zero when out of bounds.
func (t *Table) At(x, y, z int32) uint16 {
	i, ok := t.Index(x, y, z)
	if !ok {
		return 0
	}
	return t.Data[i]
}

// Set writes a cell when in bounds.
func (t *Table) Set(x, y, z int32, v uint16) {
	if i, ok := t.Index(x, y, z); ok {
		t.Data[i] = v
	}
}

// Bytes serializes the table in RGSS _dump format.
func (t *Table) Bytes() []byte {
	if t == nil {
		return nil
	}
	// The payload is five uint32 words followed by uint16 cells, little endian.
	out := make([]byte, 20+len(t.Data)*2)
	binary.LittleEndian.PutUint32(out[0:], uint32(t.Dims))
	binary.LittleEndian.PutUint32(out[4:], uint32(t.XSize))
	binary.LittleEndian.PutUint32(out[8:], uint32(t.YSize))
	binary.LittleEndian.PutUint32(out[12:], uint32(t.ZSize))
	binary.LittleEndian.PutUint32(out[16:], uint32(len(t.Data)))
	for i, v := range t.Data {
		binary.LittleEndian.PutUint16(out[20+i*2:], v)
	}
	return out
}

// ParseTable decodes an RGSS Table payload.
func ParseTable(data []byte) (*Table, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("table payload too short: %d", len(data))
	}
	dims := int32(binary.LittleEndian.Uint32(data[0:]))
	xs := int32(binary.LittleEndian.Uint32(data[4:]))
	ys := int32(binary.LittleEndian.Uint32(data[8:]))
	zs := int32(binary.LittleEndian.Uint32(data[12:]))
	count := binary.LittleEndian.Uint32(data[16:])
	if xs < 0 || ys < 0 || zs < 0 {
		return nil, fmt.Errorf("table has negative dimensions: %dx%dx%d", xs, ys, zs)
	}
	product := int64(xs) * int64(ys) * int64(zs)
	if product < 0 || int64(count) != product {
		return nil, fmt.Errorf("table cell count %d does not match dimensions %dx%dx%d", count, xs, ys, zs)
	}
	if uint64(len(data)-20) < uint64(count)*2 {
		return nil, fmt.Errorf("table data truncated: need %d bytes, have %d", uint64(count)*2, len(data)-20)
	}
	cells := make([]uint16, count)
	for i := range cells {
		cells[i] = binary.LittleEndian.Uint16(data[20+i*2:])
	}
	return &Table{Dims: dims, XSize: xs, YSize: ys, ZSize: zs, Data: cells}, nil
}

// Color is the RGSS Color user-defined object.
type Color struct {
	Red   float64
	Green float64
	Blue  float64
	Alpha float64
}

// Tone is the RGSS Tone user-defined object.
type Tone struct {
	Red   float64
	Green float64
	Blue  float64
	Gray  float64
}

func parseDoubles4(data []byte) ([4]float64, error) {
	var out [4]float64
	if len(data) != 32 {
		return out, fmt.Errorf("expected 32 bytes of doubles, got %d", len(data))
	}
	for i := 0; i < 4; i++ {
		out[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8:]))
	}
	return out, nil
}

func doubles4Bytes(a, b, c, d float64) []byte {
	out := make([]byte, 32)
	binary.LittleEndian.PutUint64(out[0:], math.Float64bits(a))
	binary.LittleEndian.PutUint64(out[8:], math.Float64bits(b))
	binary.LittleEndian.PutUint64(out[16:], math.Float64bits(c))
	binary.LittleEndian.PutUint64(out[24:], math.Float64bits(d))
	return out
}

// ParseColor decodes an RGSS Color payload.
func ParseColor(data []byte) (*Color, error) {
	v, err := parseDoubles4(data)
	if err != nil {
		return nil, err
	}
	return &Color{Red: v[0], Green: v[1], Blue: v[2], Alpha: v[3]}, nil
}

// Bytes serializes the color.
func (c *Color) Bytes() []byte {
	return doubles4Bytes(c.Red, c.Green, c.Blue, c.Alpha)
}

// ParseTone decodes an RGSS Tone payload.
func ParseTone(data []byte) (*Tone, error) {
	v, err := parseDoubles4(data)
	if err != nil {
		return nil, err
	}
	return &Tone{Red: v[0], Green: v[1], Blue: v[2], Gray: v[3]}, nil
}

// Bytes serializes the tone.
func (t *Tone) Bytes() []byte {
	return doubles4Bytes(t.Red, t.Green, t.Blue, t.Gray)
}

// UserDefined is an unknown Marshal TYPE_USERDEF value. Data is the raw
// _dump payload.
type UserDefined struct {
	Class string
	Data  []byte
}

// UserMarshal is a Marshal TYPE_USRMARSHAL value (marshal_dump).
type UserMarshal struct {
	Class string
	Value any
}

// UserClass is a Marshal TYPE_UCLASS value.
type UserClass struct {
	Class string
	Value any
}

// Bignum is a Ruby Bignum. Value is the arbitrary precision signed integer.
type Bignum struct {
	Value *big.Int
}

// Regexp is a Ruby Regexp.
type Regexp struct {
	Source  string
	Options byte
}

// ClassRef is a Marshal TYPE_CLASS/TYPE_MODULE reference.
type ClassRef struct {
	Name   string
	Module bool
}

// Bytes is a binary string. The writer emits it without the UTF-8 encoding
// instance-variable wrapper, which is required for zlib payloads.
type Bytes []byte

// AsObject asserts v is an *Object.
func AsObject(v any) (*Object, bool) {
	o, ok := v.(*Object)
	return o, ok
}

// AsArray asserts v is an *Array.
func AsArray(v any) (*Array, bool) {
	a, ok := v.(*Array)
	return a, ok
}

// AsHash asserts v is a *Hash.
func AsHash(v any) (*Hash, bool) {
	h, ok := v.(*Hash)
	return h, ok
}

// AsString returns a string or Symbol as a string.
func AsString(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case Symbol:
		return string(x), true
	}
	return "", false
}

// AsInt returns an integer value.
func AsInt(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		return x, true
	case int:
		return int64(x), true
	case int32:
		return int64(x), true
	case uint32:
		return int64(x), true
	case float64:
		if x == math.Trunc(x) {
			return int64(x), true
		}
	}
	return 0, false
}

// AsFloat returns a float value.
func AsFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int64:
		return float64(x), true
	case int:
		return float64(x), true
	}
	return 0, false
}

// AsBool returns a boolean.
func AsBool(v any) (bool, bool) {
	b, ok := v.(bool)
	return b, ok
}

// IvarString helps typed adapters read a string ivar.
func IvarString(o *Object, name string) string {
	v, _ := o.Get(name)
	s, _ := AsString(v)
	return s
}

// IvarInt helps typed adapters read an integer ivar.
func IvarInt(o *Object, name string) int64 {
	v, _ := o.Get(name)
	n, _ := AsInt(v)
	return n
}

// IvarFloat helps typed adapters read a float ivar.
func IvarFloat(o *Object, name string) float64 {
	v, _ := o.Get(name)
	f, _ := AsFloat(v)
	return f
}

// IvarBool helps typed adapters read a boolean ivar.
func IvarBool(o *Object, name string) bool {
	v, _ := o.Get(name)
	b, _ := AsBool(v)
	return b
}

// IvarObject helps typed adapters read an object ivar.
func IvarObject(o *Object, name string) *Object {
	v, _ := o.Get(name)
	obj, _ := v.(*Object)
	return obj
}

// IvarArray helps typed adapters read an array ivar.
func IvarArray(o *Object, name string) *Array {
	v, _ := o.Get(name)
	arr, _ := v.(*Array)
	return arr
}

// IvarHash helps typed adapters read a hash ivar.
func IvarHash(o *Object, name string) *Hash {
	v, _ := o.Get(name)
	h, _ := v.(*Hash)
	return h
}

// IvarTable helps typed adapters read a Table ivar.
func IvarTable(o *Object, name string) *Table {
	v, _ := o.Get(name)
	t, _ := v.(*Table)
	return t
}

// ErrUnsupported reports a Marshal feature this package does not implement.
var ErrUnsupported = errors.New("rvdata: unsupported marshal feature")

// IsValidUTF8 reports whether s can be serialized as a UTF-8 Ruby string.
func IsValidUTF8(s string) bool {
	return utf8.ValidString(s)
}

// SymbolName normalizes a class or ivar name.
func SymbolName(s string) string {
	return strings.TrimPrefix(s, ":")
}
