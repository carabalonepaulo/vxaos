package game

import (
	"context"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
)

// ShopGood is one 605/302 shop entry. Kind is 0 item, 1 weapon, 2 armor.
type ShopGood struct {
	Kind      int
	ItemID    int
	PriceType int
	Price     int
}

func (w *World) openShop(p *Player, frame *eventFrame, cmd *rpg.EventCommand) bool {
	goods := make([]ShopGood, 0, 8)
	sellDisabled, _ := paramInt(cmd, 4)
	index := frame.index
	for index+1 < len(frame.list) {
		next := frame.list[index+1]
		if next == nil || next.Code != 605 {
			break
		}
		index++
		kind, _ := paramInt(next, 0)
		id, _ := paramInt(next, 1)
		priceType, _ := paramInt(next, 2)
		price, _ := paramInt(next, 3)
		if id <= 0 {
			continue
		}
		goods = append(goods, ShopGood{Kind: kind, ItemID: id, PriceType: priceType, Price: price})
	}
	if len(goods) == 0 {
		frame.index = index + 1
		return false
	}
	p.ShopGoods = goods
	p.ShopSellDisabled = sellDisabled != 0
	p.ShopEventID = frame.eventID
	p.ShopIndex = frame.index
	p.Event.waitingShop = true
	frame.index = index + 1
	_ = p.Client.Send(protocol.ServerShopOpen, protocol.ShopOpen{
		EventID: uint16(clampInt(frame.eventID, 0, 65535)),
		Index:   uint16(clampInt(p.ShopIndex, 0, 65535)),
	})
	return true
}

func (w *World) shopPrice(good ShopGood) int {
	if good.PriceType != 0 {
		return good.Price
	}
	switch good.Kind {
	case 0:
		if item := w.item(good.ItemID); item != nil {
			return item.Price
		}
	case 1:
		if weapon := w.weapon(good.ItemID); weapon != nil {
			return weapon.Price
		}
	case 2:
		if armor := w.armor(good.ItemID); armor != nil {
			return armor.Price
		}
	}
	return good.Price
}

// ShopBuy buys amount units of one displayed shop good.
func (w *World) ShopBuy(ctx context.Context, p *Player, index uint8, amount uint16) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if int(index) >= len(p.ShopGoods) {
			return ErrInvalidTarget
		}
		if amount == 0 {
			return nil
		}
		good := p.ShopGoods[index]
		price := w.shopPrice(good)
		total := int64(price) * int64(amount)
		if total <= 0 || p.Character.Gold < total {
			return ErrRequirement
		}
		kind := good.Kind + 1
		if addInventoryItem(p.State, w.opts.MaxItems, kind, good.ItemID, int(amount)) != nil {
			return ErrFullInventory
		}
		p.Character.Gold -= total
		w.sendItemTo(p, kind, good.ItemID, int(amount), true, true)
		w.addQuestItemNotice(p, kind, good.ItemID)
		_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: -total, ShopSound: true, Popup: true})
		w.saveQuietly(p)
		return nil
	})
}

// ShopSell sells inventory items for half of their database price.
func (w *World) ShopSell(ctx context.Context, p *Player, itemID uint16, kind uint8, amount uint16) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if amount == 0 {
			return nil
		}
		k := int(kind)
		id := int(itemID)
		if k < 1 || k > 3 || findInventory(p.State, k, id) < 0 {
			return ErrNotOwned
		}
		var price int
		switch k {
		case KindItem:
			if item := w.item(id); item != nil {
				price = item.Price
			}
		case KindWeapon:
			if weapon := w.weapon(id); weapon != nil {
				price = weapon.Price
			}
		case KindArmor:
			if armor := w.armor(id); armor != nil {
				price = armor.Price
			}
		}
		if price <= 0 {
			return ErrInvalidTarget
		}
		if !removeInventoryItem(p.State, k, id, int(amount)) {
			return ErrNotOwned
		}
		total := int64(price/2) * int64(amount)
		p.Character.Gold += total
		w.sendItemTo(p, k, id, -int(amount), false, false)
		_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: total, ShopSound: true, Popup: true})
		w.saveQuietly(p)
		return nil
	})
}

// CloseWindow resumes an event waiting for a shop or bank window.
func (w *World) CloseWindow(ctx context.Context, p *Player) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if p.ShopGoods != nil {
			p.ShopGoods = nil
			p.ShopSellDisabled = false
		}
		p.CreatingGuild = false
		if p.Trade != nil {
			w.closeTradeLocked(p)
		}
		if p.Event != nil && p.Event.waitingShop {
			p.Event.waitingShop = false
			w.runEvent(p)
			return nil
		}
		if p.BankOpen || (p.Event != nil && p.Event.waitingBank) {
			w.closeBank(p)
		}
		return nil
	})
}
