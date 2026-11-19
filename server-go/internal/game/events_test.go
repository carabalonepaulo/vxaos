package game

import (
	"context"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/rpg"
)

func TestEventShowText(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Talker", 37, 13)
	p.Character.Direction = 6
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	w.startEvents(p)
	if p.Event == nil || !p.Event.waitingMessage {
		t.Fatalf("event did not wait for text: %+v", p.Event)
	}
	if len(p.Client.(*fakeClient).find(protocol.ServerEventCommand)) == 0 {
		t.Fatal("no EventCommand sent")
	}
	if err := w.EventNext(ctx, p); err != nil {
		t.Fatal(err)
	}
	if p.Event != nil {
		t.Fatalf("event did not finish: %+v", p.Event)
	}
}

func TestEventScriptNatives(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Scripter", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	ok := w.runEventScript(p, &eventFrame{eventID: 1}, `(begin (start-quest 1) (check-point 2 5 6) (chat-add "hello" 5))`)
	if !ok {
		t.Fatal("script failed")
	}
	if len(p.State.Quests) != 1 || p.State.Quests[0].QuestID != 0 || p.State.Quests[0].State != 0 {
		t.Fatalf("quests = %+v", p.State.Quests)
	}
	if p.Character.ReviveMapID != 2 || p.Character.ReviveX != 5 || p.Character.ReviveY != 6 {
		t.Fatalf("revive = %+v", p.Character)
	}
	client := p.Client.(*fakeClient)
	if len(client.find(protocol.ServerQuestAdd)) == 0 || len(client.find(protocol.ServerChat)) == 0 {
		t.Fatal("script messages missing")
	}
}

func TestEventChoiceBranch(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Chooser", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	list := []*rpg.EventCommand{
		{Code: 101, Indent: 0},
		{Code: 401, Indent: 1},
		{Code: 102, Indent: 0},
		{Code: 402, Indent: 0, Parameters: []any{0}},
		{Code: 101, Indent: 1},
		{Code: 401, Indent: 2},
		{Code: 402, Indent: 0, Parameters: []any{1}},
		{Code: 101, Indent: 1},
		{Code: 401, Indent: 2},
		{Code: 404, Indent: 0},
		{Code: 0, Indent: 0},
	}
	w.beginEvent(p, 1, list)
	if p.Event == nil || !p.Event.waitingChoice {
		t.Fatalf("event did not wait for choice: %+v", p.Event)
	}
	client := p.Client.(*fakeClient)
	before := len(client.find(protocol.ServerEventCommand))
	if err := w.EventChoice(ctx, p, 1); err != nil {
		t.Fatal(err)
	}
	if len(client.find(protocol.ServerEventCommand)) <= before {
		t.Fatal("choice branch did not send another command range")
	}
	if p.Event == nil || !p.Event.waitingMessage {
		t.Fatalf("choice branch did not wait for text: %+v", p.Event)
	}
	if err := w.EventNext(ctx, p); err != nil {
		t.Fatal(err)
	}
	if p.Event != nil {
		t.Fatalf("event did not finish: %+v", p.Event)
	}
}
