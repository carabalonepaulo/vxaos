package game

import (
	"context"
	"testing"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
)

func TestGlobalSwitchAndAutorun(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Miner", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	// Switch 103 is a server-wide switch (above the per-player range).
	if err := w.do(ctx, func() error {
		w.applySwitchChange(p, &rpg.EventCommand{Code: 121, Parameters: []any{103, 103, 0}})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !w.globalSwitchValue(103) {
		t.Fatal("global switch 103 was not set")
	}
	raw, err := st.GetServerState(ctx, globalSwitchKey)
	if err != nil || raw == "" {
		t.Fatalf("global switches were not persisted: %q %v", raw, err)
	}
	if len(p.Client.(*fakeClient).find(protocol.ServerSwitch)) == 0 {
		t.Fatal("switch change was not sent to the player")
	}

	// Move the player to the mining map. The rock's page 1 is an autorun
	// (trigger 4) that waits 600 frames and then clears switch 103.
	event := w.opts.Data.Maps[2].Events[13]
	var page *rpg.Page
	if err := w.do(ctx, func() error {
		page = w.globalPage(w.ensureMap(2), event)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if page == nil || page.Trigger != 4 {
		t.Fatalf("autorun page not selected: %+v", page)
	}
	if err := w.do(ctx, func() error {
		w.transferPlayer(p, 2, 4, 20, 2)
		w.tickAutoruns()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	ms := w.ensureMap(2)
	if ms.autorun == nil || ms.autorun.waitUntil.IsZero() {
		t.Fatalf("autorun did not start and wait: %+v", ms.autorun)
	}
	if !w.globalSwitchValue(103) {
		t.Fatal("switch cleared before the wait elapsed")
	}

	// Let the wait elapse: the next step clears switch 103 and ends the run.
	base := time.Now().Add(11 * time.Second)
	if err := w.do(ctx, func() error {
		w.now = func() time.Time { return base }
		w.tickAutoruns()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if w.globalSwitchValue(103) {
		t.Fatal("autorun did not clear switch 103")
	}
	if ms.autorun != nil {
		t.Fatalf("autorun did not finish: %+v", ms.autorun)
	}
}
