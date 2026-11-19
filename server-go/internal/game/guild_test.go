package game

import (
	"context"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
)

func TestGuildFlow(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	leader := createPlayer(t, st, "Leader", 20, 12)
	member := createPlayer(t, st, "Member", 22, 12)
	if err := w.Join(ctx, leader); err != nil {
		t.Fatal(err)
	}
	if err := w.Join(ctx, member); err != nil {
		t.Fatal(err)
	}
	flag := make([]byte, 64)
	for i := range flag {
		flag[i] = 255
	}
	flag[0] = 3
	if err := w.do(ctx, func() error {
		leader.CreatingGuild = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := w.CreateGuild(ctx, leader, "  test   guild ", flag); err != nil {
		t.Fatal(err)
	}
	if leader.GuildName != "Test Guild" || leader.Character.GuildID == 0 {
		t.Fatalf("leader guild = %q id=%d", leader.GuildName, leader.Character.GuildID)
	}
	if len(leader.Client.(*fakeClient).find(protocol.ServerGuildName)) == 0 {
		t.Fatal("guild name was not broadcast")
	}
	guilds, err := st.ListGuilds(ctx)
	if err != nil || len(guilds) != 1 || guilds[0].Name != "Test Guild" {
		t.Fatalf("guilds = %+v, %v", guilds, err)
	}
	if err := w.CreateGuild(ctx, leader, "Another", flag); err != ErrInGuild && err != ErrBadRequest {
		t.Fatalf("second guild = %v", err)
	}

	// Invite the second player and accept through the request flow.
	if err := w.InviteGuildMember(ctx, leader, "Member"); err != nil {
		t.Fatal(err)
	}
	if member.Request == nil || member.Request.Type != requestGuild {
		t.Fatalf("pending request = %+v", member.Request)
	}
	if len(member.Client.(*fakeClient).find(protocol.ServerRequest)) == 0 {
		t.Fatal("invite was not sent")
	}
	if err := w.AcceptRequest(ctx, member); err != nil {
		t.Fatal(err)
	}
	if member.GuildName != "Test Guild" {
		t.Fatalf("member guild = %q", member.GuildName)
	}
	if err := w.OpenGuild(ctx, leader); err != nil {
		t.Fatal(err)
	}
	frames := leader.Client.(*fakeClient).find(protocol.ServerGuildOpen)
	if len(frames) != 1 {
		t.Fatalf("guild open frames = %d", len(frames))
	}
	open := frames[0].(protocol.GuildOpen)
	if open.Online != 2 || len(open.Members) != 2 || open.Leader != "Leader" {
		t.Fatalf("guild open = %+v", open)
	}

	if err := w.SetGuildNotice(ctx, leader, "Bem-vindos"); err != nil {
		t.Fatal(err)
	}
	if len(member.Client.(*fakeClient).find(protocol.ServerGuildNotice)) == 0 {
		t.Fatal("notice was not broadcast")
	}
	if err := w.SetGuildLeader(ctx, leader, "Member"); err != nil {
		t.Fatal(err)
	}
	if len(leader.Client.(*fakeClient).find(protocol.ServerGuildLeader)) == 0 {
		t.Fatal("leader change was not broadcast")
	}
	if err := w.LeaveGuild(ctx, leader); err != nil {
		t.Fatal(err)
	}
	if leader.GuildName != "" {
		t.Fatalf("leader guild after leave = %q", leader.GuildName)
	}
	if err := w.LeaveGuild(ctx, member); err != nil {
		t.Fatal(err)
	}
	if member.GuildName != "" {
		t.Fatalf("member guild after delete = %q", member.GuildName)
	}
	guilds, err = st.ListGuilds(ctx)
	if err != nil || len(guilds) != 0 {
		t.Fatalf("guilds after delete = %+v, %v", guilds, err)
	}
}
