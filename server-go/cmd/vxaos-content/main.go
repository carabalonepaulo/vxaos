// Command vxaos-content builds the generated Ruby modules inside
// Scripts.rvdata2 from the content source files.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/carabalonepaulo/vxaos/server/internal/content"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	dataDir := fs.String("data", "../Client/Data", "client data directory containing Scripts.rvdata2")
	contentDir := fs.String("content", "./content", "content source directory")
	locale := fs.String("locale", "ptBR", "locale CSV name (without .csv)")
	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	opts := content.Options{DataDir: *dataDir, ContentDir: *contentDir, Locale: *locale}
	switch cmd {
	case "build":
		res, err := content.Build(opts)
		if err != nil {
			fail(err)
		}
		fmt.Printf("built %s: %d enums, 1 configs, %d vocab entries, %d quests\n", res.ScriptsPath, res.Enums, res.VocabEntries, res.Quests)
	case "check":
		res, err := content.Check(opts)
		if err != nil {
			fail(err)
		}
		fmt.Printf("ok: %d vocab entries, %d quests\n", res.VocabEntries, res.Quests)
		fmt.Println("no files written")
	default:
		usage()
		os.Exit(2)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "vxaos-content: %v\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: vxaos-content <build|check> [flags]

  build   render content sources and replace the generated scripts in Scripts.rvdata2
  check   render and validate without writing

flags:
  -data     client data directory (default ../Client/Data)
  -content  content source directory (default ./content)
  -locale   locale CSV name without .csv (default ptBR)`)
}
