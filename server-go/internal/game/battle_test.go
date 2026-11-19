package game

import (
	"context"
	"testing"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/store"
)

func firstEnemy(t *testing.T, w *World, mapID int) *enemyState {
	t.Helper()
	ms := w.maps[mapID]
	if ms == nil {
		t.Fatalf("map %d missing", mapID)
	}
	for _, enemy := range ms.enemies {
		return enemy
	}
	t.Fatalf("map %d has no enemies", mapID)
	return nil
}

func TestFormulaEvaluator(t *testing.T) {
	a := battleStats{Params: [8]int{100, 20, 10, 3, 4, 5, 6, 7}, MHP: 100, MMP: 20, Level: 1}
	b := battleStats{Params: [8]int{80, 10, 8, 2, 1, 1, 1, 1}, MHP: 80, MMP: 10, Level: 1}
	if got := evalFormula(`(- (* (get a "atk") 4) (* (get b "def") 2))`, a, b, nil); got != 36 {
		t.Fatalf("attack formula = %v", got)
	}
	if got := evalFormula(`(+ (get v 1) (sqrt 9))`, a, b, []int32{0, 2}); got != 5 {
		t.Fatalf("variable formula = %v", got)
	}
	if got := evalFormula(`(/ (+ (get a "atk") 2) 2)`, a, b, nil); got != 6 {
		t.Fatalf("parenthesized formula = %v", got)
	}
	if got := evalFormula(`(this is not a formula)`, a, b, nil); got != 0 {
		t.Fatalf("invalid formula = %v", got)
	}
}

func TestPlayerAttackKillsEnemy(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	enemy := firstEnemy(t, w, 1)
	p := createPlayer(t, st, "Fighter", enemy.X-1, enemy.Y)
	p.Character.Direction = 6
	p.Character.ATK = 200
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	client := p.Client.(*fakeClient)
	for i := 0; i < 10 && enemy.Alive; i++ {
		p.lastAttack = w.now().Add(-time.Second)
		if err := w.Attack(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	if enemy.Alive {
		t.Fatalf("enemy still alive after attacks: hp=%d", enemy.HP)
	}
	if p.Character.Exp <= 0 {
		t.Fatal("enemy gave no experience")
	}
	if len(client.find(protocol.ServerAttackEnemy)) == 0 {
		t.Fatal("no AttackEnemy message")
	}
	enemy.ReviveAt = w.now().Add(-1)
	w.tickCombat()
	if !enemy.Alive || enemy.HP != enemy.Enemy.Params[0] {
		t.Fatalf("enemy did not revive: %+v", enemy)
	}
}

func TestEnemyAttacksPlayer(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	enemy := firstEnemy(t, w, 1)
	p := createPlayer(t, st, "Victim", enemy.X, enemy.Y+1)
	p.Character.Direction = 8
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	enemy.Direction = 2
	enemy.lastAction = w.now().Add(-10 * enemyAttackDelay)
	before := p.Character.HP
	for i := 0; i < 20 && p.Character.HP == before; i++ {
		w.enemyAttack(enemy, p)
	}
	if p.Character.HP >= before {
		t.Fatalf("enemy did not damage player: %d -> %d", before, p.Character.HP)
	}
	if len(p.Client.(*fakeClient).find(protocol.ServerAttackPlayer)) == 0 {
		t.Fatal("no AttackPlayer message")
	}
}

func TestUseItemAndSkill(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	item := firstUsableItem(t, w)
	p := createPlayer(t, st, "Healer", 21, 12)
	p.Character.Level = 2
	p.Character.HP = 1
	p.State.Inventory = []store.ItemStack{{Kind: KindItem, ItemID: item.ID, Amount: 2}}
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := w.UseItem(ctx, p, uint16(item.ID)); err != nil {
		t.Fatal(err)
	}
	if p.Character.HP <= 1 {
		t.Fatalf("item did not heal: %d/%d", p.Character.HP, p.Character.MHP)
	}
	slot := findInventory(p.State, KindItem, item.ID)
	if slot < 0 || p.State.Inventory[slot].Amount != 1 {
		t.Fatalf("item was not consumed: %+v", p.State.Inventory)
	}

	enemy := firstEnemy(t, w, 1)
	p.Character.MapID = 1
	p.Character.X = enemy.X - 1
	p.Character.Y = enemy.Y
	p.Character.Direction = 6
	p.TargetType, p.TargetID = 2, uint16(enemy.Event.ID)
	p.State.Skills = append(p.State.Skills, 1)
	before := enemy.HP
	for i := 0; i < 20 && enemy.HP == before; i++ {
		p.Character.MP = p.Character.MMP
		p.lastAttack = w.now().Add(-time.Second)
		if err := w.UseSkill(ctx, p, 1); err != nil {
			t.Fatal(err)
		}
	}
	if enemy.HP >= before {
		t.Fatalf("skill did not damage enemy: %d -> %d", before, enemy.HP)
	}
}
