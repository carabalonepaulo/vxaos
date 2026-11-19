package game

import (
	"context"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
)

func TestPartyFlow(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	leader := createPlayer(t, st, "Leader", 20, 12)
	ally := createPlayer(t, st, "Ally", 22, 12)
	if err := w.Join(ctx, leader); err != nil {
		t.Fatal(err)
	}
	if err := w.Join(ctx, ally); err != nil {
		t.Fatal(err)
	}
	if err := w.PlayerRequest(ctx, leader, requestParty, ally.ID); err != nil {
		t.Fatal(err)
	}
	if ally.Request == nil || ally.Request.Type != requestParty {
		t.Fatalf("pending party request = %+v", ally.Request)
	}
	if len(ally.Client.(*fakeClient).find(protocol.ServerRequest)) == 0 {
		t.Fatal("party invite was not sent")
	}
	if err := w.AcceptRequest(ctx, ally); err != nil {
		t.Fatal(err)
	}
	if leader.PartyID == 0 || leader.PartyID != ally.PartyID {
		t.Fatalf("party ids = %d/%d", leader.PartyID, ally.PartyID)
	}
	if len(leader.Client.(*fakeClient).find(protocol.ServerPartyJoin)) != 1 {
		t.Fatal("leader did not receive the member")
	}
	if len(ally.Client.(*fakeClient).find(protocol.ServerPartyJoin)) != 1 {
		t.Fatal("ally did not receive the leader")
	}

	// Party chat reaches both members.
	if err := w.Chat(ctx, leader, 2, "hello", ""); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*Player{leader, ally} {
		frames := p.Client.(*fakeClient).find(protocol.ServerChat)
		found := false
		for _, frame := range frames {
			chat := frame.(protocol.ChatMessage)
			if chat.Color == 2 && chat.Message == "Leader: hello" {
				found = true
			}
		}
		if !found {
			t.Fatalf("party chat missing for %s", p.Character.Name)
		}
	}

	// EXP is split with the party bonus (2 members => +10%).
	members := w.partyMembersOnMap(leader)
	if len(members) != 2 {
		t.Fatalf("party members on map = %d", len(members))
	}
	leader.Character.Exp, ally.Character.Exp = 0, 0
	w.sharePartyExp(leader, members, 10)
	if leader.Character.Exp != 6 || ally.Character.Exp != 6 {
		t.Fatalf("shared exp = %d/%d", leader.Character.Exp, ally.Character.Exp)
	}
	leader.Character.Gold, ally.Character.Gold = 0, 0
	w.sharePartyGold(leader, members, 10)
	if leader.Character.Gold != 6 || ally.Character.Gold != 6 {
		t.Fatalf("shared gold = %d/%d", leader.Character.Gold, ally.Character.Gold)
	}

	// Leaving with two members dissolves the party for both.
	if err := w.LeaveParty(ctx, ally); err != nil {
		t.Fatal(err)
	}
	if ally.PartyID != 0 || leader.PartyID != 0 {
		t.Fatalf("party after leave = %d/%d", leader.PartyID, ally.PartyID)
	}
	if len(ally.Client.(*fakeClient).find(protocol.ServerPartyDissolve)) == 0 {
		t.Fatal("ally did not receive dissolve")
	}
	if len(leader.Client.(*fakeClient).find(protocol.ServerPartyDissolve)) == 0 {
		t.Fatal("leader did not receive dissolve")
	}
	if len(w.parties) != 0 {
		t.Fatalf("parties left = %d", len(w.parties))
	}
	if err := w.Chat(ctx, ally, 2, "hello", ""); err != ErrNotInParty {
		t.Fatalf("party chat without party = %v", err)
	}
}
