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

// syncPlayerGuild restores the character's guild membership on join.
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

func (w *World) playerByName(name string) *Player {
	name = strings.TrimSpace(name)
	for _, p := range w.players {
		if p == nil {
			continue
		}
		if strings.EqualFold(p.Character.Name, name) {
			return p
		}
	}
	return nil
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

// CreateGuild creates a guild for the player who opened the create window.
func (w *World) CreateGuild(ctx context.Context, p *Player, name string, flag []byte) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p || !p.CreatingGuild {
			return ErrBadRequest
		}
		if p.GuildName != "" {
			return ErrInGuild
		}
		name = normalizeGuildName(name)
		if !validGuildName(name) || !validGuildFlag(flag) {
			return ErrBadGuild
		}
		key := store.NameKey(name)
		if w.guildByName[key] != nil {
			return ErrGuildName
		}
		row := &store.Guild{
			Name: name, LeaderCharacterID: p.Character.ID,
			Flag: append([]byte(nil), flag...), CreatedAt: w.now().Unix(),
		}
		if err := w.opts.Store.CreateGuild(context.Background(), row); err != nil {
			return err
		}
		state := &guildState{
			ID: row.ID, Name: row.Name, LeaderID: p.Character.ID,
			Flag: row.Flag, Members: map[int64]string{p.Character.ID: p.Character.Name},
			Order: []int64{p.Character.ID},
		}
		w.guilds[state.ID] = state
		w.guildByName[key] = state
		p.CreatingGuild = false
		p.Character.GuildID = state.ID
		p.GuildName = state.Name
		if err := w.opts.Store.SetCharacterGuild(context.Background(), p.Character.ID, state.ID); err != nil {
			w.log.Error("set character guild", "character", p.Character.ID, "err", err)
		}
		w.broadcastGuildName(p)
		w.questChat(p, fmt.Sprintf("Você criou a guilda %s.", state.Name))
		w.saveQuietly(p)
		return nil
	})
}

// OpenGuild sends the guild window contents.
func (w *World) OpenGuild(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		g := w.guildByName[store.NameKey(p.GuildName)]
		if g == nil {
			return ErrNoGuild
		}
		members := make([]string, 0, len(g.Members))
		seen := make(map[int64]bool, len(g.Members))
		online := 0
		for _, id := range g.Order {
			if other := w.byCharacter[id]; other != nil && other.GuildName == g.Name {
				members = append(members, other.Character.Name)
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
		_ = p.Client.Send(protocol.ServerGuildOpen, protocol.GuildOpen{
			Leader:  g.Members[g.LeaderID],
			Notice:  g.Notice,
			Flag:    g.Flag,
			Members: members,
			Online:  online,
		})
		return nil
	})
}

// SetGuildLeader transfers leadership to another member.
func (w *World) SetGuildLeader(ctx context.Context, p *Player, name string) error {
	return w.do(ctx, func() error {
		g, err := w.leaderGuild(p)
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

// SetGuildNotice changes the guild notice.
func (w *World) SetGuildNotice(ctx context.Context, p *Player, notice string) error {
	return w.do(ctx, func() error {
		g, err := w.leaderGuild(p)
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
		g, err := w.leaderGuild(p)
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
			other.Character.GuildID = 0
			w.broadcastGuildName(other)
			w.questChat(other, fmt.Sprintf("Você foi removido da guilda %s.", g.Name))
			w.saveQuietly(other)
		}
		w.broadcastGuild(g, protocol.ServerGuildMemberRemove, protocol.GuildMemberRemove{Name: memberName})
		w.guildChat(g, fmt.Sprintf("%s foi removido da guilda.", memberName))
		return nil
	})
}

// InviteGuildMember sends a guild invitation to an online player.
func (w *World) InviteGuildMember(ctx context.Context, p *Player, name string) error {
	return w.do(ctx, func() error { return w.inviteGuildLocked(p, name) })
}

func (w *World) inviteGuildLocked(p *Player, name string) error {
	g, err := w.leaderGuild(p)
	if err != nil {
		return err
	}
	target := w.playerByName(name)
	if target == nil || target == p {
		return ErrInvalidTarget
	}
	if target.GuildName != "" {
		return ErrInGuild
	}
	if len(g.Members) >= w.opts.MaxGuildMembers {
		return ErrGuildFull
	}
	target.Request = &pendingRequest{Type: requestGuild, FromID: p.ID}
	_ = target.Client.Send(protocol.ServerRequest, protocol.PlayerRequest{
		Type: requestGuild, FromID: p.ID,
		Text: fmt.Sprintf("%s te convidou para a guilda %s.", p.Character.Name, g.Name),
	})
	w.questChat(p, fmt.Sprintf("Convite enviado para %s.", target.Character.Name))
	return nil
}

// AcceptRequest accepts the pending invitation.
func (w *World) AcceptRequest(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		request := p.Request
		p.Request = nil
		if request == nil {
			return nil
		}
		switch request.Type {
		case requestParty:
			from := w.playerByID(request.FromID)
			if from == nil {
				return ErrNotInGame
			}
			return w.acceptPartyLocked(p, from)
		case requestFriend:
			from := w.playerByID(request.FromID)
			if from == nil {
				return ErrNotInGame
			}
			return w.acceptFriendLocked(p, from)
		case requestTrade:
			from := w.playerByID(request.FromID)
			if from == nil {
				return ErrNotInGame
			}
			return w.openTradeLocked(p, from)
		case requestFinishTrade:
			from := w.playerByID(request.FromID)
			if from == nil {
				return ErrNotInGame
			}
			return w.finishTradeLocked(p, from)
		case requestGuild:
		default:
			return ErrUnsupported
		}
		from := w.playerByID(request.FromID)
		if from == nil || from.GuildName == "" {
			return ErrNotInGame
		}
		g := w.guildByName[store.NameKey(from.GuildName)]
		if g == nil || g.LeaderID != from.Character.ID {
			return ErrNotLeader
		}
		if p.GuildName != "" {
			return ErrInGuild
		}
		if len(g.Members) >= w.opts.MaxGuildMembers {
			return ErrGuildFull
		}
		g.Members[p.Character.ID] = p.Character.Name
		g.Order = append(g.Order, p.Character.ID)
		p.GuildName = g.Name
		p.Character.GuildID = g.ID
		if err := w.opts.Store.SetCharacterGuild(context.Background(), p.Character.ID, g.ID); err != nil {
			return err
		}
		w.broadcastGuildName(p)
		w.guildChat(g, fmt.Sprintf("%s entrou na guilda.", p.Character.Name))
		w.questChat(p, fmt.Sprintf("Você entrou na guilda %s.", g.Name))
		w.saveQuietly(p)
		return nil
	})
}

// DeclineRequest declines the pending invitation.
func (w *World) DeclineRequest(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		request := p.Request
		p.Request = nil
		if request == nil {
			return nil
		}
		if from := w.playerByID(request.FromID); from != nil {
			switch request.Type {
			case requestParty:
				w.questChat(from, fmt.Sprintf("%s recusou entrar no grupo.", p.Character.Name))
			case requestFriend:
				w.questChat(from, fmt.Sprintf("%s recusou a amizade.", p.Character.Name))
			case requestGuild:
				w.questChat(from, fmt.Sprintf("%s recusou o convite.", p.Character.Name))
			}
		}
		return nil
	})
}

// LeaveGuild leaves the guild; a leader deletes it.
func (w *World) LeaveGuild(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		g := w.guildByName[store.NameKey(p.GuildName)]
		if g == nil {
			return ErrNoGuild
		}
		if g.LeaderID == p.Character.ID {
			return w.deleteGuild(g)
		}
		memberName := p.Character.Name
		delete(g.Members, p.Character.ID)
		for i, id := range g.Order {
			if id == p.Character.ID {
				g.Order = append(g.Order[:i], g.Order[i+1:]...)
				break
			}
		}
		p.GuildName = ""
		p.Character.GuildID = 0
		_ = w.opts.Store.SetCharacterGuild(context.Background(), p.Character.ID, 0)
		w.broadcastGuildName(p)
		w.broadcastGuild(g, protocol.ServerGuildMemberRemove, protocol.GuildMemberRemove{Name: memberName})
		w.guildChat(g, fmt.Sprintf("%s saiu da guilda.", memberName))
		w.saveQuietly(p)
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
	online := make([]*Player, 0, len(g.Members))
	for _, id := range g.Order {
		if p := w.byCharacter[id]; p != nil {
			online = append(online, p)
		}
	}
	for _, member := range online {
		member.GuildName = ""
		member.Character.GuildID = 0
		w.broadcastGuildName(member)
		w.questChat(member, fmt.Sprintf("A guilda %s foi deletada.", g.Name))
		w.saveQuietly(member)
	}
	if err := w.opts.Store.DeleteGuild(context.Background(), g.ID); err != nil {
		return err
	}
	delete(w.guildByName, store.NameKey(g.Name))
	delete(w.guilds, g.ID)
	return nil
}

func (w *World) leaderGuild(p *Player) (*guildState, error) {
	if w.playerByID(p.ID) != p || p.GuildName == "" {
		return nil, ErrNoGuild
	}
	g := w.guildByName[store.NameKey(p.GuildName)]
	if g == nil {
		return nil, ErrNoGuild
	}
	if g.LeaderID != p.Character.ID {
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

func (w *World) broadcastGuildName(p *Player) {
	w.broadcastToMap(p, protocol.ServerGuildName, protocol.GuildName{PlayerID: p.ID, Name: p.GuildName})
}

func (w *World) broadcastGuild(g *guildState, op protocol.ServerOp, msg protocol.ServerMessage) {
	for _, id := range g.Order {
		if p := w.byCharacter[id]; p != nil && p.GuildName == g.Name {
			_ = p.Client.Send(op, msg)
		}
	}
}

func (w *World) guildChat(g *guildState, message string) {
	w.broadcastGuild(g, protocol.ServerChat, protocol.ChatMessage{Color: 16, Message: message})
}
