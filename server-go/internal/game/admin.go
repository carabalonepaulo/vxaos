package game

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/store"
)

// Admin panel commands, mirroring Enums::Command in the client.
const (
	AdminKick      uint8 = 0
	AdminTeleport  uint8 = 1
	AdminGo        uint8 = 2
	AdminPull      uint8 = 3
	AdminItem      uint8 = 4
	AdminWeapon    uint8 = 5
	AdminArmor     uint8 = 6
	AdminGold      uint8 = 7
	AdminBanIP     uint8 = 8
	AdminBanAcc    uint8 = 9
	AdminUnban     uint8 = 10
	AdminSwitch    uint8 = 11
	AdminMotd      uint8 = 12
	AdminMute      uint8 = 13
	AdminMessage   uint8 = 14
	adminMuteDelay       = 30 * time.Second
	adminKickDelay       = 250 * time.Millisecond
)

var (
	ErrAdminOnly   = errors.New("game: administrator only")
	ErrBadAdminCmd = errors.New("game: invalid admin command")
)

// AdminCommand executes one administrator panel command. It runs on the
// coordinator; actions that need map state are queued to the owning actor.
func (w *World) AdminCommand(ctx context.Context, p *Player, m protocol.AdminCommand) error {
	return w.do(ctx, func() error {
		entry := w.online[p.ID]
		if entry == nil {
			return ErrNotInGame
		}
		group := 0
		if entry.Account != nil {
			group = entry.Account.GroupID
		}
		if group < 1 {
			return ErrAdminOnly
		}
		if group < 2 {
			switch m.Command {
			case AdminGo, AdminPull, AdminMute:
			default:
				return ErrAdminOnly
			}
		}
		switch m.Command {
		case AdminKick:
			return w.adminKick(entry, m.Text)
		case AdminTeleport:
			return w.adminTeleport(entry, m.Text, int(m.Int1), int(m.Int2), int(m.Int3))
		case AdminGo:
			return w.adminGo(entry, m.Text)
		case AdminPull:
			return w.adminPull(entry, m.Text)
		case AdminItem, AdminWeapon, AdminArmor:
			kind := KindItem
			if m.Command == AdminWeapon {
				kind = KindWeapon
			} else if m.Command == AdminArmor {
				kind = KindArmor
			}
			return w.adminGiveItem(entry, m.Text, kind, int(m.Int1), int(m.Int2))
		case AdminGold:
			return w.adminGiveGold(entry, m.Text, int(m.Int2))
		case AdminBanIP, AdminBanAcc:
			return w.adminBan(entry, m.Command, m.Text, int(m.Int1))
		case AdminUnban:
			return w.adminUnban(entry, m.Text)
		case AdminSwitch:
			id, err := strconv.Atoi(strings.TrimSpace(m.Text))
			if err != nil || id <= w.opts.MaxPlayerSwitches {
				return ErrBadAdminCmd
			}
			if !w.setGlobalSwitch(id, m.Int1 == 1) {
				return ErrBadAdminCmd
			}
			w.saveGlobalSwitches()
			return nil
		case AdminMotd:
			w.motd = strings.TrimSpace(m.Text)
			_ = w.opts.Store.SetServerState(context.Background(), "motd", w.motd)
			w.broadcastChatColor(w.motd, 17)
			return nil
		case AdminMute:
			target := w.onlineByName(m.Text)
			if target == nil || isAdminEntry(target) {
				return ErrBadAdminCmd
			}
			target.MuteUntil = w.now().Add(adminMuteDelay)
			w.systemMessageEntry(target, "Você foi silenciado por 30 segundos.", 10)
			return nil
		case AdminMessage:
			notice := protocol.AdminCommandNotice{Command: AdminMessage, Message: m.Text}
			for _, other := range w.online {
				_ = other.Client.Send(protocol.ServerAdminCommand, notice)
			}
			w.broadcastChatColor(m.Text, 17)
			return nil
		}
		return ErrBadAdminCmd
	})
}

func isAdminEntry(entry *onlineEntry) bool {
	return entry != nil && entry.Account != nil && entry.Account.GroupID >= 1
}

func (w *World) systemMessageEntry(entry *onlineEntry, message string, color uint8) {
	if entry == nil {
		return
	}
	_ = entry.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: color, Message: message})
}

func (w *World) broadcastChatColor(message string, color uint8) {
	if strings.TrimSpace(message) == "" {
		return
	}
	msg := protocol.ChatMessage{Color: color, Message: message}
	for _, entry := range w.online {
		_ = entry.Client.Send(protocol.ServerChat, msg)
	}
}

func (w *World) adminKick(admin *onlineEntry, name string) error {
	target := w.onlineByName(name)
	if target == nil || isAdminEntry(target) {
		return ErrBadAdminCmd
	}
	w.broadcastChatColor(target.Name+" foi expulso.", 10)
	w.kickNoticeEntry(target, AdminKick, "")
	return nil
}

func (w *World) adminTeleport(admin *onlineEntry, name string, mapID, x, y int) error {
	if !w.mapTemplateValid(mapID, x, y) {
		return ErrBadAdminCmd
	}
	if name == "all" {
		for _, entry := range w.online {
			entry := entry
			w.enqueueMap(entry.Key, func(a *mapActor) {
				if p := a.playerByID(entry.ID); p != nil {
					w.transferPlayer(p, mapID, x, y, p.Character.Direction)
				}
			})
		}
		return nil
	}
	target := w.onlineByName(name)
	if target == nil {
		return ErrBadAdminCmd
	}
	w.enqueueMap(target.Key, func(a *mapActor) {
		if p := a.playerByID(target.ID); p != nil {
			w.transferPlayer(p, mapID, x, y, p.Character.Direction)
		}
	})
	return nil
}

func (w *World) adminGo(admin *onlineEntry, name string) error {
	target := w.onlineByName(name)
	if target == nil {
		return ErrBadAdminCmd
	}
	adminKey := admin.Key
	adminID := admin.ID
	w.enqueueMap(target.Key, func(a *mapActor) {
		p := a.playerByID(target.ID)
		if p == nil {
			return
		}
		x, y := p.Character.X, p.Character.Y
		dir := p.Character.Direction
		_ = w.coordErr(func() error {
			w.enqueueMap(adminKey, func(adminActor *mapActor) {
				if adminPlayer := adminActor.playerByID(adminID); adminPlayer != nil {
					w.transferPlayer(adminPlayer, target.Key.TemplateID, x, y, dir)
				}
			})
			return nil
		})
	})
	return nil
}

func (w *World) adminPull(admin *onlineEntry, name string) error {
	if name == "all" {
		for _, entry := range w.online {
			if entry.ID == admin.ID {
				continue
			}
			w.enqueueMap(entry.Key, func(a *mapActor) {
				w.pullToAdmin(a, entry, admin)
			})
		}
		return nil
	}
	target := w.onlineByName(name)
	if target == nil {
		return ErrBadAdminCmd
	}
	w.enqueueMap(target.Key, func(a *mapActor) {
		w.pullToAdmin(a, target, admin)
	})
	return nil
}

// pullToAdmin runs on the target's map: it reads the admin position from the
// admin's map and then transfers the target there.
func (w *World) pullToAdmin(a *mapActor, target, admin *onlineEntry) {
	p := a.playerByID(target.ID)
	if p == nil {
		return
	}
	_ = w.coordErr(func() error {
		w.enqueueMap(admin.Key, func(adminActor *mapActor) {
			ap := adminActor.playerByID(admin.ID)
			if ap == nil {
				return
			}
			x, y := ap.Character.X, ap.Character.Y
			w.enqueueMap(target.Key, func(targetActor *mapActor) {
				if tp := targetActor.playerByID(target.ID); tp != nil {
					w.transferPlayer(tp, admin.Key.TemplateID, x, y, tp.Character.Direction)
				}
			})
		})
		return nil
	})
}

func (w *World) adminGiveItem(admin *onlineEntry, name string, kind, id, amount int) error {
	if amount == 0 || w.itemKind(kind, id) == nil {
		return ErrBadAdminCmd
	}
	targets := w.adminTargets(name)
	if len(targets) == 0 {
		return ErrBadAdminCmd
	}
	for _, target := range targets {
		target := target
		w.enqueueMap(target.Key, func(a *mapActor) {
			p := a.playerByID(target.ID)
			if p == nil {
				return
			}
			if amount > 0 {
				if err := addInventoryItem(p.State, w.opts.MaxItems, kind, id, amount); err != nil {
					w.systemMessageEntry(admin, target.Name+": inventário cheio.", 10)
					return
				}
				w.sendItemTo(p, kind, id, amount, false, true)
			} else {
				if !removeInventoryItem(p.State, kind, id, -amount) {
					return
				}
				w.sendItemTo(p, kind, id, amount, false, true)
			}
			w.saveQuietly(p)
		})
	}
	w.systemMessageEntry(admin, "Item entregue.", 5)
	return nil
}

func (w *World) adminGiveGold(admin *onlineEntry, name string, amount int) error {
	if amount == 0 {
		return ErrBadAdminCmd
	}
	targets := w.adminTargets(name)
	if len(targets) == 0 {
		return ErrBadAdminCmd
	}
	for _, target := range targets {
		target := target
		w.enqueueMap(target.Key, func(a *mapActor) {
			p := a.playerByID(target.ID)
			if p == nil {
				return
			}
			p.Character.Gold = clampInt64(p.Character.Gold+int64(amount), 0, w.opts.MaxGold)
			_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: int64(amount), Popup: true})
			w.saveQuietly(p)
		})
	}
	w.systemMessageEntry(admin, "Ouro entregue.", 5)
	return nil
}

func (w *World) adminTargets(name string) []*onlineEntry {
	if name == "all" {
		out := make([]*onlineEntry, 0, len(w.online))
		for _, entry := range w.online {
			out = append(out, entry)
		}
		return out
	}
	if target := w.onlineByName(name); target != nil {
		return []*onlineEntry{target}
	}
	return nil
}

func (w *World) adminBan(admin *onlineEntry, command uint8, name string, days int) error {
	if days <= 0 {
		days = 1
	}
	expires := w.now().Unix() + int64(days)*86400
	target := w.onlineByName(name)
	kind := "account"
	value := ""
	var accountID int64
	var ip string
	if target != nil {
		if isAdminEntry(target) {
			return ErrBadAdminCmd
		}
		accountID = target.Account.ID
		ip = target.IP
	} else if character, err := w.opts.Store.GetCharacterByName(context.Background(), name); err == nil && character != nil {
		accountID = character.AccountID
	} else {
		return ErrBadAdminCmd
	}
	switch command {
	case AdminBanAcc:
		kind = "account"
		value = strconv.FormatInt(accountID, 10)
	case AdminBanIP:
		kind = "ip"
		value = ip
		if value == "" {
			return ErrBadAdminCmd
		}
	}
	ban := store.Ban{
		Kind: kind, Value: value, Reason: "admin",
		ExpiresAt: expires, CreatedAt: w.now().Unix(), CreatedBy: admin.Account.ID,
	}
	if err := w.opts.Store.SaveBan(context.Background(), ban); err != nil {
		w.log.Error("save ban", "err", err)
		return err
	}
	for _, other := range w.online {
		if isAdminEntry(other) {
			continue
		}
		if kind == "account" && other.Account.ID == accountID {
			w.kickNoticeEntry(other, AdminBanAcc, "")
		}
		if kind == "ip" && ip != "" && other.IP == ip {
			w.kickNoticeEntry(other, AdminBanIP, "")
		}
	}
	w.broadcastChatColor(name+" foi banido por "+strconv.Itoa(days)+" dia(s).", 10)
	return nil
}

func (w *World) adminUnban(admin *onlineEntry, text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return ErrBadAdminCmd
	}
	value := ""
	if character, err := w.opts.Store.GetCharacterByName(context.Background(), text); err == nil && character != nil {
		value = strconv.FormatInt(character.AccountID, 10)
		if err := w.opts.Store.DeleteBan(context.Background(), "account", value); err != nil {
			return err
		}
	}
	if _, err := strconv.ParseInt(text, 10, 64); err == nil {
		value = text
	}
	if value != "" {
		_ = w.opts.Store.DeleteBan(context.Background(), "account", value)
	}
	if strings.Contains(text, ".") || strings.Contains(text, ":") {
		_ = w.opts.Store.DeleteBan(context.Background(), "ip", text)
	}
	if target := w.onlineByName(text); target != nil {
		_ = w.opts.Store.DeleteBan(context.Background(), "account", strconv.FormatInt(target.Account.ID, 10))
	}
	w.systemMessageEntry(admin, "Banimento removido.", 5)
	return nil
}

// kickNoticeEntry sends the admin notification, then closes the connection
// after a short delay so the client can display the alert.
func (w *World) kickNoticeEntry(target *onlineEntry, command uint8, message string) {
	_ = target.Client.Send(protocol.ServerAdminCommand, protocol.AdminCommandNotice{Command: command, Message: message})
	client := target.Client
	time.AfterFunc(adminKickDelay, func() { _ = client.Close() })
}

// sendMOTD shows the persisted message of the day after entering the world.
func (w *World) sendMOTD(p *Player) {
	if w.motd != "" {
		w.systemMessage(p, w.motd, 17)
	}
}

func (w *World) sendMOTDText(p *Player, motd string) {
	if motd != "" {
		_ = p.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: 17, Message: motd})
	}
}

func (w *World) systemMessage(p *Player, message string, color uint8) {
	if p == nil {
		return
	}
	_ = p.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: color, Message: message})
}
