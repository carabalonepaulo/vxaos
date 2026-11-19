package game

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

const (
	enemyDefaultRespawn = 5 * time.Second
	enemyAttackDelay    = 800 * time.Millisecond
	playerAttackDelay   = 800 * time.Millisecond
	playerRespawnDelay  = 3 * time.Second
	recoverInterval     = 5 * time.Second
	recoverHPAmount     = 20
	recoverMPAmount     = 5
	vipRecoverBonus     = 1.5
)

// enemyState is one enemy map event. The client already has the event sprite
// and page from Map*.rvdata2; this is the authoritative HP/position/timer.
type enemyState struct {
	Event           *rpg.Event
	Enemy           *rpg.Enemy
	ID              int
	X, Y            int
	Direction       int
	HP, MP          int
	Alive           bool
	ReviveAt        time.Time
	lastAction      time.Time
	Status          statusSet
	moveType        int
	moveSpeed       int
	moveFrequency   int
	battleFrequency int
	regionID        int
	TargetID        uint16
	nextMove        time.Time
	through         bool
	priority        int
}

func newEnemyState(data *rpg.Data, mapDef *rpg.Map, event *rpg.Event) *enemyState {
	if data == nil || event == nil || len(event.Pages) == 0 {
		return nil
	}
	id := enemyIDFromPage(event.Pages[0])
	if id <= 0 || id >= len(data.Enemies) || data.Enemies[id] == nil {
		return nil
	}
	enemy := data.Enemies[id]
	hp := enemy.Params[0]
	if hp < 1 {
		hp = 1
	}
	mp := enemy.Params[1]
	if mp < 0 {
		mp = 0
	}
	direction := event.Pages[0].Graphic.Direction
	if direction == 0 {
		direction = 2
	}
	page := event.Pages[0]
	frequency := page.MoveFrequency
	if value := pageNoteInt(page, "Frequency", 0); value > 0 {
		frequency = value
	}
	region := pageNoteInt(page, "Region", 1)
	return &enemyState{
		Event:           event,
		Enemy:           enemy,
		ID:              id,
		X:               event.X,
		Y:               event.Y,
		Direction:       direction,
		HP:              hp,
		MP:              mp,
		Alive:           true,
		moveType:        page.MoveType,
		moveSpeed:       page.MoveSpeed,
		moveFrequency:   page.MoveFrequency,
		battleFrequency: frequency,
		regionID:        region,
		through:         page.Through,
		priority:        page.PriorityType,
	}
}

func enemyIDFromPage(page *rpg.Page) int {
	if page == nil || len(page.List) == 0 || page.List[0] == nil || page.List[0].Code != 108 {
		return 0
	}
	if len(page.List[0].Parameters) == 0 {
		return 0
	}
	text, _ := page.List[0].Parameters[0].(string)
	if !strings.HasPrefix(text, "Enemy") {
		return 0
	}
	parts := strings.SplitN(text, "=", 2)
	if len(parts) != 2 {
		return 0
	}
	id, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return id
}

// pageNoteInt reads one "Key=value" comment line (event command 408).
func pageNoteInt(page *rpg.Page, key string, def int) int {
	if page == nil {
		return def
	}
	prefix := key + "="
	for _, cmd := range page.List {
		if cmd == nil || cmd.Code != 408 || len(cmd.Parameters) == 0 {
			continue
		}
		text, _ := cmd.Parameters[0].(string)
		text = strings.TrimSpace(text)
		if !strings.HasPrefix(text, prefix) {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(text[len(prefix):]))
		if err == nil {
			return value
		}
	}
	return def
}

func (ms *mapState) enemyByID(id int) *enemyState {
	if id >= 0 && id < len(ms.enemies) {
		return ms.enemies[id]
	}
	return nil
}

func (ms *mapState) enemyAt(x, y int) *enemyState {
	for _, enemy := range ms.enemies {
		if enemy == nil {
			continue
		}
		if enemy.Alive && enemy.X == x && enemy.Y == y {
			return enemy
		}
	}
	return nil
}

func (ms *mapState) hasPlayers() bool {
	for _, player := range ms.players {
		if player != nil {
			return true
		}
	}
	return false
}

func (ms *mapState) playerAt(x, y int) *Player {
	for _, player := range ms.players {
		if player == nil {
			continue
		}
		if player.Character.X == x && player.Character.Y == y && player.Character.HP > 0 {
			return player
		}
	}
	return nil
}

// Attack performs a normal weapon attack against the tile in front.
func (w *World) Attack(ctx context.Context, p *Player) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		if w.restricted(&p.Status, 4) {
			return ErrRequirement
		}
		now := w.now()
		if !p.lastAttack.IsZero() && now.Sub(p.lastAttack) < playerAttackDelay {
			return ErrRateLimited
		}
		p.lastAttack = now
		x, y := frontTile(p.Character.X, p.Character.Y, p.Character.Direction)
		if enemy := ms.enemyAt(x, y); enemy != nil {
			w.hitEnemy(p, enemy, w.attackSkill(), w.attackAnimation(p), w.attackAniIndex(p), w.playerStats(p), w.enemyStats(enemy))
			w.startEvents(p)
			return nil
		}
		if ms.def.PvP {
			if target := ms.playerAt(x, y); target != nil && w.canPvP(p, target) {
				w.hitPlayer(p, target, w.attackSkill(), w.attackAnimation(p), w.attackAniIndex(p))
			}
		}
		w.startEvents(p)
		return nil
	})
}

func (w *World) SetTarget(ctx context.Context, p *Player, typ uint8, id uint16) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		switch typ {
		case 0:
			p.TargetType, p.TargetID = 0, 0
		case 1:
			if target := ms.playerByID(id); target != nil && target.Character.MapID == p.Character.MapID {
				p.TargetType, p.TargetID = typ, id
			}
		case 2:
			if enemy := ms.enemyByID(int(id)); enemy != nil && enemy.Alive {
				p.TargetType, p.TargetID = typ, id
			}
		default:
			return ErrInvalidTarget
		}
		return nil
	})
}

// UseSkill casts one learned skill.
func (w *World) UseSkill(ctx context.Context, p *Player, skillID uint16) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		skill := w.skill(int(skillID))
		if skill == nil || !hasSkill(p.State, int(skillID)) {
			return ErrNotOwned
		}
		if p.Character.Level < skill.Level {
			return ErrRequirement
		}
		features := w.playerFeatures(p)
		for _, f := range features {
			if f.Code == 42 && f.DataID == skill.STypeID {
				return ErrRequirement
			}
		}
		cost := skill.MPCost
		if cost > 0 {
			cost = int(math.Round(float64(cost) * sparam(features, 4)))
		}
		if p.Character.MP < cost {
			return ErrRequirement
		}
		now := w.now()
		if !p.lastAttack.IsZero() && now.Sub(p.lastAttack) < playerAttackDelay {
			return ErrRateLimited
		}
		p.lastAttack = now
		p.Character.MP -= cost
		w.useUsableOnPlayer(p, &skill.UsableItem, int(skillID))
		w.sendVitalsToMap(p)
		w.saveQuietly(p)
		return nil
	})
}

// UseItem consumes an inventory item.
func (w *World) UseItem(ctx context.Context, p *Player, itemID uint16) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		item := w.item(int(itemID))
		if item == nil || findInventory(p.State, KindItem, int(itemID)) < 0 {
			return ErrNotOwned
		}
		if p.Character.Level < item.Level {
			return ErrRequirement
		}
		now := w.now()
		if !p.lastAttack.IsZero() && now.Sub(p.lastAttack) < playerAttackDelay {
			return ErrRateLimited
		}
		p.lastAttack = now
		if item.Consumable {
			removeInventoryItem(p.State, KindItem, int(itemID), 1)
			w.sendItemTo(p, KindItem, int(itemID), -1, true, false)
		}
		w.useUsableOnPlayer(p, &item.UsableItem, int(itemID))
		w.saveQuietly(p)
		return nil
	})
}

// UseHotbar dispatches the item or skill stored in one hotbar slot.
func (w *World) UseHotbar(ctx context.Context, p *Player, slot uint8) error {
	entry := store.HotbarSlot{}
	ok := false
	_ = w.do(ctx, func() error {
		w.ensureHotbar(p)
		if int(slot) >= len(p.State.Hotbar) {
			return ErrInvalidHotbar
		}
		entry = p.State.Hotbar[slot]
		ok = true
		return nil
	})
	if !ok {
		return ErrNotInGame
	}
	switch entry.Type {
	case 1:
		return w.UseItem(ctx, p, uint16(entry.ItemID))
	case 2:
		return w.UseSkill(ctx, p, uint16(entry.ItemID))
	}
	return nil
}

func (w *World) attackSkill() *rpg.Skill {
	if len(w.opts.Data.Skills) > 1 {
		return w.opts.Data.Skills[1]
	}
	return nil
}

func (w *World) attackAnimation(p *Player) int {
	if p.State != nil && len(p.State.Equips) > 0 {
		if weapon := w.weapon(p.State.Equips[0]); weapon != nil {
			return weapon.AnimationID
		}
	}
	if skill := w.attackSkill(); skill != nil {
		return skill.AnimationID
	}
	return 0
}

func (w *World) attackAniIndex(p *Player) int {
	if p.State != nil && len(p.State.Equips) > 0 {
		if weapon := w.weapon(p.State.Equips[0]); weapon != nil && weapon.AniIndex >= 0 {
			return weapon.AniIndex
		}
	}
	return p.Character.CharacterIndex
}

func (w *World) hitEnemy(attacker *Player, enemy *enemyState, skill *rpg.Skill, animationID, aniIndex int, user, target battleStats) {
	if skill == nil || !enemy.Alive {
		return
	}
	ms := attacker.ms
	if ms == nil {
		return
	}
	damage, critical, missed := rollDamage(user, target, skill.Damage, user.Vars)
	if missed {
		damage, critical = 0, false
	}
	if damage > enemy.HP {
		damage = enemy.HP
	}
	enemy.HP -= damage
	w.log.Debug("hit enemy", "player", attacker.Character.Name, "event", enemy.Event.ID, "damage", damage, "hp", enemy.HP, "atk", user.Params[2], "def", target.Params[3], "formula", skill.Damage.Formula)
	msg := protocol.AttackEnemy{
		AttackerID:   attacker.ID,
		AttackerType: 0,
		AniIndex:     uint8(clampInt(aniIndex, 0, 255)),
		EventID:      uint16(enemy.Event.ID),
		HPDamage:     -int32(damage),
		Critical:     critical,
		AnimationID:  uint16(clampInt(animationID, 0, 65535)),
	}
	w.broadcastToMap(attacker, protocol.ServerAttackEnemy, msg)
	w.onDamaged(&enemy.Status, nil)
	if !missed {
		w.applyAttackStates(w.playerFeatures(attacker), w.enemyFeatures(enemy), &enemy.Status, nil)
	}
	if enemy.HP <= 0 {
		enemy.HP = 0
		enemy.Alive = false
		enemy.ReviveAt = w.now().Add(respawnDelay(enemy.Enemy))
		enemy.TargetID = 0
		w.clearStatus(&enemy.Status, nil)
		w.disableEnemy(attacker, enemy)
		attacker.TargetType, attacker.TargetID = 0, 0
		w.awardKill(attacker, enemy)
	}
}

func (w *World) hitPlayer(attacker *Player, target *Player, skill *rpg.Skill, animationID, aniIndex int) {
	if skill == nil || target.Character.HP <= 0 {
		return
	}
	damage, critical, missed := rollDamage(w.playerStats(attacker), w.playerStats(target), skill.Damage, attacker.State.Variables)
	if missed {
		damage, critical = 0, false
	}
	if damage > target.Character.HP {
		damage = target.Character.HP
	}
	target.Character.HP -= damage
	w.broadcastToMap(attacker, protocol.ServerAttackPlayer, protocol.AttackPlayer{
		AttackerID:   attacker.ID,
		AttackerType: 0,
		AniIndex:     uint8(clampInt(aniIndex, 0, 255)),
		PlayerID:     target.ID,
		HPDamage:     -int32(damage),
		Critical:     critical,
		AnimationID:  uint16(clampInt(animationID, 0, 65535)),
	})
	w.sendVitalsToMap(target)
	w.onDamaged(&target.Status, target)
	if !missed {
		w.applyAttackStates(w.playerFeatures(attacker), w.playerFeatures(target), &target.Status, target)
	}
	if target.Character.HP <= 0 {
		w.playerDied(target)
	}
}

// playerDied mirrors the original death handling: lose gold/EXP, full heal,
// clear states/buffs and transfer to the revive point.
func (w *World) playerDied(p *Player) {
	if p == nil {
		return
	}
	const loseGoldPercent = 5
	const loseExpPercent = 4
	const loseVIPExpPercent = 2
	if p.Character.Gold > 0 {
		loss := int(p.Character.Gold) * loseGoldPercent / 100
		p.Character.Gold -= int64(loss)
		_ = p.Client.Send(protocol.ServerGold, protocol.Gold{Amount: -int64(loss)})
	}
	rate := loseExpPercent
	if p.vipActive(w.now().Unix()) {
		rate = loseVIPExpPercent
	}
	if p.Character.Exp > 0 {
		loss := int(p.Character.Exp) * rate / 100
		w.loseExp(p, loss)
	}
	w.clearStatus(&p.Status, p)
	p.Character.HP = p.Character.MHP
	p.Character.MP = p.Character.MMP
	p.respawnAt = time.Time{}
	p.TargetType, p.TargetID = 0, 0
	p.Event = nil
	mapID, x, y := p.Character.ReviveMapID, p.Character.ReviveX, p.Character.ReviveY
	if !w.mapTemplateValid(mapID, x, y) {
		mapID = w.opts.Data.System.StartMapID
		x = w.opts.Data.System.StartX
		y = w.opts.Data.System.StartY
	}
	w.transferPlayer(p, mapID, x, y, 2)
	w.sendVitalsToMap(p)
	w.saveQuietly(p)
}

func (w *World) loseExp(p *Player, amount int) {
	if p == nil || amount <= 0 || p.Character.ClassID <= 0 || p.Character.ClassID >= len(w.opts.Data.Classes) {
		return
	}
	class := w.opts.Data.Classes[p.Character.ClassID]
	if class == nil {
		return
	}
	floor := class.ExpForLevel(p.Character.Level)
	next := p.Character.Exp - amount
	if next < floor {
		next = floor
	}
	if next < 0 {
		next = 0
	}
	if next == p.Character.Exp {
		return
	}
	delta := next - p.Character.Exp
	p.Character.Exp = next
	_ = p.Client.Send(protocol.ServerExp, protocol.Exp{PlayerID: p.ID, Amount: int32(delta)})
}

func (w *World) useUsableOnPlayer(p *Player, item *rpg.UsableItem, itemID int) {
	ms := p.ms
	if ms == nil {
		return
	}
	user := w.playerStats(p)
	scope := item.Scope
	if scope == 8 { // all allies
		targets := w.partyMembersOnMap(p)
		if len(targets) == 0 {
			targets = []*Player{p}
		}
		for _, target := range targets {
			if target.Character.HP > 0 || scope == 10 {
				w.applyEffectsToPlayer(target, item, user, item.AnimationID, 0)
			}
		}
		w.broadcastItemAnimation(p, p.ID, 1, item.AnimationID, 0)
		return
	}
	if scope == 2 { // all enemies
		radius := item.AOE
		if radius < 1 {
			radius = item.Range
		}
		if radius < 1 {
			radius = 1
		}
		w.applyAreaItem(p, item, user, radius)
		return
	}
	if scope >= 3 && scope <= 6 {
		count := scope - 2
		w.applyRandomEnemies(p, item, count)
		return
	}
	if item.AOE > 0 {
		w.applyAreaItem(p, item, user, item.AOE)
		return
	}
	if item.Scope >= 1 && item.Scope <= 6 {
		if p.TargetType == 1 {
			if target := ms.playerByID(p.TargetID); target != nil && target.Character.MapID == p.Character.MapID && target.Character.HP > 0 {
				rangeTiles := item.Range
				if rangeTiles < 1 {
					rangeTiles = 1
				}
				if absInt(target.Character.X-p.Character.X) <= rangeTiles &&
					absInt(target.Character.Y-p.Character.Y) <= rangeTiles &&
					!w.lineBlocked(ms, p.Character.X, p.Character.Y, target.Character.X, target.Character.Y, p.Character.Direction) &&
					w.canPvP(p, target) {
					w.hitPlayer(p, target, &rpg.Skill{UsableItem: *item}, item.AnimationID, p.Character.CharacterIndex)
				}
				return
			}
		}
		target := w.resolveTarget(p, item.Scope, ms)
		if target == nil {
			if item.Scope == 11 {
				w.applyEffectsToPlayer(p, item, user, item.AnimationID, 0)
				w.broadcastAttackSelf(p, item.AnimationID, 0)
			}
			return
		}
		rangeTiles := item.Range
		if rangeTiles < 1 {
			rangeTiles = 1
		}
		if absInt(target.X-p.Character.X) > rangeTiles || absInt(target.Y-p.Character.Y) > rangeTiles {
			return
		}
		fx, fy := w.projectileEnd(ms, p.Character.X, p.Character.Y, target.X, target.Y)
		blocked := w.lineBlocked(ms, p.Character.X, p.Character.Y, target.X, target.Y, p.Character.Direction)
		w.broadcastSkillProjectile(p, itemID, fx, fy, target.X, target.Y)
		if blocked {
			return
		}
		w.applyUsableToEnemy(p, target, item)
		return
	}
	// Ally scopes: use the selected player when valid, otherwise the caster.
	target := w.resolveAllyTarget(p, ms, item.Range)
	if target == nil {
		target = p
	}
	w.applyEffectsToPlayer(target, item, user, item.AnimationID, 0)
	w.broadcastItemAnimation(p, target.ID, 1, item.AnimationID, 0)
}

// applyAreaItem hits every enemy (and PvP-valid player) within the item's AOE
// radius around the caster, matching the original area branch.
func (w *World) applyAreaItem(p *Player, item *rpg.UsableItem, user battleStats, radius int) {
	ms := p.ms
	if ms == nil {
		return
	}
	if radius < 1 {
		radius = 1
	}
	for _, enemy := range ms.enemies {
		if enemy == nil {
			continue
		}
		if !enemy.Alive || absInt(enemy.X-p.Character.X) > radius || absInt(enemy.Y-p.Character.Y) > radius {
			continue
		}
		w.applyUsableToEnemy(p, enemy, item)
	}
	if ms.def.PvP {
		for _, other := range ms.players {
			if other == nil || other == p || other.Character.HP <= 0 || !w.canPvP(p, other) {
				continue
			}
			if absInt(other.Character.X-p.Character.X) > radius || absInt(other.Character.Y-p.Character.Y) > radius {
				continue
			}
			w.hitPlayer(p, other, &rpg.Skill{UsableItem: *item}, item.AnimationID, p.Character.CharacterIndex)
		}
	}
	w.broadcastItemAnimation(p, p.ID, 1, item.AnimationID, 0)
}

func (w *World) resolveAllyTarget(p *Player, ms *mapState, rangeTiles int) *Player {
	if p.TargetType == 1 {
		if target := ms.playerByID(p.TargetID); target != nil && target.Character.MapID == p.Character.MapID && target.Character.HP > 0 {
			if rangeTiles < 1 || (absInt(target.Character.X-p.Character.X) <= rangeTiles && absInt(target.Character.Y-p.Character.Y) <= rangeTiles) {
				return target
			}
		}
	}
	return nil
}

func (w *World) applyRandomEnemies(p *Player, item *rpg.UsableItem, count int) {
	ms := p.ms
	if ms == nil || count <= 0 {
		return
	}
	rangeTiles := item.Range
	if rangeTiles < 1 {
		rangeTiles = 1
	}
	candidates := make([]*enemyState, 0, len(ms.enemies))
	for _, enemy := range ms.enemies {
		if enemy == nil || !enemy.Alive {
			continue
		}
		if absInt(enemy.X-p.Character.X) <= rangeTiles && absInt(enemy.Y-p.Character.Y) <= rangeTiles {
			candidates = append(candidates, enemy)
		}
	}
	for i := len(candidates) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	if count > len(candidates) {
		count = len(candidates)
	}
	for i := 0; i < count; i++ {
		w.applyUsableToEnemy(p, candidates[i], item)
	}
	w.broadcastItemAnimation(p, p.ID, 1, item.AnimationID, 0)
}

func (w *World) broadcastItemAnimation(p *Player, characterID uint16, characterType uint8, animationID, aniIndex int) {
	if animationID <= 0 {
		return
	}
	w.broadcastToMap(p, protocol.ServerAnimation, protocol.Animation{
		AttackerID:    p.ID,
		AttackerType:  0,
		AniIndex:      uint8(clampInt(aniIndex, 0, 255)),
		CharacterID:   characterID,
		CharacterType: characterType,
		AnimationID:   uint16(clampInt(animationID, 0, 65535)),
	})
}

// broadcastSkillProjectile sends the visual projectile for configured ranged
// skills. The client only knows projectile definitions for its Configs list.
func (w *World) broadcastSkillProjectile(p *Player, skillID, finishX, finishY, targetX, targetY int) {
	if p == nil || !containsInt(w.opts.ProjectileSkills, skillID) {
		return
	}
	w.broadcastToMap(p, protocol.ServerProjectile, protocol.Projectile{
		StartX: uint16(clampInt(p.Character.X, 0, 65535)), StartY: uint16(clampInt(p.Character.Y, 0, 65535)),
		FinishX: uint16(clampInt(finishX, 0, 65535)), FinishY: uint16(clampInt(finishY, 0, 65535)),
		TargetX: uint16(clampInt(targetX, 0, 65535)), TargetY: uint16(clampInt(targetY, 0, 65535)),
		ProjectileType: 1, ProjectileID: uint8(clampInt(skillID, 0, 255)),
	})
}

// projectileEnd returns the last tile a straight shot can reach.
func (w *World) projectileEnd(ms *mapState, x0, y0, x1, y1 int) (int, int) {
	dx := x1 - x0
	dy := y1 - y0
	steps := absInt(dx)
	if absInt(dy) > steps {
		steps = absInt(dy)
	}
	if steps == 0 {
		return x0, y0
	}
	lastX, lastY := x0, y0
	for step := 1; step <= steps; step++ {
		x := x0 + dx*step/steps
		y := y0 + dy*step/steps
		if !ms.valid(x, y) {
			break
		}
		if step < steps && !ms.passable(x, y, 2) {
			break
		}
		lastX, lastY = x, y
	}
	return lastX, lastY
}

// lineBlocked reports whether terrain interrupts a straight shot.
func (w *World) lineBlocked(ms *mapState, x0, y0, x1, y1, direction int) bool {
	dx := x1 - x0
	dy := y1 - y0
	if absInt(dx) <= 1 && absInt(dy) <= 1 {
		return false
	}
	steps := absInt(dx)
	if absInt(dy) > steps {
		steps = absInt(dy)
	}
	if steps <= 0 {
		return false
	}
	dir := uint8(direction)
	for step := 1; step < steps; step++ {
		x := x0 + dx*step/steps
		y := y0 + dy*step/steps
		if !ms.valid(x, y) || !ms.passable(x, y, dir) {
			return true
		}
	}
	return false
}

// canPvP applies the original level protection and admin immunity.
func (w *World) canPvP(attacker, target *Player) bool {
	if attacker == nil || target == nil || attacker == target || isAdmin(target) {
		return false
	}
	const pvpMinLevel = 2
	return attacker.Character.Level >= pvpMinLevel && target.Character.Level >= pvpMinLevel
}

func scopeIsSelf(scope int) bool {
	return scope == 7 || scope == 8 || scope == 10 || scope == 11
}

func (w *World) resolveTarget(p *Player, scope int, ms *mapState) *enemyState {
	if scopeIsSelf(scope) {
		return nil
	}
	if p.TargetType == 2 {
		if enemy := ms.enemyByID(int(p.TargetID)); enemy != nil && enemy.Alive {
			return enemy
		}
	}
	x, y := frontTile(p.Character.X, p.Character.Y, p.Character.Direction)
	return ms.enemyAt(x, y)
}

func (w *World) applyUsableToEnemy(p *Player, enemy *enemyState, item *rpg.UsableItem) {
	ms := p.ms
	if ms == nil || enemy == nil || !enemy.Alive {
		return
	}
	user := w.playerStats(p)
	target := w.enemyStats(enemy)
	damage, critical, missed := rollDamage(user, target, item.Damage, user.Vars)
	if missed {
		damage, critical = 0, false
	}
	if damage > enemy.HP {
		damage = enemy.HP
	}
	enemy.HP -= damage
	w.onDamaged(&enemy.Status, nil)
	w.broadcastToMap(p, protocol.ServerAttackEnemy, protocol.AttackEnemy{
		AttackerID:   p.ID,
		AttackerType: 0,
		AniIndex:     uint8(clampInt(p.Character.CharacterIndex, 0, 255)),
		EventID:      uint16(enemy.Event.ID),
		HPDamage:     -int32(damage),
		Critical:     critical,
		AnimationID:  uint16(clampInt(item.AnimationID, 0, 65535)),
	})
	if enemy.HP > 0 {
		features := w.enemyFeatures(enemy)
		for _, effect := range item.Effects {
			w.applyEffectState(item, effect, &enemy.Status, features, nil)
		}
	}
	if enemy.HP <= 0 {
		enemy.HP = 0
		enemy.Alive = false
		enemy.ReviveAt = w.now().Add(respawnDelay(enemy.Enemy))
		enemy.TargetID = 0
		w.clearStatus(&enemy.Status, nil)
		w.disableEnemy(p, enemy)
		p.TargetType, p.TargetID = 0, 0
		w.awardKill(p, enemy)
	}
}

func (w *World) applyEffectsToPlayer(p *Player, item *rpg.UsableItem, user battleStats, animationID, aniIndex int) {
	changedVitals := false
	for _, effect := range item.Effects {
		switch effect.Code {
		case 11:
			value := int(math.Round((float64(user.MHP)*effect.Value1 + effect.Value2)))
			if value > 0 {
				p.Character.HP += value
				if p.Character.HP > p.Character.MHP {
					p.Character.HP = p.Character.MHP
				}
				w.broadcastToMap(p, protocol.ServerAttackPlayer, protocol.AttackPlayer{
					AttackerID: p.ID, AttackerType: 0, PlayerID: p.ID,
					HPDamage: int32(value), AnimationID: uint16(clampInt(animationID, 0, 65535)),
					NotShowMissed: true,
				})
				changedVitals = true
			}
		case 12:
			value := int(math.Round((float64(user.MMP)*effect.Value1 + effect.Value2)))
			if value > 0 {
				p.Character.MP += value
				if p.Character.MP > p.Character.MMP {
					p.Character.MP = p.Character.MMP
				}
				w.broadcastToMap(p, protocol.ServerAttackPlayer, protocol.AttackPlayer{
					AttackerID: p.ID, AttackerType: 0, PlayerID: p.ID,
					MPDamage: int32(value), AnimationID: uint16(clampInt(animationID, 0, 65535)),
					NotShowMissed: true,
				})
				changedVitals = true
			}
		case 21, 22, 31, 32, 33:
			features := w.playerFeatures(p)
			w.applyEffectState(item, effect, &p.Status, features, p)
		case 42:
			if effect.DataID >= 0 && effect.DataID < 8 {
				value := int(effect.Value1)
				w.applyParamBonus(p, effect.DataID, value)
			}
		case 43:
			if !hasSkill(p.State, effect.DataID) {
				p.State.Skills = append(p.State.Skills, effect.DataID)
				_ = p.Client.Send(protocol.ServerSkill, protocol.Skill{SkillID: uint16(clampInt(effect.DataID, 0, 65535)), Learned: true})
			}
		case 44:
			w.queueCommonEvent(p, effect.DataID)
		}
	}
	if changedVitals {
		w.sendVitalsToMap(p)
	}
}

func (w *World) broadcastAttackSelf(p *Player, animationID, aniIndex int) {
	w.broadcastToMap(p, protocol.ServerAnimation, protocol.Animation{
		AttackerID:    p.ID,
		AttackerType:  0,
		AniIndex:      uint8(clampInt(aniIndex, 0, 255)),
		CharacterID:   p.ID,
		CharacterType: 1,
		AnimationID:   uint16(clampInt(animationID, 0, 65535)),
	})
}

func (w *World) applyParamBonus(p *Player, paramID, value int) {
	if value <= 0 {
		return
	}
	switch paramID {
	case 0:
		p.Character.MHP += value
		p.Character.HP += value
	case 1:
		p.Character.MMP += value
		p.Character.MP += value
	case 2:
		p.Character.ATK += value
	case 3:
		p.Character.DEF += value
	case 4:
		p.Character.MAT += value
	case 5:
		p.Character.MDF += value
	case 6:
		p.Character.AGI += value
	case 7:
		p.Character.LUK += value
	}
	w.broadcastToMap(p, protocol.ServerParam, protocol.Param{PlayerID: p.ID, ParamID: uint8(paramID), Value: int32(value)})
}

func (w *World) recoverVitals(p *Player) {
	if p.Character.HP >= p.Character.MHP && p.Character.MP >= p.Character.MMP {
		return
	}
	n := p.Character.AGI/100 + 1
	bonus := 1.0
	if p.vipActive(w.now().Unix()) {
		bonus = vipRecoverBonus
	}
	hpGain := int(math.Round(float64(recoverHPAmount) * bonus * float64(n)))
	mpGain := int(math.Round(float64(recoverMPAmount) * bonus * float64(n)))
	p.Character.HP = clampInt(p.Character.HP+hpGain, 0, p.Character.MHP)
	p.Character.MP = clampInt(p.Character.MP+mpGain, 0, p.Character.MMP)
	w.sendVitalsToMap(p)
}

// tickCombat is the compatibility wrapper for tests and callers already on the
// world goroutine. Map actors call tickCombatMap directly.
func (w *World) tickCombat() {
	now := w.now()
	for _, ms := range w.maps {
		if ms != nil {
			w.tickCombatMap(ms, now)
		}
	}
}

// tickCombatMap advances one map's players and enemies. The map actor calls it
// with the world read lock held.
func (w *World) tickCombatMap(ms *mapState, now time.Time) {
	for _, p := range ms.players {
		if p == nil {
			continue
		}
		if p.Character.HP <= 0 && !p.respawnAt.IsZero() && now.After(p.respawnAt) {
			w.clearStatus(&p.Status, p)
			p.Character.HP = p.Character.MHP
			p.Character.MP = p.Character.MMP
			p.respawnAt = time.Time{}
			w.sendVitalsToMap(p)
		}
		if p.Character.HP > 0 {
			if p.nextRecover.IsZero() {
				p.nextRecover = now.Add(recoverInterval)
			} else if !now.Before(p.nextRecover) {
				p.nextRecover = now.Add(recoverInterval)
				w.recoverVitals(p)
			}
			w.tickStatus(now, &p.Status, p, w.playerFeatures(p), p.Character.MHP, p.Character.MMP, &p.Character.HP, &p.Character.MP)
		}
	}
	for _, enemy := range ms.enemies {
		if enemy == nil {
			continue
		}
		if !enemy.Alive {
			if !enemy.ReviveAt.IsZero() && now.After(enemy.ReviveAt) {
				w.reviveEnemy(ms, enemy, now)
			}
			continue
		}
		w.tickStatus(now, &enemy.Status, nil, w.enemyFeatures(enemy), enemy.Enemy.Params[0], enemy.Enemy.Params[1], &enemy.HP, &enemy.MP)
		w.updateEnemyAI(ms, enemy, now)
	}
}

// moveEnemy steps an enemy toward the requested direction.
func (w *World) moveEnemy(ms *mapState, enemy *enemyState, direction int) bool {
	dx, dy := dirDelta(direction)
	nx, ny := enemy.X+dx, enemy.Y+dy
	enemy.Direction = direction
	if ms.valid(nx, ny) && (enemy.through || (ms.passable(nx, ny, uint8(direction)) && !w.npcBlocked(ms, enemy.Event.ID, nx, ny))) {
		enemy.X, enemy.Y = nx, ny
		w.broadcastEventMove(ms, enemy.Event.ID, enemy.X, enemy.Y, enemy.Direction)
		return true
	}
	w.broadcastEventMove(ms, enemy.Event.ID, enemy.X, enemy.Y, enemy.Direction)
	return false
}

func (w *World) enemyAttack(enemy *enemyState, target *Player) {
	skill := w.attackSkill()
	if skill == nil {
		return
	}
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
		AttackerID:   uint16(enemy.Event.ID),
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
		w.applyAttackStates(w.enemyFeatures(enemy), w.playerFeatures(target), &target.Status, target)
	}
	if target.Character.HP <= 0 {
		w.playerDied(target)
	}
}

func (w *World) sendVitalsToMap(p *Player) {
	msg := protocol.Vitals{PlayerID: p.ID, HP: int32(p.Character.HP), MP: int32(p.Character.MP)}
	w.broadcastToMap(p, protocol.ServerVitals, msg)
}

func respawnDelay(enemy *rpg.Enemy) time.Duration {
	if enemy != nil && enemy.ReviveTime > 0 {
		return time.Duration(enemy.ReviveTime) * time.Second
	}
	return enemyDefaultRespawn
}

func isAdmin(p *Player) bool {
	return p != nil && p.Account != nil && p.Account.GroupID >= 2
}

func frontTile(x, y, direction int) (int, int) {
	switch direction {
	case 2:
		return x, y + 1
	case 4:
		return x - 1, y
	case 6:
		return x + 1, y
	case 8:
		return x, y - 1
	}
	return x, y
}

// battleStats exposes the fields used by RPG Maker damage formulas.
type battleStats struct {
	Level          int
	HP, MP         int
	MHP            int
	MMP            int
	Params         [8]int
	Vars           []int32
	Hit            float64
	Eva            float64
	Cri            float64
	Counter        float64
	HRG            float64
	MRG            float64
	ElementRates   [11]float64
	AttackElements []int
}

func (w *World) playerStats(p *Player) battleStats {
	c := p.Character
	s := battleStats{
		Level: c.Level, HP: c.HP, MP: c.MP, MHP: c.MHP, MMP: c.MMP,
		Params: [8]int{c.MHP, c.MMP, c.ATK, c.DEF, c.MAT, c.MDF, c.AGI, c.LUK},
		Vars:   p.State.Variables,
	}
	for slot, id := range p.State.Equips {
		if id <= 0 {
			continue
		}
		var params [8]int
		if slot == 0 {
			if weapon := w.weapon(id); weapon != nil {
				params = weapon.Params
			}
		} else if armor := w.armor(id); armor != nil {
			params = armor.Params
		}
		for i := range s.Params {
			s.Params[i] += params[i]
		}
	}
	features := w.playerFeatures(p)
	applyParamRates(&s, features, &p.Status.Buffs)
	applyXParams(&s, features)
	applyElementRates(&s, features)
	s.MHP, s.MMP = s.Params[0], s.Params[1]
	return s
}

func (w *World) enemyStats(enemy *enemyState) battleStats {
	s := battleStats{
		Level: 1, HP: enemy.HP, MP: enemy.MP,
		Params: enemy.Enemy.Params,
	}
	features := w.enemyFeatures(enemy)
	applyParamRates(&s, features, &enemy.Status.Buffs)
	applyXParams(&s, features)
	applyElementRates(&s, features)
	s.MHP, s.MMP = s.Params[0], s.Params[1]
	return s
}

func rollDamage(user, target battleStats, damage rpg.Damage, vars []int32) (int, bool, bool) {
	if damage.Type != 1 {
		return 0, false, false
	}
	if rand.Float64() > user.Hit {
		return 0, false, true
	}
	if rand.Float64() < target.Eva {
		return 0, false, true
	}
	value := evalFormula(damage.Formula, user, target, vars)
	if rate := elementRate(user, target, damage.ElementID); rate != 1 {
		value *= rate
	}
	critical := damage.Critical && rand.Float64() < user.Cri
	if critical {
		value *= 3
	}
	amp := math.Abs(value) * float64(damage.Variance) / 100
	value += (rand.Float64()*2 - 1) * amp
	if value < 0 {
		value = 0
	}
	return int(value), critical, false
}

func (w *World) awardKill(p *Player, enemy *enemyState) {
	if enemy == nil || enemy.Enemy == nil || p == nil {
		return
	}
	members := w.partyMembersOnMap(p)
	gold := 0
	if enemy.Enemy.Gold > 0 {
		gold = rand.IntN(enemy.Enemy.Gold + 1)
	}
	if len(members) > 1 {
		w.sharePartyExp(p, members, enemy.Enemy.Exp)
		w.sharePartyGold(p, members, gold)
		for _, member := range members {
			w.addQuestKills(member, enemy.ID)
			w.saveQuietly(member)
		}
	} else {
		if enemy.Enemy.Exp > 0 {
			w.gainExp(p, enemy.Enemy.Exp)
		}
		w.gainGold(p, gold)
		w.addQuestKills(p, enemy.ID)
	}
	ms := p.ms
	for _, drop := range enemy.Enemy.DropItems {
		if drop.Kind < 1 || drop.Kind > 3 || drop.DataID <= 0 {
			continue
		}
		denominator := drop.Denominator
		if denominator <= 0 {
			continue
		}
		if rand.IntN(denominator) != 0 {
			continue
		}
		if ms == nil || !w.addMapDrop(ms, drop.Kind, drop.DataID, 1, enemy.X, enemy.Y, p.Character.Name, p.PartyID) {
			break
		}
	}
	w.saveQuietly(p)
}

func (w *World) gainExp(p *Player, amount int) {
	if amount <= 0 || p.Character.ClassID <= 0 || p.Character.ClassID >= len(w.opts.Data.Classes) {
		return
	}
	class := w.opts.Data.Classes[p.Character.ClassID]
	if class == nil {
		return
	}
	maxLevel := w.opts.MaxLevel
	if maxLevel <= 0 {
		maxLevel = 99
	}
	oldLevel := p.Character.Level
	p.Character.Exp += amount
	if p.Character.Exp > class.ExpForLevel(maxLevel) {
		p.Character.Exp = class.ExpForLevel(maxLevel)
	}
	level := p.Character.Level
	for level < maxLevel && p.Character.Exp >= class.ExpForLevel(level+1) {
		level++
	}
	if level != p.Character.Level {
		w.applyLevelUp(p, class, p.Character.Level, level)
		p.Character.Level = level
	}
	_ = p.Client.Send(protocol.ServerExp, protocol.Exp{PlayerID: p.ID, Amount: int32(amount)})
	_ = oldLevel
}

func (w *World) applyLevelUp(p *Player, class *rpg.Class, oldLevel, newLevel int) {
	hpGain, mpGain := 0, 0
	for paramID := 0; paramID < 8; paramID++ {
		delta := class.ParamAt(paramID, newLevel) - class.ParamAt(paramID, oldLevel)
		if delta == 0 {
			continue
		}
		switch paramID {
		case 0:
			p.Character.MHP += delta
			p.Character.HP += delta
			hpGain += delta
		case 1:
			p.Character.MMP += delta
			p.Character.MP += delta
			mpGain += delta
		case 2:
			p.Character.ATK += delta
		case 3:
			p.Character.DEF += delta
		case 4:
			p.Character.MAT += delta
		case 5:
			p.Character.MDF += delta
		case 6:
			p.Character.AGI += delta
		case 7:
			p.Character.LUK += delta
		}
	}
	for _, learn := range class.Learnings {
		if learn.Level > oldLevel && learn.Level <= newLevel && !hasSkill(p.State, learn.SkillID) {
			p.State.Skills = append(p.State.Skills, learn.SkillID)
			_ = p.Client.Send(protocol.ServerSkill, protocol.Skill{SkillID: uint16(clampInt(learn.SkillID, 0, 65535)), Learned: true})
		}
	}
	if hpGain != 0 || mpGain != 0 {
		w.sendVitalsToMap(p)
	}
}

func (w *World) broadcastToMapPlayers(ms *mapState, op protocol.ServerOp, msg protocol.ServerMessage) {
	for _, p := range ms.players {
		if p == nil {
			continue
		}
		_ = p.Client.Send(op, msg)
	}
}

var ErrInvalidTarget = errors.New("game: invalid target")
