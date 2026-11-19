package content

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/rvdata"
)

func repoPath(parts ...string) string {
	all := append([]string{"..", "..", ".."}, parts...)
	return filepath.Join(all...)
}

func contentPath(parts ...string) string {
	all := append([]string{"..", ".."}, parts...)
	return filepath.Join(all...)
}

func TestLoadVocabCSV(t *testing.T) {
	entries, err := LoadVocabCSV(contentPath("content", "locales", "ptBR.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 196 {
		t.Fatalf("entries = %d", len(entries))
	}
	if entries[0].Key != "Username" || entries[0].Text != "Usuário" {
		t.Fatalf("first = %+v", entries[0])
	}
	seenTeleport := 0
	for _, e := range entries {
		if e.Key == "Teleport" {
			seenTeleport++
		}
	}
	if seenTeleport != 1 {
		t.Fatalf("Teleport entries = %d, want deduplicated", seenTeleport)
	}
}

func TestRenderVocab(t *testing.T) {
	entries := []VocabEntry{
		{Key: "Greeting", Text: `olá "mundo"`},
		{Key: "Interpolation", Text: "#{evil}"},
		{Key: "Multiline", Text: "a\nb"},
	}
	b, err := RenderVocab(entries)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "module Vocab") {
		t.Fatalf("missing module: %s", s)
	}
	if !strings.Contains(s, `Greeting = "olá \"mundo\""`) {
		t.Fatalf("bad string escaping: %s", s)
	}
	if !strings.Contains(s, `Interpolation = "\#{evil}"`) {
		t.Fatalf("interpolation not escaped: %s", s)
	}
	if !strings.Contains(s, `Multiline = "a\nb"`) {
		t.Fatalf("newline not escaped: %s", s)
	}
}

func TestLoadQuests(t *testing.T) {
	quests, err := LoadQuests(contentPath("content", "quests.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(quests) != 2 {
		t.Fatalf("quests = %d", len(quests))
	}
	if quests[0].Name != "Estimulante Mágico" || quests[0].ItemID != 5 || quests[0].ItemKind != 1 {
		t.Fatalf("quest 0 = %+v", quests[0])
	}
	if !quests[1].Repeat || quests[1].EnemyAmount != 10 || quests[1].RewItemKind != 2 {
		t.Fatalf("quest 1 = %+v", quests[1])
	}
}

func TestRenderQuests(t *testing.T) {
	quests := []Quest{{Name: "Q1", Desc: "D1", ItemKind: 1, RewItemKind: 2, EnemyAmount: 3}}
	b, err := RenderQuests(quests)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "DATA = []") || !strings.Contains(s, "DATA << {") {
		t.Fatalf("bad quests output: %s", s)
	}
	if !strings.Contains(s, ":name            => \"Q1\"") || !strings.Contains(s, ":enemy_amount    => 3") {
		t.Fatalf("bad quest fields: %s", s)
	}
}

func TestCheck(t *testing.T) {
	res, err := Check(Options{DataDir: repoPath("Client", "Data"), ContentDir: contentPath("content")})
	if err != nil {
		t.Fatal(err)
	}
	if res.VocabEntries != 196 || res.Quests != 2 || res.Enums < 15 {
		t.Fatalf("result = %+v", res)
	}
	if len(res.VocabSource) == 0 || len(res.QuestSource) == 0 || len(res.EnumSource) == 0 {
		t.Fatal("empty generated sources")
	}
}

// TestBuildRubyOracle checks that the generated Vocab and Quests modules are
// valid Ruby and that their constants are correct. Ruby is a development-only
// oracle; the test is skipped when it is not installed.
func TestBuildRubyOracle(t *testing.T) {
	ruby, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not installed")
	}
	tmp := t.TempDir()
	src := repoPath("Client", "Data", "Scripts.rvdata2")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	scriptsPath := filepath.Join(tmp, "Scripts.rvdata2")
	if err := os.WriteFile(scriptsPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(Options{DataDir: tmp, ContentDir: contentPath("content")}); err != nil {
		t.Fatal(err)
	}
	check := filepath.Join(tmp, "check.rb")
	script := `
require 'zlib'
module Font
  class << self
    attr_accessor :default_name, :default_outline, :default_shadow, :default_bold, :default_size
  end
end
s = Marshal.load(File.binread(ARGV[0]))
v = s.find { |e| e[1] == '[VS] Vocab' }
q = s.find { |e| e[1] == '[VS] Quests' }
e = s.find { |e| e[1] == '[VS] Enums' }
c = s.find { |e| e[1] == '[VS] Configs' }
raise 'missing vocab' unless v
raise 'missing quests' unless q
raise 'missing enums' unless e
raise 'missing configs' unless c
eval(Zlib::Inflate.inflate(v[2]).force_encoding('UTF-8'))
eval(Zlib::Inflate.inflate(q[2]).force_encoding('UTF-8'))
eval(Zlib::Inflate.inflate(e[2]).force_encoding('UTF-8'))
eval(Zlib::Inflate.inflate(c[2]).force_encoding('UTF-8'))
raise 'bad username' unless Vocab::Username == 'Usuário'
raise 'bad quest count' unless Quests::DATA.size == 2
raise 'bad first quest' unless Quests::DATA[0][:name] == 'Estimulante Mágico'
raise 'bad enum' unless Enums::Dir::RIGHT == 6 && Enums::Target::ENEMY == 2 && Enums::Command::MSG == 14
raise 'bad config' unless Configs::PORT == 5000 && Configs::RANGE_WEAPONS[49][:mp_cost] == 3
puts 'content-ok'
`
	if err := os.WriteFile(check, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(ruby, check, scriptsPath).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby generated content: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "content-ok") {
		t.Fatalf("unexpected ruby output: %s", out)
	}
}

func TestBuild(t *testing.T) {
	tmp := t.TempDir()
	src := repoPath("Client", "Data", "Scripts.rvdata2")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "Scripts.rvdata2"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Build(Options{DataDir: tmp, ContentDir: contentPath("content")})
	if err != nil {
		t.Fatal(err)
	}
	if res.VocabEntries != 196 || res.Quests != 2 || res.Enums < 15 {
		t.Fatalf("result = %+v", res)
	}
	scripts, err := rvdata.LoadScripts(filepath.Join(tmp, "Scripts.rvdata2"))
	if err != nil {
		t.Fatal(err)
	}
	i := rvdata.FindScript(scripts, "[VS] Vocab")
	if i < 0 || !strings.Contains(string(scripts[i].Source), "Generated by vxaos-content") {
		t.Fatalf("vocab script not replaced")
	}
	i = rvdata.FindScript(scripts, "[VS] Quests")
	if i < 0 || !strings.Contains(string(scripts[i].Source), "Estimulante Mágico") {
		t.Fatalf("quests script not replaced")
	}
	i = rvdata.FindScript(scripts, "[VS] Enums")
	if i < 0 || !strings.Contains(string(scripts[i].Source), "Dir.const_set(:RIGHT, 6)") {
		t.Fatalf("enums script not replaced")
	}
	i = rvdata.FindScript(scripts, "[VS] Configs")
	if i < 0 || !strings.Contains(string(scripts[i].Source), "RANGE_WEAPONS[49]") {
		t.Fatalf("configs script not replaced")
	}
	if _, err := os.Stat(filepath.Join(tmp, "Scripts.rvdata2.bak")); err != nil {
		t.Fatalf("missing backup: %v", err)
	}
}
