package game

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/store"
)

var (
	ErrGuildName  = errors.New("game: guild name unavailable")
	ErrGuildFull  = errors.New("game: guild is full")
	ErrNotLeader  = errors.New("game: only the guild leader can do that")
	ErrInGuild    = errors.New("game: already in a guild")
	ErrNoGuild    = errors.New("game: not in a guild")
	ErrBadGuild   = errors.New("game: invalid guild")
	ErrBadRequest = errors.New("game: invalid request")
)

const (
	requestGuild = 5
	maxGuildName = 13
	minGuildName = 3
)

// guildState is the in-memory guild shared by all players.
type guildState struct {
	ID       int64
	Name     string
	LeaderID int64
	Notice   string
	Flag     []byte
	Order    []int64
	Members  map[int64]string
}

func (w *World) loadGuilds() {
	ctx := context.Background()
	guilds, err := w.opts.Store.ListGuilds(ctx)
	if err != nil {
		w.log.Warn("load guilds", "err", err)
		return
	}
	for _, row := range guilds {
		state := &guildState{
			ID: row.ID, Name: row.Name, LeaderID: row.LeaderCharacterID,
			Notice: row.Notice, Flag: row.Flag, Members: make(map[int64]string),
		}
		members, err := w.opts.Store.GuildMembers(ctx, row.ID)
		if err != nil {
			w.log.Warn("load guild members", "guild", row.Name, "err", err)
		}
		for _, member := range members {
			state.Members[member.CharacterID] = member.Name
			state.Order = append(state.Order, member.CharacterID)
		}
		w.guilds[state.ID] = state
		w.guildByName[store.NameKey(state.Name)] = state
	}
}

// syncPlayerGuild restores the character's guild membership on join. It runs on
// the coordinator before the player is attached.
func (w *World) syncPlayerGuild(p *Player) {
	if p.Character.GuildID == 0 {
		return
	}
	g := w.guilds[p.Character.GuildID]
	if g == nil {
		p.Character.GuildID = 0
		_ = w.opts.Store.SetCharacterGuild(context.Background(), p.Character.ID, 0)
		return
	}
	p.GuildName = g.Name
	if _, ok := g.Members[p.Character.ID]; !ok {
		g.Members[p.Character.ID] = p.Character.Name
		g.Order = append(g.Order, p.Character.ID)
		_ = w.opts.Store.SetCharacterGuild(context.Background(), p.Character.ID, g.ID)
	}
}

func (w *World) entryGuild(entry *onlineEntry) *guildState {
	if entry == nil || entry.GuildName == "" {
		return nil
	}
	return w.guildByName[store.NameKey(entry.GuildName)]
}

func (g *guildState) memberByName(name string) (int64, bool) {
	for id, member := range g.Members {
		if strings.EqualFold(member, strings.TrimSpace(name)) {
			return id, true
		}
	}
	return 0, false
}

func normalizeGuildName(name string) string {
	words := strings.Fields(strings.TrimSpace(name))
	for i, word := range words {
		runes := []rune(strings.ToLower(word))
		if len(runes) > 0 {
			runes[0] = unicode.ToUpper(runes[0])
		}
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

func validGuildName(name string) bool {
	runes := []rune(name)
	if len(runes) < minGuildName || len(runes) > maxGuildName {
		return false
	}
	for _, r := range runes {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != ' ' {
			return false
		}
	}
	return true
}

func validGuildFlag(flag []byte) bool {
	if len(flag) != 64 {
		return false
	}
	colored := false
	for _, value := range flag {
		if value == 255 {
			continue
		}
		if value > 15 {
			return false
		}
		colored = true
	}
	return colored
}

// CreateGuild creates a guild for the player who opened the create window. The
// validation runs on the map; state/DB live on the coordinator.
func (w *World) CreateGuild(ctx context.Context, p *Player, name string, flag []byte) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if !p.CreatingGuild {
			return ErrBadRequest
		}
		if p.GuildName != "" {
			return ErrInGuild
		}
		name = normalizeGuildName(name)
		if !validGuildName(name) || !validGuildFlag(flag) {
			return ErrBadGuild
		}
		var state *guildState
		var err error
		_ = w.coordErr(func() error {
			state, err = w.createGuildEntry(w.online[p.ID], p.Character.ID, p.Character.Name, name, flag)
			return err
		})
		if err != nil {
			return err
		}
		p.CreatingGuild = false
		p.Character.GuildID = state.ID
		p.GuildName = state.Name
		w.broadcastGuildName(p)
		w.questChat(p, fmt.Sprintf("Você criou a guilda %s.", state.Name))
		w.saveQuietly(p)
		return nil
	})
}

func (w *World) createGuildEntry(entry *onlineEntry, charID int64, charName, name string, flag []byte) (*guildState, error) {
	if entry == nil {
		return nil, ErrNotInGame
	}
	if entry.GuildName != "" {
		return nil, ErrInGuild
	}
	key := store.NameKey(name)
	if w.guildByName[key] != nil {
		return nil, ErrGuildName
	}
	row := &store.Guild{
		Name: name, LeaderCharacterID: charID,
		Flag: append([]byte(nil), flag...), CreatedAt: w.now().Unix(),
	}
	if err := w.opts.Store.CreateGuild(context.Background(), row); err != nil {
		return nil, err
	}
	state := &guildState{
		ID: row.ID, Name: row.Name, LeaderID: charID,
		Flag: row.Flag, Members: map[int64]string{charID: charName},
		Order: []int64{charID},
	}
	w.guilds[state.ID] = state
	w.guildByName[key] = state
	entry.GuildName = state.Name
	entry.CreatingGuild = false
	if err := w.opts.Store.SetCharacterGuild(context.Background(), charID, state.ID); err != nil {
		w.log.Error("set character guild", "character", charID, "err", err)
	}
	return state, nil
}

// OpenGuild sends the guild window contents.
func (w *World) OpenGuild(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error {
		entry := w.online[p.ID]
		if entry == nil {
			return ErrNotInGame
		}
		g := w.entryGuild(entry)
		if g == nil {
			return ErrNoGuild
		}
		members := make([]string, 0, len(g.Members))
		seen := make(map[int64]bool, len(g.Members))
		online := 0
		for _, id := range g.Order {
			if other := w.byCharacter[id]; other != nil && other.GuildName == g.Name {
				members = append(members, other.Name)
				seen[id] = true
				online++
			}
		}
		for _, id := range g.Order {
			if seen[id] {
				continue
			}
			if name, ok := g.Members[id]; ok {
				members = append(members, name)
			}
		}
		_ = entry.Client.Send(protocol.ServerGuildOpen, protocol.GuildOpen{
			Leader:  g.Members[g.LeaderID],
			Notice:  g.Notice,
			Flag:    g.Flag,
			Members: members,
			Online:  online,
		})
		return nil
	})
}

func (w *World) SetGuildLeader(ctx context.Context, p *Player, name string) error {
	return w.do(ctx, func() error {
		g, err := w.leaderGuildEntry(w.online[p.ID])
		if err != nil {
			return err
		}
		id, ok := g.memberByName(name)
		if !ok || id == g.LeaderID {
			return ErrInvalidTarget
		}
		g.LeaderID = id
		if err := w.opts.Store.SaveGuild(context.Background(), w.guildRow(g)); err != nil {
			return err
		}
		w.broadcastGuild(g, protocol.ServerGuildLeader, protocol.GuildLeader{Name: g.Members[id]})
		return nil
	})
}

func (w *World) SetGuildNotice(ctx context.Context, p *Player, notice string) error {
	return w.do(ctx, func() error {
		g, err := w.leaderGuildEntry(w.online[p.ID])
		if err != nil {
			return err
		}
		notice = strings.TrimSpace(notice)
		if len([]rune(notice)) > 200 {
			return ErrBadGuild
		}
		g.Notice = notice
		if err := w.opts.Store.SaveGuild(context.Background(), w.guildRow(g)); err != nil {
			return err
		}
		w.broadcastGuild(g, protocol.ServerGuildNotice, protocol.GuildNotice{Notice: notice})
		return nil
	})
}

// KickGuildMember removes a member (online or offline) from the guild.
func (w *World) KickGuildMember(ctx context.Context, p *Player, name string) error {
	return w.do(ctx, func() error {
		g, err := w.leaderGuildEntry(w.online[p.ID])
		if err != nil {
			return err
		}
		id, ok := g.memberByName(name)
		if !ok || id == g.LeaderID {
			return ErrInvalidTarget
		}
		memberName := g.Members[id]
		delete(g.Members, id)
		for i, memberID := range g.Order {
			if memberID == id {
				g.Order = append(g.Order[:i], g.Order[i+1:]...)
				break
			}
		}
		_ = w.opts.Store.SetCharacterGuild(context.Background(), id, 0)
		if other := w.byCharacter[id]; other != nil {
			other.GuildName = ""
			w.patchSocial(other.ID, func(player *Player) {
				player.GuildName = ""
				player.Character.GuildID = 0
			})
			w.broadcastGuildNameEntry(other)
			_ = other.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: 17, Message: fmt.Sprintf("Você foi removido da guilda %s.", g.Name)})
			w.patchSave(other.ID)
		}
		w.broadcastGuild(g, protocol.ServerGuildMemberRemove, protocol.GuildMemberRemove{Name: memberName})
		w.guildChat(g, fmt.Sprintf("%s foi removido da guilda.", memberName))
		return nil
	})
}

// InviteGuildMember sends a guild invitation to an online player.
func (w *World) InviteGuildMember(ctx context.Context, p *Player, name string) error {
	return w.do(ctx, func() error {
		target := w.onlineByName(name)
		if target == nil {
			return ErrInvalidTarget
		}
		return w.inviteGuildEntry(w.online[p.ID], target.ID)
	})
}

func (w *World) inviteGuildEntry(leader *onlineEntry, targetID uint16) error {
	if leader == nil {
		return ErrNotInGame
	}
	g, err := w.leaderGuildEntry(leader)
	if err != nil {
		return err
	}
	target := w.online[targetID]
	if target == nil || target == leader {
		return ErrInvalidTarget
	}
	if target.GuildName != "" {
		return ErrInGuild
	}
	if len(g.Members) >= w.opts.MaxGuildMembers {
		return ErrGuildFull
	}
	req := &pendingRequest{Type: requestGuild, FromID: leader.ID}
	target.Request = req
	w.patchSocial(target.ID, func(p *Player) { p.Request = cloneRequest(req) })
	_ = target.Client.Send(protocol.ServerRequest, protocol.PlayerRequest{
		Type: requestGuild, FromID: leader.ID,
		Text: fmt.Sprintf("%s te convidou para a guilda %s.", leader.Name, g.Name),
	})
	return nil
}

// AcceptRequest accepts the pending invitation. Trade requests are handled by
// the map; social requests are forwarded to the coordinator.
func (w *World) AcceptRequest(ctx context.Context, p *Player) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		req := p.Request
		p.Request = nil
		if req == nil {
			return nil
		}
		switch req.Type {
		case requestTrade:
			from := ms.playerByID(req.FromID)
			if from == nil {
				return ErrNotInGame
			}
			return w.openTradeLocked(p, from)
		case requestFinishTrade:
			from := ms.playerByID(req.FromID)
			if from == nil {
				return ErrNotInGame
			}
			return w.finishTradeLocked(p, from)
		case requestParty:
			return w.coordErr(func() error {
				return w.acceptPartyEntry(w.online[p.ID], w.online[req.FromID])
			})
		case requestFriend:
			return w.coordErr(func() error {
				return w.acceptFriendEntry(w.online[p.ID], w.online[req.FromID])
			})
		case requestGuild:
			return w.coordErr(func() error {
				return w.acceptGuildEntry(w.online[p.ID], w.online[req.FromID])
			})
		default:
			return ErrUnsupported
		}
	})
}

func (w *World) acceptGuildEntry(accepting, from *onlineEntry) error {
	if accepting == nil || from == nil || from.GuildName == "" {
		return ErrNotInGame
	}
	g := w.guildByName[store.NameKey(from.GuildName)]
	if g == nil || g.LeaderID != from.CharacterID {
		return ErrNotLeader
	}
	if accepting.GuildName != "" {
		return ErrInGuild
	}
	if len(g.Members) >= w.opts.MaxGuildMembers {
		return ErrGuildFull
	}
	g.Members[accepting.CharacterID] = accepting.Name
	g.Order = append(g.Order, accepting.CharacterID)
	accepting.GuildName = g.Name
	accepting.Request = nil
	if err := w.opts.Store.SetCharacterGuild(context.Background(), accepting.CharacterID, g.ID); err != nil {
		return err
	}
	w.patchSocial(accepting.ID, func(p *Player) {
		p.GuildName = g.Name
		p.Character.GuildID = g.ID
		p.Request = nil
	})
	w.broadcastGuildNameEntry(accepting)
	w.guildChat(g, fmt.Sprintf("%s entrou na guilda.", accepting.Name))
	_ = accepting.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: 17, Message: fmt.Sprintf("Você entrou na guilda %s.", g.Name)})
	w.patchSave(accepting.ID)
	return nil
}

// DeclineRequest declines the pending invitation.
func (w *World) DeclineRequest(ctx context.Context, p *Player) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		req := p.Request
		p.Request = nil
		if req == nil {
			return nil
		}
		if from := w.online[req.FromID]; from != nil {
			switch req.Type {
			case requestParty:
				_ = from.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: 17, Message: fmt.Sprintf("%s recusou entrar no grupo.", p.Character.Name)})
			case requestFriend:
				_ = from.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: 17, Message: fmt.Sprintf("%s recusou a amizade.", p.Character.Name)})
			case requestGuild:
				_ = from.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: 17, Message: fmt.Sprintf("%s recusou o convite.", p.Character.Name)})
			}
			from.Request = nil
		}
		return nil
	})
}

// LeaveGuild leaves the guild; a leader deletes it.
func (w *World) LeaveGuild(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error {
		entry := w.online[p.ID]
		if entry == nil {
			return ErrNotInGame
		}
		g := w.entryGuild(entry)
		if g == nil {
			return ErrNoGuild
		}
		if g.LeaderID == entry.CharacterID {
			return w.deleteGuild(g)
		}
		memberName := entry.Name
		delete(g.Members, entry.CharacterID)
		for i, id := range g.Order {
			if id == entry.CharacterID {
				g.Order = append(g.Order[:i], g.Order[i+1:]...)
				break
			}
		}
		entry.GuildName = ""
		w.patchSocial(entry.ID, func(player *Player) {
			player.GuildName = ""
			player.Character.GuildID = 0
		})
		_ = w.opts.Store.SetCharacterGuild(context.Background(), entry.CharacterID, 0)
		w.broadcastGuildNameEntry(entry)
		w.broadcastGuild(g, protocol.ServerGuildMemberRemove, protocol.GuildMemberRemove{Name: memberName})
		w.guildChat(g, fmt.Sprintf("%s saiu da guilda.", memberName))
		w.patchSave(entry.ID)
		return nil
	})
}

// DeleteCharacter removes an offline character and cleans up guild membership.
func (w *World) DeleteCharacter(ctx context.Context, c *store.Character) error {
	if c == nil {
		return ErrNotInGame
	}
	return w.do(ctx, func() error {
		if w.byCharacter[c.ID] != nil {
			return ErrAlreadyOnline
		}
		if c.GuildID != 0 {
			if g := w.guilds[c.GuildID]; g != nil {
				if g.LeaderID == c.ID {
					if err := w.deleteGuild(g); err != nil {
						return err
					}
				} else {
					delete(g.Members, c.ID)
					for i, id := range g.Order {
						if id == c.ID {
							g.Order = append(g.Order[:i], g.Order[i+1:]...)
							break
						}
					}
				}
			}
		}
		return w.opts.Store.DeleteCharacter(context.Background(), c.ID)
	})
}

func (w *World) deleteGuild(g *guildState) error {
	online := make([]*onlineEntry, 0, len(g.Members))
	for _, id := range g.Order {
		if entry := w.byCharacter[id]; entry != nil {
			online = append(online, entry)
		}
	}
	for _, member := range online {
		member.GuildName = ""
		w.patchSocial(member.ID, func(p *Player) {
			p.GuildName = ""
			p.Character.GuildID = 0
		})
		w.broadcastGuildNameEntry(member)
		_ = member.Client.Send(protocol.ServerChat, protocol.ChatMessage{Color: 17, Message: fmt.Sprintf("A guilda %s foi deletada.", g.Name)})
		w.patchSave(member.ID)
	}
	if err := w.opts.Store.DeleteGuild(context.Background(), g.ID); err != nil {
		return err
	}
	delete(w.guildByName, store.NameKey(g.Name))
	delete(w.guilds, g.ID)
	return nil
}

func (w *World) leaderGuildEntry(entry *onlineEntry) (*guildState, error) {
	if entry == nil || entry.GuildName == "" {
		return nil, ErrNoGuild
	}
	g := w.guildByName[store.NameKey(entry.GuildName)]
	if g == nil {
		return nil, ErrNoGuild
	}
	if g.LeaderID != entry.CharacterID {
		return nil, ErrNotLeader
	}
	return g, nil
}

func (w *World) guildRow(g *guildState) *store.Guild {
	return &store.Guild{
		ID: g.ID, Name: g.Name, LeaderCharacterID: g.LeaderID,
		Notice: g.Notice, Flag: g.Flag,
	}
}

// broadcastGuildName is called from a map actor after a local guild change.
func (w *World) broadcastGuildName(p *Player) {
	if p == nil || p.ms == nil {
		return
	}
	w.broadcastToMapPlayers(p.ms, protocol.ServerGuildName, protocol.GuildName{PlayerID: p.ID, Name: p.GuildName})
}

// broadcastGuildNameEntry runs on the coordinator and notifies every player on
// the same map instance without touching map state.
func (w *World) broadcastGuildNameEntry(entry *onlineEntry) {
	if entry == nil {
		return
	}
	msg := protocol.GuildName{PlayerID: entry.ID, Name: entry.GuildName}
	for _, other := range w.online {
		if other.Key == entry.Key {
			_ = other.Client.Send(protocol.ServerGuildName, msg)
		}
	}
}

func (w *World) broadcastGuild(g *guildState, op protocol.ServerOp, msg protocol.ServerMessage) {
	for _, id := range g.Order {
		if entry := w.byCharacter[id]; entry != nil && entry.GuildName == g.Name {
			_ = entry.Client.Send(op, msg)
		}
	}
}

func (w *World) guildChat(g *guildState, message string) {
	w.broadcastGuild(g, protocol.ServerChat, protocol.ChatMessage{Color: 16, Message: message})
}

// patchSave asks the owning map to persist the player after a social patch.
func (w *World) patchSave(id uint16) {
	w.patchSocial(id, func(p *Player) { w.saveQuietly(p) })
}
