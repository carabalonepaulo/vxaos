package game

import (
	"math/rand/v2"
	"time"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
)

var moveDirections = []int{2, 4, 6, 8}

// refreshNPC selects the highest global page and updates its movement settings.
func (w *World) refreshNPC(ms *mapState, n *npcState) {
	page := w.globalPage(ms, n.event)
	if page == n.page {
		return
	}
	oldVisible := n.page != nil
	n.page = page
	if page == nil {
		n.moveType = 0
		if ms.actor != nil {
			ms.actor.aoiEventChanged(n.event.ID, n.x, n.y, oldVisible)
		}
		return
	}
	n.moveType = page.MoveType
	n.moveSpeed = page.MoveSpeed
	n.moveFrequency = page.MoveFrequency
	n.priority = page.PriorityType
	n.trigger = page.Trigger
	n.through = page.Through
	n.directionFix = page.DirectionFix
	n.route = page.MoveRoute
	n.routeIndex = 0
	if page.Graphic.Direction > 0 {
		n.fixedDirection = page.Graphic.Direction
		if n.directionFix {
			n.direction = n.fixedDirection
		}
	}
	if ms.actor != nil {
		ms.actor.aoiEventChanged(n.event.ID, n.x, n.y, oldVisible)
	}
}

func dirDelta(direction int) (int, int) {
	switch direction {
	case 2:
		return 0, 1
	case 4:
		return -1, 0
	case 6:
		return 1, 0
	case 8:
		return 0, -1
	}
	return 0, 0
}

// tickNPCs is the compatibility wrapper for callers already on the world
// goroutine. Map actors call tickNPCsMap directly.
func (w *World) tickNPCs(now time.Time) {
	for _, ms := range w.maps {
		if ms != nil {
			w.tickNPCsMap(ms, now)
		}
	}
}

func (w *World) tickNPCsMap(ms *mapState, now time.Time) {
	if !ms.hasPlayers() {
		return
	}
	for _, npc := range ms.npcs {
		if npc == nil {
			continue
		}
		w.refreshNPC(ms, npc)
		if npc.page == nil || npc.moveType == 0 || now.Before(npc.nextMove) || w.npcLocked(ms, npc) {
			continue
		}
		frequency := clampInt(npc.moveFrequency, 1, 5)
		npc.nextMove = now.Add(time.Duration(5-frequency) * 750 * time.Millisecond)
		switch npc.moveType {
		case 1: // random
			w.moveNPC(ms, npc, moveDirections[rand.IntN(len(moveDirections))])
		case 2: // approach
			if target := w.npcNearestPlayer(ms, npc, 10); target != nil {
				w.moveNPCToward(ms, npc, target)
			} else {
				w.moveNPC(ms, npc, moveDirections[rand.IntN(len(moveDirections))])
			}
		case 3: // custom move route
			w.runNPCMoveRoute(ms, npc, now)
		}
	}
}

// runNPCMoveRoute advances a page's custom move route (move type 3). Each
// movement tick consumes one route command, matching the per-frame stepping of
// RPG Maker's routine movement closely enough for server-driven events.
func (w *World) runNPCMoveRoute(ms *mapState, npc *npcState, now time.Time) {
	route := npc.route
	if route == nil || len(route.List) == 0 {
		return
	}
	if npc.routeIndex < 0 || npc.routeIndex >= len(route.List) {
		if !route.Repeat {
			npc.routeIndex = len(route.List)
			return
		}
		npc.routeIndex = 0
	}
	cmd := route.List[npc.routeIndex]
	npc.routeIndex++
	if npc.routeIndex >= len(route.List) {
		if route.Repeat {
			npc.routeIndex = 0
		}
	}
	switch cmd.Code {
	case 0: // end
		if !route.Repeat {
			npc.routeIndex = len(route.List)
		}
	case 1, 2, 3, 4: // straight movement
		w.moveNPC(ms, npc, []int{2, 4, 6, 8}[cmd.Code-1])
	case 5: // lower left
		w.moveNPCDiagonal(ms, npc, -1, 1)
	case 6: // lower right
		w.moveNPCDiagonal(ms, npc, 1, 1)
	case 7: // upper left
		w.moveNPCDiagonal(ms, npc, -1, -1)
	case 8: // upper right
		w.moveNPCDiagonal(ms, npc, 1, -1)
	case 9: // random
		w.moveNPC(ms, npc, moveDirections[rand.IntN(len(moveDirections))])
	case 10: // toward player
		if target := w.npcNearestPlayer(ms, npc, 20); target != nil {
			w.moveNPCToward(ms, npc, target)
		}
	case 11: // away from player
		if target := w.npcNearestPlayer(ms, npc, 20); target != nil {
			w.moveNPCAway(ms, npc, target)
		}
	case 12: // forward
		w.moveNPC(ms, npc, npc.direction)
	case 13: // backward
		w.moveNPC(ms, npc, oppositeDirection(npc.direction))
	case 14: // jump: treat as a straight step land check
		dx, dy := intParam(cmd.Parameters, 0), intParam(cmd.Parameters, 1)
		w.jumpNPC(ms, npc, dx, dy)
	case 15: // wait
		frames := intParam(cmd.Parameters, 0)
		if frames > 0 {
			npc.nextMove = now.Add(time.Duration(frames) * 16 * time.Millisecond)
		}
	case 16, 17, 18, 19: // turn
		w.faceNPC(ms, npc, []int{2, 4, 6, 8}[cmd.Code-16])
	case 20: // turn 90 right
		w.faceNPC(ms, npc, turnRight(npc.direction))
	case 21: // turn 90 left
		w.faceNPC(ms, npc, turnLeft(npc.direction))
	case 22: // turn 180
		w.faceNPC(ms, npc, oppositeDirection(npc.direction))
	case 23: // turn 90 random
		if rand.IntN(2) == 0 {
			w.faceNPC(ms, npc, turnRight(npc.direction))
		} else {
			w.faceNPC(ms, npc, turnLeft(npc.direction))
		}
	case 24: // turn random
		w.faceNPC(ms, npc, moveDirections[rand.IntN(len(moveDirections))])
	case 25: // turn toward player
		if target := w.npcNearestPlayer(ms, npc, 20); target != nil {
			w.faceNPC(ms, npc, directionTowardPoint(npc.x, npc.y, target.Character.X, target.Character.Y))
		}
	case 26: // turn away from player
		if target := w.npcNearestPlayer(ms, npc, 20); target != nil {
			w.faceNPC(ms, npc, oppositeDirection(directionTowardPoint(npc.x, npc.y, target.Character.X, target.Character.Y)))
		}
	case 27, 28: // switch on/off (global switches only)
		id := intParam(cmd.Parameters, 0)
		if id > w.opts.MaxPlayerSwitches && w.mapSetGlobalSwitch(nil, id, cmd.Code == 27) {
			_ = w.coordErr(func() error { w.saveGlobalSwitches(); return nil })
		}
	case 29: // change speed
		npc.moveSpeed = clampInt(intParam(cmd.Parameters, 0), 1, 6)
	case 30: // change frequency
		npc.moveFrequency = clampInt(intParam(cmd.Parameters, 0), 1, 5)
	case 35: // direction fix on
		npc.directionFix = true
		npc.fixedDirection = npc.direction
	case 36: // direction fix off
		npc.directionFix = false
	case 37: // through on
		npc.through = true
	case 38: // through off
		npc.through = false
	}
}

func (w *World) moveNPCDiagonal(ms *mapState, npc *npcState, dx, dy int) bool {
	nx, ny := npc.x+dx, npc.y+dy
	horiz, vert := 6, 2
	if dx < 0 {
		horiz = 4
	}
	if dy < 0 {
		vert = 8
	}
	face := npc.direction
	if npc.directionFix && npc.fixedDirection > 0 {
		face = npc.fixedDirection
	}
	ok := ms.valid(nx, ny) && ms.valid(npc.x+dx, npc.y) && ms.valid(npc.x, npc.y+dy) &&
		(npc.through || (ms.passable(npc.x+dx, npc.y, uint8(horiz)) &&
			ms.passable(npc.x, npc.y+dy, uint8(vert)) &&
			ms.passable(nx, ny, uint8(vert)) &&
			!w.npcBlocked(ms, npc.event.ID, nx, ny)))
	if ok {
		npc.x, npc.y = nx, ny
		npc.direction = face
		w.broadcastEventMove(ms, npc.event.ID, npc.x, npc.y, npc.direction)
		w.eventTouch(ms, npc)
		return true
	}
	npc.direction = face
	w.broadcastEventMove(ms, npc.event.ID, npc.x, npc.y, npc.direction)
	return false
}

func (w *World) jumpNPC(ms *mapState, npc *npcState, dx, dy int) bool {
	nx, ny := npc.x+dx, npc.y+dy
	if !ms.valid(nx, ny) || (!npc.through && (ms.blocked(nx, ny, nil) || w.npcBlocked(ms, npc.event.ID, nx, ny))) {
		return false
	}
	npc.x, npc.y = nx, ny
	w.broadcastEventMove(ms, npc.event.ID, npc.x, npc.y, npc.direction)
	w.eventTouch(ms, npc)
	return true
}

func (w *World) moveNPCAway(ms *mapState, npc *npcState, target *Player) bool {
	dx := npc.x - target.Character.X
	dy := npc.y - target.Character.Y
	primary, secondary := axisDirections(dx, dy)
	if w.moveNPC(ms, npc, primary) {
		return true
	}
	if secondary != 0 {
		return w.moveNPC(ms, npc, secondary)
	}
	return false
}

func (w *World) faceNPC(ms *mapState, npc *npcState, direction int) {
	if direction == 0 {
		return
	}
	if npc.directionFix && npc.fixedDirection > 0 {
		direction = npc.fixedDirection
	}
	npc.direction = direction
	w.broadcastEventMove(ms, npc.event.ID, npc.x, npc.y, npc.direction)
}

func oppositeDirection(direction int) int {
	if direction == 0 {
		return 0
	}
	return 10 - direction
}

func turnRight(direction int) int {
	switch direction {
	case 2:
		return 4
	case 4:
		return 8
	case 8:
		return 6
	case 6:
		return 2
	}
	return direction
}

func turnLeft(direction int) int {
	switch direction {
	case 2:
		return 6
	case 6:
		return 8
	case 8:
		return 4
	case 4:
		return 2
	}
	return direction
}

func directionTowardPoint(x, y, tx, ty int) int {
	dx := tx - x
	dy := ty - y
	if dx == 0 && dy == 0 {
		return 0
	}
	if absInt(dx) >= absInt(dy) {
		if dx > 0 {
			return 6
		}
		return 4
	}
	if dy > 0 {
		return 2
	}
	return 8
}

func intParam(params []any, index int) int {
	if index < 0 || index >= len(params) {
		return 0
	}
	switch value := params[index].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float64:
		return int(value)
	}
	return 0
}

func (w *World) npcLocked(ms *mapState, npc *npcState) bool {
	for _, p := range ms.players {
		if p == nil || p.Event == nil {
			continue
		}
		for _, frame := range p.Event.frames {
			if frame.eventID == npc.event.ID {
				return true
			}
		}
	}
	return false
}

func (w *World) npcNearestPlayer(ms *mapState, npc *npcState, sight int) *Player {
	var best *Player
	bestDistance := sight + 1
	for _, p := range ms.players {
		if p == nil {
			continue
		}
		if p.Character.HP <= 0 {
			continue
		}
		dx := absInt(p.Character.X - npc.x)
		dy := absInt(p.Character.Y - npc.y)
		if dx > sight || dy > sight {
			continue
		}
		if dx+dy < bestDistance {
			bestDistance = dx + dy
			best = p
		}
	}
	return best
}

func (w *World) moveNPCToward(ms *mapState, npc *npcState, target *Player) bool {
	dx := target.Character.X - npc.x
	dy := target.Character.Y - npc.y
	primary, secondary := axisDirections(dx, dy)
	if w.moveNPC(ms, npc, primary) {
		return true
	}
	if secondary != 0 {
		return w.moveNPC(ms, npc, secondary)
	}
	return false
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (w *World) moveNPC(ms *mapState, npc *npcState, direction int) bool {
	dx, dy := dirDelta(direction)
	nx, ny := npc.x+dx, npc.y+dy
	face := direction
	if npc.directionFix && npc.fixedDirection > 0 {
		face = npc.fixedDirection
	}
	if ms.valid(nx, ny) && (npc.through || (ms.passable(nx, ny, uint8(direction)) && !w.npcBlocked(ms, npc.event.ID, nx, ny))) {
		npc.x, npc.y = nx, ny
		npc.direction = face
		w.broadcastEventMove(ms, npc.event.ID, npc.x, npc.y, npc.direction)
		w.eventTouch(ms, npc)
		return true
	}
	npc.direction = face
	w.broadcastEventMove(ms, npc.event.ID, npc.x, npc.y, npc.direction)
	return false
}

// npcBlocked reports whether a tile is occupied by a visible player, enemy or
// event other than the mover.
func (w *World) npcBlocked(ms *mapState, moverID, x, y int) bool {
	for _, p := range ms.players {
		if p != nil && p.Character.X == x && p.Character.Y == y {
			return true
		}
	}
	for _, enemy := range ms.enemies {
		if enemy != nil && enemy.Alive && enemy.Event.ID != moverID && enemy.X == x && enemy.Y == y {
			return true
		}
	}
	for _, npc := range ms.npcs {
		if npc == nil || npc.event.ID == moverID {
			continue
		}
		if npc.priority >= 1 && !npc.through && npc.x == x && npc.y == y {
			return true
		}
	}
	return false
}

// eventTouch starts a trigger-2 page when an NPC moves into a player.
func (w *World) eventTouch(ms *mapState, npc *npcState) {
	for _, p := range ms.players {
		if p == nil || p.Event != nil || p.Character.X != npc.x || p.Character.Y != npc.y {
			continue
		}
		page := w.properPage(p, npc.event)
		if page != nil && page.Trigger == 2 {
			w.beginEvent(p, npc.event.ID, page.List)
			return
		}
	}
}

// checkPlayerTouch starts trigger-1 pages after a player step or bump.
func (w *World) checkPlayerTouch(p *Player, ms *mapState, x, y int, bumped bool) {
	if p.Event != nil {
		return
	}
	for _, npc := range ms.npcs {
		if npc == nil || npc.x != x || npc.y != y {
			continue
		}
		page := w.properPage(p, npc.event)
		if page == nil || page.Trigger != 1 {
			continue
		}
		if (!bumped && page.PriorityType == 0) || (bumped && page.PriorityType == 1) {
			w.beginEvent(p, npc.event.ID, page.List)
			return
		}
	}
}

func (w *World) broadcastEventMove(ms *mapState, eventID, x, y, direction int) {
	if ms != nil && ms.actor != nil {
		ms.actor.aoiMovedEventID(eventID)
	}
	w.broadcastNear(ms, x, y, protocol.ServerEventMove, protocol.EventMove{
		EventID:   uint16(clampInt(eventID, 0, 65535)),
		X:         uint16(clampInt(x, 0, 65535)),
		Y:         uint16(clampInt(y, 0, 65535)),
		Direction: uint8(clampInt(direction, 0, 255)),
	})
}

// sendMapEventsTo syncs every NPC position to one client after a map entry.
func (w *World) sendMapEventsTo(p *Player, ms *mapState) {
	for _, npc := range ms.npcs {
		if npc == nil {
			continue
		}
		w.refreshNPC(ms, npc)
		_ = p.Client.Send(protocol.ServerEventData, protocol.EventData{
			EventID:   uint16(clampInt(npc.event.ID, 0, 65535)),
			X:         uint16(clampInt(npc.x, 0, 65535)),
			Y:         uint16(clampInt(npc.y, 0, 65535)),
			Direction: uint8(clampInt(npc.direction, 0, 255)),
		})
	}
}
