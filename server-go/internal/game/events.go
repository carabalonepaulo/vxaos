package game

import (
	"context"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

// eventFrame is one event or common-event command list.
type eventFrame struct {
	eventID int
	list    []*rpg.EventCommand
	index   int
	branch  map[int]bool
	choice  map[int]int
}

// eventRun is the server-side interpreter for one player interaction. The
// client renders messages and choices; the server owns branches and state.
type eventRun struct {
	frames         []*eventFrame
	waitingMessage bool
	waitingChoice  bool
	choiceIndent   int
	inputVariable  int
	waitingShop    bool
	waitingBank    bool
	waitUntil      time.Time
	queuedCommon   []int
}

func (w *World) beginEvent(p *Player, eventID int, list []*rpg.EventCommand) {
	if p.Event != nil {
		return
	}
	p.Event = &eventRun{frames: []*eventFrame{{
		eventID: eventID,
		list:    list,
		branch:  make(map[int]bool),
		choice:  make(map[int]int),
	}}}
	w.log.Debug("event start", "player", p.Character.Name, "event", eventID, "commands", len(list))
	w.runEvent(p)
}

func (w *World) startEvents(p *Player) {
	if p.Event != nil {
		return
	}
	ms := p.ms
	if ms == nil {
		return
	}
	if w.startMapEventAt(p, ms, p.Character.X, p.Character.Y, []int{0}, false) {
		return
	}
	x, y := frontTile(p.Character.X, p.Character.Y, p.Character.Direction)
	w.startMapEventAt(p, ms, x, y, []int{0, 1, 2}, true)
}

func (w *World) startMapEventAt(p *Player, ms *mapState, x, y int, triggers []int, normal bool) bool {
	for _, event := range ms.events {
		if event == nil {
			continue
		}
		npc := ms.npcs[event.ID]
		if npc == nil || npc.x != x || npc.y != y || p.eventErased(event.ID) {
			continue
		}
		page := w.properPage(p, event)
		if page == nil || !containsInt(triggers, page.Trigger) {
			continue
		}
		if (page.PriorityType == 1) != normal {
			continue
		}
		w.beginEvent(p, event.ID, page.List)
		return true
	}
	return false
}

func containsInt(values []int, value int) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func (w *World) properPage(p *Player, event *rpg.Event) *rpg.Page {
	for i := len(event.Pages) - 1; i >= 0; i-- {
		page := event.Pages[i]
		if page == nil {
			continue
		}
		if w.pageConditionsMet(p, event.ID, page.Condition) {
			return page
		}
	}
	return nil
}

func (w *World) pageConditionsMet(p *Player, eventID int, c rpg.Condition) bool {
	if c.Switch1Valid && !w.switchValue(p, c.Switch1ID) {
		return false
	}
	if c.Switch2Valid && !w.switchValue(p, c.Switch2ID) {
		return false
	}
	if c.VariableValid && w.variableValue(p, c.VariableID) < c.VariableValue {
		return false
	}
	if c.SelfSwitchValid {
		if !w.selfSwitchValue(p, p.Character.MapID, eventID, c.SelfSwitchCh) {
			return false
		}
	}
	if c.ItemValid && findInventory(p.State, KindItem, c.ItemID) < 0 {
		return false
	}
	// The server runs one actor per player, so an actor condition is met
	// whenever it refers to the playing character.
	if c.ActorValid && c.ActorID > 1 {
		return false
	}
	return true
}

func (w *World) switchValue(p *Player, id int) bool {
	if id <= 0 {
		return false
	}
	if id > w.opts.MaxPlayerSwitches {
		if p.ms != nil {
			return p.ms.globalSwitchValue(w, id)
		}
		return w.globalSwitchValue(id)
	}
	index := id - 1
	if index >= 0 && index < len(p.State.Switches) {
		return p.State.Switches[index]
	}
	return false
}

// ensureEventState gives every player the full switch/variable arrays the
// event interpreter and the client expect.
func (w *World) ensureEventState(p *Player) {
	for len(p.State.Switches) < w.opts.MaxPlayerSwitches {
		p.State.Switches = append(p.State.Switches, false)
	}
	for len(p.State.Variables) < w.opts.MaxPlayerVariables {
		p.State.Variables = append(p.State.Variables, 0)
	}
	if p.ErasedEvents == nil {
		p.ErasedEvents = make([]bool, 0)
	}
}

func (p *Player) eventErased(id int) bool {
	return id >= 0 && id < len(p.ErasedEvents) && p.ErasedEvents[id]
}

func (p *Player) eraseEvent(id int) {
	if id <= 0 {
		return
	}
	p.ErasedEvents = ensureIndex(p.ErasedEvents, id)
	p.ErasedEvents[id] = true
}

func (w *World) variableValue(p *Player, id int) int {
	if id <= 0 {
		return 0
	}
	index := id - 1
	if index >= 0 && index < len(p.State.Variables) {
		return int(p.State.Variables[index])
	}
	return 0
}

func (w *World) runEvent(p *Player) {
	for {
		for p.Event != nil && len(p.Event.frames) > 0 {
			frame := p.Event.frames[len(p.Event.frames)-1]
			if frame.index >= len(frame.list) {
				p.Event.frames = p.Event.frames[:len(p.Event.frames)-1]
				continue
			}
			cmd := frame.list[frame.index]
			if cmd == nil {
				frame.index++
				continue
			}
			stopped, waiting := w.executeEventCommand(p, p.Event, frame, cmd)
			if stopped {
				w.log.Debug("event stopped", "player", p.Character.Name)
				p.Event = nil
				return
			}
			if waiting {
				return
			}
		}
		if p.Event == nil || len(p.Event.queuedCommon) == 0 {
			break
		}
		id := p.Event.queuedCommon[0]
		p.Event.queuedCommon = p.Event.queuedCommon[1:]
		w.pushCommonEventFrame(p, id)
	}
	w.log.Debug("event end", "player", p.Character.Name)
	p.Event = nil
}

// queueCommonEvent runs a common event triggered by an item effect (command
// 355/44). If another event is already running, it is queued so the client
// command stream stays ordered.
func (w *World) queueCommonEvent(p *Player, id int) {
	if id <= 0 || id >= len(w.opts.Data.CommonEvents) || w.opts.Data.CommonEvents[id] == nil {
		return
	}
	if p.Event == nil {
		p.Event = &eventRun{frames: make([]*eventFrame, 0, 1)}
		w.pushCommonEventFrame(p, id)
		w.runEvent(p)
		return
	}
	p.Event.queuedCommon = append(p.Event.queuedCommon, id)
}

func (w *World) pushCommonEventFrame(p *Player, id int) bool {
	if p.Event == nil || id <= 0 || id >= len(w.opts.Data.CommonEvents) || w.opts.Data.CommonEvents[id] == nil {
		return false
	}
	p.Event.frames = append(p.Event.frames, &eventFrame{
		eventID: -id,
		list:    w.opts.Data.CommonEvents[id].List,
		branch:  make(map[int]bool),
		choice:  make(map[int]int),
	})
	return true
}

// executeEventCommand returns (stop, waiting).
func (w *World) executeEventCommand(p *Player, run *eventRun, frame *eventFrame, cmd *rpg.EventCommand) (bool, bool) {
	switch cmd.Code {
	case 0:
		frame.index = len(frame.list)
		return false, false
	case 101:
		start := frame.index
		frame.index++
		for frame.index < len(frame.list) && frame.list[frame.index] != nil && frame.list[frame.index].Code == 401 {
			frame.index++
		}
		if frame.index < len(frame.list) && frame.list[frame.index] != nil && frame.list[frame.index].Code == 102 {
			choice := frame.list[frame.index]
			frame.index++
			w.sendEventRange(p, frame.eventID, start, frame.index-1)
			run.waitingChoice = true
			run.choiceIndent = choice.Indent
			frame.choice[choice.Indent] = -1
			return false, true
		}
		w.sendEventRange(p, frame.eventID, start, frame.index-1)
		run.waitingMessage = true
		return false, true
	case 102:
		run.waitingChoice = true
		run.choiceIndent = cmd.Indent
		frame.choice[cmd.Indent] = -1
		w.sendEventRange(p, frame.eventID, frame.index, frame.index)
		frame.index++
		return false, true
	case 401, 405, 408, 655:
		frame.index++
		return false, false
	case 402:
		choice, _ := paramInt(cmd, 0)
		if frame.choice[cmd.Indent] != choice {
			frame.index = skipChoice(frame.list, frame.index, cmd.Indent)
		} else {
			frame.index++
		}
		return false, false
	case 404:
		frame.index++
		return false, false
	case 103, 104:
		// Number/item input is delivered through the same Choice message.
		run.waitingChoice = true
		run.choiceIndent = cmd.Indent
		run.inputVariable, _ = paramInt(cmd, 0)
		frame.choice[cmd.Indent] = -1
		w.sendEventRange(p, frame.eventID, frame.index, frame.index)
		frame.index++
		return false, true
	case 108:
		frame.index++
		return false, false
	case 111:
		result := w.evalEventCondition(p, frame, cmd)
		frame.branch[cmd.Indent] = result
		frame.index++
		if !result {
			frame.index = skipToElseOrEnd(frame.list, frame.index, cmd.Indent)
		}
		return false, false
	case 411:
		if frame.branch[cmd.Indent] {
			frame.index = skipBranchEnd(frame.list, frame.index, cmd.Indent)
		} else {
			frame.index++
		}
		return false, false
	case 412:
		frame.index++
		return false, false
	case 413:
		// The loop start is the previous 413 at the same indent (RPG Maker's
		// Repeat Above convention); if there is none, continue.
		for i := frame.index - 1; i >= 0; i-- {
			if frame.list[i] != nil && frame.list[i].Code == 413 && frame.list[i].Indent <= cmd.Indent {
				frame.index = i
				return false, false
			}
		}
		frame.index++
		return false, false
	case 115:
		return true, false
	case 117:
		id, _ := paramInt(cmd, 0)
		if id > 0 && id < len(w.opts.Data.CommonEvents) && w.opts.Data.CommonEvents[id] != nil {
			frame.index++
			run.frames = append(run.frames, &eventFrame{
				eventID: -id,
				list:    w.opts.Data.CommonEvents[id].List,
				branch:  make(map[int]bool),
				choice:  make(map[int]int),
			})
			return false, false
		}
		frame.index++
		return false, false
	case 119:
		label, _ := paramString(cmd, 0)
		for i, command := range frame.list {
			if command != nil && command.Code == 118 {
				if name, _ := paramString(command, 0); name == label {
					frame.index = i
					return false, false
				}
			}
		}
		frame.index++
		return false, false
	case 121:
		w.applySwitchChange(p, cmd)
		frame.index++
		return false, false
	case 122:
		w.applyVariableChange(p, cmd)
		frame.index++
		return false, false
	case 123:
		w.applySelfSwitchChange(p, frame, cmd)
		frame.index++
		return false, false
	case 125:
		w.applyGoldChange(p, cmd)
		frame.index++
		return false, false
	case 126, 127, 128:
		w.applyItemChange(p, cmd)
		frame.index++
		return false, false
	case 201:
		if w.transferFromEvent(p, cmd) {
			return true, false
		}
		frame.index++
		return false, false
	case 230:
		frames, _ := paramInt(cmd, 0)
		if frames <= 0 {
			frames = 1
		}
		run.waitUntil = w.now().Add(time.Duration(frames) * time.Second / 60)
		frame.index++
		return false, true
	case 302:
		if w.openShop(p, frame, cmd) {
			return false, true
		}
		return false, false
	case 311, 312:
		w.applyVitalChange(p, cmd)
		frame.index++
		return false, false
	case 315:
		w.applyExpChange(p, cmd)
		frame.index++
		return false, false
	case 318:
		w.applySkillChange(p, cmd)
		frame.index++
		return false, false
	case 355:
		start := frame.index
		script, _ := paramString(cmd, 0)
		for frame.index+1 < len(frame.list) {
			next := frame.list[frame.index+1]
			if next == nil || next.Code != 655 {
				break
			}
			frame.index++
			text, _ := paramString(next, 0)
			script += "\n" + text
		}
		endIndex := frame.index
		frame.index = endIndex + 1
		if strings.HasPrefix(strings.TrimSpace(script), "[NG]") {
			w.sendEventRange(p, frame.eventID, start, endIndex)
			return false, false
		}
		if w.runEventScript(p, frame, script) {
			if run.waitingBank {
				return false, true
			}
			return false, false
		}
		w.sendEventRange(p, frame.eventID, start, endIndex)
		return false, false
	case 212, 213, 250:
		w.sendEventRange(p, frame.eventID, frame.index, frame.index)
		frame.index++
		return false, false
	case 214:
		if frame.eventID > 0 {
			p.eraseEvent(frame.eventID)
		}
		return true, false
	case 322:
		actorID, _ := paramInt(cmd, 0)
		if actorID == 1 {
			name, _ := paramString(cmd, 1)
			index, _ := paramInt(cmd, 2)
			face, _ := paramString(cmd, 3)
			faceIndex, _ := paramInt(cmd, 4)
			p.Character.CharacterName = name
			p.Character.CharacterIndex = index
			p.Character.FaceName = face
			p.Character.FaceIndex = faceIndex
			w.broadcastToMap(p, protocol.ServerGraphic, protocol.Graphic{
				PlayerID:       p.ID,
				CharacterName:  name,
				CharacterIndex: uint8(clampInt(index, 0, 255)),
				FaceName:       face,
				FaceIndex:      uint8(clampInt(faceIndex, 0, 255)),
			})
			w.saveQuietly(p)
		}
		frame.index++
		return false, false
	default:
		if isClientVisualCommand(cmd.Code) {
			w.sendEventRange(p, frame.eventID, frame.index, frame.index)
		}
		frame.index++
		return false, false
	}
}

// isClientVisualCommand reports commands with no server-side state that the
// client can execute directly from its own event data.
func isClientVisualCommand(code int) bool {
	switch code {
	case 203, 204, 205, 206, 207, 208, 209,
		221, 222, 223, 224, 225,
		231, 232, 233, 234, 235, 236,
		241, 242, 243, 244, 245, 246, 247, 248, 249, 251:
		return true
	}
	return false
}

func (w *World) sendEventRange(p *Player, eventID, initial, final int) {
	if final < initial || initial < 0 {
		return
	}
	_ = p.Client.Send(protocol.ServerEventCommand, protocol.EventCommand{
		EventID:      int32(eventID),
		InitialIndex: uint16(clampInt(initial, 0, 65535)),
		FinalIndex:   uint16(clampInt(final, 0, 65535)),
	})
}

// EventNext resumes a message window.
func (w *World) EventNext(ctx context.Context, p *Player) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if p.Event == nil || !p.Event.waitingMessage {
			return nil
		}
		p.Event.waitingMessage = false
		w.runEvent(p)
		return nil
	})
}

// EventChoice resumes a choice or number/item input.
func (w *World) EventChoice(ctx context.Context, p *Player, index int32) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if p.Event == nil || !p.Event.waitingChoice || len(p.Event.frames) == 0 {
			return nil
		}
		frame := p.Event.frames[len(p.Event.frames)-1]
		if p.Event.inputVariable > 0 {
			id := p.Event.inputVariable
			slot := id - 1
			if slot >= 0 && slot < len(p.State.Variables) {
				value := index
				p.State.Variables[slot] = value
				_ = p.Client.Send(protocol.ServerVariable, protocol.Variable{ID: uint16(clampInt(id, 0, 65535)), Value: value})
			}
			p.Event.inputVariable = 0
		} else {
			frame.choice[p.Event.choiceIndent] = int(index)
		}
		p.Event.waitingChoice = false
		w.runEvent(p)
		return nil
	})
}

// tickEvents is the inline/test compatibility wrapper.
func (w *World) tickEvents() {
	now := w.now()
	for _, ms := range w.maps {
		if ms != nil {
			w.tickEventsMap(ms, now)
		}
	}
}

// tickAutoruns is the inline/test compatibility wrapper.
func (w *World) tickAutoruns() {
	now := w.now()
	for _, ms := range w.maps {
		if ms != nil {
			w.tickAutorunsMap(ms, now)
		}
	}
}

// tickEventsMap advances event timers for the players on this map.
func (w *World) tickEventsMap(ms *mapState, now time.Time) {
	for _, p := range ms.players {
		if p == nil {
			continue
		}
		if p.Event == nil || p.Event.waitUntil.IsZero() || now.Before(p.Event.waitUntil) {
			continue
		}
		p.Event.waitUntil = time.Time{}
		w.runEvent(p)
	}
}

// tickAutorunsMap advances map-level parallel/autorun processes. Only pages
// whose conditions are global may run here, so one runner per map is enough.
func (w *World) tickAutorunsMap(ms *mapState, now time.Time) {
	var p *Player
	for _, candidate := range ms.players {
		if candidate == nil {
			continue
		}
		p = candidate
		break
	}
	if p == nil {
		return
	}
	if ms.autorun != nil {
		if !ms.autorun.waitUntil.IsZero() && now.Before(ms.autorun.waitUntil) {
			return
		}
		ms.autorun.waitUntil = time.Time{}
		w.runAutorun(ms, p)
		return
	}
	for _, event := range ms.events {
		if event == nil {
			continue
		}
		page := w.globalPage(ms, event)
		if page == nil || (page.Trigger != 3 && page.Trigger != 4) || len(page.List) == 0 {
			continue
		}
		ms.autorun = &autorunState{
			eventID: event.ID,
			run: &eventRun{frames: []*eventFrame{{
				eventID: event.ID,
				list:    page.List,
				branch:  make(map[int]bool),
				choice:  make(map[int]int),
			}}},
		}
		break
	}
	if ms.autorun != nil {
		w.runAutorun(ms, p)
	}
}

func (w *World) runAutorun(ms *mapState, p *Player) {
	state := ms.autorun
	if state == nil || state.run == nil {
		return
	}
	run := state.run
	for len(run.frames) > 0 {
		frame := run.frames[len(run.frames)-1]
		if frame.index >= len(frame.list) {
			run.frames = run.frames[:len(run.frames)-1]
			continue
		}
		cmd := frame.list[frame.index]
		if cmd == nil {
			frame.index++
			continue
		}
		stopped, waiting := w.executeEventCommand(p, run, frame, cmd)
		if !run.waitUntil.IsZero() {
			state.waitUntil = run.waitUntil
			run.waitUntil = time.Time{}
			return
		}
		if stopped || waiting || run.waitingMessage || run.waitingChoice || run.waitingShop || run.waitingBank {
			// A map-level process cannot answer player prompts; drop it rather
			// than blocking the map forever.
			ms.autorun = nil
			return
		}
	}
	ms.autorun = nil
}

func (w *World) evalEventCondition(p *Player, frame *eventFrame, cmd *rpg.EventCommand) bool {
	kind, _ := paramInt(cmd, 0)
	switch kind {
	case 0:
		id, _ := paramInt(cmd, 1)
		value, _ := paramInt(cmd, 2)
		return w.switchValue(p, id) == (value == 0)
	case 1:
		id, _ := paramInt(cmd, 1)
		operandType, _ := paramInt(cmd, 2)
		operand, _ := paramInt(cmd, 3)
		op, _ := paramInt(cmd, 4)
		value1 := w.variableValue(p, id)
		value2 := operand
		if operandType == 1 {
			value2 = w.variableValue(p, operand)
		}
		switch op {
		case 0:
			return value1 == value2
		case 1:
			return value1 >= value2
		case 2:
			return value1 <= value2
		case 3:
			return value1 > value2
		case 4:
			return value1 < value2
		case 5:
			return value1 != value2
		}
	case 2:
		ch, _ := paramString(cmd, 1)
		value, _ := paramInt(cmd, 2)
		return w.selfSwitchValue(p, p.Character.MapID, frame.eventID, ch) == (value == 0)
	case 4:
		sub, _ := paramInt(cmd, 2)
		switch sub {
		case 0:
			return true
		case 1:
			name, _ := paramString(cmd, 3)
			return p.Character.Name == name
		case 2:
			classID, _ := paramInt(cmd, 3)
			return p.Character.ClassID == classID
		case 3:
			skillID, _ := paramInt(cmd, 3)
			return hasSkill(p.State, skillID)
		case 4:
			weaponID, _ := paramInt(cmd, 3)
			return len(p.State.Equips) > 0 && p.State.Equips[0] == weaponID
		case 5:
			armorID, _ := paramInt(cmd, 3)
			for i := 1; i < len(p.State.Equips); i++ {
				if p.State.Equips[i] == armorID {
					return true
				}
			}
			return false
		case 6:
			stateID, _ := paramInt(cmd, 3)
			return p.Status.hasState(stateID)
		}
	case 7:
		amount, _ := paramInt(cmd, 1)
		op, _ := paramInt(cmd, 2)
		switch op {
		case 0:
			return p.Character.Gold >= int64(amount)
		case 1:
			return p.Character.Gold <= int64(amount)
		case 2:
			return p.Character.Gold < int64(amount)
		}
	case 8:
		id, _ := paramInt(cmd, 1)
		return findInventory(p.State, KindItem, id) >= 0
	case 9, 10:
		kind := KindWeapon
		if kindRaw, _ := paramInt(cmd, 0); kindRaw == 10 {
			kind = KindArmor
		}
		id, _ := paramInt(cmd, 1)
		if findInventory(p.State, kind, id) >= 0 {
			return true
		}
		for _, equipped := range p.State.Equips {
			if equipped == id {
				return true
			}
		}
		return false
	case 12:
		script, _ := paramString(cmd, 1)
		return w.evalScriptCondition(p, script)
	}
	return false
}

func (w *World) evalScriptCondition(p *Player, text string) bool {
	text = strings.TrimSpace(text)
	switch {
	case strings.Contains(text, "@client.admin?"):
		return isAdmin(p)
	case strings.Contains(text, "@client.standard?"):
		return p.Account != nil && p.Account.GroupID == 0
	case strings.Contains(text, "@client.vip?"):
		return p.vipActive(w.now().Unix())
	}
	if match := questInProgressRE.FindStringSubmatch(text); match != nil {
		id, _ := strconv.Atoi(match[1])
		return w.questState(p, id-1) == 0
	}
	if match := questFinishedRE.FindStringSubmatch(text); match != nil {
		id, _ := strconv.Atoi(match[1])
		return w.questState(p, id-1) == 1
	}
	if match := questRequirementsRE.FindStringSubmatch(text); match != nil {
		id, _ := strconv.Atoi(match[1])
		id--
		index := questIndex(p, id)
		def := w.questDef(id)
		if index < 0 || def == nil || p.State.Quests[index].State != QuestInProgress {
			return false
		}
		return w.questRequirementsMet(p, &p.State.Quests[index], def)
	}
	if match := itemNumberRE.FindStringSubmatch(text); match != nil {
		kind := KindItem
		switch match[1] {
		case "weapons":
			kind = KindWeapon
		case "armors":
			kind = KindArmor
		}
		id, _ := strconv.Atoi(match[2])
		value, _ := strconv.Atoi(match[4])
		count := 0
		if index := findInventory(p.State, kind, id); index >= 0 {
			count = p.State.Inventory[index].Amount
		}
		return compareInt(count, match[3], value)
	}
	if match := sexConditionRE.FindStringSubmatch(text); match != nil {
		want := 0
		if match[1] == "FEMALE" {
			want = 1
		}
		return p.Character.Sex == want
	}
	if originalGraphicEmptyRE.MatchString(text) {
		return p.OriginalCharacterName == ""
	}
	if match := levelConditionRE.FindStringSubmatch(text); match != nil {
		value, _ := strconv.Atoi(match[2])
		return compareInt(p.Character.Level, match[1], value)
	}
	if match := goldConditionRE.FindStringSubmatch(text); match != nil {
		value, _ := strconv.Atoi(match[2])
		return compareInt(int(p.Character.Gold), match[1], value)
	}
	if match := itemConditionRE.FindStringSubmatch(text); match != nil {
		id, _ := strconv.Atoi(match[1])
		return findInventory(p.State, KindItem, id) >= 0
	}
	return false
}

func (w *World) questState(p *Player, questID int) int {
	for _, quest := range p.State.Quests {
		if quest.QuestID == questID {
			return quest.State
		}
	}
	return -1
}

func compareInt(left int, op string, right int) bool {
	switch op {
	case ">=":
		return left >= right
	case "<=":
		return left <= right
	case ">":
		return left > right
	case "<":
		return left < right
	case "==":
		return left == right
	case "!=":
		return left != right
	}
	return false
}

var (
	questInProgressRE      = regexp.MustCompile(`@client\.quest_in_progress\?\((\d+)\)`)
	questFinishedRE        = regexp.MustCompile(`@client\.quest_finished\?\((\d+)\)`)
	questRequirementsRE    = regexp.MustCompile(`@client\.finished_quest_requirements\?\((\d+)\)`)
	itemNumberRE           = regexp.MustCompile(`@client\.item_number\(\$data_(items|weapons|armors)\[(\d+)\]\)\s*(>=|<=|==|!=|>|<)\s*(\d+)`)
	sexConditionRE         = regexp.MustCompile(`@client\.sex\s*==\s*Enums::Sex::(MALE|FEMALE)`)
	originalGraphicEmptyRE = regexp.MustCompile(`@client\.original_character_name\.empty\?`)
	levelConditionRE       = regexp.MustCompile(`@client\.level\s*(>=|<=|==|!=|>|<)\s*(\d+)`)
	goldConditionRE        = regexp.MustCompile(`@client\.gold\s*(>=|<=|==|!=|>|<)\s*(\d+)`)
	itemConditionRE        = regexp.MustCompile(`@client\.has_item\?\(\$data_items\[(\d+)\]\)`)
)

func (w *World) selfSwitchValue(p *Player, mapID, eventID int, ch string) bool {
	for _, s := range p.State.SelfSwitches {
		if s.MapID == mapID && s.EventID == eventID && s.Ch == ch {
			return s.Value
		}
	}
	return false
}

func (w *World) applySwitchChange(p *Player, cmd *rpg.EventCommand) {
	start, _ := paramInt(cmd, 0)
	end, _ := paramInt(cmd, 1)
	value, _ := paramInt(cmd, 2)
	globalChanged := false
	for id := start; id <= end; id++ {
		if id <= 0 {
			continue
		}
		if id > w.opts.MaxPlayerSwitches {
			globalChanged = w.mapSetGlobalSwitch(p, id, value == 0) || globalChanged
			continue
		}
		index := id - 1
		if index >= len(p.State.Switches) {
			continue
		}
		p.State.Switches[index] = value == 0
		_ = p.Client.Send(protocol.ServerSwitch, protocol.Switch{ID: uint16(clampInt(id, 0, 65535)), Value: value == 0})
	}
	if globalChanged {
		_ = w.coordErr(func() error { w.saveGlobalSwitches(); return nil })
	}
}

func (w *World) applyVariableChange(p *Player, cmd *rpg.EventCommand) {
	start, _ := paramInt(cmd, 0)
	end, _ := paramInt(cmd, 1)
	operation, _ := paramInt(cmd, 2)
	operandType, _ := paramInt(cmd, 3)
	operand, _ := paramInt(cmd, 4)
	value := operand
	if operandType == 1 {
		value = w.variableValue(p, operand)
	} else if operandType == 2 {
		high, _ := paramInt(cmd, 5)
		if high < operand {
			high = operand
		}
		value = operand + rand.IntN(high-operand+1)
	}
	for id := start; id <= end; id++ {
		index := id - 1
		if index < 0 || index >= len(p.State.Variables) {
			continue
		}
		current := int(p.State.Variables[index])
		switch operation {
		case 0:
			current = value
		case 1:
			current += value
		case 2:
			current -= value
		case 3:
			current *= value
		case 4:
			if value != 0 {
				current /= value
			}
		case 5:
			if value != 0 {
				current %= value
			}
		}
		p.State.Variables[index] = int32(current)
		_ = p.Client.Send(protocol.ServerVariable, protocol.Variable{ID: uint16(clampInt(id, 0, 65535)), Value: int32(current)})
	}
}

func (w *World) applySelfSwitchChange(p *Player, frame *eventFrame, cmd *rpg.EventCommand) {
	ch, _ := paramString(cmd, 0)
	value, _ := paramInt(cmd, 1)
	w.setStateSelfSwitch(p, frame.eventID, ch, value == 0)
}

func (w *World) setStateSelfSwitch(p *Player, eventID int, ch string, value bool) {
	found := false
	for i := range p.State.SelfSwitches {
		s := &p.State.SelfSwitches[i]
		if s.MapID == p.Character.MapID && s.EventID == eventID && s.Ch == ch {
			s.Value = value
			found = true
			break
		}
	}
	if !found {
		p.State.SelfSwitches = append(p.State.SelfSwitches, store.SelfSwitchState{
			MapID: p.Character.MapID, EventID: eventID, Ch: ch, Value: value,
		})
	}
	_ = p.Client.Send(protocol.ServerSelfSwitch, protocol.SelfSwitch{
		MapID: uint16(clampInt(p.Character.MapID, 0, 65535)), EventID: uint16(clampInt(eventID, 0, 65535)),
		Ch: ch, Value: value,
	})
}

func (w *World) applyGoldChange(p *Player, cmd *rpg.EventCommand) {
	operation, _ := paramInt(cmd, 0)
	operandType, _ := paramInt(cmd, 1)
	operand, _ := paramInt(cmd, 2)
	value := operand
	if operandType == 1 {
		value = w.variableValue(p, operand)
	}
	if operation == 1 {
		value = -value
	}
	if value == 0 {
		return
	}
	p.Character.Gold += int64(value)
	if p.Character.Gold < 0 {
		p.Character.Gold = 0
	}
	_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: int64(value), Popup: value > 0})
}

func (w *World) applyItemChange(p *Player, cmd *rpg.EventCommand) {
	id, _ := paramInt(cmd, 0)
	operation, _ := paramInt(cmd, 1)
	operandType, _ := paramInt(cmd, 2)
	operand, _ := paramInt(cmd, 3)
	value := operand
	if operandType == 1 {
		value = w.variableValue(p, operand)
	}
	if operation == 1 {
		value = -value
	}
	kind := KindItem
	switch cmd.Code {
	case 127:
		kind = KindWeapon
	case 128:
		kind = KindArmor
	}
	if value > 0 {
		if addInventoryItem(p.State, w.opts.MaxItems, kind, id, value) == nil {
			w.sendItemTo(p, kind, id, value, true, true)
			w.addQuestItemNotice(p, kind, id)
		}
	} else if value < 0 {
		removeInventoryItem(p.State, kind, id, -value)
		w.sendItemTo(p, kind, id, value, true, false)
	}
}

func (w *World) transferFromEvent(p *Player, cmd *rpg.EventCommand) bool {
	mode, _ := paramInt(cmd, 0)
	mapID := p.Character.MapID
	if mode == 0 {
		mapID, _ = paramInt(cmd, 1)
	} else {
		mapID = w.variableValue(p, mustInt(cmd, 1))
	}
	x, y := 0, 0
	if mode == 0 {
		x, _ = paramInt(cmd, 2)
		y, _ = paramInt(cmd, 3)
	} else {
		x = w.variableValue(p, mustInt(cmd, 2))
		y = w.variableValue(p, mustInt(cmd, 3))
	}
	direction, _ := paramInt(cmd, 4)
	w.transferPlayer(p, mapID, x, y, direction)
	return true
}

// Teleport moves the player to a destination chosen in the teleport window.
func (w *World) Teleport(ctx context.Context, p *Player, index uint8) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		group := p.TeleportGroup
		selected := -1
		var destination Teleport
		for _, teleport := range w.opts.Teleports {
			if teleport.Group != group {
				continue
			}
			selected++
			if selected == int(index) {
				destination = teleport
				break
			}
		}
		if selected < int(index) {
			return nil
		}
		if p.Character.Gold < int64(destination.Gold) {
			return ErrRequirement
		}
		p.TeleportGroup = -1
		if destination.Gold > 0 {
			p.Character.Gold -= int64(destination.Gold)
			_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: -int64(destination.Gold), Popup: true})
		}
		if !w.transferPlayer(p, destination.MapID, destination.X, destination.Y, 2) {
			return ErrInvalidMove
		}
		w.saveQuietly(p)
		return nil
	})
}

// transferPlayer moves one player between maps. Inline mode keeps the same
// object graph (single goroutine); production hands off an immutable snapshot.
func (w *World) transferPlayer(p *Player, mapID, x, y, direction int) bool {
	if !w.mapTemplateValid(mapID, x, y) {
		return false
	}
	if w.opts.InlineMaps {
		return w.transferInline(p, mapID, x, y, direction)
	}
	old := p.ms
	if old == nil || p.actor == nil {
		return false
	}
	if old.key.TemplateID == mapID && old.key.InstanceID == 0 {
		p.Character.X, p.Character.Y = x, y
		if direction > 0 {
			p.Character.Direction = direction
		}
		p.TargetType, p.TargetID = 0, 0
		_ = p.Client.Send(protocol.ServerTransfer, protocol.Transfer{
			MapID: uint16(mapID), X: uint16(x), Y: uint16(y), Direction: uint8(direction),
		})
		return true
	}
	snap := p.actor.detach(p.ID, true)
	if snap == nil {
		return false
	}
	snap.Character.MapID = mapID
	snap.Character.X, snap.Character.Y = x, y
	if direction > 0 {
		snap.Character.Direction = direction
	}
	snap.TargetType, snap.TargetID = 0, 0
	snap.ErasedEvents = nil
	snap.Trade = nil
	key := MapKey{TemplateID: mapID}
	_ = w.coordErr(func() error {
		w.commitTransfer(snap, key)
		return nil
	})
	return true
}

// commitTransfer runs on the coordinator and installs the snapshot in the
// destination actor. It never waits for a map.
func (w *World) commitTransfer(snap *playerSnapshot, key MapKey) {
	entry := w.online[snap.ID]
	if entry == nil {
		return
	}
	epoch := w.allocEpoch()
	entry.Key = key
	entry.Epoch = epoch
	snap.Epoch = epoch
	snap.GuildName = entry.GuildName
	snap.PartyID = entry.PartyID
	snap.Friends = append([]string(nil), entry.Friends...)
	if tmp := snap.materialize(); tmp != nil {
		entry.View = partyViewOf(tmp)
	}
	a := w.ensureMapKey(key)
	if a == nil {
		w.attachBack(snap, entry)
		return
	}
	w.attachPlayer(a, attachPayload{
		snap:    snap,
		enter:   false,
		epoch:   epoch,
		gen:     w.nextAttachGen(key),
		motd:    w.motd,
		globals: append([]bool(nil), w.globalSwitches...),
	})
}

func (w *World) attachBack(snap *playerSnapshot, entry *onlineEntry) {
	key := entry.Key
	a := w.ensureMapKey(key)
	if a == nil {
		w.log.Error("map transfer lost player", "player", snap.ID)
		return
	}
	w.attachPlayer(a, attachPayload{
		snap: snap, enter: false, epoch: entry.Epoch,
		gen: w.nextAttachGen(key), motd: w.motd,
		globals: append([]bool(nil), w.globalSwitches...),
	})
}

func (w *World) transferInline(p *Player, mapID, x, y, direction int) bool {
	target := w.ensureMap(mapID)
	if target == nil || !target.valid(x, y) {
		return false
	}
	old := p.ms
	if old != nil && old != target {
		p.actor.detach(p.ID, false)
		p.ErasedEvents = nil
	}
	p.Character.MapID = mapID
	p.Character.X, p.Character.Y = x, y
	if direction > 0 {
		p.Character.Direction = direction
	}
	p.TargetType, p.TargetID = 0, 0
	if old != target {
		actor := w.actors[target.key]
		if actor == nil {
			return false
		}
		p.Epoch = w.allocEpoch()
		p.MapKey = target.key
		actor.putPlayer(p)
		w.online[p.ID].Key = target.key
		w.online[p.ID].Epoch = p.Epoch
	}
	_ = p.Client.Send(protocol.ServerTransfer, protocol.Transfer{
		MapID: uint16(mapID), X: uint16(x), Y: uint16(y), Direction: uint8(direction),
	})
	if old != target {
		snap := w.snapshot(p)
		for _, other := range target.players {
			if other == nil || other == p {
				continue
			}
			_ = p.Client.Send(protocol.ServerPlayerData, protocol.PlayerData{Player: w.snapshot(other)})
			_ = other.Client.Send(protocol.ServerPlayerData, protocol.PlayerData{Player: snap})
		}
		for _, enemy := range target.enemies {
			if enemy == nil {
				continue
			}
			_ = p.Client.Send(protocol.ServerEventData, protocol.EventData{
				EventID: uint16(enemy.Event.ID), X: uint16(enemy.X), Y: uint16(enemy.Y),
				Direction: uint8(enemy.Direction), HP: int32(enemy.HP),
			})
		}
		w.sendMapEventsTo(p, target)
		w.sendDropsTo(p, target)
	}
	return true
}

func (w *World) applyVitalChange(p *Player, cmd *rpg.EventCommand) {
	operation, _ := paramInt(cmd, 0)
	operandType, _ := paramInt(cmd, 1)
	operand, _ := paramInt(cmd, 2)
	value := operand
	if operandType == 1 {
		value = w.variableValue(p, operand)
	}
	if operation == 1 {
		value = -value
	}
	if cmd.Code == 311 {
		p.Character.HP += value
		if p.Character.HP < 0 {
			p.Character.HP = 0
		}
		if p.Character.HP > p.Character.MHP {
			p.Character.HP = p.Character.MHP
		}
	} else {
		p.Character.MP += value
		if p.Character.MP < 0 {
			p.Character.MP = 0
		}
		if p.Character.MP > p.Character.MMP {
			p.Character.MP = p.Character.MMP
		}
	}
	w.sendVitalsToMap(p)
}

func (w *World) applyExpChange(p *Player, cmd *rpg.EventCommand) {
	operation, _ := paramInt(cmd, 0)
	operandType, _ := paramInt(cmd, 1)
	operand, _ := paramInt(cmd, 2)
	value := operand
	if operandType == 1 {
		value = w.variableValue(p, operand)
	}
	if operation == 1 {
		value = -value
	}
	if value > 0 {
		w.gainExp(p, value)
	} else if value < 0 {
		p.Character.Exp += value
		if p.Character.Exp < 0 {
			p.Character.Exp = 0
		}
		_ = p.Client.Send(protocol.ServerExp, protocol.Exp{PlayerID: p.ID, Amount: int32(value)})
	}
}

func (w *World) applySkillChange(p *Player, cmd *rpg.EventCommand) {
	operation, _ := paramInt(cmd, 2)
	id, _ := paramInt(cmd, 3)
	if id <= 0 {
		return
	}
	if operation == 0 {
		if !hasSkill(p.State, id) {
			p.State.Skills = append(p.State.Skills, id)
		}
		_ = p.Client.Send(protocol.ServerSkill, protocol.Skill{SkillID: uint16(clampInt(id, 0, 65535)), Learned: true})
	} else {
		for i, skill := range p.State.Skills {
			if skill == id {
				p.State.Skills = append(p.State.Skills[:i], p.State.Skills[i+1:]...)
				break
			}
		}
		_ = p.Client.Send(protocol.ServerSkill, protocol.Skill{SkillID: uint16(clampInt(id, 0, 65535)), Learned: false})
	}
}

func skipChoice(list []*rpg.EventCommand, index, indent int) int {
	for i := index + 1; i < len(list); i++ {
		if list[i] == nil {
			continue
		}
		if list[i].Indent < indent {
			return i
		}
		if list[i].Indent == indent && (list[i].Code == 402 || list[i].Code == 404) {
			return i
		}
	}
	return len(list)
}

func skipToElseOrEnd(list []*rpg.EventCommand, index, indent int) int {
	for i := index; i < len(list); i++ {
		if list[i] == nil {
			continue
		}
		if list[i].Indent < indent {
			return i
		}
		if list[i].Indent == indent && (list[i].Code == 411 || list[i].Code == 412 || list[i].Code == 413) {
			return i
		}
	}
	return len(list)
}

func skipBranchEnd(list []*rpg.EventCommand, index, indent int) int {
	for i := index + 1; i < len(list); i++ {
		if list[i] == nil {
			continue
		}
		if list[i].Indent < indent {
			return i
		}
		if list[i].Indent == indent && (list[i].Code == 412 || list[i].Code == 413) {
			return i
		}
	}
	return len(list)
}

func paramInt(cmd *rpg.EventCommand, index int) (int, bool) {
	if cmd == nil || index < 0 || index >= len(cmd.Parameters) {
		return 0, false
	}
	switch v := cmd.Parameters[index].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	}
	return 0, false
}

func paramString(cmd *rpg.EventCommand, index int) (string, bool) {
	if cmd == nil || index < 0 || index >= len(cmd.Parameters) {
		return "", false
	}
	value, ok := cmd.Parameters[index].(string)
	return value, ok
}

func mustInt(cmd *rpg.EventCommand, index int) int {
	value, _ := paramInt(cmd, index)
	return value
}
