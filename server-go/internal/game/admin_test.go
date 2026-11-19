package game

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
)

func TestAdminCommandPermissionsAndTeleport(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	admin := createPlayer(t, st, "Admin", 21, 12)
	admin.Account.GroupID = 2
	admin.IP = "10.0.0.1"
	if err := w.Join(ctx, admin); err != nil {
		t.Fatal(err)
	}
	hero := createPlayer(t, st, "Target", 21, 13)
	if err := w.Join(ctx, hero); err != nil {
		t.Fatal(err)
	}

	if err := w.AdminCommand(ctx, hero, protocol.AdminCommand{Command: AdminTeleport, Text: "Target", Int1: 1, Int2: 5, Int3: 5}); err != ErrAdminOnly {
		t.Fatalf("standard account error = %v", err)
	}
	if err := w.AdminCommand(ctx, admin, protocol.AdminCommand{Command: AdminTeleport, Text: "Target", Int1: 1, Int2: 5, Int3: 5}); err != nil {
		t.Fatalf("teleport: %v", err)
	}
	if hero.Character.X != 5 || hero.Character.Y != 5 {
		t.Fatalf("target position = (%d,%d)", hero.Character.X, hero.Character.Y)
	}
	if err := w.AdminCommand(ctx, admin, protocol.AdminCommand{Command: AdminGo, Text: "Target"}); err != nil {
		t.Fatalf("go: %v", err)
	}
	if admin.Character.X != 5 || admin.Character.Y != 5 {
		t.Fatalf("admin position = (%d,%d)", admin.Character.X, admin.Character.Y)
	}
}

func TestAdminItemGoldMuteAndSwitch(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	admin := createPlayer(t, st, "Admin2", 21, 12)
	admin.Account.GroupID = 2
	if err := w.Join(ctx, admin); err != nil {
		t.Fatal(err)
	}
	hero := createPlayer(t, st, "Receiver", 21, 13)
	if err := w.Join(ctx, hero); err != nil {
		t.Fatal(err)
	}
	if err := w.AdminCommand(ctx, admin, protocol.AdminCommand{Command: AdminItem, Text: "Receiver", Int1: 1, Int2: 3}); err != nil {
		t.Fatalf("give item: %v", err)
	}
	if idx := findInventory(hero.State, KindItem, 1); idx < 0 || hero.State.Inventory[idx].Amount != 3 {
		t.Fatalf("inventory = %+v", hero.State.Inventory)
	}
	if err := w.AdminCommand(ctx, admin, protocol.AdminCommand{Command: AdminGold, Text: "Receiver", Int2: 500}); err != nil {
		t.Fatalf("give gold: %v", err)
	}
	if hero.Character.Gold != 510 {
		t.Fatalf("gold = %d", hero.Character.Gold)
	}
	if err := w.AdminCommand(ctx, admin, protocol.AdminCommand{Command: AdminMute, Text: "Receiver"}); err != nil {
		t.Fatalf("mute: %v", err)
	}
	if err := w.Chat(ctx, hero, 0, "hi", ""); err != ErrMuted {
		t.Fatalf("muted chat error = %v", err)
	}
	switchID := w.opts.MaxPlayerSwitches + 1
	if err := w.AdminCommand(ctx, admin, protocol.AdminCommand{Command: AdminSwitch, Text: strconv.Itoa(switchID), Int1: 1}); err != nil {
		t.Fatalf("switch: %v", err)
	}
	if !w.globalSwitchValue(switchID) {
		t.Fatal("global switch was not set")
	}
	if err := w.AdminCommand(ctx, admin, protocol.AdminCommand{Command: AdminMessage, Text: "servidor reinicia em 5 min"}); err != nil {
		t.Fatalf("message: %v", err)
	}
	if len(hero.Client.(*fakeClient).find(protocol.ServerAdminCommand)) == 0 {
		t.Fatal("admin message was not delivered")
	}
}

func TestAdminBanAndUnban(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	admin := createPlayer(t, st, "Admin3", 21, 12)
	admin.Account.GroupID = 2
	if err := w.Join(ctx, admin); err != nil {
		t.Fatal(err)
	}
	hero := createPlayer(t, st, "Banned", 21, 13)
	hero.IP = "192.0.2.7"
	if err := w.Join(ctx, hero); err != nil {
		t.Fatal(err)
	}
	if err := w.AdminCommand(ctx, admin, protocol.AdminCommand{Command: AdminBanAcc, Text: "Banned", Int1: 2}); err != nil {
		t.Fatalf("ban account: %v", err)
	}
	value := strconv.FormatInt(hero.Account.ID, 10)
	ban, err := st.GetActiveBan(ctx, "account", value, time.Now().Unix())
	if err != nil || ban == nil {
		t.Fatalf("account ban = %+v err=%v", ban, err)
	}
	if err := w.AdminCommand(ctx, admin, protocol.AdminCommand{Command: AdminUnban, Text: "Banned"}); err != nil {
		t.Fatalf("unban: %v", err)
	}
	ban, err = st.GetActiveBan(ctx, "account", value, time.Now().Unix())
	if err != nil || ban != nil {
		t.Fatalf("ban survived unban: %+v err=%v", ban, err)
	}
}
