package game

import (
	"context"
	"errors"
	"time"
)

var (
	ErrDropFull    = errors.New("game: too many ground drops")
	ErrNotOnDrop   = errors.New("game: move onto the drop to pick it up")
	ErrDropOwner   = errors.New("game: this drop is not yours yet")
	ErrInvalidDrop = errors.New("game: invalid drop")
)

// dropState is one item stack lying on a map.
type dropState struct {
	ID        uint32
	slot      int
	ItemID    int
	Kind      int
	Amount    int
	X, Y      int
	Owner     string
	PartyID   int
	PickupAt  time.Time
	DespawnAt time.Time
}

// dropSlot is one slab slot. generation protects clients from reusing a stale
// drop id after the slot is recycled.
type dropSlot struct {
	generation uint16
	live       bool
	drop       *dropState
}

// dropSlab is a generation-checked slab. Lookup by id is O(1); allocation and
// release are O(1); no per-drop map allocation is needed.
type dropSlab struct {
	slots []dropSlot
	free  []int
}

func dropID(slot int, generation uint16) uint32 {
	return uint32(generation)<<16 | uint32(slot&0xffff)
}

func (s *dropSlab) add(drop *dropState) uint32 {
	index := -1
	if n := len(s.free); n > 0 {
		index = s.free[n-1]
		s.free = s.free[:n-1]
	} else {
		index = len(s.slots)
		s.slots = append(s.slots, dropSlot{})
	}
	slot := &s.slots[index]
	slot.generation++
	if slot.generation == 0 {
		slot.generation = 1
	}
	slot.live = true
	slot.drop = drop
	drop.slot = index
	drop.ID = dropID(index, slot.generation)
	return drop.ID
}

func (s *dropSlab) get(id uint32) *dropState {
	index := int(id & 0xffff)
	generation := uint16(id >> 16)
	if index < 0 || index >= len(s.slots) {
		return nil
	}
	slot := &s.slots[index]
	if !slot.live || slot.generation != generation {
		return nil
	}
	return slot.drop
}

func (s *dropSlab) remove(id uint32) *dropState {
	index := int(id & 0xffff)
	generation := uint16(id >> 16)
	if index < 0 || index >= len(s.slots) {
		return nil
	}
	slot := &s.slots[index]
	if !slot.live || slot.generation != generation {
		return nil
	}
	drop := slot.drop
	slot.live = false
	slot.drop = nil
	s.free = append(s.free, index)
	return drop
}

func (s *dropSlab) len() int {
	n := 0
	for i := range s.slots {
		if s.slots[i].live {
			n++
		}
	}
	return n
}

func (s *dropSlab) each(fn func(*dropState)) {
	for i := range s.slots {
		if s.slots[i].live && s.slots[i].drop != nil {
			fn(s.slots[i].drop)
		}
	}
}

func (w *World) addMapDrop(ms *mapState, kind, id, amount, x, y int, owner string, partyID int) bool {
	if ms == nil || len(ms.drops) >= w.opts.MaxMapDrops {
		return false
	}
	now := w.now()
	drop := &dropState{
		ItemID: id, Kind: kind, Amount: amount, X: x, Y: y,
		Owner: owner, PartyID: partyID,
		PickupAt:  now.Add(time.Duration(w.opts.DropPickupSeconds) * time.Second),
		DespawnAt: now.Add(time.Duration(w.opts.DropDespawnSeconds) * time.Second),
	}
	ms.dropSlab.add(drop)
	ms.drops = append(ms.drops, drop)
	w.aoiDropAdded(ms, drop)
	return true
}

// removeMapDrop removes a dense-list index (used by the expiry tick and tests).
func (w *World) removeMapDrop(ms *mapState, index int) {
	if ms == nil || index < 0 || index >= len(ms.drops) {
		return
	}
	w.removeMapDropByID(ms, ms.drops[index].ID)
}

func (w *World) removeMapDropByID(ms *mapState, id uint32) {
	drop := ms.dropSlab.remove(id)
	if drop == nil {
		return
	}
	for i, candidate := range ms.drops {
		if candidate == drop {
			ms.drops = append(ms.drops[:i], ms.drops[i+1:]...)
			break
		}
	}
	w.aoiDropRemoved(ms, id)
}

// sendDropsTo sends the current drop list in list order (the client uses the
// index to pick up and remove drops).
func (w *World) sendDropsTo(p *Player, ms *mapState) {
	w.aoiSendDrops(p)
}

// DropItem drops items from the inventory onto the player's tile.
func (w *World) DropItem(ctx context.Context, p *Player, itemID uint16, kind uint8, amount uint16) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if p.Trade != nil || amount == 0 || kind < KindItem || kind > KindArmor || w.itemKind(int(kind), int(itemID)) == nil {
			return ErrInvalidDrop
		}
		if w.soulbound(int(kind), int(itemID)) {
			return ErrInvalidDrop
		}
		if len(ms.drops) >= w.opts.MaxMapDrops {
			return ErrDropFull
		}
		if !removeInventoryItem(p.State, int(kind), int(itemID), int(amount)) {
			return ErrNotOwned
		}
		w.sendItemTo(p, int(kind), int(itemID), -int(amount), false, false)
		if !w.addMapDrop(ms, int(kind), int(itemID), int(amount), p.Character.X, p.Character.Y, "", -1) {
			_ = addInventoryItem(p.State, w.opts.MaxItems, int(kind), int(itemID), int(amount))
			w.sendItemTo(p, int(kind), int(itemID), int(amount), false, false)
			return ErrDropFull
		}
		w.saveQuietly(p)
		return nil
	})
}

// PickupDrop picks up one ground drop if the player is on its tile and owns it
// (or the pickup delay elapsed).
func (w *World) PickupDrop(ctx context.Context, p *Player, id uint32) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		drop := ms.dropSlab.get(id)
		if drop == nil {
			return ErrInvalidDrop
		}
		if p.Character.X != drop.X || p.Character.Y != drop.Y {
			return ErrNotOnDrop
		}
		if !w.canPickupDrop(drop, p) {
			return ErrDropOwner
		}
		if addInventoryItem(p.State, w.opts.MaxItems, drop.Kind, drop.ItemID, drop.Amount) != nil {
			return ErrFullInventory
		}
		w.sendItemTo(p, drop.Kind, drop.ItemID, drop.Amount, true, true)
		w.addQuestItemNotice(p, drop.Kind, drop.ItemID)
		w.removeMapDropByID(ms, id)
		w.saveQuietly(p)
		return nil
	})
}

func (w *World) canPickupDrop(drop *dropState, p *Player) bool {
	if drop.Owner == "" || equalName(drop.Owner, p.Character.Name) {
		return true
	}
	if drop.PartyID != 0 && drop.PartyID == p.PartyID {
		return true
	}
	return !w.now().Before(drop.PickupAt)
}

// tickDrops is the compatibility wrapper for callers already on the world
// goroutine. Map actors call tickDropsMap directly.
func (w *World) tickDrops(now time.Time) {
	for _, ms := range w.maps {
		if ms != nil {
			w.tickDropsMap(ms, now)
		}
	}
}

func (w *World) tickDropsMap(ms *mapState, now time.Time) {
	for i := len(ms.drops) - 1; i >= 0; i-- {
		if now.Before(ms.drops[i].DespawnAt) {
			continue
		}
		w.removeMapDrop(ms, i)
	}
}
