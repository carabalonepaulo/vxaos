package game

import (
	"context"
	"testing"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
)

func TestPlayerAreaSkillHitsEveryEnemy(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Mage", 20, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.State.Skills = []int{8}
	p.Character.MP = 500
	ms := w.maps[1]
	first := &enemyState{
		Event: &rpg.Event{ID: 900}, Enemy: &rpg.Enemy{Params: [8]int{1000, 100, 50, 10, 10, 10, 10, 10}},
		ID: 1, X: 20, Y: 13, Alive: true, HP: 1000,
	}
	second := &enemyState{
		Event: &rpg.Event{ID: 901}, Enemy: &rpg.Enemy{Params: [8]int{1000, 100, 50, 10, 10, 10, 10, 10}},
		ID: 2, X: 21, Y: 12, Alive: true, HP: 1000,
	}
	ms.enemies = ensureIndex(ms.enemies, 901)
	ms.enemies[900], ms.enemies[901] = first, second
	defer func() { ms.enemies[900], ms.enemies[901] = nil, nil }()

	for attempt := 0; attempt < 8 && (first.HP >= 1000 || second.HP >= 1000); attempt++ {
		p.Character.MP = 500
		p.lastAttack = time.Time{}
		if err := w.UseSkill(ctx, p, 8); err != nil {
			t.Fatalf("skill: %v", err)
		}
	}
	if first.HP >= 1000 || second.HP >= 1000 {
		t.Fatalf("area skill did not hit both: %d %d", first.HP, second.HP)
	}
}

func TestProjectileSkillBroadcast(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Archer", 20, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.State.Skills = []int{51}
	p.Character.MP = 500
	ms := w.maps[1]
	enemy := &enemyState{
		Event: &rpg.Event{ID: 902}, Enemy: &rpg.Enemy{Params: [8]int{1000, 100, 50, 10, 10, 10, 10, 10}},
		ID: 3, X: 25, Y: 12, Alive: true, HP: 1000,
	}
	ms.enemies = ensureIndex(ms.enemies, 902)
	ms.enemies[902] = enemy
	defer func() { ms.enemies[902] = nil }()
	if err := w.SetTarget(ctx, p, 2, 902); err != nil {
		t.Fatal(err)
	}
	if err := w.UseSkill(ctx, p, 51); err != nil {
		t.Fatalf("skill: %v", err)
	}
	if len(p.Client.(*fakeClient).find(protocol.ServerProjectile)) == 0 {
		t.Fatal("projectile was not broadcast")
	}
}

func TestPlayerDeathLosesGoldExpAndTransfers(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Fallen", 20, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.Character.Gold = 1000
	p.Character.Exp = 10000
	p.Character.ReviveMapID = 1
	p.Character.ReviveX, p.Character.ReviveY = 21, 12
	p.Character.HP = 0
	if err := w.do(ctx, func() error {
		w.addState(&p.Status, 2, 1, p)
		w.playerDied(p)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if p.Character.Gold != 950 {
		t.Fatalf("gold after death = %d", p.Character.Gold)
	}
	if p.Character.Exp != 9600 {
		t.Fatalf("exp after death = %d", p.Character.Exp)
	}
	if p.Character.HP != p.Character.MHP || p.Character.MP != p.Character.MMP {
		t.Fatalf("vitals after death = %d/%d %d/%d", p.Character.HP, p.Character.MHP, p.Character.MP, p.Character.MMP)
	}
	if p.Character.X != 21 || p.Character.Y != 12 {
		t.Fatalf("death position = (%d,%d)", p.Character.X, p.Character.Y)
	}
	if p.Status.any() {
		t.Fatalf("states after death = %+v", p.Status.States)
	}
}
