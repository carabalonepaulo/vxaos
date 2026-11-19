package game

import (
	"context"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/rpg"
)

func TestEventGraphicEraseAndVisualCommands(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Shapeshifter", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	list := []*rpg.EventCommand{
		{Code: 223, Parameters: []any{1, 1, 1}},
		{Code: 322, Parameters: []any{1, "Charset05", 1, "Charset05", 1}},
		{Code: 214},
	}
	if err := w.do(ctx, func() error {
		w.beginEvent(p, 99, list)
		w.runEvent(p)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if p.Character.CharacterName != "Charset05" || p.Character.CharacterIndex != 1 || p.Character.FaceName != "Charset05" {
		t.Fatalf("graphic = %+v", p.Character)
	}
	client := p.Client.(*fakeClient)
	if len(client.find(protocol.ServerGraphic)) == 0 {
		t.Fatal("graphic change was not broadcast")
	}
	if len(client.find(protocol.ServerEventCommand)) != 1 {
		t.Fatalf("visual command ranges = %d", len(client.find(protocol.ServerEventCommand)))
	}
	if !p.eventErased(99) {
		t.Fatal("erase event was not recorded")
	}
	if p.Event != nil {
		t.Fatalf("event did not stop: %+v", p.Event)
	}

	// Erased events are skipped by the action trigger and restored on a map
	// transfer.
	erased, restored := false, false
	if err := w.do(ctx, func() error {
		p.eraseEvent(18)
		ms := w.maps[p.Character.MapID]
		erased = !w.startMapEventAt(p, ms, 38, 13, []int{0, 1, 2}, true)
		p.Event = nil
		w.transferPlayer(p, 2, 4, 20, 2)
		w.transferPlayer(p, 1, 21, 12, 2)
		restored = w.startMapEventAt(p, ms, 38, 13, []int{0, 1, 2}, true)
		p.Event = nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !erased {
		t.Fatal("erased event still triggered")
	}
	if !restored {
		t.Fatal("event did not return after a map transfer")
	}
}
