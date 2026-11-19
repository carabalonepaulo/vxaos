package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/kelaia/vxaos/server/internal/protocol"
)

var (
	ErrNotInTrade = errors.New("game: not trading")
	ErrTradeBusy  = errors.New("game: player is busy")
	ErrTradeFull  = errors.New("game: trade is full")
)

const (
	requestTrade       = 1
	requestFinishTrade = 2
	maxTradeItems      = 12
)

func (w *World) tradeReady(p *Player) bool {
	return p.Trade != nil && p.Trade.Partner != 0
}

// inviteTradeLocked queues a trade invitation.
func (w *World) inviteTradeLocked(p, target *Player) error {
	if p.Trade != nil || target.Trade != nil {
		return ErrTradeBusy
	}
	if p.ShopGoods != nil || target.ShopGoods != nil || p.BankOpen || target.BankOpen {
		return ErrTradeBusy
	}
	if target.Character.MapID != p.Character.MapID {
		return ErrPartyRange
	}
	dx := target.Character.X - p.Character.X
	dy := target.Character.Y - p.Character.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > 10 || dy > 10 {
		return ErrPartyRange
	}
	target.Request = &pendingRequest{Type: requestTrade, FromID: p.ID}
	_ = target.Client.Send(protocol.ServerRequest, protocol.PlayerRequest{
		Type: requestTrade, FromID: p.ID,
		Text: fmt.Sprintf("%s quer negociar com você.", p.Character.Name),
	})
	return nil
}

// openTradeLocked links both players and opens both trade windows.
func (w *World) openTradeLocked(p, from *Player) error {
	if p.Trade != nil || from.Trade != nil {
		return ErrTradeBusy
	}
	if p.ShopGoods != nil || from.ShopGoods != nil || p.BankOpen || from.BankOpen {
		return ErrTradeBusy
	}
	if p.Character.MapID != from.Character.MapID {
		return ErrPartyRange
	}
	p.Trade = &tradeState{Partner: from.ID, Items: make(map[tradeKey]int)}
	from.Trade = &tradeState{Partner: p.ID, Items: make(map[tradeKey]int)}
	_ = p.Client.Send(protocol.ServerRequestAccept, protocol.RequestAccepted{Type: requestTrade})
	_ = from.Client.Send(protocol.ServerRequestAccept, protocol.RequestAccepted{Type: requestTrade})
	return nil
}

// TradeItem adds or removes an offered item.
func (w *World) TradeItem(ctx context.Context, p *Player, itemID uint16, kind uint8, amount int16) error {
	return w.do(ctx, func() error {
		if w.players[p.ID] != p || p.Trade == nil {
			return ErrNotInTrade
		}
		k := int(kind)
		id := int(itemID)
		if amount == 0 || k < KindItem || k > KindArmor {
			return ErrInvalidTarget
		}
		item := w.itemKind(k, id)
		if item == nil {
			return ErrInvalidTarget
		}
		state := p.Trade
		key := tradeKey{Kind: k, ID: id}
		offered := state.Items[key]
		n := int(amount)
		if n > 0 {
			if offered == 0 && len(state.Items) >= maxTradeItems {
				return ErrTradeFull
			}
			index := findInventory(p.State, k, id)
			if index < 0 || p.State.Inventory[index].Amount-offered < n {
				return ErrNotOwned
			}
			if w.soulbound(k, id) {
				return ErrInvalidTarget
			}
			state.Items[key] = offered + n
		} else {
			if offered < -n {
				return ErrNotOwned
			}
			state.Items[key] = offered + n
			if state.Items[key] == 0 {
				delete(state.Items, key)
			}
		}
		w.broadcastTrade(p, protocol.ServerTradeItem, protocol.TradeItem{
			PlayerID: p.ID, ItemID: itemID, Kind: kind, Amount: amount,
		})
		return nil
	})
}

// TradeGold adds or removes offered gold.
func (w *World) TradeGold(ctx context.Context, p *Player, amount int32) error {
	return w.do(ctx, func() error {
		if w.players[p.ID] != p || p.Trade == nil {
			return ErrNotInTrade
		}
		if amount == 0 {
			return nil
		}
		state := p.Trade
		delta := int64(amount)
		if delta > 0 {
			if p.Character.Gold-state.Gold < delta {
				return ErrRequirement
			}
		} else if state.Gold < -delta {
			return ErrRequirement
		}
		state.Gold += delta
		w.broadcastTrade(p, protocol.ServerTradeGold, protocol.TradeGold{PlayerID: p.ID, Amount: amount})
		return nil
	})
}

// RequestFinishTrade asks the partner to confirm the trade.
func (w *World) RequestFinishTrade(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error { return w.requestFinishTradeLocked(p) })
}

func (w *World) requestFinishTradeLocked(p *Player) error {
	if w.players[p.ID] != p || p.Trade == nil {
		return ErrNotInTrade
	}
	partner := w.players[p.Trade.Partner]
	if partner == nil || partner.Trade == nil {
		return ErrNotInTrade
	}
	partner.Request = &pendingRequest{Type: requestFinishTrade, FromID: p.ID}
	_ = partner.Client.Send(protocol.ServerRequest, protocol.PlayerRequest{
		Type: requestFinishTrade, FromID: p.ID,
		Text: fmt.Sprintf("%s quer finalizar a troca.", p.Character.Name),
	})
	return nil
}

// finishTradeLocked executes the exchange. Called when the partner accepts.
func (w *World) finishTradeLocked(p, from *Player) error {
	if p.Trade == nil || from.Trade == nil || p.Trade.Partner != from.ID || from.Trade.Partner != p.ID {
		return ErrNotInTrade
	}
	// Revalidate both sides before moving anything.
	for key, amount := range p.Trade.Items {
		index := findInventory(p.State, key.Kind, key.ID)
		if index < 0 || p.State.Inventory[index].Amount < amount {
			return ErrNotOwned
		}
	}
	for key, amount := range from.Trade.Items {
		index := findInventory(from.State, key.Kind, key.ID)
		if index < 0 || from.State.Inventory[index].Amount < amount {
			return ErrNotOwned
		}
	}
	if p.Character.Gold < p.Trade.Gold || from.Character.Gold < from.Trade.Gold {
		return ErrRequirement
	}
	w.transferTradeItems(p, from)
	w.transferTradeItems(from, p)
	pOffered := p.Trade.Gold
	fromOffered := from.Trade.Gold
	pDelta := fromOffered - pOffered
	fromDelta := pOffered - fromOffered
	p.Character.Gold = clampInt64(p.Character.Gold-pOffered+fromOffered, 0, w.opts.MaxGold)
	from.Character.Gold = clampInt64(from.Character.Gold-fromOffered+pOffered, 0, w.opts.MaxGold)
	_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: pDelta, Popup: true})
	_ = from.Client.Send(protocol.ServerGold, protocol.Gold{Amount: fromDelta, Popup: true})
	_ = p.Client.Send(protocol.ServerWindowClose, protocol.WindowClose{})
	_ = from.Client.Send(protocol.ServerWindowClose, protocol.WindowClose{})
	p.Trade = nil
	from.Trade = nil
	w.questChat(p, "Troca concluída.")
	w.questChat(from, "Troca concluída.")
	w.saveQuietly(p)
	w.saveQuietly(from)
	return nil
}

func (w *World) transferTradeItems(owner, receiver *Player) {
	for key, amount := range owner.Trade.Items {
		if amount <= 0 {
			continue
		}
		index := findInventory(receiver.State, key.Kind, key.ID)
		current := 0
		if index >= 0 {
			current = receiver.State.Inventory[index].Amount
		} else if inventoryEntries(receiver.State) >= w.opts.MaxItems {
			continue
		}
		move := amount
		if room := w.opts.MaxStack - current; move > room {
			move = room
		}
		if move <= 0 {
			continue
		}
		if !removeInventoryItem(owner.State, key.Kind, key.ID, move) {
			continue
		}
		if err := addInventoryItem(receiver.State, w.opts.MaxItems, key.Kind, key.ID, move); err != nil {
			// Should not happen after the checks; return the item to the owner.
			_ = addInventoryItem(owner.State, w.opts.MaxItems, key.Kind, key.ID, move)
			continue
		}
		w.sendItemTo(owner, key.Kind, key.ID, -move, false, false)
		w.sendItemTo(receiver, key.Kind, key.ID, move, false, true)
		w.addQuestItemNotice(receiver, key.Kind, key.ID)
	}
}

// closeTradeLocked closes both trade windows.
func (w *World) closeTradeLocked(p *Player) {
	if p.Trade == nil {
		return
	}
	partner := w.players[p.Trade.Partner]
	_ = p.Client.Send(protocol.ServerWindowClose, protocol.WindowClose{})
	p.Trade = nil
	if partner != nil && partner.Trade != nil {
		_ = partner.Client.Send(protocol.ServerWindowClose, protocol.WindowClose{})
		partner.Trade = nil
	}
}

func (w *World) broadcastTrade(p *Player, op protocol.ServerOp, msg protocol.ServerMessage) {
	if p.Trade == nil {
		return
	}
	_ = p.Client.Send(op, msg)
	if partner := w.players[p.Trade.Partner]; partner != nil {
		_ = partner.Client.Send(op, msg)
	}
}

func (w *World) soulbound(kind, id int) bool {
	switch kind {
	case KindItem:
		if item := w.item(id); item != nil {
			return item.Soulbound
		}
	case KindWeapon:
		if weapon := w.weapon(id); weapon != nil {
			return weapon.Soulbound
		}
	case KindArmor:
		if armor := w.armor(id); armor != nil {
			return armor.Soulbound
		}
	}
	return false
}
