// Command vxaos-clientpatch applies the v2 client scripts to Scripts.rvdata2.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/carabalonepaulo/vxaos/server/internal/rvdata"
)

type patch struct {
	Name   string `toml:"name"`
	File   string `toml:"file"`
	After  string `toml:"after"`
	Before string `toml:"before"`
}

type manifest struct {
	Scripts []patch `toml:"script"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	dataDir := fs.String("data", "../client/Data", "client data directory")
	patchesDir := fs.String("patches", "./client-patches", "patch directory")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	scriptsPath := filepath.Join(*dataDir, "Scripts.rvdata2")
	scripts, err := rvdata.LoadScripts(scriptsPath)
	if err != nil {
		fail(err)
	}
	var m manifest
	if _, err := toml.DecodeFile(filepath.Join(*patchesDir, "manifest.toml"), &m); err != nil {
		fail(err)
	}
	if err := apply(&scripts, *patchesDir, m.Scripts); err != nil {
		fail(err)
	}
	if cmd == "check" {
		fmt.Printf("ok: %d scripts, %d patches\n", len(scripts), len(m.Scripts))
		fmt.Println("no files written")
		return
	}
	if cmd != "apply" {
		usage()
		os.Exit(2)
	}
	if err := saveAtomic(scriptsPath, scripts); err != nil {
		fail(err)
	}
	if _, err := rvdata.LoadScripts(scriptsPath); err != nil {
		fail(fmt.Errorf("patched Scripts.rvdata2 is unreadable: %w", err))
	}
	fmt.Printf("applied %d patches to %s (%d scripts)\n", len(m.Scripts), scriptsPath, len(scripts))
}

func apply(scriptsPtr *[]rvdata.Script, dir string, patches []patch) error {
	scripts := *scriptsPtr
	maxID := int64(0)
	for _, s := range scripts {
		if s.ID > maxID {
			maxID = s.ID
		}
	}
	for _, p := range patches {
		if p.Name == "" || p.File == "" {
			return fmt.Errorf("clientpatch: manifest entry missing name/file")
		}
		if p.After != "" && p.Before != "" {
			return fmt.Errorf("clientpatch: %q sets both after and before", p.Name)
		}
		src, err := os.ReadFile(filepath.Join(dir, p.File))
		if err != nil {
			return err
		}
		idx := rvdata.FindScript(scripts, p.Name)
		if idx >= 0 {
			scripts[idx].Source = src
			if p.After == "" && p.Before == "" {
				continue
			}
			// Existing patches may need to move when their anchor changes.
			entry := scripts[idx]
			scripts = append(scripts[:idx], scripts[idx+1:]...)
			target := -1
			if p.After != "" {
				target = rvdata.FindScript(scripts, p.After)
				if target < 0 {
					return fmt.Errorf("clientpatch: %q: after target %q not found", p.Name, p.After)
				}
				target++
			} else {
				target = rvdata.FindScript(scripts, p.Before)
				if target < 0 {
					return fmt.Errorf("clientpatch: %q: before target %q not found", p.Name, p.Before)
				}
			}
			scripts = append(scripts, rvdata.Script{})
			copy(scripts[target+1:], scripts[target:])
			scripts[target] = entry
			continue
		}
		var target int
		switch {
		case p.After != "":
			target = rvdata.FindScript(scripts, p.After)
			if target < 0 {
				return fmt.Errorf("clientpatch: %q: after target %q not found", p.Name, p.After)
			}
			target++
		case p.Before != "":
			target = rvdata.FindScript(scripts, p.Before)
			if target < 0 {
				return fmt.Errorf("clientpatch: %q: before target %q not found", p.Name, p.Before)
			}
		default:
			return fmt.Errorf("clientpatch: %q does not exist and has no after/before", p.Name)
		}
		maxID++
		entry := rvdata.Script{ID: maxID, Name: p.Name, Source: src}
		scripts = append(scripts, rvdata.Script{})
		copy(scripts[target+1:], scripts[target:])
		scripts[target] = entry
	}
	*scriptsPtr = scripts
	return nil
}

func saveAtomic(path string, scripts []rvdata.Script) error {
	data, err := rvdata.EncodeScripts(scripts)
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
	tmp, err := os.CreateTemp(dir, ".vxaos-patch-*.tmp")
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
	fmt.Fprintf(os.Stderr, "vxaos-clientpatch: %v\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: vxaos-clientpatch <check|apply> [flags]

  check  load scripts and patches, validate only
  apply  replace/insert entries in Scripts.rvdata2 (keeps a .bak)

flags:
  -data     client data directory (default ../client/Data)
  -patches  patch directory (default ./client-patches)`)
}
