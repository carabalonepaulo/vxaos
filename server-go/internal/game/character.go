package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

// Inventory kinds used by the character aggregate and the Item message.
const (
	KindItem   = 1
	KindWeapon = 2
	KindArmor  = 3
)

var (
	ErrInvalidEquip   = errors.New("game: invalid equipment")
	ErrNotOwned       = errors.New("game: item not owned")
	ErrRequirement    = errors.New("game: equipment requirement not met")
	ErrFullInventory  = errors.New("game: inventory is full")
	ErrInvalidHotbar  = errors.New("game: invalid hotbar slot")
	ErrHotbarNotOwned = errors.New("game: hotbar item not owned")
	ErrNoPoints       = errors.New("game: no points left")
	ErrInvalidParam   = errors.New("game: invalid parameter")
)

func (w *World) ensureEquipSlots(p *Player) {
	for len(p.State.Equips) < w.opts.MaxEquips {
		p.State.Equips = append(p.State.Equips, 0)
	}
}

func (w *World) ensureHotbar(p *Player) {
	for len(p.State.Hotbar) < w.opts.MaxHotbar {
		p.State.Hotbar = append(p.State.Hotbar, store.HotbarSlot{})
	}
}

func inventoryEntries(st *store.CharacterState) int {
	n := 0
	for _, item := range st.Inventory {
		if item.Amount > 0 {
			n++
		}
	}
	return n
}

func findInventory(st *store.CharacterState, kind, id int) int {
	for i, item := range st.Inventory {
		if item.Kind == kind && item.ItemID == id && item.Amount > 0 {
			return i
		}
	}
	return -1
}

func addInventoryItem(st *store.CharacterState, maxItems, kind, id, amount int) error {
	if amount <= 0 {
		return nil
	}
	if i := findInventory(st, kind, id); i >= 0 {
		st.Inventory[i].Amount += amount
		return nil
	}
	if inventoryEntries(st) >= maxItems {
		return ErrFullInventory
	}
	st.Inventory = append(st.Inventory, store.ItemStack{Kind: kind, ItemID: id, Amount: amount})
	return nil
}

func removeInventoryItem(st *store.CharacterState, kind, id, amount int) bool {
	if amount <= 0 {
		return false
	}
	i := findInventory(st, kind, id)
	if i < 0 || st.Inventory[i].Amount < amount {
		return false
	}
	st.Inventory[i].Amount -= amount
	if st.Inventory[i].Amount == 0 {
		st.Inventory = append(st.Inventory[:i], st.Inventory[i+1:]...)
	}
	return true
}

func hasSkill(st *store.CharacterState, id int) bool {
	for _, skill := range st.Skills {
		if skill == id {
			return true
		}
	}
	return false
}

func (w *World) weapon(id int) *rpg.Weapon {
	if id <= 0 || id >= len(w.opts.Data.Weapons) {
		return nil
	}
	return w.opts.Data.Weapons[id]
}

func (w *World) armor(id int) *rpg.Armor {
	if id <= 0 || id >= len(w.opts.Data.Armors) {
		return nil
	}
	return w.opts.Data.Armors[id]
}

func (w *World) item(id int) *rpg.Item {
	if id <= 0 || id >= len(w.opts.Data.Items) {
		return nil
	}
	return w.opts.Data.Items[id]
}

func (w *World) skill(id int) *rpg.Skill {
	if id <= 0 || id >= len(w.opts.Data.Skills) {
		return nil
	}
	return w.opts.Data.Skills[id]
}

// Equip changes one equipment slot. itemID 0 unequips. The item may come from
// the inventory or from another equipment slot.
func (w *World) Equip(ctx context.Context, p *Player, slot uint8, itemID uint16) error {
	return w.do(ctx, func() error {
		if w.players[p.ID] != p {
			return ErrNotInGame
		}
		w.ensureEquipSlots(p)
		s := int(slot)
		if s < 0 || s >= w.opts.MaxEquips {
			return ErrInvalidEquip
		}
		kind := KindArmor
		if s == 0 {
			kind = KindWeapon
		}
		// Unequip.
		if itemID == 0 {
			old := p.State.Equips[s]
			if old == 0 {
				return nil
			}
			if !canAddInventoryItem(p.State, w.opts.MaxItems, kind, old) {
				return ErrFullInventory
			}
			_ = addInventoryItem(p.State, w.opts.MaxItems, kind, old, 1)
			p.State.Equips[s] = 0
			w.sendItemTo(p, kind, old, 1, false, false)
			w.broadcastEquip(p, s, 0)
			w.saveQuietly(p)
			return nil
		}
		id := int(itemID)
		if err := w.validateEquip(p, kind, id); err != nil {
			return err
		}
		old := p.State.Equips[s]
		if old == id {
			return nil
		}
		sourceInventory := findInventory(p.State, kind, id) >= 0
		sourceSlot := -1
		if !sourceInventory {
			for i, equipped := range p.State.Equips {
				if i != s && equipped == id {
					sourceSlot = i
					break
				}
			}
		}
		if !sourceInventory && sourceSlot < 0 {
			return ErrNotOwned
		}
		if old != 0 {
			oldKind := KindArmor
			if s == 0 {
				oldKind = KindWeapon
			}
			if !canAddInventoryItem(p.State, w.opts.MaxItems, oldKind, old) {
				return ErrFullInventory
			}
		}
		// All checks passed; apply and only then emit deltas.
		if sourceInventory {
			removeInventoryItem(p.State, kind, id, 1)
			w.sendItemTo(p, kind, id, -1, true, false)
		} else {
			p.State.Equips[sourceSlot] = 0
			w.broadcastEquip(p, sourceSlot, 0)
		}
		if old != 0 {
			oldKind := KindArmor
			if s == 0 {
				oldKind = KindWeapon
			}
			_ = addInventoryItem(p.State, w.opts.MaxItems, oldKind, old, 1)
			w.sendItemTo(p, oldKind, old, 1, false, false)
		}
		p.State.Equips[s] = id
		w.broadcastEquip(p, s, id)
		w.saveQuietly(p)
		return nil
	})
}

func (w *World) validateEquip(p *Player, kind, id int) error {
	if kind == KindWeapon {
		weapon := w.weapon(id)
		if weapon == nil || weapon.ETypeID != 0 {
			return ErrInvalidEquip
		}
		if p.Character.Level < weapon.Level {
			return fmt.Errorf("%w: level", ErrRequirement)
		}
		if weapon.VIP && !w.isVIP(p) {
			return fmt.Errorf("%w: vip", ErrRequirement)
		}
		return nil
	}
	armor := w.armor(id)
	if armor == nil || armor.ETypeID == 0 {
		return ErrInvalidEquip
	}
	if p.Character.Level < armor.Level {
		return fmt.Errorf("%w: level", ErrRequirement)
	}
	if armor.VIP && !w.isVIP(p) {
		return fmt.Errorf("%w: vip", ErrRequirement)
	}
	if armor.Sex < 2 && armor.Sex != p.Character.Sex {
		return fmt.Errorf("%w: sex", ErrRequirement)
	}
	return nil
}

func canAddInventoryItem(st *store.CharacterState, maxItems, kind, id int) bool {
	if findInventory(st, kind, id) >= 0 {
		return true
	}
	return inventoryEntries(st) < maxItems
}

func (w *World) isVIP(p *Player) bool {
	return p.Account.VIPUntil > w.now().Unix()
}

// SetHotbar stores an item or skill in one hotbar slot. Type 0 clears it.
func (w *World) SetHotbar(ctx context.Context, p *Player, slot, typ uint8, itemID uint16) error {
	return w.do(ctx, func() error {
		if w.players[p.ID] != p {
			return ErrNotInGame
		}
		w.ensureHotbar(p)
		s := int(slot)
		if s < 0 || s >= w.opts.MaxHotbar {
			return ErrInvalidHotbar
		}
		id := int(itemID)
		switch typ {
		case 0:
			id = 0
		case 1:
			if w.item(id) == nil || findInventory(p.State, KindItem, id) < 0 {
				return ErrHotbarNotOwned
			}
		case 2:
			if w.skill(id) == nil || !hasSkill(p.State, id) {
				return ErrHotbarNotOwned
			}
		default:
			return ErrInvalidHotbar
		}
		p.State.Hotbar[s] = store.HotbarSlot{Type: int(typ), ItemID: id}
		p.Client.Send(protocol.ServerHotbar, protocol.Hotbar{Slot: slot, Type: typ, ItemID: itemID})
		w.saveQuietly(p)
		return nil
	})
}

// SpendPoint converts one creation point into a permanent parameter bonus.
func (w *World) SpendPoint(ctx context.Context, p *Player, paramID uint8) error {
	return w.do(ctx, func() error {
		if w.players[p.ID] != p {
			return ErrNotInGame
		}
		param := int(paramID)
		if param < 0 || param > 7 {
			return ErrInvalidParam
		}
		if p.Character.Points <= 0 {
			return ErrNoPoints
		}
		delta := 1
		if param < 2 {
			delta = 10
		}
		p.Character.Points--
		switch param {
		case 0:
			p.Character.MHP += delta
			p.Character.HP += delta
		case 1:
			p.Character.MMP += delta
			p.Character.MP += delta
		case 2:
			p.Character.ATK += delta
		case 3:
			p.Character.DEF += delta
		case 4:
			p.Character.MAT += delta
		case 5:
			p.Character.MDF += delta
		case 6:
			p.Character.AGI += delta
		case 7:
			p.Character.LUK += delta
		}
		msg := protocol.Param{PlayerID: p.ID, ParamID: paramID, Value: int32(delta)}
		w.broadcastToMap(p, protocol.ServerParam, msg)
		p.Client.Send(protocol.ServerPoints, protocol.Points{Value: int32(p.Character.Points)})
		if param < 2 {
			vitals := protocol.Vitals{PlayerID: p.ID, HP: int32(p.Character.HP), MP: int32(p.Character.MP)}
			w.broadcastToMap(p, protocol.ServerVitals, vitals)
		}
		w.saveQuietly(p)
		return nil
	})
}

func (w *World) sendItemTo(p *Player, kind, id, amount int, dropSound, popup bool) {
	_ = p.Client.Send(protocol.ServerItem, protocol.Item{
		Kind:      uint8(kind),
		ItemID:    uint16(id),
		Amount:    int32(amount),
		DropSound: dropSound,
		Popup:     popup,
	})
}

func (w *World) broadcastEquip(p *Player, slot, id int) {
	msg := protocol.Equip{PlayerID: p.ID, Slot: uint8(slot), ItemID: uint16(id)}
	w.broadcastToMap(p, protocol.ServerEquip, msg)
}

func (w *World) broadcastToMap(p *Player, op protocol.ServerOp, msg protocol.ServerMessage) {
	ms := w.maps[p.Character.MapID]
	if ms == nil {
		return
	}
	for _, other := range ms.players {
		if err := other.Client.Send(op, msg); err != nil {
			w.log.Warn("broadcast", "player", other.ID, "op", uint16(op), "err", err)
		}
	}
}

func (w *World) saveQuietly(p *Player) {
	if err := w.saveLocked(p); err != nil {
		w.log.Error("save character", "player", p.ID, "err", err)
	}
}
