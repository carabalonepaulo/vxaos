package rvdata

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRubyOracle validates that our writer produces documents Ruby itself can
// load. Ruby is a development-only oracle; the test is skipped when it is not
// installed.
func TestRubyOracle(t *testing.T) {
	ruby, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not installed")
	}
	tmp := t.TempDir()
	mapIn := repoPath("client", "Data", "Map002.rvdata2")
	mapOut := filepath.Join(tmp, "Map002.rvdata2")
	v, err := LoadFile(mapIn)
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveFile(mapOut, v); err != nil {
		t.Fatal(err)
	}

	script := filepath.Join(tmp, "check.rb")
	src := `
$LOAD_PATH.unshift(ARGV[1])
require File.join(ARGV[1], 'rgss.rb')
require File.join(ARGV[1], 'rpg.rb')
m = Marshal.load(File.binread(ARGV[0]))
raise 'bad class' unless m.is_a?(RPG::Map)
raise 'bad size' unless m.width == 40 && m.height == 25
raise 'bad events' unless m.events.size == 17
raise 'bad boss' unless m.events[14].name == 'Boss'
raise 'bad table' unless m.data[0, 0, 0] == 1567
puts 'map-ok'
`
	if err := os.WriteFile(script, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(ruby, script, mapOut, repoPath("server-rb", "Data")).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby map load: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "map-ok") {
		t.Fatalf("unexpected ruby output: %s", out)
	}

	scripts, err := LoadScripts(repoPath("client", "Data", "Scripts.rvdata2"))
	if err != nil {
		t.Fatal(err)
	}
	scriptsOut := filepath.Join(tmp, "Scripts.rvdata2")
	if err := SaveScripts(scriptsOut, scripts); err != nil {
		t.Fatal(err)
	}
	script2 := filepath.Join(tmp, "check_scripts.rb")
	src2 := `
require 'zlib'
s = Marshal.load(File.binread(ARGV[0]))
raise 'bad count' unless s.size == 168
i = s.index { |e| e[1] == '[VS] Configs' }
raise 'no configs' unless i
src = Zlib::Inflate.inflate(s[i][2])
raise 'no version' unless src.include?('GAME_VERSION')
puts 'scripts-ok'
`
	if err := os.WriteFile(script2, []byte(src2), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err = exec.Command(ruby, script2, scriptsOut).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby scripts load: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "scripts-ok") {
		t.Fatalf("unexpected ruby output: %s", out)
	}
}
