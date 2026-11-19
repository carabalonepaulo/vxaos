package chargen

import (
	"path/filepath"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/rpg"
)

func testOptions(t *testing.T) Options {
	t.Helper()
	data, err := rpg.Load(filepath.Join("..", "..", "..", "Client", "Data"))
	if err != nil {
		t.Fatal(err)
	}
	return Options{
		Data:               data,
		MaxEquips:          9,
		MaxHotbar:          9,
		MaxPlayerSwitches:  100,
		MaxPlayerVariables: 100,
		StartPoints:        10,
		MaxActors:          3,
		MinCharacters:      3,
		MaxCharacters:      13,
	}
}

func TestCreate(t *testing.T) {
	opts := testOptions(t)
	c, st, err := Create(opts, 42, 0, "Hero", 1, 0, 0, [8]int{})
	if err != nil {
		t.Fatal(err)
	}
	if c.Level != 1 || c.Exp != 0 || c.HP != 562 || c.MP != 41 {
		t.Fatalf("character = %+v", c)
	}
	if c.ATK != 20 || c.DEF != 16 || c.MAT != 10 || c.AGI != 20 {
		t.Fatalf("params = %+v", c)
	}
	if c.CharacterName != "Charset01" || c.CharacterIndex != 0 || c.FaceName != "Charset01" {
		t.Fatalf("graphics = %+v", c)
	}
	if c.MapID != 1 || c.X != 21 || c.Y != 12 || c.Direction != 2 {
		t.Fatalf("position = %+v", c)
	}
	if len(st.Equips) != 9 || st.Equips[0] != 1 || st.Equips[1] != 46 {
		t.Fatalf("equips = %+v", st.Equips)
	}
	if len(st.Skills) != 1 || st.Skills[0] != 80 {
		t.Fatalf("skills = %+v", st.Skills)
	}
	if len(st.Switches) != 100 || len(st.Variables) != 100 || len(st.Hotbar) != 9 {
		t.Fatalf("state sizes = %d/%d/%d", len(st.Switches), len(st.Variables), len(st.Hotbar))
	}
}

func TestCreateFemaleArcher(t *testing.T) {
	opts := testOptions(t)
	c, _, err := Create(opts, 1, 1, "Arqueira", 6, 1, 0, [8]int{})
	if err != nil {
		t.Fatal(err)
	}
	if c.CharacterName != "Charset03" || c.CharacterIndex != 5 {
		t.Fatalf("female archer graphics = %+v", c)
	}
}

func TestCreateWithAllocatedPoints(t *testing.T) {
	opts := testOptions(t)
	c, _, err := Create(opts, 1, 0, "  hERO  mAN  ", 1, 0, 1, [8]int{1, 0, 1, 0, 0, 0, 0, 0})
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "Hero Man" {
		t.Fatalf("titleize = %q", c.Name)
	}
	if c.MHP != 572 || c.ATK != 21 || c.Points != 8 {
		t.Fatalf("allocated stats = %+v", c)
	}
	if c.CharacterName != "Charset01" || c.CharacterIndex != 2 {
		t.Fatalf("graphic index = %+v", c)
	}
	if _, _, err := Create(opts, 1, 0, "Hero", 1, 0, 0, [8]int{10, 10, 10, 10, 10, 10, 10, 10}); err == nil {
		t.Fatal("over-allocation accepted")
	}
}

func TestInvalid(t *testing.T) {
	opts := testOptions(t)
	if _, _, err := Create(opts, 1, 0, "Hi", 1, 0, 0, [8]int{}); err == nil {
		t.Fatal("short name accepted")
	}
	if _, _, err := Create(opts, 1, 0, "Hero", 0, 0, 0, [8]int{}); err == nil {
		t.Fatal("class 0 accepted")
	}
	if _, _, err := Create(opts, 1, 0, "Hero", 1, 2, 0, [8]int{}); err == nil {
		t.Fatal("sex 2 accepted")
	}
	if _, _, err := Create(opts, 1, 3, "Hero", 1, 0, 0, [8]int{}); err == nil {
		t.Fatal("slot 3 accepted")
	}
}
