package game

import (
	"context"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/store"
)

func TestBankDepositWithdraw(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	item := firstUsableItem(t, w)
	p := createPlayer(t, st, "Banker", 21, 12)
	p.State.Inventory = []store.ItemStack{{Kind: KindItem, ItemID: item.ID, Amount: 5}}
	p.Character.Gold = 500
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := w.do(ctx, func() error {
		if !w.openBank(p) {
			return ErrUnsupported
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !p.BankOpen {
		t.Fatal("bank did not open")
	}
	if len(p.Client.(*fakeClient).find(protocol.ServerBankOpen)) != 1 {
		t.Fatal("no BankOpen sent")
	}

	if err := w.BankItem(ctx, p, uint16(item.ID), KindItem, 3); err != nil {
		t.Fatal(err)
	}
	if amount := p.State.Inventory[findInventory(p.State, KindItem, item.ID)].Amount; amount != 2 {
		t.Fatalf("inventory after deposit = %d", amount)
	}
	if p.Bank.Items[bankIndex(p.Bank, KindItem, item.ID)].Amount != 3 {
		t.Fatalf("bank after deposit = %+v", p.Bank.Items)
	}
	if err := w.BankItem(ctx, p, uint16(item.ID), KindItem, -2); err != nil {
		t.Fatal(err)
	}
	if amount := p.State.Inventory[findInventory(p.State, KindItem, item.ID)].Amount; amount != 4 {
		t.Fatalf("inventory after withdraw = %d", amount)
	}
	if p.Bank.Items[bankIndex(p.Bank, KindItem, item.ID)].Amount != 1 {
		t.Fatalf("bank after withdraw = %+v", p.Bank.Items)
	}

	if err := w.BankGold(ctx, p, 200); err != nil {
		t.Fatal(err)
	}
	if p.Character.Gold != 300 || p.Bank.Gold != 200 {
		t.Fatalf("gold after deposit: player=%d bank=%d", p.Character.Gold, p.Bank.Gold)
	}
	if err := w.BankGold(ctx, p, -50); err != nil {
		t.Fatal(err)
	}
	if p.Character.Gold != 350 || p.Bank.Gold != 150 {
		t.Fatalf("gold after withdraw: player=%d bank=%d", p.Character.Gold, p.Bank.Gold)
	}
	if err := w.BankGold(ctx, p, 1000); err != ErrRequirement {
		t.Fatalf("overdraft deposit = %v", err)
	}
	if err := w.BankItem(ctx, p, uint16(item.ID), KindItem, -5); err != ErrNotOwned {
		t.Fatalf("overdraft withdraw = %v", err)
	}

	if err := w.CloseWindow(ctx, p); err != nil {
		t.Fatal(err)
	}
	if p.BankOpen {
		t.Fatal("bank did not close")
	}
	saved, err := st.GetBank(ctx, p.Account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Gold != 150 || len(saved.Items) != 1 || saved.Items[0].Amount != 1 {
		t.Fatalf("saved bank = %+v", saved)
	}
}
