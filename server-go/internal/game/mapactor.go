package game

import (
	"errors"
	"sync"
	"time"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/store"
)

// ErrStaleRoute means a command was addressed to a map that no longer owns the
// player, usually because a transfer completed after the route lookup. The
// caller should refresh the route and retry once.
var ErrStaleRoute = errors.New("game: stale player route")

// MapKey identifies one runtime map. TemplateID is the RPG Maker map id;
// InstanceID is 0 for the open world and will identify dungeon copies later.
type MapKey struct {
	TemplateID int
	InstanceID int
}

func (k MapKey) zero() bool { return k.TemplateID <= 0 }

// mapMsg is one unit of work for a map actor. playerID/epoch are checked before
// run executes. Control messages set playerID=0.
type mapMsg struct {
	playerID uint16
	epoch    uint64
	run      func(*mapActor, *Player) error
	reply    chan error
}

// mapActor owns one map runtime. Every field below the queue is accessed only
// from the actor goroutine.
type mapActor struct {
	w   *World
	key MapKey

	mu    sync.Mutex
	queue []mapMsg
	wake  chan struct{}

	ms        *mapState
	aoi       *aoiState
	stop      chan struct{}
	done      chan struct{}
	players   []*Player
	idleSince time.Time
	gen       uint64 // last attach generation applied by this actor
	lastSave  time.Time
}

func newMapActor(w *World, key MapKey, ms *mapState) *mapActor {
	a := &mapActor{
		w:        w,
		key:      key,
		ms:       ms,
		wake:     make(chan struct{}, 1),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		players:  make([]*Player, 0),
		lastSave: w.now(),
	}
	ms.actor = a
	if w.opts.AOIRadius > 0 {
		a.aoi = newAOI(a)
	}
	return a
}

// post enqueues work without blocking the caller.
func (a *mapActor) post(msg mapMsg) {
	a.mu.Lock()
	a.queue = append(a.queue, msg)
	a.mu.Unlock()
	select {
	case a.wake <- struct{}{}:
	default:
	}
}

func (a *mapActor) drain() []mapMsg {
	a.mu.Lock()
	q := a.queue
	a.queue = nil
	a.mu.Unlock()
	return q
}

func (a *mapActor) start() {
	go a.run()
}

func (a *mapActor) run() {
	defer close(a.done)
	tick := time.NewTicker(a.w.opts.Tick)
	defer tick.Stop()
	for {
		select {
		case <-a.wake:
			for _, msg := range a.drain() {
				a.exec(msg)
			}
		case <-tick.C:
			a.tick()
		case <-a.stop:
			a.flushAll()
			return
		}
	}
}

func (a *mapActor) exec(msg mapMsg) {
	var p *Player
	if msg.playerID != 0 {
		p = a.playerByID(msg.playerID)
		if p == nil {
			if msg.reply != nil {
				msg.reply <- ErrNotInGame
			}
			return
		}
		if msg.epoch != 0 && p.Epoch != msg.epoch {
			if msg.reply != nil {
				msg.reply <- ErrStaleRoute
			}
			return
		}
	}
	var err error
	if msg.run != nil {
		err = msg.run(a, p)
	}
	if msg.reply != nil {
		msg.reply <- err
	}
}

func (a *mapActor) playerByID(id uint16) *Player {
	if int(id) < len(a.players) {
		return a.players[id]
	}
	// The slice may be shorter than the id when IDs have holes; fall back to a
	// short scan for the rare high-ID case.
	for _, p := range a.players {
		if p != nil && p.ID == id {
			return p
		}
	}
	return nil
}

func (a *mapActor) putPlayer(p *Player) {
	a.players = ensureIndex(a.players, int(p.ID))
	a.players[p.ID] = p
	a.ms.players = ensureIndex(a.ms.players, int(p.ID))
	a.ms.players[p.ID] = p
	p.ms = a.ms
	p.actor = a
}

func (a *mapActor) dropPlayer(id uint16) {
	if int(id) < len(a.players) {
		a.players[id] = nil
	}
	if int(id) < len(a.ms.players) {
		a.ms.players[id] = nil
	}
}

func (a *mapActor) hasPlayers() bool {
	for _, p := range a.players {
		if p != nil {
			return true
		}
	}
	return false
}

func (a *mapActor) tick() {
	now := a.w.now()
	if !a.hasPlayers() {
		if a.idleSince.IsZero() {
			a.idleSince = now
		}
		if now.Sub(a.idleSince) >= a.w.opts.MapIdle {
			a.exec(mapMsg{run: func(a *mapActor, _ *Player) error {
				a.sleep()
				return nil
			}})
		}
		return
	}
	a.idleSince = time.Time{}
	a.w.tickCombatMap(a.ms, now)
	a.w.tickEventsMap(a.ms, now)
	a.w.tickAutorunsMap(a.ms, now)
	a.w.tickNPCsMap(a.ms, now)
	a.w.tickDropsMap(a.ms, now)
	if now.Sub(a.lastSave) >= a.w.opts.SaveInterval {
		a.lastSave = now
		a.flushAll()
	}
}

// sleep tells the coordinator that this actor has been empty for the grace
// period. The coordinator decides whether it can be removed.
func (a *mapActor) sleep() {
	gen := a.gen
	key := a.key
	a.w.postCoordinator(func() {
		a.w.maybeUnloadMap(key, a, gen)
	})
}

// maybeUnloadMap runs on the coordinator. gen guards against an attach that was
// already posted to the actor before the idle notification was processed.
func (w *World) maybeUnloadMap(key MapKey, a *mapActor, gen uint64) {
	cur := w.actors[key]
	if cur != a || w.attachGen[key] != gen {
		return
	}
	delete(w.actors, key)
	delete(w.attachGen, key)
	close(a.stop)
	w.log.Info("map unloaded", "map", key.TemplateID, "instance", key.InstanceID)
}

func (a *mapActor) flushAll() {
	now := a.w.now()
	for _, p := range a.players {
		if p == nil {
			continue
		}
		a.w.queueSave(p)
	}
	a.lastSave = now
}

// playerSnapshot is the immutable value that crosses actor boundaries. It must
// never contain a *Player or *mapState.
type playerSnapshot struct {
	ID      uint16
	Epoch   uint64
	Account *store.Account
	Client  Client
	IP      string

	GuildName     string
	Friends       []string
	PartyID       int
	MuteUntil     time.Time
	CreatingGuild bool
	Request       *pendingRequest

	Character store.Character
	State     store.CharacterState
	Bank      *store.BankState
	BankOpen  bool

	Status statusSet

	LastMove    time.Time
	LastChat    time.Time
	LastAttack  time.Time
	RespawnAt   time.Time
	NextRecover time.Time
	TargetType  uint8
	TargetID    uint16

	OriginalCharacterName  string
	OriginalCharacterIndex int
	OriginalFaceName       string
	OriginalFaceIndex      int
	TeleportGroup          int

	ShopGoods        []ShopGood
	ShopSellDisabled bool
	ShopEventID      int
	ShopIndex        int
	ErasedEvents     []bool

	Trade *tradeSnapshot
}

type tradeSnapshot struct {
	Partner uint16
	Items   map[tradeKey]int
	Gold    int64
}

func snapshotPlayer(p *Player, epoch uint64) *playerSnapshot {
	if p == nil {
		return nil
	}
	s := &playerSnapshot{
		ID:      p.ID,
		Epoch:   epoch,
		Account: p.Account,
		Client:  p.Client,
		IP:      p.IP,

		GuildName:     p.GuildName,
		Friends:       append([]string(nil), p.Friends...),
		PartyID:       p.PartyID,
		MuteUntil:     p.MuteUntil,
		CreatingGuild: p.CreatingGuild,
		Request:       cloneRequest(p.Request),

		Character: *p.Character,
		State:     *cloneCharacterState(p.State),
		BankOpen:  p.BankOpen,

		Status: cloneStatus(p.Status),

		LastMove:    p.lastMove,
		LastChat:    p.lastChat,
		LastAttack:  p.lastAttack,
		RespawnAt:   p.respawnAt,
		NextRecover: p.nextRecover,
		TargetType:  p.TargetType,
		TargetID:    p.TargetID,

		OriginalCharacterName:  p.OriginalCharacterName,
		OriginalCharacterIndex: p.OriginalCharacterIndex,
		OriginalFaceName:       p.OriginalFaceName,
		OriginalFaceIndex:      p.OriginalFaceIndex,
		TeleportGroup:          p.TeleportGroup,

		ShopGoods:        append([]ShopGood(nil), p.ShopGoods...),
		ShopSellDisabled: p.ShopSellDisabled,
		ShopEventID:      p.ShopEventID,
		ShopIndex:        p.ShopIndex,
		ErasedEvents:     append([]bool(nil), p.ErasedEvents...),
	}
	if p.Bank != nil {
		s.Bank = cloneBankState(p.Bank)
	}
	if p.Trade != nil {
		s.Trade = &tradeSnapshot{Partner: p.Trade.Partner, Gold: p.Trade.Gold, Items: make(map[tradeKey]int, len(p.Trade.Items))}
		for k, v := range p.Trade.Items {
			s.Trade.Items[k] = v
		}
	}
	return s
}

func (s *playerSnapshot) materialize() *Player {
	if s == nil {
		return nil
	}
	character := s.Character
	state := cloneCharacterState(&s.State)
	p := &Player{
		ID:        s.ID,
		Account:   s.Account,
		Client:    s.Client,
		IP:        s.IP,
		Character: &character,
		State:     state,
		GuildName: s.GuildName,
		Friends:   append([]string(nil), s.Friends...),
		PartyID:   s.PartyID,
		MuteUntil: s.MuteUntil,

		lastMove:    s.LastMove,
		lastChat:    s.LastChat,
		lastAttack:  s.LastAttack,
		respawnAt:   s.RespawnAt,
		nextRecover: s.NextRecover,
		TargetType:  s.TargetType,
		TargetID:    s.TargetID,

		OriginalCharacterName:  s.OriginalCharacterName,
		OriginalCharacterIndex: s.OriginalCharacterIndex,
		OriginalFaceName:       s.OriginalFaceName,
		OriginalFaceIndex:      s.OriginalFaceIndex,
		TeleportGroup:          s.TeleportGroup,

		ShopGoods:        append([]ShopGood(nil), s.ShopGoods...),
		ShopSellDisabled: s.ShopSellDisabled,
		ShopEventID:      s.ShopEventID,
		ShopIndex:        s.ShopIndex,
		BankOpen:         s.BankOpen,
		ErasedEvents:     append([]bool(nil), s.ErasedEvents...),
		CreatingGuild:    s.CreatingGuild,
		Request:          cloneRequest(s.Request),
		Status:           cloneStatus(s.Status),
		Epoch:            s.Epoch,
	}
	if s.Bank != nil {
		p.Bank = cloneBankState(s.Bank)
	}
	if s.Trade != nil {
		p.Trade = &tradeState{Partner: s.Trade.Partner, Gold: s.Trade.Gold, Items: make(map[tradeKey]int, len(s.Trade.Items))}
		for k, v := range s.Trade.Items {
			p.Trade.Items[k] = v
		}
	}
	return p
}

func cloneRequest(r *pendingRequest) *pendingRequest {
	if r == nil {
		return nil
	}
	out := *r
	return &out
}

func cloneStatus(s statusSet) statusSet {
	out := s
	out.States = append([]int(nil), s.States...)
	return out
}

func cloneBankState(b *store.BankState) *store.BankState {
	if b == nil {
		return nil
	}
	out := *b
	out.Items = append([]store.ItemStack(nil), b.Items...)
	return &out
}

// attachPayload is the control message that installs a player in a map actor.
type attachPayload struct {
	snap    *playerSnapshot
	enter   bool
	epoch   uint64
	gen     uint64
	motd    string
	globals []bool
}

func (a *mapActor) attach(payload attachPayload) {
	snap := payload.snap
	if snap == nil {
		return
	}
	if !a.ms.valid(snap.Character.X, snap.Character.Y) {
		a.w.postCoordinator(func() { a.w.attachFailed(snap, "invalid position") })
		return
	}
	a.gen = payload.gen
	snap.Epoch = payload.epoch
	p := snap.materialize()
	p.Epoch = payload.epoch
	p.MapKey = a.key
	a.putPlayer(p)
	a.ms.globalSwitches = append([]bool(nil), payload.globals...)

	if payload.enter {
		if err := p.Client.Send(protocol.ServerEnterWorld, a.w.enterWorld(p)); err != nil {
			a.dropPlayer(p.ID)
			a.w.postCoordinator(func() {
				a.w.removeOnlineByID(p.ID)
				_ = p.Client.Close()
			})
			return
		}
		a.w.sendGlobalSwitches(p)
		a.w.sendMOTDText(p, payload.motd)
		if a.aoi != nil {
			_ = p.Client.Send(protocol.ServerMapReset, protocol.MapReset{})
		}
		a.aoiAttachPlayer(p, true)
		a.w.sendMapRoster(p, a.ms)
		a.w.broadcastMapPlayerEnter(a.ms, p)
		a.w.log.Info("player joined", "player", p.ID, "character", p.Character.Name, "map", a.key.TemplateID, "x", p.Character.X, "y", p.Character.Y)
		return
	}

	_ = p.Client.Send(protocol.ServerTransfer, protocol.Transfer{
		MapID: uint16(p.Character.MapID), X: uint16(p.Character.X), Y: uint16(p.Character.Y), Direction: uint8(p.Character.Direction),
	})
	a.w.sendGlobalSwitches(p)
	if a.aoi != nil {
		_ = p.Client.Send(protocol.ServerMapReset, protocol.MapReset{})
	}
	a.aoiAttachPlayer(p, true)
	a.w.sendMapRoster(p, a.ms)
	a.w.broadcastMapPlayerEnter(a.ms, p)
}

// detach removes one player and returns an immutable snapshot. It runs on the
// actor goroutine.
func (a *mapActor) detach(id uint16, save bool) *playerSnapshot {
	p := a.playerByID(id)
	if p == nil {
		return nil
	}
	if p.Trade != nil {
		a.w.closeTradeLocked(p)
	}
	p.TargetType, p.TargetID = 0, 0
	p.Event = nil
	snap := snapshotPlayer(p, p.Epoch)
	a.w.broadcastMapPlayerLeave(a.ms, p)
	if a.aoi != nil {
		a.aoiDetachPlayer(p)
	} else {
		for _, other := range a.ms.players {
			if other == nil || other == p {
				continue
			}
			_ = other.Client.Send(protocol.ServerPlayerLeft, protocol.PlayerLeft{ID: p.ID})
		}
	}
	a.dropPlayer(id)
	p.ms = nil
	p.actor = nil
	if save {
		a.w.queueSave(p)
	}
	return snap
}
