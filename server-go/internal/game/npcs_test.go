package game

import (
	"context"
	"testing"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
)

func TestNPCTouchTransfer(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Walker", 5, 22)
	p.Character.Direction = 2
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := w.do(ctx, func() error {
		p.lastMove = time.Time{}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, p, 2); err != nil {
		t.Fatal(err)
	}
	if p.Character.MapID != 2 || p.Character.X != 4 || p.Character.Y != 24 {
		t.Fatalf("touch transfer = map %d (%d,%d)", p.Character.MapID, p.Character.X, p.Character.Y)
	}
}

func TestNPCWander(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Observer", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	ms := w.maps[1]
	npc := ms.npcs[18]
	if npc == nil {
		t.Fatal("guardian NPC missing")
	}
	startX, startY := npc.x, npc.y
	moved := false
	if err := w.do(ctx, func() error {
		for i := 0; i < 60 && !moved; i++ {
			npc.nextMove = time.Time{}
			w.tickNPCs(w.now())
			if npc.x != startX || npc.y != startY {
				moved = true
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !moved {
		t.Fatalf("NPC did not move from (%d,%d)", startX, startY)
	}
	if len(p.Client.(*fakeClient).find(protocol.ServerEventMove)) == 0 {
		t.Fatal("NPC movement was not broadcast")
	}
}

func TestEnemyApproachesPlayer(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Prey", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	enemy := firstEnemy(t, w, 1)
	p.Character.MapID = 1
	p.Character.HP = 100
	ms := w.maps[1]
	placed := false
	for _, off := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
		x, y := enemy.X+off[0]*2, enemy.Y+off[1]*2
		midX, midY := enemy.X+off[0], enemy.Y+off[1]
		if ms.valid(x, y) && ms.passable(x, y, 2) && ms.passable(midX, midY, 2) {
			p.Character.X, p.Character.Y = x, y
			placed = true
			break
		}
	}
	if !placed {
		t.Skip("no passable tile around the enemy")
	}
	startX, startY := enemy.X, enemy.Y
	if err := w.do(ctx, func() error {
		enemy.nextMove = time.Time{}
		enemy.lastAction = w.now().Add(-time.Hour)
		w.tickCombat()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if enemy.X == startX && enemy.Y == startY {
		t.Fatalf("enemy did not move toward the player at (%d,%d)", startX, startY)
	}
	if len(p.Client.(*fakeClient).find(protocol.ServerEventMove)) == 0 {
		t.Fatal("enemy movement was not broadcast")
	}
}
