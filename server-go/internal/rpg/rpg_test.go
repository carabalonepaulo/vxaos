package rpg

import (
	"path/filepath"
	"strings"
	"testing"
)

func dataDir() string {
	return filepath.Join("..", "..", "..", "Client", "Data")
}

func TestLoadData(t *testing.T) {
	d, err := Load(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Actors) != 10 {
		t.Errorf("actors = %d", len(d.Actors))
	}
	if len(d.Classes) != 10 {
		t.Errorf("classes = %d", len(d.Classes))
	}
	if len(d.Skills) != 129 {
		t.Errorf("skills = %d", len(d.Skills))
	}
	if len(d.Items) != 26 {
		t.Errorf("items = %d", len(d.Items))
	}
	if len(d.Weapons) != 63 {
		t.Errorf("weapons = %d", len(d.Weapons))
	}
	if len(d.Armors) != 64 {
		t.Errorf("armors = %d", len(d.Armors))
	}
	if len(d.Enemies) != 32 {
		t.Errorf("enemies = %d", len(d.Enemies))
	}
	if len(d.States) != 26 {
		t.Errorf("states = %d", len(d.States))
	}
	if len(d.Animations) != 111 {
		t.Errorf("animations = %d", len(d.Animations))
	}
	if len(d.Tilesets) != 7 {
		t.Errorf("tilesets = %d", len(d.Tilesets))
	}
	if len(d.CommonEvents) != 9 {
		t.Errorf("common events = %d", len(d.CommonEvents))
	}
	if len(d.MapInfos) != 2 || len(d.Maps) != 2 {
		t.Errorf("maps = %d infos = %d", len(d.Maps), len(d.MapInfos))
	}
	if d.System == nil || d.System.StartMapID <= 0 {
		t.Errorf("system = %+v", d.System)
	}
}

func TestActorAndClass(t *testing.T) {
	d, err := Load(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	a := d.Actors[1]
	if a == nil || a.Name != "Guerreiro" || a.ClassID != 1 || a.InitialLevel != 1 {
		t.Fatalf("actor 1 = %+v", a)
	}
	if len(a.Equips) != 5 || a.Equips[0] != 1 || a.Equips[1] != 46 {
		t.Errorf("actor 1 equips = %v", a.Equips)
	}
	c := d.Classes[1]
	if c == nil || c.Name != "Guerreiro" {
		t.Fatalf("class 1 = %+v", c)
	}
	if c.ExpParams != [4]int{30, 20, 30, 30} {
		t.Errorf("exp params = %v", c.ExpParams)
	}
	if len(c.Graphics.MaleSprites) != 2 || len(c.Graphics.FemaleSprites) != 1 {
		t.Errorf("graphics = %+v", c.Graphics)
	}
	if len(c.Learnings) != 5 {
		t.Errorf("learnings = %+v", c.Learnings)
	}
	if c.Params == nil || c.Params.XSize != 8 || c.Params.YSize != 100 {
		t.Errorf("params table = %+v", c.Params)
	}
}

func TestItemsAndEquips(t *testing.T) {
	d, err := Load(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	s := d.Skills[1]
	if s == nil || s.Name != "Atacar" {
		t.Fatalf("skill 1 = %+v", s)
	}
	if s.Damage.Formula == "" {
		t.Errorf("skill 1 has no damage formula")
	}
	if !strings.HasPrefix(strings.TrimSpace(s.Damage.Formula), "(") {
		t.Errorf("skill 1 formula is not Lisp: %q", s.Damage.Formula)
	}
	if s.AnimationID == 0 {
		t.Errorf("skill 1 has no animation")
	}
	it := d.Items[5]
	if it == nil || it.Name != "Estimulante" {
		t.Fatalf("item 5 = %+v", it)
	}
	if it.Range != 10 {
		t.Errorf("item 5 range = %d", it.Range)
	}
	w := d.Weapons[31]
	if w == nil || w.Name != "Arco Curto" || w.WTypeID == 0 {
		t.Fatalf("weapon 31 = %+v", w)
	}
	if w.PaperdollName != "$Bow" {
		t.Errorf("weapon 31 paperdoll = %q", w.PaperdollName)
	}
	ar := d.Armors[54]
	if ar == nil || ar.Name != "Amuleto da Natureza" {
		t.Fatalf("armor 54 = %+v", ar)
	}
	if ar.ETypeID != 5 {
		t.Errorf("armor 54 etype = %d", ar.ETypeID)
	}
	if ar.Sex != 2 {
		t.Errorf("armor 54 sex = %d", ar.Sex)
	}
}

func TestEnemies(t *testing.T) {
	d, err := Load(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	e := d.Enemies[1]
	if e == nil || e.Name != "Gosma" || e.Sight != 6 {
		t.Fatalf("enemy 1 = %+v", e)
	}
	if len(e.DropItems) != 3 || e.DropItems[0].DataID != 1 || e.DropItems[0].Denominator != 2 {
		t.Errorf("enemy 1 drops = %+v", e.DropItems)
	}
	b := d.Enemies[2]
	if b == nil || b.ReviveTime != 10 || !b.Escape {
		t.Errorf("enemy 2 = %+v", b)
	}
	boss := d.Enemies[31]
	if boss == nil || !boss.Boss || boss.DisableSwitchID != 102 || boss.Sight != 10 {
		t.Errorf("enemy 31 = %+v", boss)
	}
}

func TestMapsAndEvents(t *testing.T) {
	d, err := Load(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	m := d.Maps[2]
	if m == nil || m.Name != "Caverna" || m.Width != 40 || m.Height != 25 {
		t.Fatalf("map 2 = %+v", m)
	}
	if !m.PvP {
		t.Errorf("map 2 is not PvP")
	}
	if len(m.Events) != 17 {
		t.Errorf("map 2 events = %d", len(m.Events))
	}
	boss := m.Events[14]
	if boss == nil || boss.Name != "Boss" || len(boss.Pages) != 1 {
		t.Fatalf("map 2 boss = %+v", boss)
	}
	page := boss.Pages[0]
	if len(page.List) < 2 || page.List[0].Code != 108 {
		t.Fatalf("boss list = %+v", page.List)
	}
	if s, _ := page.List[0].Parameters[0].(string); !strings.HasPrefix(s, "Enemy=31") {
		t.Errorf("boss enemy parameter = %#v", page.List[0].Parameters)
	}
	event3 := m.Events[3]
	if event3 == nil || event3.Name != "$Boss" {
		t.Fatalf("map 2 event 3 = %+v", event3)
	}
}

func TestCommonEvents(t *testing.T) {
	d, err := Load(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	ce := d.CommonEvents[1]
	if ce == nil || ce.Name != "Aprender Cura" || ce.SwitchID != 1 {
		t.Fatalf("common event 1 = %+v", ce)
	}
	found := false
	for _, c := range ce.List {
		if c.Code == 355 {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("common event 1 has no script command")
	}
}

func TestClassExpAndParams(t *testing.T) {
	d, err := Load(dataDir())
	if err != nil {
		t.Fatal(err)
	}
	c := d.Classes[1]
	if c == nil {
		t.Fatal("class 1 missing")
	}
	cases := []struct {
		level int
		exp   int
	}{
		{1, 0},
		{2, 50},
		{3, 162},
		{10, 5296},
		{50, 529492},
		{99, 2547133},
	}
	for _, tc := range cases {
		if got := c.ExpForLevel(tc.level); got != tc.exp {
			t.Errorf("ExpForLevel(%d) = %d, want %d", tc.level, got, tc.exp)
		}
	}
	wantParams := []int{562, 41, 20, 16, 10, 10, 20, 15}
	for i, want := range wantParams {
		if got := c.ParamAt(i, 1); got != want {
			t.Errorf("ParamAt(%d, 1) = %d, want %d", i, got, want)
		}
	}
}

func TestNoteParsing(t *testing.T) {
	note := "Sight=6\nReviveTime=10\nEscape=true\nAniIndex=1\nBoss=true\n"
	if got := NoteNumber(note, "Sight"); got != 6 {
		t.Errorf("Sight = %d", got)
	}
	if got := NoteNumber(note, "ReviveTime"); got != 10 {
		t.Errorf("ReviveTime = %d", got)
	}
	if !NoteBool(note, "Escape") || !NoteBool(note, "Boss") {
		t.Errorf("boolean notes not parsed")
	}
	if got, ok := NoteNumberPresent(note, "AniIndex"); !ok || got != 1 {
		t.Errorf("AniIndex = %d, %v", got, ok)
	}
	if _, ok := NoteNumberPresent(note, "Missing"); ok {
		t.Errorf("missing tag reported present")
	}
	refs := ParseSpriteRefs("Charset01/0, Charset02/3")
	if len(refs) != 2 || refs[0].Name != "Charset01" || refs[0].Index != 0 || refs[1].Index != 3 {
		t.Errorf("refs = %+v", refs)
	}
}
