package rvdata

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
)

// Script is one entry of Scripts.rvdata2: an id, a display name and the
// decompressed Ruby source.
type Script struct {
	ID     int64
	Name   string
	Source []byte
}

// LoadScripts decodes and inflates Scripts.rvdata2.
func LoadScripts(path string) ([]Script, error) {
	v, err := LoadFile(path)
	if err != nil {
		return nil, err
	}
	root, ok := v.(*Array)
	if !ok {
		return nil, fmt.Errorf("rvdata: Scripts root is %T, want Array", v)
	}
	out := make([]Script, 0, len(root.Elems))
	for i, el := range root.Elems {
		entry, ok := el.(*Array)
		if !ok || len(entry.Elems) != 3 {
			return nil, fmt.Errorf("rvdata: Scripts[%d] is not a 3-element array", i)
		}
		id, ok := AsInt(entry.Elems[0])
		if !ok {
			return nil, fmt.Errorf("rvdata: Scripts[%d] id is not an integer", i)
		}
		name, ok := AsString(entry.Elems[1])
		if !ok {
			return nil, fmt.Errorf("rvdata: Scripts[%d] name is not a string", i)
		}
		compressed, ok := AsString(entry.Elems[2])
		if !ok {
			return nil, fmt.Errorf("rvdata: Scripts[%d] payload is not a string", i)
		}
		zr, err := zlib.NewReader(bytes.NewReader([]byte(compressed)))
		if err != nil {
			return nil, fmt.Errorf("rvdata: Scripts[%d] %q zlib: %w", i, name, err)
		}
		src, err := io.ReadAll(zr)
		closeErr := zr.Close()
		if err != nil {
			return nil, fmt.Errorf("rvdata: Scripts[%d] %q inflate: %w", i, name, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("rvdata: Scripts[%d] %q close: %w", i, name, closeErr)
		}
		out = append(out, Script{ID: id, Name: name, Source: src})
	}
	return out, nil
}

// EncodeScripts serializes a Scripts archive.
func EncodeScripts(scripts []Script) ([]byte, error) {
	root := &Array{Elems: make([]any, 0, len(scripts))}
	for i, s := range scripts {
		var compressed bytes.Buffer
		zw := zlib.NewWriter(&compressed)
		if _, err := zw.Write(s.Source); err != nil {
			return nil, fmt.Errorf("rvdata: Scripts[%d] %q compress: %w", i, s.Name, err)
		}
		if err := zw.Close(); err != nil {
			return nil, fmt.Errorf("rvdata: Scripts[%d] %q compress close: %w", i, s.Name, err)
		}
		root.Elems = append(root.Elems, &Array{Elems: []any{
			s.ID,
			s.Name,
			Bytes(compressed.Bytes()),
		}})
	}
	return Encode(root)
}

// SaveScripts encodes and writes a Scripts archive.
func SaveScripts(path string, scripts []Script) error {
	data, err := EncodeScripts(scripts)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// FindScript returns the index of the script with name, or -1.
func FindScript(scripts []Script, name string) int {
	for i := range scripts {
		if scripts[i].Name == name {
			return i
		}
	}
	return -1
}

// SetScriptSource replaces the source of an existing script. It returns false
// when the script does not exist.
func SetScriptSource(scripts []Script, name string, source []byte) bool {
	i := FindScript(scripts, name)
	if i < 0 {
		return false
	}
	scripts[i].Source = append([]byte(nil), source...)
	return true
}
