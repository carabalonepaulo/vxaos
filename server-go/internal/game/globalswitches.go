package game

import (
	"context"
	"encoding/json"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
)

const (
	globalSwitchKey   = "global_switches"
	globalSwitchCount = 100
)

// loadGlobalSwitches restores the server-wide switches from server_state. A
// missing or invalid row yields all-off switches.
func (w *World) loadGlobalSwitches() {
	w.globalSwitches = make([]bool, globalSwitchCount)
	raw, err := w.opts.Store.GetServerState(context.Background(), globalSwitchKey)
	if err != nil {
		w.log.Warn("load global switches", "err", err)
		return
	}
	if raw == "" {
		return
	}
	var values []bool
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		w.log.Warn("decode global switches", "err", err)
		return
	}
	for i := 0; i < len(w.globalSwitches) && i < len(values); i++ {
		w.globalSwitches[i] = values[i]
	}
}

func (w *World) saveGlobalSwitches() {
	values, err := json.Marshal(w.globalSwitches)
	if err != nil {
		w.log.Error("encode global switches", "err", err)
		return
	}
	if err := w.opts.Store.SetServerState(context.Background(), globalSwitchKey, string(values)); err != nil {
		w.log.Error("save global switches", "err", err)
	}
}

// globalSwitchIndex maps a switch id to its slot. Global switch ids start
// right after the per-player range.
func (w *World) globalSwitchIndex(id int) (int, bool) {
	if id <= w.opts.MaxPlayerSwitches {
		return 0, false
	}
	index := id - w.opts.MaxPlayerSwitches - 1
	if index < 0 || index >= globalSwitchCount {
		return 0, false
	}
	return index, true
}

func (w *World) globalSwitchValue(id int) bool {
	index, ok := w.globalSwitchIndex(id)
	if !ok || index >= len(w.globalSwitches) {
		return false
	}
	return w.globalSwitches[index]
}

// setGlobalSwitch runs on the coordinator. It updates the authoritative slice,
// notifies online clients and patches every active map actor.
func (w *World) setGlobalSwitch(id int, value bool) bool {
	index, ok := w.globalSwitchIndex(id)
	if !ok {
		return false
	}
	if index >= len(w.globalSwitches) {
		grown := make([]bool, globalSwitchCount)
		copy(grown, w.globalSwitches)
		w.globalSwitches = grown
	}
	w.globalSwitches[index] = value
	msg := protocol.Switch{ID: uint16(clampInt(id, 0, 65535)), Value: value}
	for _, other := range w.online {
		_ = other.Client.Send(protocol.ServerSwitch, msg)
	}
	for key := range w.actors {
		idx := index
		w.enqueueMap(key, func(a *mapActor) {
			if len(a.ms.globalSwitches) <= idx {
				grown := make([]bool, globalSwitchCount)
				copy(grown, a.ms.globalSwitches)
				a.ms.globalSwitches = grown
			}
			a.ms.globalSwitches[idx] = value
		})
	}
	return true
}

// mapSetGlobalSwitch is called from a map actor. It updates the coordinator and
// then applies the value locally for immediate page evaluation.
func (w *World) mapSetGlobalSwitch(p *Player, id int, value bool) bool {
	index, ok := w.globalSwitchIndex(id)
	if !ok {
		return false
	}
	var applied bool
	_ = w.coordErr(func() error {
		applied = w.setGlobalSwitch(id, value)
		return nil
	})
	if p != nil && p.ms != nil {
		if len(p.ms.globalSwitches) <= index {
			grown := make([]bool, globalSwitchCount)
			copy(grown, p.ms.globalSwitches)
			p.ms.globalSwitches = grown
		}
		p.ms.globalSwitches[index] = value
	}
	return applied
}

// sendGlobalSwitches sends the full server-wide switch block to one player.
func (w *World) sendGlobalSwitches(p *Player) {
	values := make([]bool, globalSwitchCount)
	if p != nil && p.ms != nil {
		copy(values, p.ms.globalSwitches)
	}
	_ = p.Client.Send(protocol.ServerGlobalSwitches, protocol.GlobalSwitches{Values: values})
}

// mapGlobalSwitchValue reads a global switch from the map-local copy.
func (ms *mapState) globalSwitchValue(w *World, id int) bool {
	index, ok := w.globalSwitchIndex(id)
	if !ok || index >= len(ms.globalSwitches) {
		return false
	}
	return ms.globalSwitches[index]
}

// globalPage returns the highest active page whose conditions only reference
// global switches. Autorun/parallel events must not depend on per-player state.
func (w *World) globalPage(ms *mapState, event *rpg.Event) *rpg.Page {
	for i := len(event.Pages) - 1; i >= 0; i-- {
		page := event.Pages[i]
		if page == nil {
			continue
		}
		c := page.Condition
		if c.SelfSwitchValid || c.VariableValid || c.ItemValid || c.ActorValid {
			continue
		}
		if c.Switch1Valid {
			if c.Switch1ID <= w.opts.MaxPlayerSwitches {
				continue
			}
			if !ms.globalSwitchValue(w, c.Switch1ID) {
				continue
			}
		}
		if c.Switch2Valid {
			if c.Switch2ID <= w.opts.MaxPlayerSwitches {
				continue
			}
			if !ms.globalSwitchValue(w, c.Switch2ID) {
				continue
			}
		}
		return page
	}
	return nil
}
