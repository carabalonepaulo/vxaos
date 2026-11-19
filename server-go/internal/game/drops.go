package game

import (
	"context"
	"errors"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
)

var (
	ErrDropFull    = errors.New("game: too many ground drops")
	ErrNotOnDrop   = errors.New("game: move onto the drop to pick it up")
	ErrDropOwner   = errors.New("game: this drop is not yours yet")
	ErrInvalidDrop = errors.New("game: invalid drop")
)

// dropState is one item stack lying on a map.
type dropState struct {
	ItemID    int
	Kind      int
	Amount    int
	X, Y      int
	Owner     string
	PartyID   int
	PickupAt  time.Time
	DespawnAt time.Time
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
	ms.drops = append(ms.drops, drop)
	w.broadcastToMapPlayers(ms, protocol.ServerDropAdd, protocol.DropAdd{
		ItemID: uint16(clampInt(id, 0, 65535)), Kind: uint8(clampInt(kind, 0, 255)),
		Amount: uint16(clampInt(amount, 0, 65535)),
		X:      uint16(clampInt(x, 0, 65535)), Y: uint16(clampInt(y, 0, 65535)),
	})
	return true
}

func (w *World) removeMapDrop(ms *mapState, index int) {
	if ms == nil || index < 0 || index >= len(ms.drops) {
		return
	}
	ms.drops = append(ms.drops[:index], ms.drops[index+1:]...)
	w.broadcastToMapPlayers(ms, protocol.ServerDropRemove, protocol.DropRemove{DropID: uint8(clampInt(index, 0, 255))})
}

// sendDropsTo sends the current drop list in list order (the client uses the
// index to pick up and remove drops).
func (w *World) sendDropsTo(p *Player, ms *mapState) {
	for _, drop := range ms.drops {
		_ = p.Client.Send(protocol.ServerDropAdd, protocol.DropAdd{
			ItemID: uint16(clampInt(drop.ItemID, 0, 65535)),
			Kind:   uint8(clampInt(drop.Kind, 0, 255)),
			Amount: uint16(clampInt(drop.Amount, 0, 65535)),
			X:      uint16(clampInt(drop.X, 0, 65535)),
			Y:      uint16(clampInt(drop.Y, 0, 65535)),
		})
	}
}

// DropItem drops items from the inventory onto the player's tile.
func (w *World) DropItem(ctx context.Context, p *Player, itemID uint16, kind uint8, amount uint16) error {
	return w.do(ctx, func() error {
		if w.players[p.ID] != p {
			return ErrNotInGame
		}
		ms := w.maps[p.Character.MapID]
		if ms == nil {
			return ErrNotInGame
		}
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
func (w *World) PickupDrop(ctx context.Context, p *Player, index uint8) error {
	return w.do(ctx, func() error {
		if w.players[p.ID] != p {
			return ErrNotInGame
		}
		ms := w.maps[p.Character.MapID]
		i := int(index)
		if ms == nil || i < 0 || i >= len(ms.drops) {
			return ErrInvalidDrop
		}
		drop := ms.drops[i]
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
		w.removeMapDrop(ms, i)
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

// tickDrops removes expired ground drops.
func (w *World) tickDrops(now time.Time) {
	for _, ms := range w.maps {
		if ms == nil {
			continue
		}
		for i := len(ms.drops) - 1; i >= 0; i-- {
			if now.Before(ms.drops[i].DespawnAt) {
				continue
			}
			w.removeMapDrop(ms, i)
		}
	}
}
