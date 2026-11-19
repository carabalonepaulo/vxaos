package game

import (
	"context"
	"errors"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/rpg"
	"github.com/carabalonepaulo/vxaos/server/internal/store"
)

func firstUsableWeapon(t *testing.T, w *World) *rpg.Weapon {
	t.Helper()
	for _, weapon := range w.opts.Data.Weapons {
		if weapon != nil && weapon.ETypeID == 0 && weapon.Level <= 1 && !weapon.VIP {
			return weapon
		}
	}
	t.Fatal("no level-1 weapon in data")
	return nil
}

func firstUsableArmor(t *testing.T, w *World) *rpg.Armor {
	t.Helper()
	for _, armor := range w.opts.Data.Armors {
		if armor != nil && armor.ETypeID > 0 && armor.Level <= 1 && !armor.VIP && armor.Sex >= 2 {
			return armor
		}
	}
	t.Fatal("no level-1 unisex armor in data")
	return nil
}

func firstUsableItem(t *testing.T, w *World) *rpg.Item {
	t.Helper()
	for _, item := range w.opts.Data.Items {
		if item != nil {
			return item
		}
	}
	t.Fatal("no item in data")
	return nil
}

func TestEquipUnequip(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	weapon := firstUsableWeapon(t, w)
	armor := firstUsableArmor(t, w)
	p := createPlayer(t, st, "Equip", 21, 12)
	p.State.Equips = make([]int, 0)
	p.State.Inventory = []store.ItemStack{
		{Kind: KindWeapon, ItemID: weapon.ID, Amount: 1},
		{Kind: KindArmor, ItemID: armor.ID, Amount: 1},
	}
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	client := p.Client.(*fakeClient)

	if err := w.Equip(ctx, p, 0, uint16(weapon.ID)); err != nil {
		t.Fatal(err)
	}
	if p.State.Equips[0] != weapon.ID {
		t.Fatalf("weapon slot = %+v", p.State.Equips)
	}
	if len(p.State.Inventory) != 1 || p.State.Inventory[0].ItemID != armor.ID {
		t.Fatalf("inventory after equip = %+v", p.State.Inventory)
	}
	if len(client.find(protocol.ServerEquip)) == 0 || len(client.find(protocol.ServerItem)) == 0 {
		t.Fatal("equip did not send Equip/Item messages")
	}

	if err := w.Equip(ctx, p, 1, uint16(armor.ID)); err != nil {
		t.Fatal(err)
	}
	if p.State.Equips[1] != armor.ID || len(p.State.Inventory) != 0 {
		t.Fatalf("armor equip: equips=%+v inventory=%+v", p.State.Equips, p.State.Inventory)
	}

	if err := w.Equip(ctx, p, 0, 0); err != nil {
		t.Fatal(err)
	}
	if p.State.Equips[0] != 0 {
		t.Fatalf("weapon not unequipped: %+v", p.State.Equips)
	}
	if findInventory(p.State, KindWeapon, weapon.ID) < 0 {
		t.Fatal("unequipped weapon not returned to inventory")
	}

	if err := w.Equip(ctx, p, 0, 65000); !errors.Is(err, ErrInvalidEquip) {
		t.Fatalf("unknown weapon err = %v", err)
	}
}

func TestEquipRequirementAndCapacity(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	armor := firstUsableArmor(t, w)
	p := createPlayer(t, st, "Limits", 21, 12)
	p.State.Equips = make([]int, w.opts.MaxEquips)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	p.State.Equips[1] = armor.ID
	for i := 0; i < w.opts.MaxItems; i++ {
		p.State.Inventory = append(p.State.Inventory, store.ItemStack{Kind: KindItem, ItemID: 1000 + i, Amount: 1})
	}
	if err := w.Equip(ctx, p, 1, 0); !errors.Is(err, ErrFullInventory) {
		t.Fatalf("full inventory unequip err = %v", err)
	}

	p.State.Inventory = nil
	p.State.Equips[1] = 0
	high := &rpg.Armor{ID: 1, ETypeID: 2, Level: 99, Sex: 2}
	w.opts.Data.Armors = append(w.opts.Data.Armors, nil)
	w.opts.Data.Armors[len(w.opts.Data.Armors)-1] = high
	high.ID = len(w.opts.Data.Armors) - 1
	w.opts.Data.Armors[high.ID] = high
	p.State.Inventory = []store.ItemStack{{Kind: KindArmor, ItemID: high.ID, Amount: 1}}
	if err := w.Equip(ctx, p, 1, uint16(high.ID)); !errors.Is(err, ErrRequirement) {
		t.Fatalf("level requirement err = %v", err)
	}
}

func TestHotbarAndPoints(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	item := firstUsableItem(t, w)
	p := createPlayer(t, st, "Points", 21, 12)
	p.State.Equips = make([]int, 0)
	p.State.Inventory = []store.ItemStack{{Kind: KindItem, ItemID: item.ID, Amount: 2}}
	p.Character.Points = 2
	p.Character.MHP, p.Character.HP = 100, 50
	p.Character.ATK = 10
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := w.SetHotbar(ctx, p, 0, 1, uint16(item.ID)); err != nil {
		t.Fatal(err)
	}
	if p.State.Hotbar[0].Type != 1 || p.State.Hotbar[0].ItemID != item.ID {
		t.Fatalf("hotbar = %+v", p.State.Hotbar)
	}
	if err := w.SetHotbar(ctx, p, 1, 1, 2); !errors.Is(err, ErrHotbarNotOwned) {
		t.Fatalf("unowned hotbar item err = %v", err)
	}
	if err := w.SetHotbar(ctx, p, 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if p.State.Hotbar[0].Type != 0 {
		t.Fatalf("hotbar not cleared: %+v", p.State.Hotbar[0])
	}

	if err := w.SpendPoint(ctx, p, 0); err != nil {
		t.Fatal(err)
	}
	if p.Character.MHP != 110 || p.Character.HP != 60 || p.Character.Points != 1 {
		t.Fatalf("hp point = %+v", p.Character)
	}
	if err := w.SpendPoint(ctx, p, 2); err != nil {
		t.Fatal(err)
	}
	if p.Character.ATK != 11 || p.Character.Points != 0 {
		t.Fatalf("atk point = %+v", p.Character)
	}
	if err := w.SpendPoint(ctx, p, 3); !errors.Is(err, ErrNoPoints) {
		t.Fatalf("spend without points err = %v", err)
	}
	if err := w.SpendPoint(ctx, p, 9); !errors.Is(err, ErrInvalidParam) {
		t.Fatalf("invalid param err = %v", err)
	}
}
