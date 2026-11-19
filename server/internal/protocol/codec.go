package protocol

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf8"
)

// Writer encodes a protocol payload. Methods record the first error and
// subsequent calls become no-ops; call Done to retrieve it.
type Writer struct {
	buf []byte
	err error
}

// NewWriter creates an empty payload writer.
func NewWriter() *Writer { return &Writer{} }

func (w *Writer) fail(format string, args ...any) {
	if w.err == nil {
		w.err = fmt.Errorf("protocol: "+format, args...)
	}
}

// U8 writes an unsigned byte.
func (w *Writer) U8(v uint8) {
	if w.err == nil {
		w.buf = append(w.buf, v)
	}
}

// Bool writes a boolean as one byte.
func (w *Writer) Bool(v bool) {
	if v {
		w.U8(1)
	} else {
		w.U8(0)
	}
}

// U16 writes a little-endian uint16.
func (w *Writer) U16(v uint16) {
	if w.err != nil {
		return
	}
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

// I16 writes a little-endian int16.
func (w *Writer) I16(v int16) { w.U16(uint16(v)) }

// U32 writes a little-endian uint32.
func (w *Writer) U32(v uint32) {
	if w.err != nil {
		return
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.buf = append(w.buf, b[:]...)
}

// I32 writes a little-endian int32.
func (w *Writer) I32(v int32) { w.U32(uint32(v)) }

// I64 writes a little-endian int64.
func (w *Writer) I64(v int64) {
	if w.err != nil {
		return
	}
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(v))
	w.buf = append(w.buf, b[:]...)
}

// Count writes a uint16 count. It fails when n does not fit.
func (w *Writer) Count(n int) {
	if n < 0 || n > math.MaxUint16 {
		w.fail("count %d out of uint16 range", n)
		return
	}
	w.U16(uint16(n))
}

// String writes a UTF-8 string with a uint16 byte length.
func (w *Writer) String(s string) {
	if w.err != nil {
		return
	}
	if !utf8.ValidString(s) {
		w.fail("invalid UTF-8 string")
		return
	}
	if len(s) > math.MaxUint16 {
		w.fail("string length %d exceeds uint16", len(s))
		return
	}
	w.U16(uint16(len(s)))
	w.buf = append(w.buf, s...)
}

// Blob writes a binary blob with a uint16 length.
func (w *Writer) Blob(b []byte) {
	if w.err != nil {
		return
	}
	if len(b) > math.MaxUint16 {
		w.fail("blob length %d exceeds uint16", len(b))
		return
	}
	w.U16(uint16(len(b)))
	w.buf = append(w.buf, b...)
}

// Done returns the encoded payload or the first error.
func (w *Writer) Done() ([]byte, error) {
	return w.buf, w.err
}

// Reader decodes a protocol payload. Methods record the first error; check
// Err after reading all fields.
type Reader struct {
	buf []byte
	pos int
	err error
}

// NewReader creates a payload reader.
func NewReader(buf []byte) *Reader { return &Reader{buf: buf} }

func (r *Reader) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf("protocol: "+format, args...)
	}
}

func (r *Reader) need(n int) bool {
	if r.err != nil {
		return false
	}
	if n < 0 || n > len(r.buf)-r.pos {
		r.fail("short payload: need %d bytes, have %d", n, len(r.buf)-r.pos)
		return false
	}
	return true
}

// U8 reads an unsigned byte.
func (r *Reader) U8() uint8 {
	if !r.need(1) {
		return 0
	}
	v := r.buf[r.pos]
	r.pos++
	return v
}

// Bool reads a boolean byte.
func (r *Reader) Bool() bool { return r.U8() != 0 }

// U16 reads a little-endian uint16.
func (r *Reader) U16() uint16 {
	if !r.need(2) {
		return 0
	}
	v := binary.LittleEndian.Uint16(r.buf[r.pos:])
	r.pos += 2
	return v
}

// I16 reads a little-endian int16.
func (r *Reader) I16() int16 { return int16(r.U16()) }

// U32 reads a little-endian uint32.
func (r *Reader) U32() uint32 {
	if !r.need(4) {
		return 0
	}
	v := binary.LittleEndian.Uint32(r.buf[r.pos:])
	r.pos += 4
	return v
}

// I32 reads a little-endian int32.
func (r *Reader) I32() int32 { return int32(r.U32()) }

// I64 reads a little-endian int64.
func (r *Reader) I64() int64 {
	if !r.need(8) {
		return 0
	}
	v := int64(binary.LittleEndian.Uint64(r.buf[r.pos:]))
	r.pos += 8
	return v
}

// Count reads a uint16 count.
func (r *Reader) Count() int { return int(r.U16()) }

// String reads a UTF-8 string with a uint16 byte length.
func (r *Reader) String() string {
	n := int(r.U16())
	if !r.need(n) {
		return ""
	}
	b := r.buf[r.pos : r.pos+n]
	r.pos += n
	if !utf8.Valid(b) {
		r.fail("invalid UTF-8 string")
		return ""
	}
	return string(b)
}

// Blob reads a binary blob with a uint16 length.
func (r *Reader) Blob() []byte {
	n := int(r.U16())
	if !r.need(n) {
		return nil
	}
	out := make([]byte, n)
	copy(out, r.buf[r.pos:r.pos+n])
	r.pos += n
	return out
}

// Err returns the first decoding error, including trailing bytes.
func (r *Reader) Err() error {
	if r.err != nil {
		return r.err
	}
	if r.pos != len(r.buf) {
		return fmt.Errorf("protocol: %d trailing payload bytes", len(r.buf)-r.pos)
	}
	return nil
}

// Remaining returns the number of unread bytes.
func (r *Reader) Remaining() int { return len(r.buf) - r.pos }
