package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

// Magic identifies a v2 client. It is sent before the framed stream starts.
const Magic = "VXAOS2\r\n"

// ProtocolVersion is the wire protocol implemented by this package.
const ProtocolVersion uint16 = 1

// MaxFrameSize is the largest accepted frame, including the two-byte opcode.
const MaxFrameSize = 1 << 20

// MinFrameSize is the length of a frame containing only an opcode.
const MinFrameSize = 2

var (
	ErrMagic         = errors.New("protocol: invalid handshake magic")
	ErrFrameTooBig   = errors.New("protocol: frame exceeds maximum size")
	ErrFrameTooShort = errors.New("protocol: frame is shorter than an opcode")
)

// Frame is one framed message.
type Frame struct {
	Op      uint16
	Payload []byte
}

// ReadFrame reads one frame. It is safe against TCP fragmentation because it
// uses io.ReadFull for the header and the payload.
func ReadFrame(r io.Reader) (Frame, error) {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Frame{}, err
	}
	length := binary.LittleEndian.Uint32(header[:])
	if length < MinFrameSize {
		return Frame{}, ErrFrameTooShort
	}
	if length > MaxFrameSize {
		return Frame{}, ErrFrameTooBig
	}
	var opBuf [2]byte
	if _, err := io.ReadFull(r, opBuf[:]); err != nil {
		return Frame{}, err
	}
	op := binary.LittleEndian.Uint16(opBuf[:])
	payload := make([]byte, int(length)-MinFrameSize)
	if len(payload) > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return Frame{}, err
		}
	}
	return Frame{Op: op, Payload: payload}, nil
}

// WriteFrame writes one frame. The length includes the uint16 opcode.
func WriteFrame(w io.Writer, op uint16, payload []byte) error {
	length := MinFrameSize + len(payload)
	if length > MaxFrameSize {
		return ErrFrameTooBig
	}
	header := make([]byte, 6)
	binary.LittleEndian.PutUint32(header[0:], uint32(length))
	binary.LittleEndian.PutUint16(header[4:], op)
	if err := writeFull(w, header); err != nil {
		return err
	}
	if len(payload) > 0 {
		return writeFull(w, payload)
	}
	return nil
}

func writeFull(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}

// Hello is the first message from a v2 client, before framing starts.
type Hello struct {
	ProtocolVersion uint16
	GameVersion     uint16
	Build           string
}

// HelloAck is the server's handshake response.
type HelloAck struct {
	ProtocolVersion uint16
	GameVersion     uint16
	ServerTime      int64
	MaxFrameSize    uint32
	Status          uint8
	Message         string
}

// Handshake status codes.
const (
	StatusOK           uint8 = 0
	StatusVersion      uint8 = 1
	StatusServerFull   uint8 = 2
	StatusMaintenance  uint8 = 3
	StatusBanned       uint8 = 4
	StatusBadHandshake uint8 = 5
)

// ReadHello reads magic and the client hello.
func ReadHello(r io.Reader) (Hello, error) {
	if err := readMagic(r); err != nil {
		return Hello{}, err
	}
	var h Hello
	var err error
	if h.ProtocolVersion, err = readU16(r); err != nil {
		return Hello{}, err
	}
	if h.GameVersion, err = readU16(r); err != nil {
		return Hello{}, err
	}
	if h.Build, err = readStringU16(r); err != nil {
		return Hello{}, err
	}
	return h, nil
}

// WriteHello writes magic and the client hello.
func WriteHello(w io.Writer, h Hello) error {
	var wr Writer
	wr.U16(h.ProtocolVersion)
	wr.U16(h.GameVersion)
	wr.String(h.Build)
	body, err := wr.Done()
	if err != nil {
		return err
	}
	if err := writeFull(w, []byte(Magic)); err != nil {
		return err
	}
	return writeFull(w, body)
}

// ReadHelloAck reads magic and the server handshake response.
func ReadHelloAck(r io.Reader) (HelloAck, error) {
	if err := readMagic(r); err != nil {
		return HelloAck{}, err
	}
	var a HelloAck
	var err error
	if a.ProtocolVersion, err = readU16(r); err != nil {
		return a, err
	}
	if a.GameVersion, err = readU16(r); err != nil {
		return a, err
	}
	if a.ServerTime, err = readI64(r); err != nil {
		return a, err
	}
	if a.MaxFrameSize, err = readU32(r); err != nil {
		return a, err
	}
	status, err := readU8(r)
	if err != nil {
		return a, err
	}
	a.Status = status
	if a.Message, err = readStringU16(r); err != nil {
		return a, err
	}
	return a, nil
}

// WriteHelloAck writes magic and the server handshake response.
func WriteHelloAck(w io.Writer, a HelloAck) error {
	var wr Writer
	wr.U16(a.ProtocolVersion)
	wr.U16(a.GameVersion)
	wr.I64(a.ServerTime)
	wr.U32(a.MaxFrameSize)
	wr.U8(a.Status)
	wr.String(a.Message)
	body, err := wr.Done()
	if err != nil {
		return err
	}
	if err := writeFull(w, []byte(Magic)); err != nil {
		return err
	}
	return writeFull(w, body)
}

func readMagic(r io.Reader) error {
	buf := make([]byte, len(Magic))
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	if string(buf) != Magic {
		return ErrMagic
	}
	return nil
}

func readU8(r io.Reader) (uint8, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func readU16(r io.Reader) (uint16, error) {
	var b [2]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b[:]), nil
}

func readU32(r io.Reader) (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b[:]), nil
}

func readI64(r io.Reader) (int64, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(b[:])), nil
}

func readStringU16(r io.Reader) (string, error) {
	n, err := readU16(r)
	if err != nil {
		return "", err
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	if !utf8.Valid(b) {
		return "", fmt.Errorf("protocol: invalid UTF-8 handshake string")
	}
	return string(b), nil
}
