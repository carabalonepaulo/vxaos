// Package game owns the authoritative world state. A single goroutine owns all
// mutable game data; network sessions submit intents and receive packets.
package game

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kelaia/vxaos/server/internal/content"
	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

var (
	ErrClosed        = errors.New("game: world closed")
	ErrAlreadyOnline = errors.New("game: character already online")
	ErrNotInGame     = errors.New("game: player is not in the world")
	ErrRateLimited   = errors.New("game: rate limited")
	ErrUnsupported   = errors.New("game: unsupported")
	ErrInvalidMove   = errors.New("game: invalid move")
	ErrMuted         = errors.New("game: muted")
)

// Client is the packet sink for one session. Send must not block on the
// network; the net session enqueues into its writer.
type Client interface {
	Send(op protocol.ServerOp, msg protocol.ServerMessage) error
	Close() error
}

// Options configures a World.
type Options struct {
	Data               *rpg.Data
	Store              *store.Store
	Tick               time.Duration
	SaveInterval       time.Duration
	MaxEquips          int
	MaxHotbar          int
	MaxItems           int
	MaxBankItems       int
	MaxStack           int
	MaxGold            int64
	MaxLevel           int
	MaxPlayerSwitches  int
	MaxPlayerVariables int
	MaxGuildMembers    int
	MaxPartyMembers    int
	MaxFriends         int
	MaxMapDrops        int
	DropDespawnSeconds int
	DropPickupSeconds  int
	MapIdle            time.Duration
	InlineMaps         bool
	AOIRadius          int
	AOIHysteresis      int
	AOICellSize        int
	ProjectileSkills   []int
	Teleports          []Teleport
	Quests             []content.Quest
	Logger             *slog.Logger
	Now                func() time.Time
}

// Teleport is one event-driven teleport destination.
type Teleport struct {
	Group int
	MapID int
	X     int
	Y     int
	Gold  int
}

// Player is the runtime state of one logged-in character. While attached, it is
// owned by exactly one map actor. The network session only keeps the player id
// and never reads these fields.
type Player struct {
	ID        uint16
	Account   *store.Account
	Character *store.Character
	State     *store.CharacterState
	Client    Client
	GuildName string
	IP        string
	MuteUntil time.Time

	Epoch  uint64
	MapKey MapKey
	ms     *mapState
	actor  *mapActor

	lastMove    time.Time
	lastChat    time.Time
	lastAttack  time.Time
	respawnAt   time.Time
	nextRecover time.Time

	TargetType uint8
	TargetID   uint16
	Event      *eventRun

	OriginalCharacterName  string
	OriginalCharacterIndex int
	OriginalFaceName       string
	OriginalFaceIndex      int
	TeleportGroup          int
	ShopGoods              []ShopGood
	ShopSellDisabled       bool
	ShopEventID            int
	ShopIndex              int
	Bank                   *store.BankState
	BankOpen               bool
	ErasedEvents           []bool
	CreatingGuild          bool
	Request                *pendingRequest
	PartyID                int
	Friends                []string
	Trade                  *tradeState
	Status                 statusSet
}

type tradeKey struct {
	Kind int
	ID   int
}

type tradeState struct {
	Partner uint16
	Items   map[tradeKey]int
	Gold    int64
}

type pendingRequest struct {
	Type   uint8
	FromID uint16
}

// World is the coordinator. It owns the online directory, social state, global
// switches and the map registry. Map runtime state belongs to map actors.
type World struct {
	opts Options
	log  *slog.Logger
	now  func() time.Time

	commands   chan func()
	stop       chan struct{}
	stopped    chan struct{}
	stopOnce   sync.Once
	actorWG    sync.WaitGroup
	routeCache sync.Map // player id -> playerRoute

	// All fields below are owned by the coordinator goroutine.
	online         map[uint16]*onlineEntry
	byCharacter    map[int64]*onlineEntry
	actors         map[MapKey]*mapActor
	attachGen      map[MapKey]uint64
	maps           []*mapState // inline mode only
	guilds         map[int64]*guildState
	guildByName    map[string]*guildState
	parties        map[int]*partyState
	nextPartyID    int
	nextID         uint16
	nextEpoch      uint64
	globalSwitches []bool
	motd           string
	persist        *persistWorker
}

// onlineEntry is the coordinator's authoritative view of one online player.
// It never holds map runtime state.
type onlineEntry struct {
	ID            uint16
	CharacterID   int64
	Name          string
	Account       *store.Account
	Client        Client
	IP            string
	Key           MapKey
	Epoch         uint64
	GuildName     string
	PartyID       int
	Friends       []string
	MuteUntil     time.Time
	CreatingGuild bool
	Request       *pendingRequest
	LastChat      time.Time
	View          partyView
}

// partyView is the coordinator-side display snapshot used for party join
// packets. It is refreshed on join and transfer.
type partyView struct {
	ID             uint16
	Name           string
	CharacterName  string
	CharacterIndex int
	Sex            int
	Level          int
	ClassID        int
	HP             int
	MHP            int
	EXP            int
	Equips         [protocol.MaxEquips]uint16
}

type mapState struct {
	key            MapKey
	actor          *mapActor
	def            *rpg.Map
	tileset        *rpg.Tileset
	events         []*rpg.Event
	npcs           []*npcState
	enemies        []*enemyState
	players        []*Player
	autorun        *autorunState
	drops          []*dropState
	dropSlab       dropSlab
	globalSwitches []bool
}

// npcState is the runtime position and autonomous movement of a non-enemy map
// event. Page conditions are evaluated against global switches only; NPCs with
// player-specific pages stay fixed.
type npcState struct {
	event          *rpg.Event
	page           *rpg.Page
	x, y           int
	direction      int
	priority       int
	trigger        int
	through        bool
	directionFix   bool
	fixedDirection int
	moveType       int
	moveSpeed      int
	moveFrequency  int
	nextMove       time.Time
	route          *rpg.MoveRoute
	routeIndex     int
}

// autorunState is one map-level parallel/autorun process. These pages may
// only depend on global switches, so a single runner per map is enough.
type autorunState struct {
	eventID   int
	run       *eventRun
	waitUntil time.Time
}

// New creates and starts a world.
func New(opts Options) (*World, error) {
	if opts.Data == nil {
		return nil, fmt.Errorf("game: nil data")
	}
	if opts.Store == nil {
		return nil, fmt.Errorf("game: nil store")
	}
	if opts.Tick <= 0 {
		opts.Tick = 80 * time.Millisecond
	}
	if opts.SaveInterval <= 0 {
		opts.SaveInterval = 15 * time.Minute
	}
	if opts.MaxEquips <= 0 {
		opts.MaxEquips = protocol.MaxEquips
	}
	if opts.MaxHotbar <= 0 {
		opts.MaxHotbar = 9
	}
	if opts.MaxItems <= 0 {
		opts.MaxItems = 30
	}
	if opts.MaxBankItems <= 0 {
		opts.MaxBankItems = 42
	}
	if opts.MaxStack <= 0 {
		opts.MaxStack = 999
	}
	if opts.MaxGold <= 0 {
		opts.MaxGold = 99999999
	}
	if opts.MaxPlayerSwitches <= 0 {
		opts.MaxPlayerSwitches = 100
	}
	if opts.MaxPlayerVariables <= 0 {
		opts.MaxPlayerVariables = 100
	}
	if opts.MaxGuildMembers <= 0 {
		opts.MaxGuildMembers = 50
	}
	if opts.MaxPartyMembers <= 0 {
		opts.MaxPartyMembers = 4
	}
	if opts.MaxFriends <= 0 {
		opts.MaxFriends = 20
	}
	if opts.MaxMapDrops <= 0 {
		opts.MaxMapDrops = 20
	}
	if opts.DropDespawnSeconds <= 0 {
		opts.DropDespawnSeconds = 60
	}
	if opts.DropPickupSeconds < 0 {
		opts.DropPickupSeconds = 0
	}
	if opts.DropPickupSeconds == 0 {
		opts.DropPickupSeconds = 10
	}
	if len(opts.ProjectileSkills) == 0 {
		opts.ProjectileSkills = []int{51, 70}
	}
	if opts.MaxLevel <= 0 {
		opts.MaxLevel = 99
	}
	if opts.MapIdle <= 0 {
		opts.MapIdle = 60 * time.Second
	}
	if opts.AOIRadius < 0 {
		opts.AOIRadius = 0
	}
	if opts.AOIRadius > 0 && opts.AOIHysteresis <= 0 {
		opts.AOIHysteresis = 2
	}
	if opts.AOICellSize <= 0 {
		opts.AOICellSize = 8
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	w := &World{
		opts:        opts,
		log:         opts.Logger,
		now:         opts.Now,
		commands:    make(chan func(), 4096),
		stop:        make(chan struct{}),
		stopped:     make(chan struct{}),
		online:      make(map[uint16]*onlineEntry),
		byCharacter: make(map[int64]*onlineEntry),
		actors:      make(map[MapKey]*mapActor),
		attachGen:   make(map[MapKey]uint64),
		guilds:      make(map[int64]*guildState),
		guildByName: make(map[string]*guildState),
		parties:     make(map[int]*partyState),
		nextPartyID: 1,
		nextID:      1,
		nextEpoch:   1,
	}
	if opts.InlineMaps {
		w.maps = make([]*mapState, len(opts.Data.Maps))
	}
	w.loadGlobalSwitches()
	if motd, err := opts.Store.GetServerState(context.Background(), "motd"); err == nil {
		w.motd = motd
	}
	w.loadGuilds()
	w.persist = newPersistWorker(opts.Store, opts.Logger)
	// Map runtime state is created on demand by ensureMap when the first
	// player enters a map.
	go w.loop()
	return w, nil
}

// Stop stops the coordinator and all map actors after saving online players.
func (w *World) Stop(ctx context.Context) error {
	w.stopOnce.Do(func() { close(w.stop) })
	select {
	case <-w.stopped:
	case <-ctx.Done():
		return ctx.Err()
	}
	if w.opts.InlineMaps {
		w.saveInline()
	} else {
		for key, a := range w.actors {
			delete(w.actors, key)
			close(a.stop)
		}
	}
	done := make(chan struct{})
	go func() {
		w.actorWG.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}
	if w.persist != nil {
		if err := w.persist.closeAndFlush(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Submit runs fn on the world goroutine.
func (w *World) Submit(ctx context.Context, fn func()) error {
	if fn == nil {
		return nil
	}
	select {
	case w.commands <- fn:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stop:
		return ErrClosed
	}
}

func (w *World) do(ctx context.Context, fn func() error) error {
	result := make(chan error, 1)
	select {
	case w.commands <- func() { result <- fn() }:
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stop:
		return ErrClosed
	}
	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stop:
		return ErrClosed
	}
}

// postCoordinator enqueues coordinator work without waiting. It is used by map
// actors, which must never block the coordinator.
func (w *World) postCoordinator(fn func()) {
	if fn == nil {
		return
	}
	select {
	case w.commands <- fn:
	case <-w.stop:
	}
}

// coordErr runs fn on the coordinator. Map actors use it for social/global
// state; inline mode already runs map commands on the coordinator goroutine.
func (w *World) coordErr(fn func() error) error {
	if w.opts.InlineMaps {
		return fn()
	}
	result := make(chan error, 1)
	select {
	case w.commands <- func() { result <- fn() }:
	case <-w.stop:
		return ErrClosed
	}
	select {
	case err := <-result:
		return err
	case <-w.stop:
		return ErrClosed
	}
}

// playerRoute is the immutable answer to "where is this player now".
type playerRoute struct {
	key   MapKey
	epoch uint64
	actor *mapActor
}

func (w *World) lookupRoute(ctx context.Context, id uint16) (playerRoute, error) {
	var route playerRoute
	err := w.do(ctx, func() error {
		entry := w.online[id]
		if entry == nil {
			return ErrNotInGame
		}
		route = playerRoute{key: entry.Key, epoch: entry.Epoch, actor: w.actors[entry.Key]}
		if route.actor == nil {
			return ErrNotInGame
		}
		return nil
	})
	return route, err
}

// onPlayer runs fn with exclusive access to the player on the map actor that
// currently owns it.
func (w *World) onPlayer(ctx context.Context, id uint16, fn func(*mapState, *Player) error) error {
	return w.onPlayerRetry(ctx, id, fn, 1)
}

func (w *World) onPlayerRetry(ctx context.Context, id uint16, fn func(*mapState, *Player) error, retries int) error {
	route, ok := w.cachedRoute(id)
	if !ok {
		var err error
		route, err = w.lookupRoute(ctx, id)
		if err != nil {
			return err
		}
		w.storeRoute(id, route)
	}
	if w.opts.InlineMaps {
		return w.do(ctx, func() error {
			p := route.actor.playerByID(id)
			if p == nil {
				return ErrNotInGame
			}
			if p.Epoch != route.epoch {
				return ErrStaleRoute
			}
			return fn(route.actor.ms, p)
		})
	}
	reply := make(chan error, 1)
	route.actor.post(mapMsg{playerID: id, epoch: route.epoch, run: func(a *mapActor, p *Player) error { return fn(a.ms, p) }, reply: reply})
	select {
	case err := <-reply:
		if errors.Is(err, ErrStaleRoute) && retries > 0 {
			w.routeCache.Delete(id)
			return w.onPlayerRetry(ctx, id, fn, retries-1)
		}
		return err
	case <-route.actor.done:
		w.routeCache.Delete(id)
		return ErrNotInGame
	case <-ctx.Done():
		return ctx.Err()
	case <-w.stop:
		return ErrClosed
	}
}

func (w *World) cachedRoute(id uint16) (playerRoute, bool) {
	value, ok := w.routeCache.Load(id)
	if !ok {
		return playerRoute{}, false
	}
	route, ok := value.(playerRoute)
	return route, ok
}

func (w *World) storeRoute(id uint16, route playerRoute) {
	w.routeCache.Store(id, route)
}

// enqueueMap posts fire-and-forget work to a map actor. It must be called on
// the coordinator goroutine.
func (w *World) enqueueMap(key MapKey, fn func(*mapActor)) {
	a := w.actors[key]
	if a == nil {
		return
	}
	if w.opts.InlineMaps {
		a.exec(mapMsg{run: func(a *mapActor, _ *Player) error {
			fn(a)
			return nil
		}})
		return
	}
	a.post(mapMsg{run: func(a *mapActor, _ *Player) error {
		fn(a)
		return nil
	}})
}

// patchSocial applies a social cache change to the map-owned Player. Inline
// mode runs on the same goroutine; production posts a fire-and-forget task.
func (w *World) patchSocial(id uint16, fn func(*Player)) {
	entry := w.online[id]
	if entry == nil {
		return
	}
	if w.opts.InlineMaps {
		if p := w.playerByID(id); p != nil {
			fn(p)
		}
		return
	}
	if a := w.actors[entry.Key]; a != nil {
		a.post(mapMsg{playerID: id, epoch: entry.Epoch, run: func(a *mapActor, p *Player) error {
			fn(p)
			return nil
		}})
	}
}

// queueSave copies the player state into the persistence worker.
func (w *World) queueSave(p *Player) {
	if p == nil || p.Character == nil || p.State == nil {
		return
	}
	if w.persist != nil {
		w.persist.queue(p.Character, p.State)
		return
	}
	if err := w.opts.Store.SaveCharacter(context.Background(), p.Character, p.State); err != nil {
		w.log.Error("save character", "player", p.ID, "err", err)
	}
}

// ensureMap returns the runtime map for a template. It is used by tests and by
// inline mode; production code goes through ensureMapKey on the coordinator.
func (w *World) ensureMap(id int) *mapState {
	if !w.opts.InlineMaps {
		a := w.ensureMapKey(MapKey{TemplateID: id})
		if a == nil {
			return nil
		}
		return a.ms
	}
	if w.maps == nil || id <= 0 || id >= len(w.maps) {
		return nil
	}
	if ms := w.maps[id]; ms != nil {
		return ms
	}
	key := MapKey{TemplateID: id}
	ms := newMapState(w, key)
	if ms == nil {
		return nil
	}
	w.maps[id] = ms
	a := newMapActor(w, key, ms)
	w.actors[key] = a
	w.log.Info("map activated", "map", id, "name", ms.def.Name, "inline", true)
	return ms
}

// ensureMapKey creates the actor for key on the coordinator.
func (w *World) ensureMapKey(key MapKey) *mapActor {
	if a := w.actors[key]; a != nil {
		return a
	}
	ms := newMapState(w, key)
	if ms == nil {
		return nil
	}
	a := newMapActor(w, key, ms)
	w.actors[key] = a
	if w.opts.InlineMaps {
		return a
	}
	w.actorWG.Add(1)
	go func() {
		defer w.actorWG.Done()
		a.run()
	}()
	w.log.Info("map activated", "map", key.TemplateID, "instance", key.InstanceID, "name", ms.def.Name)
	return a
}

func newMapState(w *World, key MapKey) *mapState {
	id := key.TemplateID
	if id <= 0 || id >= len(w.opts.Data.Maps) {
		return nil
	}
	m := w.opts.Data.Maps[id]
	if m == nil {
		return nil
	}
	ms := &mapState{
		key:            key,
		def:            m,
		tileset:        w.opts.Data.Tilesets[m.TilesetID],
		events:         m.Events,
		npcs:           make([]*npcState, len(m.Events)),
		enemies:        make([]*enemyState, len(m.Events)),
		players:        make([]*Player, 0),
		globalSwitches: append([]bool(nil), w.globalSwitches...),
	}
	for _, ev := range m.Events {
		if ev == nil || ev.Name == "notupdate" || ev.Name == "notglobal" {
			continue
		}
		if enemy := newEnemyState(w.opts.Data, m, ev); enemy != nil {
			ms.enemies[ev.ID] = enemy
			continue
		}
		direction := 2
		if len(ev.Pages) > 0 && ev.Pages[0].Graphic.Direction > 0 {
			direction = ev.Pages[0].Graphic.Direction
		}
		ms.npcs[ev.ID] = &npcState{event: ev, x: ev.X, y: ev.Y, direction: direction}
	}
	for _, npc := range ms.npcs {
		if npc != nil {
			w.refreshNPC(ms, npc)
		}
	}
	return ms
}

func (w *World) loop() {
	defer close(w.stopped)
	for {
		select {
		case fn := <-w.commands:
			fn()
		case <-w.stop:
			return
		}
	}
}

// Join adds a player to the world. The client must already be authenticated.
func (w *World) Join(ctx context.Context, p *Player) error {
	if p == nil || p.Client == nil || p.Character == nil || p.Account == nil || p.State == nil {
		return fmt.Errorf("game: incomplete player")
	}
	return w.do(ctx, func() error {
		if w.opts.InlineMaps {
			return w.joinInline(p)
		}
		return w.joinOnline(p)
	})
}

func (w *World) allocPlayerID() (uint16, error) {
	for w.online[w.nextID] != nil {
		w.nextID++
		if w.nextID == 0 {
			return 0, fmt.Errorf("game: no free player ids")
		}
	}
	if w.nextID == 0 {
		return 0, fmt.Errorf("game: no free player ids")
	}
	id := w.nextID
	w.nextID++
	return id, nil
}

func (w *World) allocEpoch() uint64 {
	e := w.nextEpoch
	w.nextEpoch++
	if w.nextEpoch == 0 {
		w.nextEpoch = 1
	}
	return e
}

// normalizeJoinPosition validates the saved position against immutable map
// templates. It does not create runtime map state.
func (w *World) normalizeJoinPosition(p *Player) error {
	valid := w.mapTemplateValid
	if valid(p.Character.MapID, p.Character.X, p.Character.Y) {
		return nil
	}
	if valid(p.Character.ReviveMapID, p.Character.ReviveX, p.Character.ReviveY) {
		p.Character.MapID = p.Character.ReviveMapID
		p.Character.X = p.Character.ReviveX
		p.Character.Y = p.Character.ReviveY
		return nil
	}
	p.Character.MapID = w.opts.Data.System.StartMapID
	p.Character.X = w.opts.Data.System.StartX
	p.Character.Y = w.opts.Data.System.StartY
	if !valid(p.Character.MapID, p.Character.X, p.Character.Y) {
		return fmt.Errorf("game: invalid save position for character %d", p.Character.ID)
	}
	return nil
}

func (w *World) mapTemplateValid(mapID, x, y int) bool {
	if mapID <= 0 || mapID >= len(w.opts.Data.Maps) || w.opts.Data.Maps[mapID] == nil {
		return false
	}
	m := w.opts.Data.Maps[mapID]
	return x >= 0 && y >= 0 && x < m.Width && y < m.Height
}

func (w *World) joinOnline(p *Player) error {
	if _, ok := w.byCharacter[p.Character.ID]; ok {
		return ErrAlreadyOnline
	}
	id, err := w.allocPlayerID()
	if err != nil {
		return err
	}
	if err := w.normalizeJoinPosition(p); err != nil {
		return err
	}
	p.ID = id
	w.ensureEventState(p)
	w.syncPlayerGuild(p)
	w.syncPlayerFriends(p)
	if p.Bank == nil {
		bank, err := w.opts.Store.GetBank(context.Background(), p.Account.ID)
		if err != nil {
			w.log.Warn("load bank", "account", p.Account.ID, "err", err)
		} else {
			p.Bank = bank
		}
	}
	key := MapKey{TemplateID: p.Character.MapID}
	epoch := w.allocEpoch()
	entry := &onlineEntry{
		ID:            p.ID,
		CharacterID:   p.Character.ID,
		Name:          p.Character.Name,
		Account:       p.Account,
		Client:        p.Client,
		IP:            p.IP,
		Key:           key,
		Epoch:         epoch,
		GuildName:     p.GuildName,
		PartyID:       p.PartyID,
		Friends:       append([]string(nil), p.Friends...),
		MuteUntil:     p.MuteUntil,
		CreatingGuild: p.CreatingGuild,
		View:          partyViewOf(p),
	}
	w.online[p.ID] = entry
	w.byCharacter[p.Character.ID] = entry
	a := w.ensureMapKey(key)
	if a == nil {
		delete(w.online, p.ID)
		delete(w.byCharacter, p.Character.ID)
		return fmt.Errorf("game: no valid map for character %d", p.Character.ID)
	}
	w.storeRoute(p.ID, playerRoute{key: key, epoch: epoch, actor: a})
	snap := snapshotPlayer(p, epoch)
	snap.GuildName = entry.GuildName
	snap.PartyID = entry.PartyID
	snap.Friends = append([]string(nil), entry.Friends...)
	w.attachPlayer(a, attachPayload{
		snap:    snap,
		enter:   true,
		epoch:   epoch,
		gen:     w.nextAttachGen(key),
		motd:    w.motd,
		globals: append([]bool(nil), w.globalSwitches...),
	})
	w.log.Info("player joining", "player", p.ID, "character", p.Character.Name, "map", key.TemplateID, "x", p.Character.X, "y", p.Character.Y)
	return nil
}

// nextAttachGen bumps the coordinator-side generation of one actor.
func (w *World) nextAttachGen(key MapKey) uint64 {
	w.attachGen[key]++
	return w.attachGen[key]
}

func (w *World) attachPlayer(a *mapActor, payload attachPayload) {
	a.post(mapMsg{run: func(a *mapActor, _ *Player) error {
		a.attach(payload)
		return nil
	}})
}

func (w *World) attachFailed(snap *playerSnapshot, reason string) {
	if snap == nil {
		return
	}
	w.log.Error("map attach failed", "player", snap.ID, "reason", reason)
	w.removeOnlineByID(snap.ID)
	if snap.Client != nil {
		_ = snap.Client.Send(protocol.ServerError, protocol.ErrorMessage{Code: protocol.CodeInternal, Message: "map attach failed"})
		_ = snap.Client.Close()
	}
}

func (w *World) joinInline(p *Player) error {
	if _, ok := w.byCharacter[p.Character.ID]; ok {
		return ErrAlreadyOnline
	}
	for w.online[w.nextID] != nil {
		w.nextID++
	}
	if w.nextID == 0 {
		return fmt.Errorf("game: no free player ids")
	}
	p.ID = w.nextID
	w.nextID++
	ms := w.ensureMap(p.Character.MapID)
	if ms == nil {
		p.Character.MapID = p.Character.ReviveMapID
		ms = w.ensureMap(p.Character.MapID)
	}
	if ms == nil {
		p.Character.MapID = w.opts.Data.System.StartMapID
		ms = w.ensureMap(p.Character.MapID)
	}
	if ms == nil {
		return fmt.Errorf("game: no valid map for character %d", p.Character.ID)
	}
	if !ms.valid(p.Character.X, p.Character.Y) {
		if revive := w.ensureMap(p.Character.ReviveMapID); revive != nil && revive.valid(p.Character.ReviveX, p.Character.ReviveY) {
			p.Character.MapID = p.Character.ReviveMapID
			p.Character.X = p.Character.ReviveX
			p.Character.Y = p.Character.ReviveY
			ms = revive
		} else {
			p.Character.X = w.opts.Data.System.StartX
			p.Character.Y = w.opts.Data.System.StartY
			if !ms.valid(p.Character.X, p.Character.Y) {
				return fmt.Errorf("game: invalid save position for character %d", p.Character.ID)
			}
		}
	}
	epoch := w.allocEpoch()
	p.Epoch = epoch
	p.MapKey = MapKey{TemplateID: p.Character.MapID}
	entry := &onlineEntry{
		ID: p.ID, CharacterID: p.Character.ID, Name: p.Character.Name,
		Account: p.Account, Client: p.Client, IP: p.IP, Key: p.MapKey, Epoch: epoch,
		View: partyViewOf(p),
	}
	w.online[p.ID] = entry
	w.byCharacter[p.Character.ID] = entry
	actor := w.actors[p.MapKey]
	w.storeRoute(p.ID, playerRoute{key: p.MapKey, epoch: epoch, actor: actor})
	actor.putPlayer(p)
	w.ensureEventState(p)
	w.syncPlayerGuild(p)
	w.syncPlayerFriends(p)
	entry.GuildName = p.GuildName
	entry.PartyID = p.PartyID
	entry.Friends = append([]string(nil), p.Friends...)
	snap := w.snapshot(p)
	if err := p.Client.Send(protocol.ServerEnterWorld, w.enterWorld(p)); err != nil {
		w.removeInlineLocked(p, false)
		return err
	}
	w.sendGlobalSwitches(p)
	w.sendMOTD(p)
	for _, enemy := range ms.enemies {
		if enemy == nil {
			continue
		}
		_ = p.Client.Send(protocol.ServerEventData, protocol.EventData{
			EventID:   uint16(enemy.Event.ID),
			X:         uint16(enemy.X),
			Y:         uint16(enemy.Y),
			Direction: uint8(enemy.Direction),
			HP:        int32(enemy.HP),
		})
	}
	w.sendMapEventsTo(p, ms)
	w.sendDropsTo(p, ms)
	for _, other := range ms.players {
		if other == nil || other == p {
			continue
		}
		if err := p.Client.Send(protocol.ServerPlayerData, protocol.PlayerData{Player: w.snapshot(other)}); err != nil {
			w.log.Warn("send player data", "player", p.ID, "err", err)
		}
		if err := other.Client.Send(protocol.ServerPlayerData, protocol.PlayerData{Player: snap}); err != nil {
			w.log.Warn("broadcast player data", "player", other.ID, "err", err)
		}
	}
	w.log.Info("player joined", "player", p.ID, "character", p.Character.Name, "map", p.Character.MapID, "x", p.Character.X, "y", p.Character.Y)
	return nil
}

// Leave removes a player and optionally saves it.
func (w *World) Leave(ctx context.Context, p *Player, save bool) error {
	if p == nil {
		return ErrNotInGame
	}
	return w.do(ctx, func() error {
		if w.opts.InlineMaps {
			return w.leaveInline(p, save)
		}
		entry := w.online[p.ID]
		if entry == nil {
			return ErrNotInGame
		}
		delete(w.online, p.ID)
		delete(w.byCharacter, entry.CharacterID)
		w.routeCache.Delete(p.ID)
		if entry.PartyID != 0 {
			w.leavePartyEntry(entry)
		}
		if a := w.actors[entry.Key]; a != nil {
			a.post(mapMsg{playerID: p.ID, epoch: entry.Epoch, run: func(a *mapActor, pl *Player) error {
				a.detach(pl.ID, save)
				return nil
			}})
		} else if save {
			// The actor is already gone; the last flush handled it.
		}
		w.log.Info("player left", "player", p.ID, "character", entry.Name)
		return nil
	})
}

func (w *World) leaveInline(p *Player, save bool) error {
	if w.online[p.ID] == nil {
		return ErrNotInGame
	}
	if save {
		if err := w.saveLocked(p); err != nil {
			w.log.Error("save player", "player", p.ID, "err", err)
		}
		w.saveBankQuietly(p)
	}
	w.removeInlineLocked(p, false)
	w.log.Info("player left", "player", p.ID, "character", p.Character.Name)
	return nil
}

func (w *World) removeInlineLocked(p *Player, save bool) {
	if p.PartyID != 0 {
		w.leavePartyLocked(p)
	}
	if a := w.actors[p.MapKey]; a != nil {
		a.detach(p.ID, save)
	} else if ms := p.ms; ms != nil {
		if int(p.ID) < len(ms.players) {
			ms.players[p.ID] = nil
		}
	}
	delete(w.online, p.ID)
	delete(w.byCharacter, p.Character.ID)
	w.routeCache.Delete(p.ID)
}

func (w *World) removeOnlineByID(id uint16) {
	entry := w.online[id]
	if entry == nil {
		return
	}
	delete(w.online, id)
	delete(w.byCharacter, entry.CharacterID)
	w.routeCache.Delete(id)
	if entry.PartyID != 0 {
		w.leavePartyEntry(entry)
	}
}

// Move processes one step/turn request.
func (w *World) Move(ctx context.Context, p *Player, direction uint8) error {
	return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
		now := w.now()
		if !p.lastMove.IsZero() && now.Sub(p.lastMove) < 80*time.Millisecond {
			return ErrRateLimited
		}
		var dx, dy int
		switch direction {
		case 2:
			dy = 1
		case 4:
			dx = -1
		case 6:
			dx = 1
		case 8:
			dy = -1
		default:
			return ErrInvalidMove
		}
		nx, ny := p.Character.X+dx, p.Character.Y+dy
		oldX, oldY := p.Character.X, p.Character.Y
		moved := false
		if ms.valid(nx, ny) && ms.passable(nx, ny, direction) && !ms.blocked(nx, ny, p) {
			p.Character.X, p.Character.Y = nx, ny
			moved = true
		}
		// Always echo the current position/direction, even for a blocked step or
		// a turn into the same direction. The client clears $wait_player_move on
		// this response; suppressing it would leave the character unable to move
		// after a server-side collision that the client cannot predict.
		p.Character.Direction = int(direction)
		p.lastMove = now
		msg := protocol.PlayerMove{ID: p.ID, X: uint16(p.Character.X), Y: uint16(p.Character.Y), Direction: direction}
		w.broadcastNear(ms, p.Character.X, p.Character.Y, protocol.ServerPlayerMove, msg)
		if moved {
			p.actor.aoiPlayerMoved(p, oldX, oldY)
		}
		if moved {
			w.checkPlayerTouch(p, ms, p.Character.X, p.Character.Y, false)
		} else {
			w.checkPlayerTouch(p, ms, nx, ny, true)
		}
		return nil
	})
}

// Chat handles map/global/party/guild/private chat on the coordinator. Chat
// never touches map runtime state.
func (w *World) Chat(ctx context.Context, p *Player, channel uint8, message, target string) error {
	return w.do(ctx, func() error {
		entry := w.online[p.ID]
		if entry == nil {
			return ErrNotInGame
		}
		now := w.now()
		if !entry.MuteUntil.IsZero() && now.Before(entry.MuteUntil) {
			return ErrMuted
		}
		if !entry.LastChat.IsZero() && now.Sub(entry.LastChat) < 300*time.Millisecond {
			return ErrRateLimited
		}
		message = strings.TrimSpace(message)
		if message == "" || len(message) > 1024 {
			return fmt.Errorf("game: invalid chat length")
		}
		text := entry.Name + ": " + message
		switch channel {
		case 0: // map
			msg := protocol.MapChat{PlayerID: entry.ID, Color: 0, Message: text}
			for _, other := range w.online {
				if other.Key != entry.Key {
					continue
				}
				if err := other.Client.Send(protocol.ServerMapChat, msg); err != nil {
					w.log.Warn("broadcast map chat", "player", other.ID, "err", err)
				}
			}
		case 1: // global
			msg := protocol.ChatMessage{Color: 1, Message: text}
			for _, other := range w.online {
				if err := other.Client.Send(protocol.ServerChat, msg); err != nil {
					w.log.Warn("broadcast global chat", "player", other.ID, "err", err)
				}
			}
		case 2: // party
			party := w.partyByID(entry.PartyID)
			if party == nil {
				return ErrNotInParty
			}
			msg := protocol.ChatMessage{Color: 2, Message: text}
			for _, id := range party.Members {
				if member := w.online[id]; member != nil {
					_ = member.Client.Send(protocol.ServerChat, msg)
				}
			}
		case 3: // guild
			g := w.guildByName[store.NameKey(entry.GuildName)]
			if g == nil {
				return ErrNoGuild
			}
			msg := protocol.ChatMessage{Color: 16, Message: text}
			for _, charID := range g.Order {
				if member := w.byCharacter[charID]; member != nil {
					_ = member.Client.Send(protocol.ServerChat, msg)
				}
			}
		case 4: // private
			target := w.onlineByName(target)
			if target == nil || target == entry {
				return ErrInvalidTarget
			}
			msg := protocol.ChatMessage{Color: 4, Message: text}
			_ = target.Client.Send(protocol.ServerChat, msg)
			_ = entry.Client.Send(protocol.ServerChat, msg)
		default:
			return ErrUnsupported
		}
		entry.LastChat = now
		w.log.Debug("chat", "player", entry.Name, "channel", channel, "text", text)
		return nil
	})
}

// CharacterOnline reports whether a character is currently in the world.
func (w *World) CharacterOnline(ctx context.Context, characterID int64) (bool, error) {
	result := make(chan bool, 1)
	select {
	case w.commands <- func() { _, ok := w.byCharacter[characterID]; result <- ok }:
	case <-ctx.Done():
		return false, ctx.Err()
	case <-w.stop:
		return false, ErrClosed
	}
	select {
	case ok := <-result:
		return ok, nil
	case <-ctx.Done():
		return false, ctx.Err()
	case <-w.stop:
		return false, ErrClosed
	}
}

func (w *World) snapshot(p *Player) protocol.PlayerSnapshot {
	c := p.Character
	s := protocol.PlayerSnapshot{
		ID:             p.ID,
		Group:          uint8(p.Account.GroupID),
		Name:           c.Name,
		CharacterName:  c.CharacterName,
		CharacterIndex: uint8(c.CharacterIndex),
		FaceName:       c.FaceName,
		FaceIndex:      uint8(c.FaceIndex),
		Sex:            uint8(c.Sex),
		ClassID:        uint16(c.ClassID),
		Level:          int32(c.Level),
		HP:             int32(c.HP),
		MP:             int32(c.MP),
		MHP:            int32(c.MHP),
		MMP:            int32(c.MMP),
		EXP:            int32(c.Exp),
		Points:         int32(c.Points),
		Gold:           c.Gold,
		GuildName:      p.GuildName,
		MapID:          uint16(c.MapID),
		X:              uint16(c.X),
		Y:              uint16(c.Y),
		Direction:      uint8(c.Direction),
	}
	if p.State != nil {
		for i := 0; i < len(s.Equips) && i < len(p.State.Equips); i++ {
			if p.State.Equips[i] >= 0 && p.State.Equips[i] <= 65535 {
				s.Equips[i] = uint16(p.State.Equips[i])
			}
		}
	}
	return s
}

func (w *World) enterWorld(p *Player) protocol.EnterWorld {
	c := p.Character
	e := protocol.EnterWorld{Player: w.snapshot(p)}
	e.Friends = append([]string(nil), p.Friends...)
	e.Params = [8]int32{
		int32(c.MHP), int32(c.MMP), int32(c.ATK), int32(c.DEF),
		int32(c.MAT), int32(c.MDF), int32(c.AGI), int32(c.LUK),
	}
	if p.State != nil {
		for _, item := range p.State.Inventory {
			entry := protocol.ItemEntry{Kind: uint8(clampInt(item.Kind, 0, 255)), ItemID: uint16(clampInt(item.ItemID, 0, 65535)), Amount: uint16(clampInt(item.Amount, 0, 65535))}
			switch item.Kind {
			case 1:
				e.Items = append(e.Items, entry)
			case 2:
				e.Weapons = append(e.Weapons, entry)
			case 3:
				e.Armors = append(e.Armors, entry)
			}
		}
		for _, skill := range p.State.Skills {
			e.Skills = append(e.Skills, uint16(clampInt(skill, 0, 65535)))
		}
		for _, q := range p.State.Quests {
			e.Quests = append(e.Quests, protocol.QuestEntry{QuestID: uint8(clampInt(q.QuestID, 0, 255)), State: uint8(clampInt(q.State, 0, 255))})
		}
		for _, h := range p.State.Hotbar {
			e.Hotbar = append(e.Hotbar, protocol.HotbarEntry{Type: uint8(clampInt(h.Type, 0, 255)), ItemID: uint16(clampInt(h.ItemID, 0, 65535))})
		}
		e.Switches = append([]bool(nil), p.State.Switches...)
		e.Variables = append([]int32(nil), p.State.Variables...)
		for _, s := range p.State.SelfSwitches {
			e.SelfSwitches = append(e.SelfSwitches, protocol.SelfSwitchEntry{
				MapID: uint16(clampInt(s.MapID, 0, 65535)), EventID: uint16(clampInt(s.EventID, 0, 65535)),
				Ch: s.Ch, Value: s.Value,
			})
		}
	}
	return e
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (p *Player) vipUntil() int64 {
	if p == nil || p.Account == nil {
		return 0
	}
	return atomic.LoadInt64(&p.Account.VIPUntil)
}

func (p *Player) setVIPUntil(value int64) {
	if p == nil || p.Account == nil {
		return
	}
	atomic.StoreInt64(&p.Account.VIPUntil, value)
}

func (p *Player) vipActive(now int64) bool {
	return p.vipUntil() > now
}

func clampInt64(v, min, max int64) int64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ensureIndex grows an id-indexed slice. RPG Maker ids are array indexes and
// may contain holes, so callers must skip nil entries while iterating.
func ensureIndex[T any](list []T, i int) []T {
	if i < 0 {
		return list
	}
	if i >= len(list) {
		list = append(list, make([]T, i-len(list)+1)...)
	}
	return list
}

func (w *World) playerByID(id uint16) *Player {
	if !w.opts.InlineMaps {
		return nil
	}
	entry := w.online[id]
	if entry == nil {
		return nil
	}
	if a := w.actors[entry.Key]; a != nil {
		return a.playerByID(id)
	}
	return nil
}

func (w *World) onlineByName(name string) *onlineEntry {
	name = strings.TrimSpace(name)
	for _, entry := range w.online {
		if strings.EqualFold(entry.Name, name) {
			return entry
		}
	}
	return nil
}

func (w *World) saveLocked(p *Player) error {
	return w.opts.Store.SaveCharacter(context.Background(), p.Character, p.State)
}

func (w *World) saveInline() {
	for _, a := range w.actors {
		for _, p := range a.players {
			if p == nil {
				continue
			}
			if err := w.saveLocked(p); err != nil {
				w.log.Error("save player", "player", p.ID, "err", err)
			}
			w.saveBankQuietly(p)
		}
	}
}

func (ms *mapState) valid(x, y int) bool {
	return x >= 0 && y >= 0 && x < ms.def.Width && y < ms.def.Height
}

func (ms *mapState) playerByID(id uint16) *Player {
	if int(id) < len(ms.players) {
		return ms.players[id]
	}
	for _, p := range ms.players {
		if p != nil && p.ID == id {
			return p
		}
	}
	return nil
}

// passable applies the target tile's reverse-direction passage bit.
func (ms *mapState) passable(x, y int, direction uint8) bool {
	reverse := uint8(10 - int(direction))
	bit := (1 << (reverse/2 - 1)) & 0x0f
	if ms.tileset == nil || ms.tileset.Flags == nil {
		return true
	}
	for _, tileID := range ms.allTiles(x, y) {
		flag := ms.tileset.Flags.At(int32(tileID), 0, 0)
		if flag&0x10 != 0 {
			continue
		}
		if flag&uint16(bit) == 0 {
			return true
		}
		if flag&uint16(bit) == uint16(bit) {
			return false
		}
	}
	return false
}

func (ms *mapState) allTiles(x, y int) []int {
	var tiles []int
	// Tile events come first, matching the original map.all_tiles ordering.
	for _, ev := range ms.events {
		if ev == nil {
			continue
		}
		npc := ms.npcs[ev.ID]
		if npc == nil || npc.x != x || npc.y != y || len(ev.Pages) == 0 {
			continue
		}
		page := ev.Pages[0]
		if page.Graphic.TileID > 0 && page.PriorityType == 0 && !page.Through {
			tiles = append(tiles, page.Graphic.TileID)
		}
	}
	for _, z := range []int{2, 1, 0} {
		tiles = append(tiles, int(ms.def.Data.At(int32(x), int32(y), int32(z))))
	}
	return tiles
}

// blocked checks normal-priority events and, on PvP maps, other players.
func (ms *mapState) blocked(x, y int, moving *Player) bool {
	for _, npc := range ms.npcs {
		if npc == nil || npc.x != x || npc.y != y || npc.through {
			continue
		}
		if npc.priority == 1 {
			return true
		}
	}
	for _, enemy := range ms.enemies {
		if enemy == nil || !enemy.Alive || enemy.X != x || enemy.Y != y {
			continue
		}
		if len(enemy.Event.Pages) > 0 {
			page := enemy.Event.Pages[0]
			if page.PriorityType == 1 && !page.Through {
				return true
			}
		}
	}
	if !ms.def.PvP {
		return false
	}
	for _, other := range ms.players {
		if other == nil || other == moving {
			continue
		}
		if other.Character.X == x && other.Character.Y == y {
			return true
		}
	}
	return false
}
