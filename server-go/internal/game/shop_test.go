package game

import (
	"context"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
)

func TestShopBuySell(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	item := firstUsableItem(t, w)
	p := createPlayer(t, st, "Merchant", 21, 12)
	p.Character.Gold = 100000
	p.ShopGoods = []ShopGood{{Kind: 0, ItemID: item.ID}}
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	price := w.shopPrice(p.ShopGoods[0])
	if price <= 0 {
		t.Fatalf("item %d has no price", item.ID)
	}
	if err := w.ShopBuy(ctx, p, 0, 2); err != nil {
		t.Fatal(err)
	}
	if findInventory(p.State, KindItem, item.ID) < 0 || p.Character.Gold != 100000-int64(price*2) {
		t.Fatalf("buy: gold=%d inv=%+v", p.Character.Gold, p.State.Inventory)
	}
	if len(p.Client.(*fakeClient).find(protocol.ServerItem)) == 0 {
		t.Fatal("buy did not send Item")
	}
	before := p.Character.Gold
	if err := w.ShopSell(ctx, p, uint16(item.ID), KindItem, 1); err != nil {
		t.Fatal(err)
	}
	if p.Character.Gold != before+int64(price/2) {
		t.Fatalf("sell: gold=%d want=%d", p.Character.Gold, before+int64(price/2))
	}
	if idx := findInventory(p.State, KindItem, item.ID); idx < 0 || p.State.Inventory[idx].Amount != 1 {
		t.Fatalf("sell inventory = %+v", p.State.Inventory)
	}
}
