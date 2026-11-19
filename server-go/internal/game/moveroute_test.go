package game

import (
	"testing"

	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

func TestCustomMoveRouteStraightAndWait(t *testing.T) {
	w, st := testWorld(t)
	p := createPlayer(t, st, "Route", 21, 12)
	ms := w.ensureMap(1)
	x, y := findOpenTile(t, ms, 2, 0)
	npc := &npcState{
		event:     &rpg.Event{ID: 999, Pages: []*rpg.Page{{}}},
		page:      &rpg.Page{MoveType: 3},
		x:         x,
		y:         y,
		moveType:  3,
		moveSpeed: 3, moveFrequency: 5,
		route: &rpg.MoveRoute{List: []rpg.MoveCommand{
			{Code: 3}, // move right
			{Code: 15, Parameters: []any{0}},
			{Code: 1}, // move down
		}},
	}
	ms.npcs = ensureIndex(ms.npcs, 999)
	ms.npcs[999] = npc
	defer func() { ms.npcs[999] = nil }()

	w.runNPCMoveRoute(ms, npc, w.now())
	if npc.x != x+1 || npc.y != y {
		t.Fatalf("after right move = (%d,%d), want (%d,%d)", npc.x, npc.y, x+1, y)
	}
	w.runNPCMoveRoute(ms, npc, w.now())
	if npc.x != x+1 || npc.y != y {
		t.Fatalf("wait command moved the npc to (%d,%d)", npc.x, npc.y)
	}
	w.runNPCMoveRoute(ms, npc, w.now())
	if npc.x != x+1 || npc.y != y+1 {
		t.Fatalf("after down move = (%d,%d), want (%d,%d)", npc.x, npc.y, x+1, y+1)
	}
	// Repeating routes wrap back to the first command.
	npc.route.Repeat = true
	w.runNPCMoveRoute(ms, npc, w.now())
	if npc.routeIndex != 1 {
		t.Fatalf("repeat index = %d", npc.routeIndex)
	}
	_ = p
}

func TestCustomMoveRouteTurnsAndSwitches(t *testing.T) {
	w, st := testWorld(t)
	p := createPlayer(t, st, "Turner", 21, 12)
	_ = p
	ms := w.ensureMap(1)
	x, y := findOpenTile(t, ms, 0, 0)
	npc := &npcState{
		event:     &rpg.Event{ID: 998},
		page:      &rpg.Page{MoveType: 3},
		x:         x,
		y:         y,
		direction: 2,
		moveType:  3,
		route: &rpg.MoveRoute{List: []rpg.MoveCommand{
			{Code: 18}, // turn right
			{Code: 27, Parameters: []any{w.opts.MaxPlayerSwitches + 1}},
		}},
	}
	ms.npcs = ensureIndex(ms.npcs, 998)
	ms.npcs[998] = npc
	defer func() { ms.npcs[998] = nil }()

	w.runNPCMoveRoute(ms, npc, w.now())
	if npc.direction != 6 {
		t.Fatalf("turn right direction = %d", npc.direction)
	}
	w.runNPCMoveRoute(ms, npc, w.now())
	if !w.globalSwitchValue(w.opts.MaxPlayerSwitches + 1) {
		t.Fatal("move route switch did not turn on")
	}
	if npc.x != x || npc.y != y {
		t.Fatalf("turn/switch commands moved the npc")
	}
}

func TestPageConditionItemAndActor(t *testing.T) {
	w, st := testWorld(t)
	p := createPlayer(t, st, "Condition", 21, 12)
	event := &rpg.Event{ID: 77, Pages: []*rpg.Page{
		{List: []*rpg.EventCommand{{Code: 0}}},
		{Condition: rpg.Condition{ItemValid: true, ItemID: 1}, List: []*rpg.EventCommand{{Code: 101}}},
	}}
	if page := w.properPage(p, event); page == nil || len(page.List) == 0 || page.List[0].Code != 0 {
		t.Fatalf("item page selected without the item")
	}
	p.State.Inventory = []store.ItemStack{{Kind: KindItem, ItemID: 1, Amount: 1}}
	if page := w.properPage(p, event); page == nil || page.List[0].Code != 101 {
		t.Fatalf("item page not selected with the item")
	}
	event.Pages = append(event.Pages, &rpg.Page{
		Condition: rpg.Condition{ActorValid: true, ActorID: 1},
		List:      []*rpg.EventCommand{{Code: 115}},
	})
	if page := w.properPage(p, event); page == nil || page.List[0].Code != 115 {
		t.Fatalf("actor page not selected")
	}
}

func findOpenTile(t *testing.T, ms *mapState, dx, dy int) (int, int) {
	t.Helper()
	for x := 1; x < ms.def.Width-2; x++ {
		for y := 1; y < ms.def.Height-2; y++ {
			ok := true
			for _, step := range [][2]int{{0, 0}, {dx, dy}} {
				tx, ty := x+step[0], y+step[1]
				if !ms.passable(tx, ty, 2) || ms.blocked(tx, ty, nil) {
					ok = false
					break
				}
			}
			if ok {
				return x, y
			}
		}
	}
	t.Fatal("no open tile")
	return 0, 0
}
