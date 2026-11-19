package game

import (
	"context"
	"testing"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
)

func TestEnemyAggroSightAndHysteresis(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Scout", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	enemy := firstEnemy(t, w, 1)
	sight := enemy.Enemy.Sight
	if sight <= 0 {
		sight = 6
	}

	// Outside sight: no target and no balloon.
	p.Character.X, p.Character.Y = enemy.X+sight+1, enemy.Y
	if err := w.do(ctx, func() error {
		enemy.nextMove = w.now().Add(time.Hour)
		w.updateEnemyAI(w.maps[1], enemy, w.now())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if enemy.TargetID != 0 {
		t.Fatalf("target acquired outside sight: %d", enemy.TargetID)
	}

	// Inside sight: target acquired and balloon sent.
	p.Character.X = enemy.X + sight
	if err := w.do(ctx, func() error {
		w.updateEnemyAI(w.maps[1], enemy, w.now())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if enemy.TargetID != p.ID {
		t.Fatalf("target = %d want %d", enemy.TargetID, p.ID)
	}
	if len(p.Client.(*fakeClient).find(protocol.ServerBalloon)) == 0 {
		t.Fatal("aggro balloon was not sent")
	}

	// Between sight and sight+5 the enemy keeps chasing.
	p.Character.X = enemy.X + sight + 4
	if err := w.do(ctx, func() error {
		w.updateEnemyAI(w.maps[1], enemy, w.now())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if enemy.TargetID != p.ID {
		t.Fatal("enemy dropped a target inside sight+5")
	}

	// Beyond sight+5 the target is dropped.
	p.Character.X = enemy.X + sight + 6
	if err := w.do(ctx, func() error {
		w.updateEnemyAI(w.maps[1], enemy, w.now())
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if enemy.TargetID != 0 {
		t.Fatalf("enemy kept a target beyond sight+5: %d", enemy.TargetID)
	}
	_ = ctx
}

func TestEnemyDisableAndReviveRegion(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Hunter", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	// Direct disable behavior: global switch 102 and variable 2.
	enemy := &enemyState{Enemy: &rpg.Enemy{DisableSwitchID: 102, DisableVariableID: 2}}
	if err := w.do(ctx, func() error {
		w.disableEnemy(p, enemy)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !w.globalSwitchValue(102) {
		t.Fatal("disable global switch was not toggled")
	}
	if len(p.State.Variables) < 2 || p.State.Variables[1] != 1 {
		t.Fatalf("disable variable = %+v", p.State.Variables)
	}

	// A region=2 enemy revives at a tile of region 2.
	ms := w.maps[2]
	var regional *enemyState
	for _, candidate := range ms.enemies {
		if candidate == nil {
			continue
		}
		if candidate.regionID == 2 {
			regional = candidate
			break
		}
	}
	if regional == nil {
		t.Fatal("no region=2 enemy")
	}
	if err := w.do(ctx, func() error {
		regional.Alive = false
		regional.ReviveAt = w.now().Add(-time.Second)
		w.tickCombat()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !regional.Alive {
		t.Fatal("enemy did not revive")
	}
	if int(ms.def.Data.At(int32(regional.X), int32(regional.Y), 3))>>8 != 2 {
		t.Fatalf("enemy revived outside region 2 at (%d,%d)", regional.X, regional.Y)
	}
}
