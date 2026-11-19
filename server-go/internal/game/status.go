package game

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
)

// buffSlot is one parameter buff. Level is -2..2, Turns is in seconds.
type buffSlot struct {
	Level int
	Turns int
}

// statusSet is the transient state/buff container of a battler. Timed states
// and buffs tick once per second while the world runs. States is indexed by
// state id; 0 means absent, otherwise the stored value is turns+1. This keeps
// the RPG Maker id-as-index convention and avoids hashing on every hit.
type statusSet struct {
	States     []int
	Buffs      [8]buffSlot
	LastSecond int64
}

func (s *statusSet) ensure(states int) {
	if len(s.States) < states {
		grown := make([]int, states)
		copy(grown, s.States)
		s.States = grown
	}
}

func (s *statusSet) hasState(id int) bool {
	return id > 0 && id < len(s.States) && s.States[id] != 0
}

func (s *statusSet) stateTurns(id int) int {
	if !s.hasState(id) {
		return 0
	}
	return s.States[id] - 1
}

func (s *statusSet) any() bool {
	for _, turns := range s.States {
		if turns != 0 {
			return true
		}
	}
	for _, b := range s.Buffs {
		if b.Level != 0 {
			return true
		}
	}
	return false
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (w *World) stateDef(id int) *rpg.State {
	if id <= 0 || id >= len(w.opts.Data.States) {
		return nil
	}
	return w.opts.Data.States[id]
}

// playerFeatures gathers class, equipment and active-state features.
func (w *World) playerFeatures(p *Player) []rpg.Feature {
	var out []rpg.Feature
	if class := w.class(p.Character.ClassID); class != nil {
		out = append(out, class.Features...)
	}
	for slot, id := range p.State.Equips {
		if id <= 0 {
			continue
		}
		if slot == 0 {
			if weapon := w.weapon(id); weapon != nil {
				out = append(out, weapon.Features...)
			}
		} else if armor := w.armor(id); armor != nil {
			out = append(out, armor.Features...)
		}
	}
	return append(out, w.stateFeatures(p.Status.States)...)
}

func (w *World) enemyFeatures(enemy *enemyState) []rpg.Feature {
	if enemy == nil || enemy.Enemy == nil {
		return nil
	}
	out := append([]rpg.Feature(nil), enemy.Enemy.Features...)
	return append(out, w.stateFeatures(enemy.Status.States)...)
}

func (w *World) stateFeatures(states []int) []rpg.Feature {
	var out []rpg.Feature
	for id, turns := range states {
		if turns == 0 {
			continue
		}
		if def := w.stateDef(id); def != nil {
			out = append(out, def.Features...)
		}
	}
	return out
}

func (w *World) stateRate(features []rpg.Feature, stateID int) float64 {
	rate := 1.0
	for _, f := range features {
		switch f.Code {
		case 13: // state rate
			if f.DataID == stateID {
				rate *= f.Value
			}
		case 14: // state resist
			if f.DataID == stateID {
				return 0
			}
		}
	}
	return rate
}

// applyParamRates applies state/equipment parameter rates and buff levels.
func applyParamRates(stats *battleStats, features []rpg.Feature, buffs *[8]buffSlot) {
	rates := [8]float64{1, 1, 1, 1, 1, 1, 1, 1}
	for _, f := range features {
		if f.Code == 21 && f.DataID >= 0 && f.DataID < 8 && f.Value != 0 {
			rates[f.DataID] *= f.Value
		}
	}
	for i, buff := range buffs {
		if buff.Level != 0 {
			rates[i] *= 1 + 0.25*float64(buff.Level)
		}
	}
	// HP/MP max rates are not applied to the live character to keep persistence
	// and the client HUD in sync; no shipped state buffs those parameters.
	for i := 2; i < 8; i++ {
		stats.Params[i] = int(math.Round(float64(stats.Params[i]) * rates[i]))
	}
}

// sparam returns a multiplicative special-parameter rate (1 = unchanged).
func sparam(features []rpg.Feature, id int) float64 {
	rate := 1.0
	for _, f := range features {
		if f.Code == 23 && f.DataID == id && f.Value >= 0 {
			rate *= f.Value
		}
	}
	return rate
}

func applyXParams(stats *battleStats, features []rpg.Feature) {
	var x [11]float64
	for _, f := range features {
		if f.Code == 22 && f.DataID >= 0 && f.DataID < len(x) {
			x[f.DataID] += f.Value
		}
	}
	stats.Hit = clampFloat(0.95+x[0], 0, 1)
	stats.Eva = clampFloat(0.05+x[1], 0, 1)
	stats.Cri = clampFloat(0.04+x[2], 0, 1)
	stats.Counter = x[6]
	stats.HRG = x[7]
	stats.MRG = x[8]
}

// applyElementRates fills element rates and attack elements.
func applyElementRates(stats *battleStats, features []rpg.Feature) {
	for i := range stats.ElementRates {
		stats.ElementRates[i] = 1
	}
	for _, f := range features {
		switch f.Code {
		case 11:
			if f.DataID > 0 && f.DataID < len(stats.ElementRates) && f.Value >= 0 {
				stats.ElementRates[f.DataID] *= f.Value
			}
		case 31:
			if f.DataID > 0 && f.DataID < len(stats.ElementRates) {
				stats.AttackElements = append(stats.AttackElements, f.DataID)
			}
		}
	}
}

func elementRate(user, target battleStats, elementID int) float64 {
	if elementID > 0 && elementID < len(target.ElementRates) {
		return target.ElementRates[elementID]
	}
	best := -1.0
	for _, id := range user.AttackElements {
		if id > 0 && id < len(target.ElementRates) {
			rate := target.ElementRates[id]
			if rate > best {
				best = rate
			}
		}
	}
	if best >= 0 {
		return best
	}
	return 1
}

// addState applies a state with a chance (0..1). owner receives the client
// notification; owners are nil for enemies.
func (w *World) addState(set *statusSet, stateID int, chance float64, owner *Player) bool {
	def := w.stateDef(stateID)
	if def == nil || chance <= 0 {
		return false
	}
	if chance < 1 && rand.Float64() >= chance {
		return false
	}
	set.ensure(len(w.opts.Data.States))
	if set.hasState(stateID) {
		return false
	}
	turns := 0
	if def.AutoRemovalTiming != 0 && def.MaxTurns > 0 {
		turns = def.MaxTurns
	}
	set.States[stateID] = turns + 1
	if owner != nil {
		_ = owner.Client.Send(protocol.ServerState, protocol.State{StateID: uint16(clampInt(stateID, 0, 65535)), Add: true})
	}
	return true
}

func (w *World) removeState(set *statusSet, stateID int, owner *Player) {
	if !set.hasState(stateID) {
		return
	}
	set.States[stateID] = 0
	if owner != nil {
		_ = owner.Client.Send(protocol.ServerState, protocol.State{StateID: uint16(clampInt(stateID, 0, 65535)), Add: false})
	}
}

// addBuff applies a +1/-1 buff step with a number of seconds. Sends one packet
// per level step so the client's ±1 protocol stays correct.
func (w *World) addBuff(set *statusSet, owner *Player, param, delta, turns int) {
	if param < 0 || param >= len(set.Buffs) || delta == 0 {
		return
	}
	slot := &set.Buffs[param]
	old := slot.Level
	next := clampInt(old+delta, -2, 2)
	slot.Level = next
	if turns <= 0 {
		turns = 5
	}
	slot.Turns = turns
	if owner == nil {
		return
	}
	if next == 0 {
		_ = owner.Client.Send(protocol.ServerBuff, protocol.Buff{ParamID: uint8(param), Level: 0})
		return
	}
	step := 1
	if next < old {
		step = -1
	}
	for level := old + step; ; level += step {
		_ = owner.Client.Send(protocol.ServerBuff, protocol.Buff{ParamID: uint8(param), Level: int16(step)})
		if level == next {
			break
		}
	}
}

func (w *World) eraseBuff(set *statusSet, owner *Player, param int) {
	if param < 0 || param >= len(set.Buffs) {
		return
	}
	if set.Buffs[param].Level == 0 {
		return
	}
	set.Buffs[param] = buffSlot{}
	if owner != nil {
		_ = owner.Client.Send(protocol.ServerBuff, protocol.Buff{ParamID: uint8(param), Level: 0})
	}
}

// clearStatus removes every state and buff (death, logout).
func (w *World) clearStatus(set *statusSet, owner *Player) {
	for id, turns := range set.States {
		if turns != 0 {
			w.removeState(set, id, owner)
		}
	}
	for i := range set.Buffs {
		if set.Buffs[i].Level != 0 {
			w.eraseBuff(set, owner, i)
		}
	}
}

// applyEffectState handles effect codes 21/22/31/32/33 for one battler.
func (w *World) applyEffectState(item *rpg.UsableItem, effect rpg.Effect, set *statusSet, features []rpg.Feature, owner *Player) {
	switch effect.Code {
	case 21:
		if effect.DataID > 0 {
			w.addState(set, effect.DataID, effect.Value1*w.stateRate(features, effect.DataID), owner)
		}
	case 22:
		if effect.DataID > 0 {
			w.removeState(set, effect.DataID, owner)
		}
	case 31:
		w.addBuff(set, owner, effect.DataID, 1, int(effect.Value1))
	case 32:
		w.addBuff(set, owner, effect.DataID, -1, int(effect.Value1))
	case 33:
		w.eraseBuff(set, owner, effect.DataID)
	}
}

// applyAttackStates adds states from the attacker's attack-state features.
func (w *World) applyAttackStates(attackerFeatures, targetFeatures []rpg.Feature, set *statusSet, owner *Player) {
	for _, f := range attackerFeatures {
		if f.Code != 32 || f.DataID <= 0 {
			continue
		}
		w.addState(set, f.DataID, f.Value*w.stateRate(targetFeatures, f.DataID), owner)
	}
}

// onDamaged removes states flagged as removable by damage.
func (w *World) onDamaged(set *statusSet, owner *Player) {
	if !set.any() {
		return
	}
	for id, encoded := range set.States {
		if encoded == 0 {
			continue
		}
		def := w.stateDef(id)
		if def == nil || !def.RemoveByDamage {
			continue
		}
		if rand.IntN(100) < def.ChanceByDamage {
			w.removeState(set, id, owner)
		}
	}
}

// tickStatus advances one second of states/buffs and applies regeneration.
func (w *World) tickStatus(now time.Time, set *statusSet, owner *Player, features []rpg.Feature, maxHP, maxMP int, hp, mp *int) {
	second := now.Unix()
	if set.LastSecond == second {
		return
	}
	set.LastSecond = second
	for id, encoded := range set.States {
		if encoded == 0 {
			continue
		}
		def := w.stateDef(id)
		turns := encoded - 1
		if def == nil || def.AutoRemovalTiming == 0 || turns <= 0 {
			continue
		}
		turns--
		if turns == 0 {
			w.removeState(set, id, owner)
		} else {
			set.States[id] = turns + 1
		}
	}
	for i := range set.Buffs {
		if set.Buffs[i].Level == 0 {
			continue
		}
		set.Buffs[i].Turns--
		if set.Buffs[i].Turns <= 0 {
			w.eraseBuff(set, owner, i)
		}
	}
	hrg, mrg := 0.0, 0.0
	for _, f := range features {
		if f.Code != 22 {
			continue
		}
		switch f.DataID {
		case 7:
			hrg += f.Value
		case 8:
			mrg += f.Value
		}
	}
	changed := false
	if hrg != 0 {
		delta := int(math.Round(float64(maxHP) * hrg))
		if delta != 0 {
			*hp = clampInt(*hp+delta, 0, maxHP)
			changed = true
		}
	}
	if mrg != 0 {
		delta := int(math.Round(float64(maxMP) * mrg))
		if delta != 0 {
			*mp = clampInt(*mp+delta, 0, maxMP)
			changed = true
		}
	}
	if changed && owner != nil {
		w.sendVitalsToMap(owner)
	}
}
