package game

import (
	"context"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/store"
)

// openBank loads the account bank on first use and sends its contents. It
// returns false when the bank cannot be loaded.
func (w *World) openBank(p *Player) bool {
	if p.Bank == nil {
		var bank *store.BankState
		err := w.coordErr(func() error {
			var loadErr error
			bank, loadErr = w.opts.Store.GetBank(context.Background(), p.Account.ID)
			return loadErr
		})
		if err != nil {
			w.log.Error("load bank", "account", p.Account.ID, "err", err)
			return false
		}
		p.Bank = bank
	}
	if p.BankOpen {
		return true
	}
	p.BankOpen = true
	w.sendBank(p)
	return true
}

// sendBank sends the full bank snapshot and opens the window.
func (w *World) sendBank(p *Player) {
	if p.Bank == nil {
		return
	}
	stacks := make([]protocol.BankStack, 0, len(p.Bank.Items))
	for _, item := range p.Bank.Items {
		if item.Amount <= 0 || item.Kind < KindItem || item.Kind > KindArmor {
			continue
		}
		stacks = append(stacks, protocol.BankStack{
			Kind:   uint8(item.Kind),
			ItemID: uint16(clampInt(item.ItemID, 0, 65535)),
			Amount: uint16(clampInt(item.Amount, 0, 65535)),
		})
	}
	_ = p.Client.Send(protocol.ServerBankOpen, protocol.BankOpen{
		Gold:   int32(clampInt64(p.Bank.Gold, 0, w.opts.MaxGold)),
		Stacks: stacks,
	})
}

func bankIndex(bank *store.BankState, kind, id int) int {
	for i, item := range bank.Items {
		if item.Kind == kind && item.ItemID == id && item.Amount > 0 {
			return i
		}
	}
	return -1
}

func bankKindEntries(bank *store.BankState, kind int) int {
	n := 0
	for _, item := range bank.Items {
		if item.Kind == kind && item.Amount > 0 {
			n++
		}
	}
	return n
}

// BankItem moves a stack between the inventory and the bank. Positive amounts
// are deposits, negative amounts withdrawals.
func (w *World) BankItem(ctx context.Context, p *Player, itemID uint16, kind uint8, amount int16) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		k := int(kind)
		id := int(itemID)
		if amount == 0 || k < KindItem || k > KindArmor || w.itemKind(k, id) == nil {
			return ErrInvalidTarget
		}
		n := int(amount)
		if n > 0 {
			if !w.hasBankRoom(p.Bank, k, id, n) {
				return ErrFullInventory
			}
			if !removeInventoryItem(p.State, k, id, n) {
				return ErrNotOwned
			}
			w.addBankItem(p.Bank, k, id, n)
		} else {
			n = -n
			if !w.hasInventoryRoom(p, k, id, n) {
				return ErrFullInventory
			}
			if !removeBankItem(p.Bank, k, id, n) {
				return ErrNotOwned
			}
			if err := addInventoryItem(p.State, w.opts.MaxItems, k, id, n); err != nil {
				return err
			}
			w.addQuestItemNotice(p, k, id)
		}
		p.Bank.Items = compactStacks(p.Bank.Items)
		_ = p.Client.Send(protocol.ServerBankItem, protocol.BankItem{
			ItemID: itemID, Kind: kind, Amount: amount,
		})
		w.sendItemTo(p, k, id, -n, false, false)
		w.saveBankQuietly(p)
		return nil
	})
}

// BankGold moves gold between the player and the bank. Positive amounts are
// deposits, negative amounts withdrawals.
func (w *World) BankGold(ctx context.Context, p *Player, amount int32) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if amount == 0 {
			return nil
		}
		delta := int64(amount)
		if delta > 0 {
			if p.Character.Gold < delta || p.Bank.Gold+delta > w.opts.MaxGold {
				return ErrRequirement
			}
		} else {
			if p.Bank.Gold < -delta || p.Character.Gold-delta > w.opts.MaxGold {
				return ErrRequirement
			}
		}
		p.Character.Gold -= delta
		p.Bank.Gold += delta
		_ = p.Client.Send(protocol.ServerBankGold, protocol.BankGold{Amount: amount})
		_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: -delta, Popup: false})
		w.saveBankQuietly(p)
		w.saveQuietly(p)
		return nil
	})
}

func (w *World) itemKind(kind, id int) any {
	switch kind {
	case KindItem:
		if item := w.item(id); item != nil {
			return item
		}
	case KindWeapon:
		if weapon := w.weapon(id); weapon != nil {
			return weapon
		}
	case KindArmor:
		if armor := w.armor(id); armor != nil {
			return armor
		}
	}
	return nil
}

func (w *World) hasBankRoom(bank *store.BankState, kind, id, amount int) bool {
	i := bankIndex(bank, kind, id)
	if i >= 0 {
		return bank.Items[i].Amount+amount <= w.opts.MaxStack
	}
	return bankKindEntries(bank, kind) < w.opts.MaxBankItems
}

func (w *World) addBankItem(bank *store.BankState, kind, id, amount int) {
	if i := bankIndex(bank, kind, id); i >= 0 {
		bank.Items[i].Amount += amount
		return
	}
	bank.Items = append(bank.Items, store.ItemStack{Kind: kind, ItemID: id, Amount: amount})
}

func removeBankItem(bank *store.BankState, kind, id, amount int) bool {
	i := bankIndex(bank, kind, id)
	if i < 0 || bank.Items[i].Amount < amount {
		return false
	}
	bank.Items[i].Amount -= amount
	if bank.Items[i].Amount == 0 {
		bank.Items = append(bank.Items[:i], bank.Items[i+1:]...)
	}
	return true
}

func (w *World) hasInventoryRoom(p *Player, kind, id, amount int) bool {
	if i := findInventory(p.State, kind, id); i >= 0 {
		return p.State.Inventory[i].Amount+amount <= w.opts.MaxStack
	}
	return inventoryEntries(p.State) < w.opts.MaxItems
}

func compactStacks(items []store.ItemStack) []store.ItemStack {
	out := items[:0]
	for _, item := range items {
		if item.Amount > 0 {
			out = append(out, item)
		}
	}
	return out
}

func (w *World) saveBankQuietly(p *Player) {
	if p.Bank == nil {
		return
	}
	if w.opts.InlineMaps {
		if err := w.opts.Store.SaveBank(context.Background(), p.Account.ID, p.Bank); err != nil {
			w.log.Error("save bank", "account", p.Account.ID, "err", err)
		}
		return
	}
	if w.persist != nil {
		w.persist.queueBank(p.Account.ID, p.Bank)
		return
	}
	if err := w.opts.Store.SaveBank(context.Background(), p.Account.ID, p.Bank); err != nil {
		w.log.Error("save bank", "account", p.Account.ID, "err", err)
	}
}

// closeBank hides the bank window and resumes an event waiting for it.
func (w *World) closeBank(p *Player) {
	if p.Bank != nil {
		w.saveBankQuietly(p)
	}
	p.BankOpen = false
	if p.Event != nil && p.Event.waitingBank {
		p.Event.waitingBank = false
		w.runEvent(p)
	}
}
