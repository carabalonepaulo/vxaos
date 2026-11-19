package game

import (
	"context"
	"testing"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/store"
)

func TestTradeFlow(t *testing.T) {
	w, st := testWorld(t)
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

	if err := w.PlayerRequest(ctx, hero, requestTrade, ally.ID); err != nil {
		t.Fatal(err)
	}
	if ally.Request == nil || ally.Request.Type != requestTrade {
		t.Fatalf("trade request = %+v", ally.Request)
	}
	if err := w.AcceptRequest(ctx, ally); err != nil {
		t.Fatal(err)
	}
	if hero.Trade == nil || ally.Trade == nil {
		t.Fatal("trade did not open")
	}
	if len(hero.Client.(*fakeClient).find(protocol.ServerRequestAccept)) == 0 ||
		len(ally.Client.(*fakeClient).find(protocol.ServerRequestAccept)) == 0 {
		t.Fatal("trade open was not sent to both sides")
	}

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
	if len(ally.Client.(*fakeClient).find(protocol.ServerTradeItem)) != 2 {
		t.Fatalf("trade item broadcasts = %d", len(ally.Client.(*fakeClient).find(protocol.ServerTradeItem)))
	}
	if len(ally.Client.(*fakeClient).find(protocol.ServerTradeGold)) != 2 {
		t.Fatalf("trade gold broadcasts = %d", len(ally.Client.(*fakeClient).find(protocol.ServerTradeGold)))
	}

	if err := w.RequestFinishTrade(ctx, hero); err != nil {
		t.Fatal(err)
	}
	if ally.Request == nil || ally.Request.Type != requestFinishTrade {
		t.Fatalf("finish request = %+v", ally.Request)
	}
	if err := w.AcceptRequest(ctx, ally); err != nil {
		t.Fatal(err)
	}
	if hero.Trade != nil || ally.Trade != nil {
		t.Fatal("trade did not close")
	}
	if hero.Character.Gold != 80 || ally.Character.Gold != 70 {
		t.Fatalf("gold after trade = %d/%d", hero.Character.Gold, ally.Character.Gold)
	}
	if findInventory(hero.State, KindWeapon, weapon.ID) < 0 {
		t.Fatal("hero did not receive the weapon")
	}
	if findInventory(ally.State, KindItem, item.ID) < 0 {
		t.Fatal("ally did not receive the item")
	}
	if idx := findInventory(hero.State, KindItem, item.ID); idx < 0 || hero.State.Inventory[idx].Amount != 1 {
		t.Fatalf("hero unexpected item stack: %+v", hero.State.Inventory)
	}
	if idx := findInventory(ally.State, KindItem, item.ID); idx < 0 || ally.State.Inventory[idx].Amount != 1 {
		t.Fatalf("ally unexpected item stack: %+v", ally.State.Inventory)
	}
	if len(hero.Client.(*fakeClient).find(protocol.ServerWindowClose)) == 0 ||
		len(ally.Client.(*fakeClient).find(protocol.ServerWindowClose)) == 0 {
		t.Fatal("trade windows were not closed")
	}
}
