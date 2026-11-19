package game

import (
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
)

// updateEnemyAI mirrors the original VXA-OS enemy loop: acquire a target within
// Sight, keep chasing while within Sight+5, use a random valid action every
// ATTACK_TIME, and move with the frequency-based stop interval.
func (w *World) updateEnemyAI(ms *mapState, enemy *enemyState, now time.Time) {
	sight := 6
	if enemy.Enemy.Sight > 0 {
		sight = enemy.Enemy.Sight
	}
	target := w.players[enemy.TargetID]
	if target != nil {
		if target.Character.MapID != ms.def.ID || target.Character.HP <= 0 || !enemyInRange(enemy, target, sight+5) {
			target = nil
			enemy.TargetID = 0
		}
	}
	if target == nil {
		target = w.findEnemyTarget(ms, enemy, sight)
		if target != nil {
			enemy.TargetID = target.ID
			_ = target.Client.Send(protocol.ServerBalloon, protocol.Balloon{
				CharacterID:   uint16(clampInt(enemy.Event.ID, 0, 65535)),
				CharacterType: 2, BalloonID: 1,
			})
		}
	}
	inBattle := target != nil

	if inBattle && !w.restricted(&enemy.Status, 4) && now.Sub(enemy.lastAction) >= enemyAttackDelay {
		enemy.lastAction = now
		w.enemyMakeAction(ms, enemy, target)
	}

	frequency := enemy.moveFrequency
	if inBattle {
		frequency = enemy.battleFrequency
	}
	if frequency < 1 {
		frequency = 1
	}
	if frequency > 5 {
		frequency = 5
	}
	if now.Before(enemy.nextMove) {
		return
	}
	enemy.nextMove = now.Add(time.Duration(5-frequency) * 750 * time.Millisecond)

	switch {
	case inBattle:
		if w.enemyInFront(enemy, target) {
			return
		}
		if enemy.Enemy.Escape && enemy.HP*3 < enemy.Enemy.Params[0] {
			w.moveEnemyAway(ms, enemy, target)
		} else {
			w.moveEnemyToward(ms, enemy, target)
		}
	case enemy.moveType == 1 || enemy.moveType == 2:
		w.moveEnemy(ms, enemy, moveDirections[rand.IntN(len(moveDirections))])
	}
}

func (w *World) findEnemyTarget(ms *mapState, enemy *enemyState, sight int) *Player {
	players := make([]*Player, 0, len(ms.players))
	for _, p := range ms.players {
		if p == nil {
			continue
		}
		if p.Character.HP > 0 {
			players = append(players, p)
		}
	}
	sort.Slice(players, func(i, j int) bool { return players[i].ID < players[j].ID })
	for _, p := range players {
		if enemyInRange(enemy, p, sight) {
			return p
		}
	}
	return nil
}

func enemyInRange(enemy *enemyState, p *Player, rangeTiles int) bool {
	return absInt(p.Character.X-enemy.X) <= rangeTiles && absInt(p.Character.Y-enemy.Y) <= rangeTiles
}

func (w *World) enemyInFront(enemy *enemyState, p *Player) bool {
	x, y := frontTile(enemy.X, enemy.Y, enemy.Direction)
	return p.Character.X == x && p.Character.Y == y
}

func (w *World) moveEnemyToward(ms *mapState, enemy *enemyState, target *Player) bool {
	dx := target.Character.X - enemy.X
	dy := target.Character.Y - enemy.Y
	primary, secondary := axisDirections(dx, dy)
	if w.moveEnemy(ms, enemy, primary) {
		return true
	}
	if secondary != 0 {
		return w.moveEnemy(ms, enemy, secondary)
	}
	return false
}

func (w *World) moveEnemyAway(ms *mapState, enemy *enemyState, target *Player) bool {
	dx := enemy.X - target.Character.X
	dy := enemy.Y - target.Character.Y
	primary, secondary := axisDirections(dx, dy)
	if w.moveEnemy(ms, enemy, primary) {
		return true
	}
	if secondary != 0 {
		return w.moveEnemy(ms, enemy, secondary)
	}
	return false
}

func axisDirections(dx, dy int) (int, int) {
	if absInt(dx) >= absInt(dy) {
		primary := 0
		if dx > 0 {
			primary = 6
		} else if dx < 0 {
			primary = 4
		}
		secondary := 0
		if dy > 0 {
			secondary = 2
		} else if dy < 0 {
			secondary = 8
		}
		return primary, secondary
	}
	primary := 0
	if dy > 0 {
		primary = 2
	} else if dy < 0 {
		primary = 8
	}
	secondary := 0
	if dx > 0 {
		secondary = 6
	} else if dx < 0 {
		secondary = 4
	}
	return primary, secondary
}

func (w *World) restricted(set *statusSet, restriction int) bool {
	for id := range set.States {
		if def := w.stateDef(id); def != nil && def.Restriction == restriction {
			return true
		}
	}
	return false
}

// enemyMakeAction selects one uniformly valid action, matching the original
// server (the Rating field is not used there).
func (w *World) enemyMakeAction(ms *mapState, enemy *enemyState, target *Player) {
	candidates := make([]rpg.EnemyAction, 0, len(enemy.Enemy.Actions))
	for _, action := range enemy.Enemy.Actions {
		if w.enemyActionValid(enemy, target, action) {
			candidates = append(candidates, action)
		}
	}
	if len(candidates) == 0 {
		return
	}
	action := candidates[rand.IntN(len(candidates))]
	skill := w.skill(action.SkillID)
	if skill == nil {
		return
	}
	if action.SkillID == 1 || skill.Scope == 0 {
		if front := w.playerInFront(ms, enemy); front != nil {
			w.enemyAttack(enemy, front)
		}
		return
	}
	if skill.Scope == 11 {
		w.enemySelfSkill(enemy, skill)
		return
	}
	if skill.AOE > 0 {
		for _, other := range w.enemyAreaTargets(ms, enemy, skill.AOE) {
			w.enemySkillOnPlayer(enemy, other, skill)
		}
		enemy.MP -= skill.MPCost
		return
	}
	if target == nil || !enemyInRange(enemy, target, maxInt(1, skill.Range)) {
		return
	}
	w.enemySkillOnPlayer(enemy, target, skill)
	enemy.MP -= skill.MPCost
}

func (w *World) enemyActionValid(enemy *enemyState, target *Player, action rpg.EnemyAction) bool {
	skill := w.skill(action.SkillID)
	if skill == nil || enemy.MP < skill.MPCost {
		return false
	}
	switch action.ConditionType {
	case 2:
		return hpRateIn(enemy.HP, enemy.Enemy.Params[0], action.ConditionParam1, action.ConditionParam2)
	case 3:
		return hpRateIn(enemy.MP, enemy.Enemy.Params[1], action.ConditionParam1, action.ConditionParam2)
	case 4:
		_, ok := enemy.Status.States[action.ConditionParam1]
		return ok
	case 5:
		return target != nil && target.Character.Level >= action.ConditionParam1
	case 6:
		return target != nil && w.switchValue(target, action.ConditionParam1)
	}
	return true
}

func hpRateIn(value, max, low, high int) bool {
	if max <= 0 {
		return false
	}
	rate := value * 100 / max
	return rate >= low && rate <= high
}

func (w *World) playerInFront(ms *mapState, enemy *enemyState) *Player {
	x, y := frontTile(enemy.X, enemy.Y, enemy.Direction)
	return ms.playerAt(x, y)
}

func (w *World) enemyAreaTargets(ms *mapState, enemy *enemyState, aoe int) []*Player {
	var out []*Player
	for _, p := range ms.players {
		if p == nil {
			continue
		}
		if p.Character.HP > 0 && enemyInRange(enemy, p, aoe) {
			out = append(out, p)
		}
	}
	return out
}

func (w *World) enemySkillOnPlayer(enemy *enemyState, target *Player, skill *rpg.Skill) {
	user := w.enemyStats(enemy)
	defender := w.playerStats(target)
	damage, critical, missed := rollDamage(user, defender, skill.Damage, nil)
	if missed {
		damage, critical = 0, false
	}
	if damage > target.Character.HP {
		damage = target.Character.HP
	}
	target.Character.HP -= damage
	w.broadcastToMap(target, protocol.ServerAttackPlayer, protocol.AttackPlayer{
		AttackerID:   uint16(clampInt(enemy.Event.ID, 0, 65535)),
		AttackerType: 1,
		AniIndex:     uint8(clampInt(enemy.Enemy.AniIndex, 0, 255)),
		PlayerID:     target.ID,
		HPDamage:     -int32(damage),
		Critical:     critical,
		AnimationID:  uint16(clampInt(skill.AnimationID, 0, 65535)),
	})
	w.sendVitalsToMap(target)
	w.onDamaged(&target.Status, target)
	if !missed {
		features := w.playerFeatures(target)
		for _, effect := range skill.Effects {
			w.applyEffectState(&skill.UsableItem, effect, &target.Status, features, target)
		}
	}
	if target.Character.HP <= 0 {
		target.respawnAt = w.now().Add(playerRespawnDelay)
		w.clearStatus(&target.Status, target)
		w.addState(&target.Status, 1, 1, target)
		target.TargetType, target.TargetID = 0, 0
	}
}

// enemySelfSkill applies recovery/buff effects to the enemy itself (scope 11).
func (w *World) enemySelfSkill(enemy *enemyState, skill *rpg.Skill) {
	enemy.MP -= skill.MPCost
	features := w.enemyFeatures(enemy)
	for _, effect := range skill.Effects {
		switch effect.Code {
		case 11:
			value := int(math.Round(float64(enemy.Enemy.Params[0])*effect.Value1 + effect.Value2))
			enemy.HP = clampInt(enemy.HP+value, 0, enemy.Enemy.Params[0])
		case 12:
			value := int(math.Round(float64(enemy.Enemy.Params[1])*effect.Value1 + effect.Value2))
			enemy.MP = clampInt(enemy.MP+value, 0, enemy.Enemy.Params[1])
		case 21, 22, 31, 32, 33:
			w.applyEffectState(&skill.UsableItem, effect, &enemy.Status, features, nil)
		}
	}
}

// disableEnemy toggles the VXA-OS disable switch/variable on death.
func (w *World) disableEnemy(killer *Player, enemy *enemyState) {
	if enemy == nil || enemy.Enemy == nil || killer == nil {
		return
	}
	if id := enemy.Enemy.DisableVariableID; id > 0 {
		slot := id - 1
		if slot < len(killer.State.Variables) {
			killer.State.Variables[slot]++
			_ = killer.Client.Send(protocol.ServerVariable, protocol.Variable{ID: uint16(clampInt(id, 0, 65535)), Value: killer.State.Variables[slot]})
		}
	}
	if id := enemy.Enemy.DisableSwitchID; id > 0 {
		if id > w.opts.MaxPlayerSwitches {
			w.setGlobalSwitch(id, !w.globalSwitchValue(id))
			w.saveGlobalSwitches()
		} else if slot := id - 1; slot < len(killer.State.Switches) {
			killer.State.Switches[slot] = !killer.State.Switches[slot]
			_ = killer.Client.Send(protocol.ServerSwitch, protocol.Switch{ID: uint16(clampInt(id, 0, 65535)), Value: killer.State.Switches[slot]})
		}
	}
}

// reviveEnemy restores HP/MP and may move a non-fixed enemy to a passable tile
// in its configured revive region.
func (w *World) reviveEnemy(ms *mapState, enemy *enemyState, now time.Time) {
	enemy.Alive = true
	enemy.HP = enemy.Enemy.Params[0]
	if enemy.HP < 1 {
		enemy.HP = 1
	}
	enemy.MP = enemy.Enemy.Params[1]
	enemy.ReviveAt = time.Time{}
	enemy.TargetID = 0
	enemy.lastAction = time.Time{}
	enemy.nextMove = now.Add(time.Duration(5-clampInt(enemy.moveFrequency, 1, 5)) * 750 * time.Millisecond)
	w.clearStatus(&enemy.Status, nil)
	if enemy.moveType != 0 {
		w.enemyChangePosition(ms, enemy)
	}
	w.broadcastToMapPlayers(ms, protocol.ServerEnemyRevive, protocol.EnemyRevive{EventID: uint16(enemy.Event.ID)})
	w.broadcastToMapPlayers(ms, protocol.ServerEventData, protocol.EventData{
		EventID: uint16(enemy.Event.ID), X: uint16(enemy.X), Y: uint16(enemy.Y),
		Direction: uint8(enemy.Direction), HP: int32(enemy.HP),
	})
}

func (w *World) enemyChangePosition(ms *mapState, enemy *enemyState) {
	region := enemy.regionID
	if region < 1 || region > 4 {
		region = 1
	}
	type pos struct{ x, y int }
	tiles := make([]pos, 0, 32)
	for x := 0; x < ms.def.Width; x++ {
		for y := 0; y < ms.def.Height; y++ {
			if int(ms.def.Data.At(int32(x), int32(y), 3))>>8 == region {
				tiles = append(tiles, pos{x, y})
			}
		}
	}
	if len(tiles) == 0 {
		return
	}
	for i := 0; i < len(tiles); i++ {
		choice := tiles[rand.IntN(len(tiles))]
		if !ms.passable(choice.x, choice.y, 0) || w.npcBlocked(ms, enemy.Event.ID, choice.x, choice.y) {
			continue
		}
		enemy.X, enemy.Y = choice.x, choice.y
		enemy.Direction = 2
		return
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
