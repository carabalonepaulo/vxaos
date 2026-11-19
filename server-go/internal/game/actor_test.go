package game

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kelaia/vxaos/server/internal/content"
	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

func testActorWorld(t *testing.T) (*World, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(context.Background(), store.Config{Driver: store.SQLite, DSN: filepath.Join(dir, "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := rpg.Load(filepath.Join("..", "..", "..", "Client", "Data"))
	if err != nil {
		t.Fatal(err)
	}
	quests, err := content.LoadQuests(filepath.Join("..", "..", "content", "quests.toml"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := New(Options{
		Data:         data,
		Store:        st,
		Tick:         20 * time.Millisecond,
		SaveInterval: time.Hour,
		MapIdle:      200 * time.Millisecond,
		Quests:       quests,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = w.Stop(ctx)
	})
	return w, st
}

func waitFrame(t *testing.T, client Client, op protocol.ServerOp, timeout time.Duration) protocol.ServerMessage {
	t.Helper()
	fc := client.(*fakeClient)
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if frames := fc.find(op); len(frames) > 0 {
			return frames[len(frames)-1]
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("no %v frame within %s", op, timeout)
	return nil
}

func waitRoute(t *testing.T, w *World, id uint16, mapID int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var key MapKey
		_ = w.do(context.Background(), func() error {
			if entry := w.online[id]; entry != nil {
				key = entry.Key
			}
			return nil
		})
		if key.TemplateID == mapID {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("player %d did not reach map %d", id, mapID)
}

func TestActorTradeFlowPersists(t *testing.T) {
	w, st := testActorWorld(t)
	ctx := context.Background()
	item := firstUsableItem(t, w)
	weapon := firstUsableWeapon(t, w)
	hero := createPlayer(t, st, "Hero", 20, 12)
	ally := createPlayer(t, st, "Ally", 22, 12)
	hero.State.Inventory = []store.ItemStack{{Kind: KindItem, ItemID: item.ID, Amount: 2}}
	ally.State.Inventory = []store.ItemStack{{Kind: KindWeapon, ItemID: weapon.ID, Amount: 1}}
	hero.Character.Gold, ally.Character.Gold = 100, 50
	if err := w.Join(ctx, hero); err != nil {
		t.Fatal(err)
	}
	if err := w.Join(ctx, ally); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, hero.Client, protocol.ServerEnterWorld, 2*time.Second)
	waitFrame(t, ally.Client, protocol.ServerEnterWorld, 2*time.Second)

	if err := w.PlayerRequest(ctx, hero, requestTrade, ally.ID); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, ally.Client, protocol.ServerRequest, 2*time.Second)
	if err := w.AcceptRequest(ctx, ally); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, hero.Client, protocol.ServerRequestAccept, 2*time.Second)
	waitFrame(t, ally.Client, protocol.ServerRequestAccept, 2*time.Second)

	if err := w.TradeItem(ctx, hero, uint16(item.ID), KindItem, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.TradeGold(ctx, hero, 30); err != nil {
		t.Fatal(err)
	}
	if err := w.TradeItem(ctx, ally, uint16(weapon.ID), KindWeapon, 1); err != nil {
		t.Fatal(err)
	}
	if err := w.TradeGold(ctx, ally, 10); err != nil {
		t.Fatal(err)
	}
	if err := w.RequestFinishTrade(ctx, hero); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, ally.Client, protocol.ServerRequest, 2*time.Second)
	if err := w.AcceptRequest(ctx, ally); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, hero.Client, protocol.ServerWindowClose, 2*time.Second)
	waitFrame(t, ally.Client, protocol.ServerWindowClose, 2*time.Second)

	if err := w.Leave(ctx, hero, true); err != nil {
		t.Fatal(err)
	}
	if err := w.Leave(ctx, ally, true); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		hc, hs, err := st.GetCharacter(ctx, hero.Character.ID)
		ac, as, err2 := st.GetCharacter(ctx, ally.Character.ID)
		if err == nil && err2 == nil && hc.Gold == 80 && ac.Gold == 70 &&
			findInventory(hs, KindWeapon, weapon.ID) >= 0 && findInventory(as, KindItem, item.ID) >= 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("trade result was not persisted")
}

func TestActorGuildFlow(t *testing.T) {
	w, st := testActorWorld(t)
	ctx := context.Background()
	leader := createPlayer(t, st, "Leader", 20, 12)
	member := createPlayer(t, st, "Member", 22, 12)
	if err := w.Join(ctx, leader); err != nil {
		t.Fatal(err)
	}
	if err := w.Join(ctx, member); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, leader.Client, protocol.ServerEnterWorld, 2*time.Second)
	waitFrame(t, member.Client, protocol.ServerEnterWorld, 2*time.Second)

	if err := w.onPlayer(ctx, leader.ID, func(ms *mapState, p *Player) error {
		p.CreatingGuild = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	flag := make([]byte, 64)
	for i := range flag {
		flag[i] = 255
	}
	flag[0] = 3
	if err := w.CreateGuild(ctx, leader, "  test   guild ", flag); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, leader.Client, protocol.ServerGuildName, 2*time.Second)

	if err := w.InviteGuildMember(ctx, leader, "Member"); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, member.Client, protocol.ServerRequest, 2*time.Second)
	if err := w.AcceptRequest(ctx, member); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, member.Client, protocol.ServerGuildName, 2*time.Second)

	var memberGuild string
	if err := w.onPlayer(ctx, member.ID, func(ms *mapState, p *Player) error {
		memberGuild = p.GuildName
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if memberGuild != "Test Guild" {
		t.Fatalf("member guild = %q", memberGuild)
	}
	if err := w.OpenGuild(ctx, leader); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, leader.Client, protocol.ServerGuildOpen, 2*time.Second)
	if err := w.LeaveGuild(ctx, member); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	cleared := false
	for time.Now().Before(deadline) {
		var name string
		_ = w.onPlayer(ctx, member.ID, func(ms *mapState, p *Player) error {
			name = p.GuildName
			return nil
		})
		if name == "" {
			cleared = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !cleared {
		t.Fatal("member guild name was not cleared")
	}
}

func TestActorJoinMoveAndTransfer(t *testing.T) {
	w, st := testActorWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Walker", 5, 22)
	p.Character.Direction = 2
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, p.Client, protocol.ServerEnterWorld, 2*time.Second)
	time.Sleep(120 * time.Millisecond)

	// This step touches the NPC that transfers the player to map 2.
	if err := w.Move(ctx, p, 2); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, p.Client, protocol.ServerTransfer, 2*time.Second)
	waitRoute(t, w, p.ID, 2, 2*time.Second)

	time.Sleep(120 * time.Millisecond)
	if err := w.Move(ctx, p, 2); err != nil {
		t.Fatalf("move after transfer: %v", err)
	}
	waitFrame(t, p.Client, protocol.ServerPlayerMove, 2*time.Second)
}

func TestActorIdleUnloadAndReactivate(t *testing.T) {
	w, st := testActorWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Sleeper", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, p.Client, protocol.ServerEnterWorld, 2*time.Second)

	if err := w.Leave(ctx, p, false); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		actors := 0
		_ = w.do(ctx, func() error {
			actors = len(w.actors)
			return nil
		})
		if actors == 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	var remaining int
	_ = w.do(ctx, func() error {
		remaining = len(w.actors)
		return nil
	})
	if remaining != 0 {
		t.Fatalf("map was not unloaded: %d actors", remaining)
	}

	again := createPlayer(t, st, "Waker", 21, 12)
	if err := w.Join(ctx, again); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, again.Client, protocol.ServerEnterWorld, 2*time.Second)
}

func TestActorPartyFlow(t *testing.T) {
	w, st := testActorWorld(t)
	ctx := context.Background()
	leader := createPlayer(t, st, "Leader", 20, 12)
	ally := createPlayer(t, st, "Ally", 22, 12)
	if err := w.Join(ctx, leader); err != nil {
		t.Fatal(err)
	}
	if err := w.Join(ctx, ally); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, leader.Client, protocol.ServerEnterWorld, 2*time.Second)
	waitFrame(t, ally.Client, protocol.ServerEnterWorld, 2*time.Second)

	if err := w.PlayerRequest(ctx, leader, requestParty, ally.ID); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, ally.Client, protocol.ServerRequest, 2*time.Second)
	if err := w.AcceptRequest(ctx, ally); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, leader.Client, protocol.ServerPartyJoin, 2*time.Second)
	waitFrame(t, ally.Client, protocol.ServerPartyJoin, 2*time.Second)

	if err := w.Chat(ctx, leader, 2, "hello", ""); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, ally.Client, protocol.ServerChat, 2*time.Second)

	if err := w.LeaveParty(ctx, ally); err != nil {
		t.Fatal(err)
	}
	waitFrame(t, ally.Client, protocol.ServerPartyDissolve, 2*time.Second)
	waitFrame(t, leader.Client, protocol.ServerPartyDissolve, 2*time.Second)
}
