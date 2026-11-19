package game

import "github.com/carabalonepaulo/vxaos/server/internal/protocol"

// Area of interest.
//
// The server is authoritative. Each player has a square window of
// 2*radius+1 tiles centered on the player. The window is shifted inward when
// it would cross a map edge, so a player next to an edge still sees the whole
// clamped camera area. Hysteresis keeps known entities in a slightly larger
// window before removing them, which avoids enter/leave flapping.

type aoiKind uint8

const (
	aoiPlayer aoiKind = 1 + iota
	aoiEnemy
	aoiEvent
	aoiDrop
)

type aoiRef struct {
	kind aoiKind
	id   uint32
}

type aoiPos struct {
	x, y int
}

type aoiView struct {
	refs                   map[aoiRef]struct{}
	minX, maxX, minY, maxY int
	cx, cy                 int
}

type aoiState struct {
	a          *mapActor
	radius     int
	hysteresis int
	cellSize   int

	views       map[uint16]*aoiView
	viewCell    map[uint16]int64
	cellViewers map[int64]map[uint16]struct{}

	entities map[aoiRef]aoiPos
	cellRefs map[int64]map[aoiRef]struct{}
}

func newAOI(a *mapActor) *aoiState {
	s := &aoiState{
		a:           a,
		radius:      a.w.opts.AOIRadius,
		hysteresis:  a.w.opts.AOIHysteresis,
		cellSize:    a.w.opts.AOICellSize,
		views:       make(map[uint16]*aoiView),
		viewCell:    make(map[uint16]int64),
		cellViewers: make(map[int64]map[uint16]struct{}),
		entities:    make(map[aoiRef]aoiPos),
		cellRefs:    make(map[int64]map[aoiRef]struct{}),
	}
	if s.cellSize <= 0 {
		s.cellSize = 8
	}
	for _, ev := range a.ms.events {
		if ev == nil || a.ms.enemies[ev.ID] != nil {
			continue
		}
		ref := aoiRef{kind: aoiEvent, id: uint32(ev.ID)}
		x, y, visible, ok := a.entityPos(ref)
		if !ok || !visible {
			continue
		}
		s.entities[ref] = aoiPos{x: x, y: y}
		s.addEntityCell(ref, x, y)
	}
	for _, enemy := range a.ms.enemies {
		if enemy == nil || !enemy.Alive {
			continue
		}
		ref := aoiRef{kind: aoiEnemy, id: uint32(enemy.Event.ID)}
		s.entities[ref] = aoiPos{x: enemy.X, y: enemy.Y}
		s.addEntityCell(ref, enemy.X, enemy.Y)
	}
	return s
}

func clampIntRange(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (s *aoiState) cell(x, y int) int64 {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return int64(x/s.cellSize)<<32 | int64(uint32(y/s.cellSize))
}

func (s *aoiState) addEntityCell(ref aoiRef, x, y int) {
	key := s.cell(x, y)
	cells := s.cellRefs[key]
	if cells == nil {
		cells = make(map[aoiRef]struct{})
		s.cellRefs[key] = cells
	}
	cells[ref] = struct{}{}
}

func (s *aoiState) removeEntityCell(ref aoiRef, pos aoiPos) {
	key := s.cell(pos.x, pos.y)
	if cells := s.cellRefs[key]; cells != nil {
		delete(cells, ref)
		if len(cells) == 0 {
			delete(s.cellRefs, key)
		}
	}
}

func (s *aoiState) putEntity(ref aoiRef, x, y int) {
	if old, ok := s.entities[ref]; ok {
		s.removeEntityCell(ref, old)
	}
	s.entities[ref] = aoiPos{x: x, y: y}
	s.addEntityCell(ref, x, y)
}

func (s *aoiState) dropEntity(ref aoiRef) {
	if old, ok := s.entities[ref]; ok {
		s.removeEntityCell(ref, old)
		delete(s.entities, ref)
	}
}

// viewBounds returns a square of 2*radius+1 tiles, shifted inward at map edges.
func (s *aoiState) viewBounds(x, y int) (minX, maxX, minY, maxY, cx, cy int) {
	width := s.a.ms.def.Width
	height := s.a.ms.def.Height
	size := s.radius * 2
	if size >= width {
		minX, maxX = 0, width-1
	} else {
		minX = clampIntRange(x-s.radius, 0, width-1-size)
		maxX = minX + size
	}
	if size >= height {
		minY, maxY = 0, height-1
	} else {
		minY = clampIntRange(y-s.radius, 0, height-1-size)
		maxY = minY + size
	}
	cx = (minX + maxX) / 2
	cy = (minY + maxY) / 2
	return
}

func (s *aoiState) inView(v *aoiView, x, y, expand int) bool {
	if v == nil {
		return false
	}
	return x >= v.minX-expand && x <= v.maxX+expand && y >= v.minY-expand && y <= v.maxY+expand
}

func (s *aoiState) addViewer(id uint16, x, y int) {
	view := &aoiView{refs: make(map[aoiRef]struct{})}
	view.minX, view.maxX, view.minY, view.maxY, view.cx, view.cy = s.viewBounds(x, y)
	s.views[id] = view
	s.indexViewer(id, view.cx, view.cy)
}

func (s *aoiState) indexViewer(id uint16, cx, cy int) {
	key := s.cell(cx, cy)
	if old, ok := s.viewCell[id]; ok {
		if old == key {
			return
		}
		if cells := s.cellViewers[old]; cells != nil {
			delete(cells, id)
			if len(cells) == 0 {
				delete(s.cellViewers, old)
			}
		}
	}
	s.viewCell[id] = key
	cells := s.cellViewers[key]
	if cells == nil {
		cells = make(map[uint16]struct{})
		s.cellViewers[key] = cells
	}
	cells[id] = struct{}{}
}

func (s *aoiState) updateViewer(id uint16, x, y int) {
	view := s.views[id]
	if view == nil {
		return
	}
	view.minX, view.maxX, view.minY, view.maxY, view.cx, view.cy = s.viewBounds(x, y)
	s.indexViewer(id, view.cx, view.cy)
}

func (s *aoiState) removeViewer(id uint16) {
	if key, ok := s.viewCell[id]; ok {
		if cells := s.cellViewers[key]; cells != nil {
			delete(cells, id)
			if len(cells) == 0 {
				delete(s.cellViewers, key)
			}
		}
	}
	delete(s.views, id)
	delete(s.viewCell, id)
}

// entityPos resolves the authoritative position/visibility of an entity.
func (a *mapActor) entityPos(ref aoiRef) (x, y int, visible bool, ok bool) {
	switch ref.kind {
	case aoiPlayer:
		p := a.playerByID(uint16(ref.id))
		if p == nil {
			return 0, 0, false, false
		}
		return p.Character.X, p.Character.Y, true, true
	case aoiEnemy:
		enemy := a.ms.enemies[int(ref.id)]
		if enemy == nil {
			return 0, 0, false, false
		}
		return enemy.X, enemy.Y, enemy.Alive, true
	case aoiEvent:
		ev := a.ms.events[int(ref.id)]
		if ev == nil {
			return 0, 0, false, false
		}
		if npc := a.ms.npcs[int(ref.id)]; npc != nil {
			return npc.x, npc.y, npc.page != nil, true
		}
		return ev.X, ev.Y, len(ev.Pages) > 0, true
	case aoiDrop:
		drop := a.ms.dropSlab.get(ref.id)
		if drop == nil {
			return 0, 0, false, false
		}
		return drop.X, drop.Y, true, true
	}
	return 0, 0, false, false
}

// candidates returns viewers whose square could intersect either position.
func (s *aoiState) candidates(oldX, oldY int, oldValid bool, newX, newY int, newValid bool) map[uint16]struct{} {
	out := make(map[uint16]struct{})
	r := s.radius + s.hysteresis
	collect := func(x, y int) {
		minCellX, maxCellX := (x-r)/s.cellSize, (x+r)/s.cellSize
		minCellY, maxCellY := (y-r)/s.cellSize, (y+r)/s.cellSize
		if minCellX < 0 {
			minCellX = 0
		}
		if minCellY < 0 {
			minCellY = 0
		}
		for cx := minCellX; cx <= maxCellX; cx++ {
			for cy := minCellY; cy <= maxCellY; cy++ {
				key := int64(cx)<<32 | int64(uint32(cy))
				for id := range s.cellViewers[key] {
					out[id] = struct{}{}
				}
			}
		}
	}
	if oldValid {
		collect(oldX, oldY)
	}
	if newValid {
		collect(newX, newY)
	}
	return out
}

func (s *aoiState) notifyViewers(ref aoiRef, oldX, oldY int, oldValid bool, newX, newY int, newVisible bool) {
	for id := range s.candidates(oldX, oldY, oldValid, newX, newY, newVisible) {
		if ref.kind == aoiPlayer && uint16(ref.id) == id {
			continue
		}
		view := s.views[id]
		viewer := s.a.playerByID(id)
		if viewer == nil || view == nil {
			continue
		}
		_, known := view.refs[ref]
		if !newVisible {
			if known {
				s.a.aoiSendRemove(viewer, ref)
				delete(view.refs, ref)
			}
			continue
		}
		if known {
			if !s.inView(view, newX, newY, s.hysteresis) {
				s.a.aoiSendRemove(viewer, ref)
				delete(view.refs, ref)
			}
			continue
		}
		if s.inView(view, newX, newY, 0) {
			s.a.aoiSendAdd(viewer, ref)
			view.refs[ref] = struct{}{}
		}
	}
}

func (s *aoiState) rebuild(id uint16, reset bool) {
	view := s.views[id]
	viewer := s.a.playerByID(id)
	if view == nil || viewer == nil {
		return
	}
	if reset {
		view.refs = make(map[aoiRef]struct{})
	} else {
		for ref := range view.refs {
			x, y, visible, ok := s.a.entityPos(ref)
			if !ok || !visible || !s.inView(view, x, y, s.hysteresis) {
				s.a.aoiSendRemove(viewer, ref)
				delete(view.refs, ref)
			}
		}
	}
	for ref, pos := range s.entities {
		if ref.kind == aoiPlayer && uint16(ref.id) == id {
			continue
		}
		if _, known := view.refs[ref]; known {
			continue
		}
		if s.inView(view, pos.x, pos.y, 0) {
			s.a.aoiSendAdd(viewer, ref)
			view.refs[ref] = struct{}{}
		}
	}
}

func (a *mapActor) aoiAttachPlayer(p *Player, reset bool) {
	if a.aoi == nil {
		a.sendLegacySnapshot(p)
		return
	}
	a.aoi.addViewer(p.ID, p.Character.X, p.Character.Y)
	ref := aoiRef{kind: aoiPlayer, id: uint32(p.ID)}
	a.aoi.putEntity(ref, p.Character.X, p.Character.Y)
	a.aoi.rebuild(p.ID, reset)
	a.aoi.notifyViewers(ref, 0, 0, false, p.Character.X, p.Character.Y, true)
}

func (a *mapActor) aoiDetachPlayer(p *Player) {
	if a.aoi == nil {
		return
	}
	ref := aoiRef{kind: aoiPlayer, id: uint32(p.ID)}
	x, y := p.Character.X, p.Character.Y
	a.aoi.notifyViewers(ref, x, y, true, 0, 0, false)
	a.aoi.removeViewer(p.ID)
	a.aoi.dropEntity(ref)
}

func (a *mapActor) aoiPlayerMoved(p *Player, oldX, oldY int) {
	a.aoiPlayerMovedInternal(p, oldX, oldY, false)
}

func (a *mapActor) aoiPlayerMovedResync(p *Player, oldX, oldY int) {
	a.aoiPlayerMovedInternal(p, oldX, oldY, true)
}

func (a *mapActor) aoiPlayerMovedInternal(p *Player, oldX, oldY int, reset bool) {
	if a.aoi == nil {
		return
	}
	ref := aoiRef{kind: aoiPlayer, id: uint32(p.ID)}
	a.aoi.updateViewer(p.ID, p.Character.X, p.Character.Y)
	a.aoi.putEntity(ref, p.Character.X, p.Character.Y)
	a.aoi.notifyViewers(ref, oldX, oldY, true, p.Character.X, p.Character.Y, true)
	a.aoi.rebuild(p.ID, reset)
}

func (a *mapActor) aoiEntityChanged(ref aoiRef, oldX, oldY int, oldVisible bool, newX, newY int, newVisible bool) {
	if a.aoi == nil {
		return
	}
	if newVisible {
		a.aoi.putEntity(ref, newX, newY)
	} else {
		a.aoi.dropEntity(ref)
	}
	a.aoi.notifyViewers(ref, oldX, oldY, oldVisible, newX, newY, newVisible)
}

func (w *World) aoiDropAdded(ms *mapState, drop *dropState) {
	if ms == nil || ms.actor == nil {
		return
	}
	a := ms.actor
	if a.aoi == nil {
		w.broadcastNear(ms, drop.X, drop.Y, protocol.ServerDropAdd, protocol.DropAdd{
			DropID: drop.ID, ItemID: uint16(clampInt(drop.ItemID, 0, 65535)),
			Kind: uint8(clampInt(drop.Kind, 0, 255)), Amount: uint16(clampInt(drop.Amount, 0, 65535)),
			X: uint16(clampInt(drop.X, 0, 65535)), Y: uint16(clampInt(drop.Y, 0, 65535)),
		})
		return
	}
	a.aoiEntityChanged(aoiRef{kind: aoiDrop, id: drop.ID}, 0, 0, false, drop.X, drop.Y, true)
}

func (w *World) aoiDropRemoved(ms *mapState, id uint32) {
	if ms == nil || ms.actor == nil {
		return
	}
	a := ms.actor
	if a.aoi == nil {
		w.broadcastNear(ms, 0, 0, protocol.ServerDropRemove, protocol.DropRemove{DropID: id})
		return
	}
	if pos, ok := a.aoi.entities[aoiRef{kind: aoiDrop, id: id}]; ok {
		a.aoiEntityChanged(aoiRef{kind: aoiDrop, id: id}, pos.x, pos.y, true, 0, 0, false)
	}
}

func (w *World) aoiSendDrops(p *Player) {
	if p == nil || p.ms == nil {
		return
	}
	if p.ms.actor == nil || p.ms.actor.aoi != nil {
		return
	}
	for _, drop := range p.ms.drops {
		_ = p.Client.Send(protocol.ServerDropAdd, protocol.DropAdd{
			DropID: drop.ID, ItemID: uint16(clampInt(drop.ItemID, 0, 65535)),
			Kind: uint8(clampInt(drop.Kind, 0, 255)), Amount: uint16(clampInt(drop.Amount, 0, 65535)),
			X: uint16(clampInt(drop.X, 0, 65535)), Y: uint16(clampInt(drop.Y, 0, 65535)),
		})
	}
}

func (a *mapActor) aoiSendAdd(viewer *Player, ref aoiRef) {
	switch ref.kind {
	case aoiPlayer:
		target := a.playerByID(uint16(ref.id))
		if target == nil {
			return
		}
		_ = viewer.Client.Send(protocol.ServerPlayerData, protocol.PlayerData{Player: a.w.snapshot(target)})
	case aoiEnemy:
		enemy := a.ms.enemies[int(ref.id)]
		if enemy == nil {
			return
		}
		_ = viewer.Client.Send(protocol.ServerEventData, protocol.EventData{
			EventID: uint16(clampInt(enemy.Event.ID, 0, 65535)), X: uint16(enemy.X), Y: uint16(enemy.Y),
			Direction: uint8(enemy.Direction), HP: int32(enemy.HP),
		})
	case aoiEvent:
		ev := a.ms.events[int(ref.id)]
		if ev == nil {
			return
		}
		if npc := a.ms.npcs[int(ref.id)]; npc != nil {
			a.w.refreshNPC(a.ms, npc)
			_ = viewer.Client.Send(protocol.ServerEventData, protocol.EventData{
				EventID: uint16(clampInt(ev.ID, 0, 65535)), X: uint16(npc.x), Y: uint16(npc.y),
				Direction: uint8(npc.direction),
			})
			return
		}
		_ = viewer.Client.Send(protocol.ServerEventData, protocol.EventData{
			EventID: uint16(clampInt(ev.ID, 0, 65535)), X: uint16(ev.X), Y: uint16(ev.Y),
			Direction: uint8(clampInt(ev.Pages[0].Graphic.Direction, 0, 255)),
		})
	case aoiDrop:
		drop := a.ms.dropSlab.get(ref.id)
		if drop == nil {
			return
		}
		_ = viewer.Client.Send(protocol.ServerDropAdd, protocol.DropAdd{
			DropID: drop.ID, ItemID: uint16(clampInt(drop.ItemID, 0, 65535)),
			Kind: uint8(clampInt(drop.Kind, 0, 255)), Amount: uint16(clampInt(drop.Amount, 0, 65535)),
			X: uint16(clampInt(drop.X, 0, 65535)), Y: uint16(clampInt(drop.Y, 0, 65535)),
		})
	}
}

func (a *mapActor) aoiSendRemove(viewer *Player, ref aoiRef) {
	switch ref.kind {
	case aoiPlayer:
		_ = viewer.Client.Send(protocol.ServerPlayerLeft, protocol.PlayerLeft{ID: uint16(ref.id)})
	case aoiEnemy, aoiEvent:
		_ = viewer.Client.Send(protocol.ServerEventRemove, protocol.EventRemove{EventID: uint16(clampInt(int(ref.id), 0, 65535))})
	case aoiDrop:
		_ = viewer.Client.Send(protocol.ServerDropRemove, protocol.DropRemove{DropID: ref.id})
	}
}

// sendLegacySnapshot preserves the pre-AOI behavior when the feature is off.
func (a *mapActor) sendLegacySnapshot(p *Player) {
	for _, other := range a.ms.players {
		if other == nil || other == p {
			continue
		}
		_ = p.Client.Send(protocol.ServerPlayerData, protocol.PlayerData{Player: a.w.snapshot(other)})
		snap := a.w.snapshot(p)
		_ = other.Client.Send(protocol.ServerPlayerData, protocol.PlayerData{Player: snap})
	}
	for _, enemy := range a.ms.enemies {
		if enemy == nil {
			continue
		}
		_ = p.Client.Send(protocol.ServerEventData, protocol.EventData{
			EventID: uint16(enemy.Event.ID), X: uint16(enemy.X), Y: uint16(enemy.Y),
			Direction: uint8(enemy.Direction), HP: int32(enemy.HP),
		})
	}
	a.w.sendMapEventsTo(p, a.ms)
	a.w.sendDropsTo(p, a.ms)
}

// broadcastNear sends a packet to players whose AOI window contains (x,y). It
// is the spatial replacement for the old map-wide broadcasts.
func (w *World) broadcastNear(ms *mapState, x, y int, op protocol.ServerOp, msg protocol.ServerMessage) {
	if ms == nil {
		return
	}
	if w.opts.AOIRadius <= 0 || ms.actor == nil || ms.actor.aoi == nil {
		for _, other := range ms.players {
			if other != nil {
				_ = other.Client.Send(op, msg)
			}
		}
		return
	}
	for _, other := range ms.players {
		if other == nil {
			continue
		}
		view := ms.actor.aoi.views[other.ID]
		if view != nil && ms.actor.aoi.inView(view, x, y, w.opts.AOIHysteresis) {
			_ = other.Client.Send(op, msg)
		}
	}
}

func (w *World) broadcastProjectile(ms *mapState, x0, y0, x1, y1 int, op protocol.ServerOp, msg protocol.ServerMessage) {
	if ms == nil {
		return
	}
	if w.opts.AOIRadius <= 0 || ms.actor == nil || ms.actor.aoi == nil {
		for _, other := range ms.players {
			if other != nil {
				_ = other.Client.Send(op, msg)
			}
		}
		return
	}
	for _, other := range ms.players {
		if other == nil {
			continue
		}
		view := ms.actor.aoi.views[other.ID]
		if view != nil && (ms.actor.aoi.inView(view, x0, y0, w.opts.AOIHysteresis) ||
			ms.actor.aoi.inView(view, x1, y1, w.opts.AOIHysteresis)) {
			_ = other.Client.Send(op, msg)
		}
	}
}

func (w *World) broadcastMapPlayerEnter(ms *mapState, p *Player) {
	if w.opts.AOIRadius <= 0 || ms == nil || p == nil {
		return
	}
	msg := protocol.MapPlayerEnter{ID: p.ID, Name: p.Character.Name}
	for _, other := range ms.players {
		if other == nil || other == p {
			continue
		}
		_ = other.Client.Send(protocol.ServerMapPlayerEnter, msg)
	}
}

func (w *World) broadcastMapPlayerLeave(ms *mapState, p *Player) {
	if w.opts.AOIRadius <= 0 || ms == nil || p == nil {
		return
	}
	msg := protocol.MapPlayerLeave{ID: p.ID, Name: p.Character.Name}
	for _, other := range ms.players {
		if other == nil || other == p {
			continue
		}
		_ = other.Client.Send(protocol.ServerMapPlayerLeave, msg)
	}
}

func (w *World) sendMapRoster(p *Player, ms *mapState) {
	if w.opts.AOIRadius <= 0 || ms == nil || p == nil {
		return
	}
	for _, other := range ms.players {
		if other == nil || other == p {
			continue
		}
		_ = p.Client.Send(protocol.ServerMapPlayerEnter, protocol.MapPlayerEnter{ID: other.ID, Name: other.Character.Name})
	}
}

// aoiEventChanged must be called after an NPC page/position update.
func (a *mapActor) aoiEventChanged(eventID int, oldX, oldY int, oldVisible bool) {
	if a.aoi == nil {
		return
	}
	npc := a.ms.npcs[eventID]
	ev := a.ms.events[eventID]
	if ev == nil {
		return
	}
	ref := aoiRef{kind: aoiEvent, id: uint32(eventID)}
	if npc == nil {
		a.aoiEntityChanged(ref, oldX, oldY, oldVisible, ev.X, ev.Y, len(ev.Pages) > 0)
		return
	}
	a.aoiEntityChanged(ref, oldX, oldY, oldVisible, npc.x, npc.y, npc.page != nil)
}

// aoiMovedEventID handles an NPC or enemy that already moved. The old position
// comes from the AOI grid, so movement call sites do not need to carry it.
func (a *mapActor) aoiMovedEventID(eventID int) {
	if a.aoi == nil {
		return
	}
	kind := aoiEvent
	if a.ms.enemies[eventID] != nil {
		kind = aoiEnemy
	}
	ref := aoiRef{kind: kind, id: uint32(eventID)}
	old, ok := a.aoi.entities[ref]
	x, y, visible, exists := a.entityPos(ref)
	if !ok && !exists {
		return
	}
	oldX, oldY := 0, 0
	if ok {
		oldX, oldY = old.x, old.y
	}
	a.aoiEntityChanged(ref, oldX, oldY, ok, x, y, visible)
}
