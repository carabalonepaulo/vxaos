package game

import (
	"context"
	"testing"
	"time"

	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

func TestDropAndPickup(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Dropper", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.State.Inventory = []store.ItemStack{{Kind: KindItem, ItemID: 1, Amount: 3}}
	p.Character.X, p.Character.Y = 22, 12

	if err := w.DropItem(ctx, p, 1, KindItem, 2); err != nil {
		t.Fatalf("drop: %v", err)
	}
	ms := w.ensureMap(1)
	if len(ms.drops) != 1 {
		t.Fatalf("drops = %d", len(ms.drops))
	}
	drop := ms.drops[0]
	if drop.ItemID != 1 || drop.Amount != 2 || drop.X != 22 || drop.Y != 12 {
		t.Fatalf("drop = %+v", drop)
	}
	if idx := findInventory(p.State, KindItem, 1); idx < 0 || p.State.Inventory[idx].Amount != 1 {
		t.Fatalf("inventory after drop = %+v", p.State.Inventory)
	}

	// The client must be standing on the drop to take it.
	p.Character.X, p.Character.Y = 22, 13
	if err := w.PickupDrop(ctx, p, 0); err != ErrNotOnDrop {
		t.Fatalf("pickup away from drop err = %v", err)
	}
	p.Character.X, p.Character.Y = 22, 12
	if err := w.PickupDrop(ctx, p, 0); err != nil {
		t.Fatalf("pickup: %v", err)
	}
	if len(ms.drops) != 0 {
		t.Fatalf("drops after pickup = %d", len(ms.drops))
	}
	if idx := findInventory(p.State, KindItem, 1); idx < 0 || p.State.Inventory[idx].Amount != 3 {
		t.Fatalf("inventory after pickup = %+v", p.State.Inventory)
	}
}

func TestDropOwnerAndDespawn(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Owner", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	ms := w.ensureMap(1)
	if err := w.do(ctx, func() error {
		w.addMapDrop(ms, KindItem, 1, 1, 21, 12, "SomeoneElse", 0)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	drop := ms.drops[0]
	if w.canPickupDrop(drop, p) {
		t.Fatal("pickup allowed before the delay for another owner")
	}
	if err := w.do(ctx, func() error {
		drop.PickupAt = w.now().Add(-time.Second)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !w.canPickupDrop(drop, p) {
		t.Fatal("pickup still blocked after the delay")
	}
	if err := w.do(ctx, func() error {
		drop.DespawnAt = w.now().Add(-time.Second)
		w.tickDrops(w.now())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(ms.drops) != 0 {
		t.Fatalf("expired drop survived: %d", len(ms.drops))
	}
}

func TestEnemyDropSpawnsGroundItem(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Farmer", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	ms := w.ensureMap(1)
	enemy := &enemyState{
		Enemy: &rpg.Enemy{DropItems: []rpg.EnemyDrop{{Kind: KindItem, DataID: 1, Denominator: 1}}},
		X:     21, Y: 12, Alive: true,
	}
	if err := w.do(ctx, func() error {
		w.awardKill(p, enemy)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(ms.drops) != 1 {
		t.Fatalf("ground drops = %d", len(ms.drops))
	}
	if ms.drops[0].Owner != p.Character.Name {
		t.Fatalf("drop owner = %q", ms.drops[0].Owner)
	}
	if findInventory(p.State, KindItem, 1) >= 0 {
		t.Fatal("drop went straight to inventory")
	}
}
