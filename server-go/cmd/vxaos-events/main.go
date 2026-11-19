// Command vxaos-events migrates event command 355 scripts from Ruby helper
// calls to the server-side Lisp executed by the interpreter. Unsupported
// scripts are left untouched and reported.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelaia/vxaos/server/internal/eventscript"
	"github.com/kelaia/vxaos/server/internal/rvdata"
)

type stats struct {
	changed int
	failed  int
	skipped int
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	if cmd != "check" && cmd != "apply" {
		usage()
		os.Exit(2)
	}
	dataDir := "../Client/Data"
	if len(os.Args) > 2 {
		dataDir = os.Args[2]
	}
	files, err := filepath.Glob(filepath.Join(dataDir, "Map*.rvdata2"))
	if err != nil {
		fail(err)
	}
	files = append(files, filepath.Join(dataDir, "CommonEvents.rvdata2"))
	total := stats{}
	for _, path := range files {
		doc, err := rvdata.LoadFile(path)
		if err != nil {
			fail(err)
		}
		st := &stats{}
		walk(doc, st)
		fmt.Printf("%-18s changed=%d failed=%d skipped=%d\n", filepath.Base(path), st.changed, st.failed, st.skipped)
		total.changed += st.changed
		total.failed += st.failed
		total.skipped += st.skipped
		if cmd == "apply" && st.changed > 0 {
			if err := saveAtomic(path, doc); err != nil {
				fail(err)
			}
			if _, err := rvdata.LoadFile(path); err != nil {
				fail(fmt.Errorf("%s is unreadable after writing: %w", path, err))
			}
		}
	}
	if cmd == "check" {
		fmt.Println("no files written")
	}
	fmt.Printf("total changed=%d failed=%d skipped=%d\n", total.changed, total.failed, total.skipped)
}

func walk(v any, st *stats) {
	switch x := v.(type) {
	case *rvdata.Array:
		if isCommandList(x) {
			processList(x, st)
			return
		}
		for _, elem := range x.Elems {
			walk(elem, st)
		}
	case *rvdata.Object:
		for _, iv := range x.Ivars {
			walk(iv.Value, st)
		}
	case *rvdata.Hash:
		for _, pair := range x.Pairs {
			walk(pair.Value, st)
		}
	}
}

func isCommandList(a *rvdata.Array) bool {
	if a == nil {
		return false
	}
	for _, elem := range a.Elems {
		if obj, ok := elem.(*rvdata.Object); ok && obj.Class == "RPG::EventCommand" {
			return true
		}
	}
	return false
}

func processList(a *rvdata.Array, st *stats) {
	for i := 0; i < len(a.Elems); i++ {
		cmd, ok := a.Elems[i].(*rvdata.Object)
		if !ok || cmd.Class != "RPG::EventCommand" || fieldInt(cmd, "@code") != 355 {
			continue
		}
		params, _ := cmd.MustGet("@parameters").(*rvdata.Array)
		if params == nil || len(params.Elems) == 0 {
			continue
		}
		first, ok := params.Elems[0].(string)
		if !ok {
			continue
		}
		script := first
		end := i
		for end+1 < len(a.Elems) {
			next, ok := a.Elems[end+1].(*rvdata.Object)
			if !ok || next.Class != "RPG::EventCommand" || fieldInt(next, "@code") != 655 {
				break
			}
			nextParams, _ := next.MustGet("@parameters").(*rvdata.Array)
			if nextParams != nil && len(nextParams.Elems) > 0 {
				if text, ok := nextParams.Elems[0].(string); ok {
					script += "\n" + text
				}
			}
			end++
		}
		if strings.HasPrefix(strings.TrimSpace(script), "(") {
			st.skipped++
			i = end
			continue
		}
		lisp, ok := eventscript.ToLisp(script)
		if !ok {
			st.failed++
			i = end
			continue
		}
		params.Elems[0] = lisp
		if end > i {
			a.Elems = append(a.Elems[:i+1], a.Elems[end+1:]...)
		}
		st.changed++
		i++
	}
}

func fieldInt(obj *rvdata.Object, name string) int {
	switch v := obj.MustGet(name).(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func saveAtomic(path string, doc any) error {
	data, err := rvdata.Encode(doc)
	if err != nil {
		return err
	}
	old, err := os.ReadFile(path)
	if err == nil {
		if err := os.WriteFile(path+".bak", old, 0o644); err != nil {
			return err
		}
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vxaos-events-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err == nil {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(name, path)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "vxaos-events: %v\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: vxaos-events <check|apply> [data-dir]

  check  translate event 355 scripts in memory and report counts
  apply  rewrite Map*.rvdata2 / CommonEvents.rvdata2, keeping .bak files`)
}
