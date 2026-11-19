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

func (w *World) partyOnline(party *partyState) []*Player {
	if party == nil {
		return nil
	}
	out := make([]*Player, 0, len(party.Members))
	for _, id := range party.Members {
		if p := w.playerByID(id); p != nil {
			out = append(out, p)
		}
	}
	return out
}

func (w *World) partyMembersOnMap(p *Player) []*Player {
	party := w.partyByID(p.PartyID)
	if party == nil {
		return []*Player{p}
	}
	out := make([]*Player, 0, len(party.Members))
	for _, member := range w.partyOnline(party) {
		if member.Character.MapID == p.Character.MapID {
			out = append(out, member)
		}
	}
	if len(out) == 0 {
		return []*Player{p}
	}
	return out
}

func (w *World) createParty(leader *Player) *partyState {
	party := &partyState{ID: w.nextPartyID, Members: []uint16{leader.ID}}
	w.nextPartyID++
	w.parties[party.ID] = party
	leader.PartyID = party.ID
	return party
}

// PlayerRequest dispatches a target-HUD invitation.
func (w *World) PlayerRequest(ctx context.Context, p *Player, typ uint8, targetID uint16) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		if typ == requestFinishTrade {
			return w.requestFinishTradeLocked(p)
		}
		target := w.playerByID(targetID)
		if target == nil || target == p {
			return ErrInvalidTarget
		}
		switch typ {
		case requestParty:
			return w.invitePartyLocked(p, target)
		case requestFriend:
			return w.inviteFriendLocked(p, target)
		case requestTrade:
			return w.inviteTradeLocked(p, target)
		case requestGuild:
			return w.inviteGuildLocked(p, target.Character.Name)
		default:
			return ErrUnsupported
		}
	})
}

// invitePartyLocked validates and queues a party invitation.
func (w *World) invitePartyLocked(p, target *Player) error {
	if target.PartyID != 0 {
		return ErrInParty
	}
	if p.PartyID != 0 {
		if party := w.partyByID(p.PartyID); party != nil && len(party.Members) >= w.opts.MaxPartyMembers {
			return ErrPartyFull
		}
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
	target.Request = &pendingRequest{Type: requestParty, FromID: p.ID}
	_ = target.Client.Send(protocol.ServerRequest, protocol.PlayerRequest{
		Type: requestParty, FromID: p.ID,
		Text: fmt.Sprintf("%s te convidou para o grupo.", p.Character.Name),
	})
	return nil
}

// acceptPartyLocked adds the accepting player to the inviter's party.
func (w *World) acceptPartyLocked(p, from *Player) error {
	if p.PartyID != 0 {
		return ErrInParty
	}
	party := w.partyByID(from.PartyID)
	if party == nil {
		party = w.createParty(from)
	}
	if len(party.Members) >= w.opts.MaxPartyMembers {
		return ErrPartyFull
	}
	join := w.partyJoinMessage(p)
	for _, member := range w.partyOnline(party) {
		_ = member.Client.Send(protocol.ServerPartyJoin, join)
		_ = p.Client.Send(protocol.ServerPartyJoin, w.partyJoinMessage(member))
	}
	party.Members = append(party.Members, p.ID)
	p.PartyID = party.ID
	w.partyChat(party, fmt.Sprintf("%s entrou no grupo.", p.Character.Name))
	return nil
}

// LeaveParty removes the player from the party. The last member dissolves it.
func (w *World) LeaveParty(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		w.leavePartyLocked(p)
		return nil
	})
}

func (w *World) leavePartyLocked(p *Player) {
	party := w.partyByID(p.PartyID)
	if party == nil {
		p.PartyID = 0
		return
	}
	_ = p.Client.Send(protocol.ServerPartyDissolve, protocol.PartyDissolve{})
	remaining := party.Members[:0]
	for _, id := range party.Members {
		if id != p.ID {
			remaining = append(remaining, id)
		}
	}
	party.Members = remaining
	p.PartyID = 0
	if len(party.Members) == 1 {
		last := w.playerByID(party.Members[0])
		if last != nil {
			_ = last.Client.Send(protocol.ServerPartyDissolve, protocol.PartyDissolve{})
			last.PartyID = 0
		}
		delete(w.parties, party.ID)
		return
	}
	if len(party.Members) == 0 {
		delete(w.parties, party.ID)
		return
	}
	for _, id := range party.Members {
		if member := w.playerByID(id); member != nil {
			_ = member.Client.Send(protocol.ServerPartyLeave, protocol.PartyLeave{PlayerID: p.ID})
		}
	}
}

func (w *World) partyChat(party *partyState, message string) {
	msg := protocol.ChatMessage{Color: 2, Message: message}
	for _, member := range w.partyOnline(party) {
		_ = member.Client.Send(protocol.ServerChat, msg)
	}
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
