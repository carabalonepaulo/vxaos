package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/store"
)

var (
	ErrFriendFull = errors.New("game: friend list is full")
	ErrFriend     = errors.New("game: already friends")
)

// syncPlayerFriends loads the friend list on the coordinator during join.
func (w *World) syncPlayerFriends(p *Player) {
	names, err := w.opts.Store.ListFriends(context.Background(), p.Character.ID)
	if err != nil {
		w.log.Warn("load friends", "character", p.Character.ID, "err", err)
		return
	}
	online := make([]string, 0, len(names))
	offline := make([]string, 0, len(names))
	for _, name := range names {
		if friend := w.onlineByName(name); friend != nil && friend.Name != p.Character.Name {
			online = append(online, name)
		} else {
			offline = append(offline, name)
		}
	}
	p.Friends = append(online, offline...)
}

func (w *World) friendOnlineCount(entry *onlineEntry) int {
	n := 0
	for _, name := range entry.Friends {
		if friend := w.onlineByName(name); friend != nil && friend != entry {
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
		entry := w.online[p.ID]
		if entry == nil {
			return ErrNotInGame
		}
		online := make([]string, 0, len(entry.Friends))
		offline := make([]string, 0, len(entry.Friends))
		for _, name := range entry.Friends {
			if friend := w.onlineByName(name); friend != nil && friend != entry {
				online = append(online, name)
			} else {
				offline = append(offline, name)
			}
		}
		entry.Friends = append(append([]string(nil), online...), offline...)
		w.patchSocial(entry.ID, func(player *Player) {
			player.Friends = append([]string(nil), entry.Friends...)
		})
		_ = entry.Client.Send(protocol.ServerFriends, protocol.Friends{
			Online: uint8(clampInt(len(online), 0, 255)), Names: online,
		})
		return nil
	})
}

// inviteFriendEntry queues a friend request on the coordinator.
func (w *World) inviteFriendEntry(from, target *onlineEntry) error {
	if from == nil || target == nil {
		return ErrInvalidTarget
	}
	if len(from.Friends) >= w.opts.MaxFriends || len(target.Friends) >= w.opts.MaxFriends {
		return ErrFriendFull
	}
	for _, name := range from.Friends {
		if equalName(name, target.Name) {
			return ErrFriend
		}
	}
	req := &pendingRequest{Type: requestFriend, FromID: from.ID}
	target.Request = req
	w.patchSocial(target.ID, func(p *Player) { p.Request = cloneRequest(req) })
	_ = target.Client.Send(protocol.ServerRequest, protocol.PlayerRequest{
		Type: requestFriend, FromID: from.ID,
		Text: fmt.Sprintf("%s quer ser seu amigo.", from.Name),
	})
	return nil
}

// acceptFriendEntry links both characters as friends.
func (w *World) acceptFriendEntry(accepting, from *onlineEntry) error {
	if accepting == nil || from == nil {
		return ErrNotInGame
	}
	if len(accepting.Friends) >= w.opts.MaxFriends || len(from.Friends) >= w.opts.MaxFriends {
		return ErrFriendFull
	}
	for _, name := range accepting.Friends {
		if equalName(name, from.Name) {
			return ErrFriend
		}
	}
	accepting.Friends = insertFriendName(accepting.Friends, w.friendOnlineCount(accepting), from.Name)
	from.Friends = insertFriendName(from.Friends, w.friendOnlineCount(from), accepting.Name)
	if err := w.opts.Store.AddFriend(context.Background(), accepting.CharacterID, from.CharacterID); err != nil {
		return err
	}
	if err := w.opts.Store.AddFriend(context.Background(), from.CharacterID, accepting.CharacterID); err != nil {
		return err
	}
	_ = accepting.Client.Send(protocol.ServerFriendAdd, protocol.FriendAdd{Name: from.Name})
	_ = from.Client.Send(protocol.ServerFriendAdd, protocol.FriendAdd{Name: accepting.Name})
	w.patchSocial(accepting.ID, func(p *Player) { p.Friends = append([]string(nil), accepting.Friends...) })
	w.patchSocial(from.ID, func(p *Player) { p.Friends = append([]string(nil), from.Friends...) })
	return nil
}

// RemoveFriend removes one entry from the player's list by index.
func (w *World) RemoveFriend(ctx context.Context, p *Player, index uint8) error {
	return w.do(ctx, func() error {
		entry := w.online[p.ID]
		if entry == nil {
			return ErrNotInGame
		}
		i := int(index)
		if i < 0 || i >= len(entry.Friends) {
			return ErrInvalidTarget
		}
		name := entry.Friends[i]
		entry.Friends = append(entry.Friends[:i], entry.Friends[i+1:]...)
		if friend, err := w.opts.Store.GetCharacterByName(context.Background(), name); err == nil {
			_ = w.opts.Store.RemoveFriend(context.Background(), entry.CharacterID, friend.ID)
		}
		w.patchSocial(entry.ID, func(player *Player) {
			if i < len(player.Friends) {
				player.Friends = append(player.Friends[:i], player.Friends[i+1:]...)
			}
		})
		_ = entry.Client.Send(protocol.ServerFriendRemove, protocol.FriendRemove{Index: index})
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
