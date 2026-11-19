package game

import (
	"context"
	"testing"
	"time"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/rpg"
)

func TestStateAndBuffFlow(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Buffer", 21, 12)
	p.State.Skills = []int{2, 31, 41, 45}
	p.Character.MP, p.Character.MMP = 100, 100
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	use := func(id uint16) {
		t.Helper()
		if err := w.do(ctx, func() error {
			p.lastAttack = time.Time{}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if err := w.UseSkill(ctx, p, id); err != nil {
			t.Fatalf("skill %d: %v", id, err)
		}
	}

	// Defender adds state 9 to the caster.
	use(2)
	if !p.Status.hasState(9) {
		t.Fatalf("defender state missing: %+v", p.Status.States)
	}
	client := p.Client.(*fakeClient)
	if len(client.find(protocol.ServerState)) == 0 {
		t.Fatal("state add was not sent")
	}

	// Attack buff raises the ATK used by formulas by 25%.
	before := w.playerStats(p).Params[2]
	use(41)
	if p.Status.Buffs[2].Level != 1 || p.Status.Buffs[2].Turns <= 0 {
		t.Fatalf("attack buff = %+v", p.Status.Buffs[2])
	}
	after := w.playerStats(p).Params[2]
	if after <= before {
		t.Fatalf("attack buff did not raise ATK: %d -> %d", before, after)
	}
	if len(client.find(protocol.ServerBuff)) == 0 {
		t.Fatal("buff add was not sent")
	}

	// Poison and paralysis, then cleanse with Tratamento.
	if err := w.do(ctx, func() error {
		w.addState(&p.Status, 2, 1, p)
		w.addState(&p.Status, 7, 1, p)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	use(31)
	if p.Status.hasState(2) {
		t.Fatal("poison was not removed")
	}
	if p.Status.hasState(7) {
		t.Fatal("paralysis was not removed")
	}

	// Attack-state features apply states to the target.
	if err := w.do(ctx, func() error {
		w.applyAttackStates([]rpg.Feature{{Code: 32, DataID: 2, Value: 1}}, nil, &p.Status, p)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !p.Status.hasState(2) {
		t.Fatal("attack state was not applied")
	}

	// Ten seconds later the defend state and the buff expire (and poison stays
	// because it has no auto removal).
	if err := w.do(ctx, func() error {
		base := w.now()
		for i := 1; i <= 10; i++ {
			w.tickStatus(base.Add(time.Duration(i)*time.Second), &p.Status, p, w.playerFeatures(p), p.Character.MHP, p.Character.MMP, &p.Character.HP, &p.Character.MP)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if p.Status.hasState(9) {
		t.Fatal("defend state did not expire")
	}
	if p.Status.Buffs[2].Level != 0 {
		t.Fatalf("buff did not expire: %+v", p.Status.Buffs[2])
	}
	if !p.Status.hasState(2) {
		t.Fatal("permanent poison expired")
	}
}

func TestEnemyDebuffAndElementRate(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Hexer", 20, 12)
	p.State.Skills = []int{45}
	p.Character.MP, p.Character.MMP = 100, 100
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	enemy := firstEnemy(t, w, 1)
	p.Character.X, p.Character.Y = enemy.X-1, enemy.Y
	p.Character.Direction = 6
	p.TargetType, p.TargetID = 2, uint16(enemy.Event.ID)
	p.lastAttack = time.Time{}
	if err := w.UseSkill(ctx, p, 45); err != nil {
		t.Fatal(err)
	}
	if enemy.Status.Buffs[2].Level != -1 {
		t.Fatalf("enemy debuff = %+v", enemy.Status.Buffs[2])
	}

	user := battleStats{AttackElements: []int{2}}
	target := battleStats{}
	for i := range target.ElementRates {
		target.ElementRates[i] = 1
	}
	target.ElementRates[2] = 0.5
	if rate := elementRate(user, target, 0); rate != 0.5 {
		t.Fatalf("attack element rate = %v", rate)
	}
	if rate := elementRate(battleStats{}, target, 2); rate != 0.5 {
		t.Fatalf("damage element rate = %v", rate)
	}
}
