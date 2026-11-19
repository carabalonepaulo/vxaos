package content

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigs(t *testing.T) {
	file, err := LoadConfigs("../../content/configs.toml")
	if err != nil {
		t.Fatal(err)
	}
	if file.Network.Port != 5000 || file.Network.GameVersion != 12983 {
		t.Fatalf("network = %+v", file.Network)
	}
	if file.Limits.MaxEquips != 9 || file.Limits.MaxMapDrops != 20 {
		t.Fatalf("limits = %+v", file.Limits)
	}
	if len(file.Paperdoll.Down) != 9 || file.Paperdoll.Down[0] != 3 {
		t.Fatalf("paperdoll = %+v", file.Paperdoll)
	}
	if len(file.Teleports) != 1 || len(file.Teleports[0]) != 2 || file.Teleports[0][1].Gold != 20 {
		t.Fatalf("teleports = %+v", file.Teleports)
	}
	if file.RangeWeapons["49"].MPCost != 3 || !file.RangeSkills["70"].StepAnime {
		t.Fatalf("range config = %+v %+v", file.RangeWeapons, file.RangeSkills)
	}
	if file.Timing.AttackTime != 0.8 || file.Limits.MinLevelPvP != 2 {
		t.Fatalf("timing/limits = %+v %+v", file.Timing, file.Limits)
	}
}

func TestRenderConfigs(t *testing.T) {
	file, err := LoadConfigs("../../content/configs.toml")
	if err != nil {
		t.Fatal(err)
	}
	first, err := RenderConfigs(file)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderConfigs(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("render is not deterministic")
	}
	source := string(first)
	for _, want := range []string{
		"Font.default_name = \"VL Gothic\"",
		"module Configs",
		"TELEPORTS = [",
		"RANGE_WEAPONS[31] = {",
		"RANGE_WEAPONS[49] = {",
		":mp_cost         => 3",
		"RANGE_SKILLS[70] = {",
		":step_anime      => true",
		"HOTKEYS = [:KEY_1",
		"BALLOONS_KEYS = [:F3",
	} {
		if !strings.Contains(source, want) {
			t.Fatalf("generated configs missing %q:\n%s", want, source)
		}
	}
}

func TestConfigsRejectBadValues(t *testing.T) {
	file, err := LoadConfigs("../../content/configs.toml")
	if err != nil {
		t.Fatal(err)
	}
	bad := *file
	bad.Network.Port = 0
	if err := bad.Validate(); err == nil {
		t.Fatal("invalid port accepted")
	}
	bad = *file
	bad.Paperdoll.Down = bad.Paperdoll.Down[:2]
	if err := bad.Validate(); err == nil {
		t.Fatal("short paperdoll accepted")
	}
	bad = *file
	bad.Keys.Hotkeys = bad.Keys.Hotkeys[:2]
	if err := bad.Validate(); err == nil {
		t.Fatal("short hotkey list accepted")
	}
	bad = *file
	bad.Keys.Attack = "not valid"
	if err := bad.Validate(); err == nil {
		t.Fatal("invalid key symbol accepted")
	}
}

func TestRenderConfigsRubyOracle(t *testing.T) {
	ruby, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not installed")
	}
	file, err := LoadConfigs("../../content/configs.toml")
	if err != nil {
		t.Fatal(err)
	}
	source, err := RenderConfigs(file)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "configs.rb")
	if err := os.WriteFile(path, source, 0o644); err != nil {
		t.Fatal(err)
	}
	stub := `
module Font
  class << self
    attr_accessor :default_name, :default_outline, :default_shadow, :default_bold, :default_size
  end
end
`
	path2 := filepath.Join(dir, "run.rb")
	script := stub + "eval(File.read(ARGV[0], encoding: 'UTF-8'))\n" +
		"raise 'port' unless Configs::PORT == 5000\n" +
		"raise 'paperdoll' unless Configs::PAPERDOLL_DOWN_DIR.size == 9\n" +
		"raise 'range' unless Configs::RANGE_WEAPONS[49][:mp_cost] == 3\n" +
		"raise 'skill' unless Configs::RANGE_SKILLS[70][:step_anime] == true\n" +
		"raise 'teleport' unless Configs::TELEPORTS[0][1][:gold] == 20\n" +
		"raise 'keys' unless Configs::HOTKEYS.size == 9 && Configs::ATTACK_KEY == :CTRL\n" +
		"raise 'timing' unless Configs::ATTACK_TIME == 0.8\n" +
		"puts 'configs-ok'\n"
	if err := os.WriteFile(path2, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(ruby, path2, path).CombinedOutput()
	if err != nil {
		t.Fatalf("ruby generated configs: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "configs-ok") {
		t.Fatalf("unexpected ruby output: %s", out)
	}
}
