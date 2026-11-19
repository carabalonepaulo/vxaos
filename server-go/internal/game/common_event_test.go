package game

import (
	"context"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/store"
)

func TestItemCommonEventRuns(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Scholar", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	// Item 17 ("Pergaminho") fires common event 1, which teaches skill 26.
	p.State.Inventory = []store.ItemStack{{Kind: KindItem, ItemID: 17, Amount: 1}}
	p.State.Skills = nil
	if err := w.UseItem(ctx, p, 17); err != nil {
		t.Fatalf("use item: %v", err)
	}
	if !hasSkill(p.State, 26) {
		t.Fatalf("skill 26 not learned, skills=%v", p.State.Skills)
	}
	if idx := findInventory(p.State, KindItem, 17); idx >= 0 {
		t.Fatalf("scroll was not consumed: %+v", p.State.Inventory)
	}
	if p.Event != nil {
		t.Fatalf("event still running: %+v", p.Event)
	}
}
