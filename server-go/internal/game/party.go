package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
)

var (
	ErrInParty    = errors.New("game: already in a party")
	ErrNotInParty = errors.New("game: not in a party")
	ErrPartyFull  = errors.New("game: party is full")
	ErrPartyRange = errors.New("game: player is out of range")
)

const (
	requestParty  = 3
	requestFriend = 4
)

// partyState is one online party. Members are player session ids.
type partyState struct {
	ID      int
	Members []uint16
}

func (w *World) partyByID(id int) *partyState {
	if id == 0 {
		return nil
	}
	return w.parties[id]
}

func (w *World) partyEntries(party *partyState) []*onlineEntry {
	if party == nil {
		return nil
	}
	out := make([]*onlineEntry, 0, len(party.Members))
	for _, id := range party.Members {
		if entry := w.online[id]; entry != nil {
			out = append(out, entry)
		}
	}
	return out
}

// partyMembersOnMap returns the killer plus party members present on the same
// map actor. It never reads another map's runtime.
func (w *World) partyMembersOnMap(p *Player) []*Player {
	if p == nil {
		return nil
	}
	ms := p.ms
	if ms == nil {
		return []*Player{p}
	}
	out := make([]*Player, 0, 4)
	for _, member := range ms.players {
		if member == nil {
			continue
		}
		if member == p || (p.PartyID != 0 && member.PartyID == p.PartyID) {
			out = append(out, member)
		}
	}
	if len(out) == 0 {
		return []*Player{p}
	}
	return out
}

func (w *World) createPartyEntry(leader *onlineEntry) *partyState {
	party := &partyState{ID: w.nextPartyID, Members: []uint16{leader.ID}}
	w.nextPartyID++
	w.parties[party.ID] = party
	leader.PartyID = party.ID
	w.patchSocial(leader.ID, func(p *Player) { p.PartyID = party.ID })
	return party
}

// PlayerRequest dispatches a target-HUD invitation. Range checks happen on the
// map actor that owns both players; the social state stays on the coordinator.
func (w *World) PlayerRequest(ctx context.Context, p *Player, typ uint8, targetID uint16) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if typ == requestFinishTrade {
			return w.requestFinishTradeLocked(p)
		}
		target := ms.playerByID(targetID)
		if typ == requestGuild {
			var err error
			_ = w.coordErr(func() error {
				err = w.inviteGuildEntry(w.online[p.ID], targetID)
				return err
			})
			return err
		}
		if target == nil || target == p {
			return ErrInvalidTarget
		}
		switch typ {
		case requestParty:
			if err := w.checkInviteRange(p, target); err != nil {
				return err
			}
			var err error
			_ = w.coordErr(func() error {
				err = w.invitePartyEntry(w.online[p.ID], w.online[target.ID])
				return err
			})
			return err
		case requestFriend:
			if err := w.checkInviteRange(p, target); err != nil {
				return err
			}
			var err error
			_ = w.coordErr(func() error {
				err = w.inviteFriendEntry(w.online[p.ID], w.online[target.ID])
				return err
			})
			return err
		case requestTrade:
			if err := w.checkInviteRange(p, target); err != nil {
				return err
			}
			return w.inviteTradeLocked(p, target)
		default:
			return ErrUnsupported
		}
	})
}

func (w *World) checkInviteRange(p, target *Player) error {
	if p == nil || target == nil || p.ms != target.ms {
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
	return nil
}

func (w *World) invitePartyEntry(from, target *onlineEntry) error {
	if from == nil || target == nil {
		return ErrInvalidTarget
	}
	if target.PartyID != 0 {
		return ErrInParty
	}
	if from.PartyID != 0 {
		if party := w.partyByID(from.PartyID); party != nil && len(party.Members) >= w.opts.MaxPartyMembers {
			return ErrPartyFull
		}
	}
	req := &pendingRequest{Type: requestParty, FromID: from.ID}
	target.Request = req
	w.patchSocial(target.ID, func(p *Player) { p.Request = cloneRequest(req) })
	_ = target.Client.Send(protocol.ServerRequest, protocol.PlayerRequest{
		Type: requestParty, FromID: from.ID,
		Text: fmt.Sprintf("%s te convidou para o grupo.", from.Name),
	})
	return nil
}

func (w *World) acceptPartyEntry(accepting, from *onlineEntry) error {
	if accepting == nil || from == nil {
		return ErrNotInGame
	}
	if accepting.PartyID != 0 {
		return ErrInParty
	}
	party := w.partyByID(from.PartyID)
	if party == nil {
		party = w.createPartyEntry(from)
	}
	if len(party.Members) >= w.opts.MaxPartyMembers {
		return ErrPartyFull
	}
	join := partyJoinFromView(accepting.View)
	for _, member := range w.partyEntries(party) {
		_ = member.Client.Send(protocol.ServerPartyJoin, join)
		_ = accepting.Client.Send(protocol.ServerPartyJoin, partyJoinFromView(member.View))
	}
	party.Members = append(party.Members, accepting.ID)
	accepting.PartyID = party.ID
	w.patchSocial(accepting.ID, func(p *Player) { p.PartyID = party.ID })
	msg := protocol.ChatMessage{Color: 2, Message: fmt.Sprintf("%s entrou no grupo.", accepting.Name)}
	for _, member := range w.partyEntries(party) {
		_ = member.Client.Send(protocol.ServerChat, msg)
	}
	return nil
}

func partyViewOf(p *Player) partyView {
	v := partyView{
		ID:             p.ID,
		Name:           p.Character.Name,
		CharacterName:  p.Character.CharacterName,
		CharacterIndex: p.Character.CharacterIndex,
		Sex:            p.Character.Sex,
		Level:          p.Character.Level,
		ClassID:        p.Character.ClassID,
		HP:             p.Character.HP,
		MHP:            p.Character.MHP,
		EXP:            p.Character.Exp,
	}
	for i := 0; i < len(v.Equips) && i < len(p.State.Equips); i++ {
		v.Equips[i] = uint16(clampInt(p.State.Equips[i], 0, 65535))
	}
	return v
}

func partyJoinFromView(v partyView) protocol.PartyJoin {
	join := protocol.PartyJoin{
		PlayerID:       v.ID,
		Name:           v.Name,
		CharacterName:  v.CharacterName,
		CharacterIndex: uint8(clampInt(v.CharacterIndex, 0, 255)),
		Sex:            uint8(clampInt(v.Sex, 0, 255)),
		HP:             int32(v.HP),
		EXP:            int32(v.EXP),
		Equips:         v.Equips,
		MHP:            int32(v.MHP),
	}
	return join
}

func (w *World) partyJoinMessage(p *Player) protocol.PartyJoin {
	join := protocol.PartyJoin{
		PlayerID:       p.ID,
		Name:           p.Character.Name,
		CharacterName:  p.Character.CharacterName,
		CharacterIndex: uint8(clampInt(p.Character.CharacterIndex, 0, 255)),
		Sex:            uint8(clampInt(p.Character.Sex, 0, 255)),
		HP:             int32(p.Character.HP),
		EXP:            int32(p.Character.Exp),
	}
	for i := 0; i < len(join.Equips) && i < len(p.State.Equips); i++ {
		join.Equips[i] = uint16(clampInt(p.State.Equips[i], 0, 65535))
	}
	join.MHP = int32(p.Character.MHP)
	if class := w.class(p.Character.ClassID); class != nil {
		join.MHP = int32(class.ParamAt(0, p.Character.Level))
	}
	return join
}

// LeaveParty removes the player from the party on the coordinator.
func (w *World) LeaveParty(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error {
		entry := w.online[p.ID]
		if entry == nil {
			return ErrNotInGame
		}
		w.leavePartyEntry(entry)
		return nil
	})
}

func (w *World) leavePartyEntry(entry *onlineEntry) {
	if entry == nil {
		return
	}
	party := w.partyByID(entry.PartyID)
	if party == nil {
		entry.PartyID = 0
		w.patchSocial(entry.ID, func(p *Player) { p.PartyID = 0 })
		return
	}
	_ = entry.Client.Send(protocol.ServerPartyDissolve, protocol.PartyDissolve{})
	remaining := party.Members[:0]
	for _, id := range party.Members {
		if id != entry.ID {
			remaining = append(remaining, id)
		}
	}
	party.Members = remaining
	entry.PartyID = 0
	w.patchSocial(entry.ID, func(p *Player) { p.PartyID = 0 })
	if len(party.Members) == 1 {
		last := w.online[party.Members[0]]
		if last != nil {
			_ = last.Client.Send(protocol.ServerPartyDissolve, protocol.PartyDissolve{})
			last.PartyID = 0
			w.patchSocial(last.ID, func(p *Player) { p.PartyID = 0 })
		}
		delete(w.parties, party.ID)
		return
	}
	if len(party.Members) == 0 {
		delete(w.parties, party.ID)
		return
	}
	msg := protocol.PartyLeave{PlayerID: entry.ID}
	for _, id := range party.Members {
		if member := w.online[id]; member != nil {
			_ = member.Client.Send(protocol.ServerPartyLeave, msg)
		}
	}
}

// leavePartyLocked is kept for inline tests and inline leave.
func (w *World) leavePartyLocked(p *Player) {
	if entry := w.online[p.ID]; entry != nil {
		w.leavePartyEntry(entry)
		return
	}
	p.PartyID = 0
}

func (w *World) partyChat(party *partyState, message string) {
	msg := protocol.ChatMessage{Color: 2, Message: message}
	for _, member := range w.partyEntries(party) {
		_ = member.Client.Send(protocol.ServerChat, msg)
	}
}

func (w *World) class(id int) *rpg.Class {
	if id <= 0 || id >= len(w.opts.Data.Classes) {
		return nil
	}
	return w.opts.Data.Classes[id]
}

func partyBonus(size int) int {
	switch size {
	case 2:
		return 10
	case 3:
		return 15
	case 4:
		return 20
	case 5:
		return 30
	}
	return 0
}

func (w *World) sharePartyExp(killer *Player, members []*Player, exp int) {
	if exp <= 0 {
		return
	}
	if len(members) <= 1 || len(members) > exp {
		w.gainExp(killer, exp)
		return
	}
	size := len(members)
	share := exp/size + exp*partyBonus(size)/100
	remainder := exp - (exp/size)*size
	for _, member := range members {
		amount := share
		if member == killer {
			amount += remainder
		}
		w.gainExp(member, amount)
	}
}

func (w *World) sharePartyGold(killer *Player, members []*Player, gold int) {
	if gold <= 0 {
		return
	}
	if len(members) <= 1 || len(members) > gold {
		if killer != nil {
			w.gainGold(killer, gold)
		} else if len(members) == 1 {
			w.gainGold(members[0], gold)
		}
		return
	}
	size := len(members)
	share := gold/size + gold*partyBonus(size)/100
	for _, member := range members {
		w.gainGold(member, share)
	}
}

func (w *World) gainGold(p *Player, amount int) {
	if amount <= 0 {
		return
	}
	p.Character.Gold = clampInt64(p.Character.Gold+int64(amount), 0, w.opts.MaxGold)
	_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: int64(amount), ShopSound: true, Popup: true})
}
