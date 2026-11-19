package game

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/store"
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

// AdminCommand executes one administrator panel command. Monitors may use the
// navigation and mute commands only, matching the original server.
func (w *World) AdminCommand(ctx context.Context, p *Player, m protocol.AdminCommand) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		group := 0
		if p.Account != nil {
			group = p.Account.GroupID
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
			return w.adminKick(p, m.Text)
		case AdminTeleport:
			return w.adminTeleport(p, m.Text, int(m.Int1), int(m.Int2), int(m.Int3))
		case AdminGo:
			return w.adminGo(p, m.Text)
		case AdminPull:
			return w.adminPull(p, m.Text)
		case AdminItem, AdminWeapon, AdminArmor:
			kind := KindItem
			if m.Command == AdminWeapon {
				kind = KindWeapon
			} else if m.Command == AdminArmor {
				kind = KindArmor
			}
			return w.adminGiveItem(p, m.Text, kind, int(m.Int1), int(m.Int2))
		case AdminGold:
			return w.adminGiveGold(p, m.Text, int(m.Int2))
		case AdminBanIP, AdminBanAcc:
			return w.adminBan(p, m.Command, m.Text, int(m.Int1))
		case AdminUnban:
			return w.adminUnban(p, m.Text)
		case AdminSwitch:
			id, err := strconv.Atoi(strings.TrimSpace(m.Text))
			if err != nil || id <= w.opts.MaxPlayerSwitches {
				return ErrBadAdminCmd
			}
			w.setGlobalSwitch(id, m.Int1 == 1)
			w.saveGlobalSwitches()
			return nil
		case AdminMotd:
			w.motd = strings.TrimSpace(m.Text)
			_ = w.opts.Store.SetServerState(context.Background(), "motd", w.motd)
			w.broadcastChatColor(w.motd, 17)
			return nil
		case AdminMute:
			target := w.playerByName(m.Text)
			if target == nil || isAdmin(target) {
				return ErrBadAdminCmd
			}
			target.MuteUntil = w.now().Add(adminMuteDelay)
			w.systemMessage(target, "Você foi silenciado por 30 segundos.", 10)
			return nil
		case AdminMessage:
			notice := protocol.AdminCommandNotice{Command: AdminMessage, Message: m.Text}
			for _, other := range w.players {
				if other == nil {
					continue
				}
				_ = other.Client.Send(protocol.ServerAdminCommand, notice)
			}
			w.broadcastChatColor(m.Text, 17)
			return nil
		}
		return ErrBadAdminCmd
	})
}

func (w *World) systemMessage(p *Player, message string, color uint8) {
	if p == nil {
		return
	}
	_ = p.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: color, Message: message})
}

func (w *World) broadcastChatColor(message string, color uint8) {
	if strings.TrimSpace(message) == "" {
		return
	}
	msg := protocol.ChatMessage{Color: color, Message: message}
	for _, p := range w.players {
		if p == nil {
			continue
		}
		_ = p.Client.Send(protocol.ServerChat, msg)
	}
}

func (w *World) adminKick(admin *Player, name string) error {
	target := w.playerByName(name)
	if target == nil || isAdmin(target) {
		return ErrBadAdminCmd
	}
	w.broadcastChatColor(target.Character.Name+" foi expulso.", 10)
	w.kickNotice(target, AdminKick, "")
	return nil
}

func (w *World) adminTeleport(admin *Player, name string, mapID, x, y int) error {
	if name == "all" {
		for _, p := range w.players {
			if p == nil {
				continue
			}
			w.transferPlayer(p, mapID, x, y, p.Character.Direction)
		}
		return nil
	}
	target := w.playerByName(name)
	if target == nil {
		return ErrBadAdminCmd
	}
	if !w.transferPlayer(target, mapID, x, y, target.Character.Direction) {
		return ErrBadAdminCmd
	}
	return nil
}

func (w *World) adminGo(admin *Player, name string) error {
	target := w.playerByName(name)
	if target == nil {
		return ErrBadAdminCmd
	}
	if !w.transferPlayer(admin, target.Character.MapID, target.Character.X, target.Character.Y, admin.Character.Direction) {
		return ErrBadAdminCmd
	}
	return nil
}

func (w *World) adminPull(admin *Player, name string) error {
	if name == "all" {
		for _, p := range w.players {
			if p == nil || p == admin {
				continue
			}
			w.transferPlayer(p, admin.Character.MapID, admin.Character.X, admin.Character.Y, p.Character.Direction)
		}
		return nil
	}
	target := w.playerByName(name)
	if target == nil {
		return ErrBadAdminCmd
	}
	if !w.transferPlayer(target, admin.Character.MapID, admin.Character.X, admin.Character.Y, target.Character.Direction) {
		return ErrBadAdminCmd
	}
	return nil
}

func (w *World) adminGiveItem(admin *Player, name string, kind, id, amount int) error {
	if amount == 0 || w.itemKind(kind, id) == nil {
		return ErrBadAdminCmd
	}
	targets := w.adminTargets(name)
	if len(targets) == 0 {
		return ErrBadAdminCmd
	}
	for _, target := range targets {
		if amount > 0 {
			if err := addInventoryItem(target.State, w.opts.MaxItems, kind, id, amount); err != nil {
				w.systemMessage(admin, target.Character.Name+": inventário cheio.", 10)
				continue
			}
			w.sendItemTo(target, kind, id, amount, false, true)
		} else {
			if !removeInventoryItem(target.State, kind, id, -amount) {
				continue
			}
			w.sendItemTo(target, kind, id, amount, false, true)
		}
		w.saveQuietly(target)
	}
	w.systemMessage(admin, "Item entregue.", 5)
	return nil
}

func (w *World) adminGiveGold(admin *Player, name string, amount int) error {
	if amount == 0 {
		return ErrBadAdminCmd
	}
	targets := w.adminTargets(name)
	if len(targets) == 0 {
		return ErrBadAdminCmd
	}
	for _, target := range targets {
		target.Character.Gold = clampInt64(target.Character.Gold+int64(amount), 0, w.opts.MaxGold)
		_ = target.Client.Send(protocol.ServerGold, protocol.Gold{Amount: int64(amount), Popup: true})
		w.saveQuietly(target)
	}
	w.systemMessage(admin, "Ouro entregue.", 5)
	return nil
}

func (w *World) adminTargets(name string) []*Player {
	if name == "all" {
		out := make([]*Player, 0, len(w.players))
		for _, p := range w.players {
			if p == nil {
				continue
			}
			out = append(out, p)
		}
		return out
	}
	if target := w.playerByName(name); target != nil {
		return []*Player{target}
	}
	return nil
}

func (w *World) adminBan(admin *Player, command uint8, name string, days int) error {
	if days <= 0 {
		days = 1
	}
	expires := w.now().Unix() + int64(days)*86400
	target := w.playerByName(name)
	kind := "account"
	value := ""
	var accountID int64
	var ip string
	if target != nil {
		if isAdmin(target) {
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
	for _, other := range w.players {
		if other == nil || isAdmin(other) {
			continue
		}
		if kind == "account" && other.Account.ID == accountID {
			w.kickNotice(other, AdminBanAcc, "")
		}
		if kind == "ip" && ip != "" && other.IP == ip {
			w.kickNotice(other, AdminBanIP, "")
		}
	}
	w.broadcastChatColor(name+" foi banido por "+strconv.Itoa(days)+" dia(s).", 10)
	return nil
}

func (w *World) adminUnban(admin *Player, text string) error {
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
	if target := w.playerByName(text); target != nil {
		_ = w.opts.Store.DeleteBan(context.Background(), "account", strconv.FormatInt(target.Account.ID, 10))
	}
	w.systemMessage(admin, "Banimento removido.", 5)
	return nil
}

// kickNotice sends the admin notification, then closes the connection after a
// short delay so the client can display the alert.
func (w *World) kickNotice(target *Player, command uint8, message string) {
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
