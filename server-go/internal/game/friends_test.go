package game

import (
	"context"
	"testing"

	"github.com/kelaia/vxaos/server/internal/protocol"
)

func TestFriendFlow(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	hero := createPlayer(t, st, "Hero", 20, 12)
	ally := createPlayer(t, st, "Ally", 22, 12)
	if err := w.Join(ctx, hero); err != nil {
		t.Fatal(err)
	}
	if err := w.Join(ctx, ally); err != nil {
		t.Fatal(err)
	}
	if err := w.PlayerRequest(ctx, hero, requestFriend, ally.ID); err != nil {
		t.Fatal(err)
	}
	if ally.Request == nil || ally.Request.Type != requestFriend {
		t.Fatalf("friend request = %+v", ally.Request)
	}
	if len(ally.Client.(*fakeClient).find(protocol.ServerRequest)) == 0 {
		t.Fatal("friend request was not sent")
	}
	if err := w.AcceptRequest(ctx, ally); err != nil {
		t.Fatal(err)
	}
	if len(hero.Friends) != 1 || hero.Friends[0] != "Ally" {
		t.Fatalf("hero friends = %+v", hero.Friends)
	}
	if len(ally.Friends) != 1 || ally.Friends[0] != "Hero" {
		t.Fatalf("ally friends = %+v", ally.Friends)
	}
	if len(hero.Client.(*fakeClient).find(protocol.ServerFriendAdd)) == 0 {
		t.Fatal("friend add was not sent")
	}
	names, err := st.ListFriends(ctx, hero.Character.ID)
	if err != nil || len(names) != 1 || names[0] != "Ally" {
		t.Fatalf("stored friends = %+v, %v", names, err)
	}

	if err := w.OpenFriends(ctx, hero); err != nil {
		t.Fatal(err)
	}
	frames := hero.Client.(*fakeClient).find(protocol.ServerFriends)
	if len(frames) != 1 {
		t.Fatalf("friends frames = %d", len(frames))
	}
	friends := frames[0].(protocol.Friends)
	if friends.Online != 1 || len(friends.Names) != 1 || friends.Names[0] != "Ally" {
		t.Fatalf("friends message = %+v", friends)
	}
	if err := w.RemoveFriend(ctx, hero, 0); err != nil {
		t.Fatal(err)
	}
	if len(hero.Friends) != 0 {
		t.Fatalf("friends after remove = %+v", hero.Friends)
	}
	if len(hero.Client.(*fakeClient).find(protocol.ServerFriendRemove)) == 0 {
		t.Fatal("friend remove was not sent")
	}
	names, err = st.ListFriends(ctx, hero.Character.ID)
	if err != nil || len(names) != 0 {
		t.Fatalf("stored friends after remove = %+v, %v", names, err)
	}
}
