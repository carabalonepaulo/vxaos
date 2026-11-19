package game

import (
	"context"
	"fmt"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/script"
)

// runEventScript executes a migrated 355 script. The script gets a small set
// of native functions bound to this player/event; no Ruby or client state is
// involved.
func (w *World) runEventScript(p *Player, frame *eventFrame, source string) bool {
	vm := script.NewVM()
	register := func(name string, min, max int, fn func(*script.VM, []any) (any, error)) {
		vm.Globals[name] = &script.NativeFunc{Name: name, MinArgs: min, MaxArgs: max, Fn: fn}
	}
	register("start-quest", 1, 1, func(_ *script.VM, args []any) (any, error) {
		w.startQuest(p, scriptInt(args, 0))
		return nil, nil
	})
	register("finish-quest", 1, 1, func(_ *script.VM, args []any) (any, error) {
		w.finishQuest(p, scriptInt(args, 0))
		return nil, nil
	})
	register("check-point", 3, 3, func(_ *script.VM, args []any) (any, error) {
		p.Character.ReviveMapID = scriptInt(args, 0)
		p.Character.ReviveX = scriptInt(args, 1)
		p.Character.ReviveY = scriptInt(args, 2)
		return nil, nil
	})
	register("chat-add", 2, 2, func(_ *script.VM, args []any) (any, error) {
		_ = p.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: uint8(scriptInt(args, 1)), Message: scriptString(args, 0)})
		return nil, nil
	})
	register("add-vip-days", 1, 1, func(_ *script.VM, args []any) (any, error) {
		days := scriptInt(args, 0)
		var until int64
		_ = w.coordErr(func() error {
			until = p.vipUntil()
			now := w.now().Unix()
			if until < now {
				until = now
			}
			until += int64(days) * 86400
			p.setVIPUntil(until)
			return w.opts.Store.UpdateAccountVIP(context.Background(), p.Account.ID, until)
		})
		_ = p.Client.Send(protocol.ServerVIPDays, protocol.VIPDays{Until: until})
		return nil, nil
	})
	register("change-sex", 0, 0, func(_ *script.VM, args []any) (any, error) {
		p.Character.Sex = 1 - p.Character.Sex
		w.broadcastToMap(p, protocol.ServerSex, protocol.Sex{PlayerID: p.ID, Sex: uint8(p.Character.Sex)})
		return nil, nil
	})
	register("save-original-graphic", 0, 0, func(_ *script.VM, args []any) (any, error) {
		p.OriginalCharacterName = p.Character.CharacterName
		p.OriginalCharacterIndex = p.Character.CharacterIndex
		p.OriginalFaceName = p.Character.FaceName
		p.OriginalFaceIndex = p.Character.FaceIndex
		return nil, nil
	})
	register("load-original-graphic", 0, 0, func(_ *script.VM, args []any) (any, error) {
		p.Character.CharacterName = p.OriginalCharacterName
		p.Character.CharacterIndex = p.OriginalCharacterIndex
		p.Character.FaceName = p.OriginalFaceName
		p.Character.FaceIndex = p.OriginalFaceIndex
		return nil, nil
	})
	register("send-player-graphic", 0, 0, func(_ *script.VM, args []any) (any, error) {
		w.broadcastToMap(p, protocol.ServerGraphic, protocol.Graphic{
			PlayerID: p.ID, CharacterName: p.Character.CharacterName,
			CharacterIndex: uint8(p.Character.CharacterIndex),
			FaceName:       p.Character.FaceName, FaceIndex: uint8(p.Character.FaceIndex),
		})
		return nil, nil
	})
	register("send-player-sex", 0, 0, func(_ *script.VM, args []any) (any, error) {
		w.broadcastToMap(p, protocol.ServerSex, protocol.Sex{PlayerID: p.ID, Sex: uint8(p.Character.Sex)})
		return nil, nil
	})
	register("reset-parameters", 0, 0, func(_ *script.VM, args []any) (any, error) {
		w.resetParameters(p)
		return nil, nil
	})
	register("erase-buff", 1, 1, func(_ *script.VM, args []any) (any, error) {
		w.eraseBuff(&p.Status, p, scriptInt(args, 0))
		return nil, nil
	})
	register("open-teleport", 1, 1, func(_ *script.VM, args []any) (any, error) {
		group := scriptInt(args, 0) - 1
		if group < 0 {
			return nil, nil
		}
		p.TeleportGroup = group
		_ = p.Client.Send(protocol.ServerTeleportOpen, protocol.TeleportOpen{ID: uint8(clampInt(group, 0, 255))})
		return nil, nil
	})
	register("open-bank", 0, 0, func(_ *script.VM, args []any) (any, error) {
		if w.openBank(p) && p.Event != nil {
			p.Event.waitingBank = true
		}
		return nil, nil
	})
	register("open-create-guild", 0, 0, func(_ *script.VM, args []any) (any, error) {
		if p.GuildName != "" {
			return nil, nil
		}
		p.CreatingGuild = true
		_ = p.Client.Send(protocol.ServerGuildCreateOpen, protocol.GuildCreateOpen{})
		return nil, nil
	})
	register("open-shop", 0, 1, func(_ *script.VM, args []any) (any, error) { return nil, nil })
	if _, err := vm.EvalString("event", source, nil); err != nil {
		w.log.Warn("event script failed", "player", p.Character.Name, "event", frame.eventID, "err", err)
		return false
	}
	return true
}

func scriptInt(args []any, index int) int {
	if index < 0 || index >= len(args) {
		return 0
	}
	switch v := args[index].(type) {
	case int64:
		return int(v)
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func scriptString(args []any, index int) string {
	if index < 0 || index >= len(args) {
		return ""
	}
	if value, ok := args[index].(string); ok {
		return value
	}
	return fmt.Sprint(args[index])
}

func (w *World) resetParameters(p *Player) {
	if p.Character.ClassID <= 0 || p.Character.ClassID >= len(w.opts.Data.Classes) {
		return
	}
	class := w.opts.Data.Classes[p.Character.ClassID]
	if class == nil {
		return
	}
	level := p.Character.Level
	if level < 1 {
		level = 1
	}
	values := [8]int{
		class.ParamAt(0, level), class.ParamAt(1, level), class.ParamAt(2, level), class.ParamAt(3, level),
		class.ParamAt(4, level), class.ParamAt(5, level), class.ParamAt(6, level), class.ParamAt(7, level),
	}
	p.Character.MHP, p.Character.MMP = values[0], values[1]
	p.Character.ATK, p.Character.DEF, p.Character.MAT = values[2], values[3], values[4]
	p.Character.MDF, p.Character.AGI, p.Character.LUK = values[5], values[6], values[7]
	if p.Character.HP > p.Character.MHP {
		p.Character.HP = p.Character.MHP
	}
	if p.Character.MP > p.Character.MMP {
		p.Character.MP = p.Character.MMP
	}
	w.sendVitalsToMap(p)
	_ = time.Now
}
