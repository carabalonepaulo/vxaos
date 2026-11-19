// Command vxaos-formula migrates RPG Maker damage formulas to the server-side
// Lisp form. The formula field is server-only: the client Damage#eval guard
// returns 0, so the .rvdata2 string can hold Lisp directly.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kelaia/vxaos/server/internal/formula"
	"github.com/kelaia/vxaos/server/internal/rvdata"
)

var files = []string{"Skills.rvdata2", "Items.rvdata2"}

type stats struct {
	changed  int
	already  int
	invalid  int
	examples []string
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	dataDir := "../Client/Data"
	if len(os.Args) > 2 {
		dataDir = os.Args[2]
	}
	total := stats{}
	for _, name := range files {
		path := filepath.Join(dataDir, name)
		doc, err := rvdata.LoadFile(path)
		if err != nil {
			fail(err)
		}
		st := migrate(doc, cmd == "apply")
		fmt.Printf("%-16s changed=%d already=%d invalid=%d\n", name, st.changed, st.already, st.invalid)
		for _, example := range st.examples {
			fmt.Printf("  %s\n", example)
		}
		total.changed += st.changed
		total.already += st.already
		total.invalid += st.invalid
		if cmd == "apply" && st.changed > 0 {
			if err := saveAtomic(path, doc); err != nil {
				fail(err)
			}
			if _, err := rvdata.LoadFile(path); err != nil {
				fail(fmt.Errorf("%s is unreadable after writing: %w", path, err))
			}
		}
	}
	if total.invalid > 0 {
		fmt.Fprintln(os.Stderr, "some formulas cannot be translated; no partial write is silently accepted")
		os.Exit(1)
	}
	if cmd == "check" {
		fmt.Println("no files written")
	}
}

func migrate(doc any, apply bool) stats {
	var st stats
	arr, ok := doc.(*rvdata.Array)
	if !ok {
		return st
	}
	for i := 0; i < arr.Len(); i++ {
		obj, ok := arr.At(i).(*rvdata.Object)
		if !ok {
			continue
		}
		damage, ok := obj.MustGet("@damage").(*rvdata.Object)
		if !ok {
			continue
		}
		value, ok := damage.MustGet("@formula").(string)
		if !ok {
			continue
		}
		if formula.IsLisp(value) {
			st.already++
			continue
		}
		lisp, ok := formula.ToLisp(value)
		if !ok {
			st.invalid++
			if len(st.examples) < 8 {
				st.examples = append(st.examples, fmt.Sprintf("[%d] cannot translate %q", i, value))
			}
			continue
		}
		st.changed++
		if len(st.examples) < 8 {
			st.examples = append(st.examples, fmt.Sprintf("[%d] %s", i, lisp))
		}
		if apply {
			damage.Set("@formula", lisp)
		}
	}
	return st
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
	tmp, err := os.CreateTemp(dir, ".vxaos-formula-*.tmp")
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
	fmt.Fprintf(os.Stderr, "vxaos-formula: %v\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: vxaos-formula <check|apply> [data-dir]

  check  translate formulas in memory and report counts
  apply  rewrite Skills.rvdata2 and Items.rvdata2, keeping .bak files`)
}
