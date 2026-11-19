package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/store"
)

var (
	ErrFriendFull = errors.New("game: friend list is full")
	ErrFriend     = errors.New("game: already friends")
)

func (w *World) syncPlayerFriends(p *Player) {
	names, err := w.opts.Store.ListFriends(context.Background(), p.Character.ID)
	if err != nil {
		w.log.Warn("load friends", "character", p.Character.ID, "err", err)
		return
	}
	online := make([]string, 0, len(names))
	offline := make([]string, 0, len(names))
	for _, name := range names {
		if friend := w.playerByName(name); friend != nil && friend != p {
			online = append(online, name)
		} else {
			offline = append(offline, name)
		}
	}
	p.Friends = append(online, offline...)
}

func (w *World) friendOnlineCount(p *Player) int {
	n := 0
	for _, name := range p.Friends {
		if friend := w.playerByName(name); friend != nil && friend != p {
			n++
		} else {
			break
		}
	}
	return n
}

// OpenFriends reorders the list online-first and sends the online names.
func (w *World) OpenFriends(ctx context.Context, p *Player) error {
	return w.do(ctx, func() error {
		if w.players[p.ID] != p {
			return ErrNotInGame
		}
		online := make([]string, 0, len(p.Friends))
		offline := make([]string, 0, len(p.Friends))
		for _, name := range p.Friends {
			if friend := w.playerByName(name); friend != nil && friend != p {
				online = append(online, name)
			} else {
				offline = append(offline, name)
			}
		}
		p.Friends = append(append([]string(nil), online...), offline...)
		_ = p.Client.Send(protocol.ServerFriends, protocol.Friends{
			Online: uint8(clampInt(len(online), 0, 255)), Names: online,
		})
		return nil
	})
}

// inviteFriendLocked queues a friend request.
func (w *World) inviteFriendLocked(p, target *Player) error {
	if len(p.Friends) >= w.opts.MaxFriends || len(target.Friends) >= w.opts.MaxFriends {
		return ErrFriendFull
	}
	for _, name := range p.Friends {
		if equalName(name, target.Character.Name) {
			return ErrFriend
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
	target.Request = &pendingRequest{Type: requestFriend, FromID: p.ID}
	_ = target.Client.Send(protocol.ServerRequest, protocol.PlayerRequest{
		Type: requestFriend, FromID: p.ID,
		Text: fmt.Sprintf("%s quer ser seu amigo.", p.Character.Name),
	})
	return nil
}

// acceptFriendLocked links both characters as friends.
func (w *World) acceptFriendLocked(p, from *Player) error {
	if len(p.Friends) >= w.opts.MaxFriends || len(from.Friends) >= w.opts.MaxFriends {
		return ErrFriendFull
	}
	for _, name := range p.Friends {
		if equalName(name, from.Character.Name) {
			return ErrFriend
		}
	}
	p.Friends = insertFriendName(p.Friends, w.friendOnlineCount(p), from.Character.Name)
	from.Friends = insertFriendName(from.Friends, w.friendOnlineCount(from), p.Character.Name)
	if err := w.opts.Store.AddFriend(context.Background(), p.Character.ID, from.Character.ID); err != nil {
		return err
	}
	if err := w.opts.Store.AddFriend(context.Background(), from.Character.ID, p.Character.ID); err != nil {
		return err
	}
	_ = p.Client.Send(protocol.ServerFriendAdd, protocol.FriendAdd{Name: from.Character.Name})
	_ = from.Client.Send(protocol.ServerFriendAdd, protocol.FriendAdd{Name: p.Character.Name})
	return nil
}

// RemoveFriend removes one entry from the player's list by index.
func (w *World) RemoveFriend(ctx context.Context, p *Player, index uint8) error {
	return w.do(ctx, func() error {
		if w.players[p.ID] != p {
			return ErrNotInGame
		}
		i := int(index)
		if i < 0 || i >= len(p.Friends) {
			return ErrInvalidTarget
		}
		name := p.Friends[i]
		p.Friends = append(p.Friends[:i], p.Friends[i+1:]...)
		if friend, err := w.opts.Store.GetCharacterByName(context.Background(), name); err == nil {
			_ = w.opts.Store.RemoveFriend(context.Background(), p.Character.ID, friend.ID)
		}
		_ = p.Client.Send(protocol.ServerFriendRemove, protocol.FriendRemove{Index: index})
		return nil
	})
}

func equalName(a, b string) bool {
	return store.NameKey(a) == store.NameKey(b)
}

func insertFriendName(names []string, index int, name string) []string {
	if index < 0 {
		index = 0
	}
	if index > len(names) {
		index = len(names)
	}
	names = append(names, "")
	copy(names[index+1:], names[index:])
	names[index] = name
	return names
}
