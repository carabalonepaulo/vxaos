package rvdata

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"io"
	"math/big"
	"path/filepath"
	"strings"
	"testing"
)

func repoPath(parts ...string) string {
	base := filepath.Join("..", "..", "..")
	all := append([]string{base}, parts...)
	return filepath.Join(all...)
}

func mustLoad(t *testing.T, path string) any {
	t.Helper()
	v, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile(%s): %v", path, err)
	}
	return v
}

func TestEncodeKnownScalars(t *testing.T) {
	cases := []struct {
		value any
		hex   string
	}{
		{int64(0), "04086900"},
		{int64(1), "04086906"},
		{int64(2), "04086907"},
		{int64(127), "040869017f"},
		{int64(-1), "040869fa"},
		{int64(-129), "040869ff7f"},
		{int64(256), "040869020001"},
		{int64(1000), "04086902e803"},
		{true, "040854"},
		{false, "040846"},
		{nil, "040830"},
		{"abc", "0408492208616263063a064554"},
		{Bytes{0, 1, 2}, "04082208000102"},
	}
	for _, tc := range cases {
		got, err := Encode(tc.value)
		if err != nil {
			t.Fatalf("Encode(%v): %v", tc.value, err)
		}
		if hex.EncodeToString(got) != tc.hex {
			t.Errorf("Encode(%v) = %s, want %s", tc.value, hex.EncodeToString(got), tc.hex)
		}
	}
}

func TestScalarRoundTrip(t *testing.T) {
	values := []any{
		nil, true, false, int64(0), int64(1), int64(-1), int64(122), int64(123),
		int64(-122), int64(-123), int64(32767), int64(-32768), int64(1 << 20),
		"", "hello", "áéíóú", float64(1.5), float64(-0.25),
	}
	for _, v := range values {
		data, err := Encode(v)
		if err != nil {
			t.Fatalf("Encode(%v): %v", v, err)
		}
		got, err := Decode(data)
		if err != nil {
			t.Fatalf("Decode(%v): %v", v, err)
		}
		switch want := v.(type) {
		case float64:
			gotF, ok := got.(float64)
			if !ok || gotF != want {
				t.Errorf("round trip %v -> %v", v, got)
			}
		default:
			if got != want {
				t.Errorf("round trip %v -> %v", v, got)
			}
		}
	}
}

func TestLegacyFloatTail(t *testing.T) {
	// Ruby 1.9-era Marshal floats append a short binary tail after a NUL.
	data := []byte{4, 8, 'f', 0x1b}
	data = append(data, []byte("0.94999999999999996")...)
	data = append(data, 0x00, 0x66, 0x66)
	v, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	f, ok := v.(float64)
	if !ok || f < 0.949 || f > 0.951 {
		t.Fatalf("float = %#v", v)
	}
}

func TestBignumRoundTrip(t *testing.T) {
	n := new(big.Int).Lsh(big.NewInt(1), 31)
	data, err := Encode(&Bignum{Value: n})
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(data) != "04086c2b0700000080" {
		t.Fatalf("bignum bytes = %s", hex.EncodeToString(data))
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	b, ok := got.(*Bignum)
	if !ok || b.Value.Cmp(n) != 0 {
		t.Fatalf("bignum round trip = %#v", got)
	}
}

func TestTableRoundTrip(t *testing.T) {
	tbl := NewTable(2, 3, 2)
	tbl.Set(0, 0, 0, 1)
	tbl.Set(1, 2, 1, 65535)
	data, err := Encode(tbl)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	gotT, ok := got.(*Table)
	if !ok {
		t.Fatalf("got %T, want *Table", got)
	}
	if gotT.Dims != tbl.Dims || gotT.XSize != 2 || gotT.YSize != 3 || gotT.ZSize != 2 {
		t.Fatalf("table dims = %+v", gotT)
	}
	if gotT.At(0, 0, 0) != 1 || gotT.At(1, 2, 1) != 65535 {
		t.Fatalf("table cells = %v", gotT.Data)
	}
}

func TestColorToneRoundTrip(t *testing.T) {
	c := &Color{Red: 1.25, Green: 2.5, Blue: -3, Alpha: 255}
	data, err := Encode(c)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	gotC, ok := got.(*Color)
	if !ok || *gotC != *c {
		t.Fatalf("color round trip = %#v", got)
	}
	tn := &Tone{Red: -68, Green: 10, Blue: 20, Gray: 30}
	data, err = Encode(tn)
	if err != nil {
		t.Fatal(err)
	}
	got, err = Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	gotT, ok := got.(*Tone)
	if !ok || *gotT != *tn {
		t.Fatalf("tone round trip = %#v", got)
	}
}

func TestObjectRoundTrip(t *testing.T) {
	obj := &Object{Class: "RPG::EventCommand", Ivars: []Ivar{
		{Name: "@code", Value: int64(355)},
		{Name: "@indent", Value: int64(0)},
		{Name: "@parameters", Value: &Array{Elems: []any{"check_point(1, 14, 3)"}}},
	}}
	data, err := Encode(obj)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	gotO, ok := got.(*Object)
	if !ok || gotO.Class != obj.Class {
		t.Fatalf("object round trip = %#v", got)
	}
	if code := IvarInt(gotO, "@code"); code != 355 {
		t.Fatalf("code = %d", code)
	}
	params := IvarArray(gotO, "@parameters")
	if params == nil || len(params.Elems) != 1 {
		t.Fatalf("params = %#v", params)
	}
	if s, _ := AsString(params.Elems[0]); s != "check_point(1, 14, 3)" {
		t.Fatalf("param = %#v", params.Elems[0])
	}
}

func TestCycleRejected(t *testing.T) {
	a := &Array{}
	a.Elems = []any{a}
	if _, err := Encode(a); err == nil {
		t.Fatal("expected cyclic graph error")
	}
}

func TestLoadMap002(t *testing.T) {
	v := mustLoad(t, repoPath("Client", "Data", "Map002.rvdata2"))
	m, ok := v.(*Object)
	if !ok {
		t.Fatalf("root = %T", v)
	}
	if m.Class != "RPG::Map" {
		t.Fatalf("class = %s", m.Class)
	}
	if got := IvarInt(m, "@width"); got != 40 {
		t.Errorf("width = %d", got)
	}
	if got := IvarInt(m, "@height"); got != 25 {
		t.Errorf("height = %d", got)
	}
	if got := IvarInt(m, "@tileset_id"); got != 6 {
		t.Errorf("tileset = %d", got)
	}
	tbl := IvarTable(m, "@data")
	if tbl == nil || tbl.XSize != 40 || tbl.YSize != 25 || tbl.ZSize != 4 {
		t.Fatalf("data table = %#v", tbl)
	}
	if len(tbl.Data) != 4000 {
		t.Fatalf("data cells = %d", len(tbl.Data))
	}
	events := IvarHash(m, "@events")
	if events == nil || len(events.Pairs) == 0 {
		t.Fatalf("events = %#v", events)
	}
	// Event 14 is the boss and must carry an RPG::Event with pages.
	var boss *Object
	for _, p := range events.Pairs {
		if id, _ := AsInt(p.Key); id == 14 {
			boss, _ = p.Value.(*Object)
		}
	}
	if boss == nil || boss.Class != "RPG::Event" {
		t.Fatalf("boss event = %#v", boss)
	}
	if name := IvarString(boss, "@name"); name != "Boss" {
		t.Errorf("boss name = %q", name)
	}
}

func TestMapRoundTrip(t *testing.T) {
	v := mustLoad(t, repoPath("Client", "Data", "Map002.rvdata2"))
	data, err := Encode(v)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := got.(*Object)
	if !ok || m.Class != "RPG::Map" {
		t.Fatalf("round trip root = %#v", got)
	}
	if IvarInt(m, "@width") != 40 || IvarInt(m, "@height") != 25 {
		t.Fatalf("round trip dims = %d x %d", IvarInt(m, "@width"), IvarInt(m, "@height"))
	}
	events := IvarHash(m, "@events")
	if events == nil || len(events.Pairs) == 0 {
		t.Fatalf("round trip events = %#v", events)
	}
}

func TestLoadScripts(t *testing.T) {
	scripts, err := LoadScripts(repoPath("Client", "Data", "Scripts.rvdata2"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts) != 167 {
		t.Fatalf("script count = %d", len(scripts))
	}
	i := FindScript(scripts, "[VS] Configs")
	if i < 0 {
		t.Fatal("[VS] Configs not found")
	}
	if !strings.Contains(string(scripts[i].Source), "GAME_VERSION") {
		t.Errorf("[VS] Configs does not contain GAME_VERSION")
	}
	i = FindScript(scripts, "[VS] Enums")
	if i < 0 {
		t.Fatal("[VS] Enums not found")
	}
	if !strings.Contains(string(scripts[i].Source), "Generated by vxaos-content") ||
		!strings.Contains(string(scripts[i].Source), "Command.const_set(:MSG, 14)") {
		t.Errorf("[VS] Enums was not generated from enums.toml")
	}
}

func TestScriptsRoundTrip(t *testing.T) {
	scripts, err := LoadScripts(repoPath("Client", "Data", "Scripts.rvdata2"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeScripts(scripts)
	if err != nil {
		t.Fatal(err)
	}
	scripts2, err := decodeScriptsBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(scripts2) != len(scripts) {
		t.Fatalf("count = %d, want %d", len(scripts2), len(scripts))
	}
	for i := range scripts {
		if scripts[i].ID != scripts2[i].ID || scripts[i].Name != scripts2[i].Name {
			t.Fatalf("script %d mismatch: %+v vs %+v", i, scripts[i], scripts2[i])
		}
		if !bytes.Equal(scripts[i].Source, scripts2[i].Source) {
			t.Fatalf("script %d source mismatch", i)
		}
	}
}

func decodeScriptsBytes(data []byte) ([]Script, error) {
	v, err := Decode(data)
	if err != nil {
		return nil, err
	}
	root, ok := v.(*Array)
	if !ok {
		return nil, ErrUnsupported
	}
	var out []Script
	for _, el := range root.Elems {
		entry, ok := el.(*Array)
		if !ok || len(entry.Elems) != 3 {
			return nil, ErrUnsupported
		}
		id, _ := AsInt(entry.Elems[0])
		name, _ := AsString(entry.Elems[1])
		compressed, _ := AsString(entry.Elems[2])
		zr, err := zlib.NewReader(strings.NewReader(compressed))
		if err != nil {
			return nil, err
		}
		src, err := io.ReadAll(zr)
		if err != nil {
			return nil, err
		}
		if err := zr.Close(); err != nil {
			return nil, err
		}
		out = append(out, Script{ID: id, Name: name, Source: src})
	}
	return out, nil
}
