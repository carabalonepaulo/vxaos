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

// Player is the runtime state of one logged-in character.
type Player struct {
	ID        uint16
	Account   *store.Account
	Character *store.Character
	State     *store.CharacterState
	Client    Client
	GuildName string
	IP        string
	MuteUntil time.Time

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

// World is the single-owner game state.
type World struct {
	opts Options
	log  *slog.Logger
	now  func() time.Time

	commands chan func()
	stop     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
	rw       sync.RWMutex
	mapWG    sync.WaitGroup

	// All fields below are owned by the world goroutine.
	players        []*Player
	byCharacter    map[int64]*Player
	maps           []*mapState
	guilds         map[int64]*guildState
	guildByName    map[string]*guildState
	parties        map[int]*partyState
	nextPartyID    int
	nextID         uint16
	globalSwitches []bool
	motd           string
	persist        *persistWorker
}

type mapState struct {
	def     *rpg.Map
	tileset *rpg.Tileset
	events  []*rpg.Event
	npcs    []*npcState
	enemies []*enemyState
	players []*Player
	autorun *autorunState
	drops   []*dropState
	stop    chan struct{}
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
		commands:    make(chan func(), 1024),
		stop:        make(chan struct{}),
		stopped:     make(chan struct{}),
		players:     make([]*Player, 0),
		byCharacter: make(map[int64]*Player),
		maps:        make([]*mapState, len(opts.Data.Maps)),
		guilds:      make(map[int64]*guildState),
		guildByName: make(map[string]*guildState),
		parties:     make(map[int]*partyState),
		nextPartyID: 1,
		nextID:      1,
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

// Stop stops the loop after saving online players.
func (w *World) Stop(ctx context.Context) error {
	w.stopOnce.Do(func() { close(w.stop) })
	select {
	case <-w.stopped:
	case <-ctx.Done():
		return ctx.Err()
	}
	for _, ms := range w.maps {
		if ms != nil && ms.stop != nil {
			close(ms.stop)
		}
	}
	w.mapWG.Wait()
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

// ensureMap creates the runtime state for one map template on demand. Callers
// are the world goroutine today; this is the seam that will become a map
// actor start later.
func (w *World) ensureMap(id int) *mapState {
	if id <= 0 || id >= len(w.maps) {
		return nil
	}
	if ms := w.maps[id]; ms != nil {
		return ms
	}
	m := w.opts.Data.Maps[id]
	if m == nil {
		return nil
	}
	ms := &mapState{
		def:     m,
		tileset: w.opts.Data.Tilesets[m.TilesetID],
		events:  m.Events,
		npcs:    make([]*npcState, len(m.Events)),
		enemies: make([]*enemyState, len(m.Events)),
		players: make([]*Player, 0),
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
			w.refreshNPC(npc)
		}
	}
	w.maps[id] = ms
	ms.stop = make(chan struct{})
	w.mapWG.Add(1)
	go w.runMapActor(ms)
	w.log.Info("map activated", "map", id, "name", m.Name)
	return ms
}

func (w *World) runMapActor(ms *mapState) {
	defer w.mapWG.Done()
	tick := time.NewTicker(w.opts.Tick)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			w.rw.RLock()
			if ms.hasPlayers() {
				now := w.now()
				w.tickCombatMap(ms, now)
				w.tickDropsMap(ms, now)
				w.tickNPCsMap(ms, now)
			}
			w.rw.RUnlock()
		case <-ms.stop:
			return
		}
	}
}

func (w *World) loop() {
	defer close(w.stopped)
	tick := time.NewTicker(w.opts.Tick)
	defer tick.Stop()
	save := time.NewTicker(w.opts.SaveInterval)
	defer save.Stop()
	for {
		select {
		case fn := <-w.commands:
			w.rw.Lock()
			fn()
			w.rw.Unlock()
		case <-tick.C:
			w.rw.Lock()
			w.tickEvents()
			w.tickAutoruns()
			w.rw.Unlock()
		case <-save.C:
			w.rw.Lock()
			w.saveAll()
			w.rw.Unlock()
		case <-w.stop:
			w.rw.Lock()
			w.saveAll()
			w.rw.Unlock()
			return
		}
	}
}

// Join adds a player to the map. The client must already be authenticated.
func (w *World) Join(ctx context.Context, p *Player) error {
	if p == nil || p.Client == nil || p.Character == nil || p.Account == nil || p.State == nil {
		return fmt.Errorf("game: incomplete player")
	}
	return w.do(ctx, func() error { return w.join(p) })
}

func (w *World) join(p *Player) error {
	if _, ok := w.byCharacter[p.Character.ID]; ok {
		return ErrAlreadyOnline
	}
	for int(w.nextID) < len(w.players) && w.playerByID(w.nextID) != nil {
		w.nextID++
		if w.nextID == 0 {
			return fmt.Errorf("game: no free player ids")
		}
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
	// Validate the saved position; fall back to the revive point and then the
	// system start point.
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
	w.players = ensureIndex(w.players, int(p.ID))
	w.players[p.ID] = p
	w.byCharacter[p.Character.ID] = p
	ms.players = ensureIndex(ms.players, int(p.ID))
	ms.players[p.ID] = p
	w.ensureEventState(p)
	w.syncPlayerGuild(p)
	w.syncPlayerFriends(p)
	snap := w.snapshot(p)
	if err := p.Client.Send(protocol.ServerEnterWorld, w.enterWorld(p)); err != nil {
		w.removeLocked(p)
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
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		if save {
			if err := w.saveLocked(p); err != nil {
				w.log.Error("save player", "player", p.ID, "err", err)
			}
			w.saveBankQuietly(p)
		}
		w.removeLocked(p)
		w.log.Info("player left", "player", p.ID, "character", p.Character.Name)
		return nil
	})
}

func (w *World) removeLocked(p *Player) {
	if p.Trade != nil {
		w.closeTradeLocked(p)
	}
	if p.PartyID != 0 {
		w.leavePartyLocked(p)
	}
	w.players[p.ID] = nil
	delete(w.byCharacter, p.Character.ID)
	if ms := w.maps[p.Character.MapID]; ms != nil {
		ms.players[p.ID] = nil
		for _, other := range ms.players {
			if other == nil {
				continue
			}
			if err := other.Client.Send(protocol.ServerPlayerLeft, protocol.PlayerLeft{ID: p.ID}); err != nil {
				w.log.Warn("broadcast player left", "player", other.ID, "err", err)
			}
		}
	}
}

// Move processes one step/turn request.
func (w *World) Move(ctx context.Context, p *Player, direction uint8) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		now := w.now()
		if !p.lastMove.IsZero() && now.Sub(p.lastMove) < 80*time.Millisecond {
			return ErrRateLimited
		}
		ms := w.maps[p.Character.MapID]
		if ms == nil {
			return ErrInvalidMove
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
		for _, other := range ms.players {
			if other == nil {
				continue
			}
			if err := other.Client.Send(protocol.ServerPlayerMove, msg); err != nil {
				w.log.Warn("broadcast move", "player", other.ID, "err", err)
			}
		}
		if moved {
			w.checkPlayerTouch(p, ms, p.Character.X, p.Character.Y, false)
		} else {
			w.checkPlayerTouch(p, ms, nx, ny, true)
		}
		return nil
	})
}

// Chat handles map/global chat. Party/guild/private chat arrives in a later
// slice.
func (w *World) Chat(ctx context.Context, p *Player, channel uint8, message, target string) error {
	return w.do(ctx, func() error {
		if w.playerByID(p.ID) != p {
			return ErrNotInGame
		}
		now := w.now()
		if !p.MuteUntil.IsZero() && now.Before(p.MuteUntil) {
			return ErrMuted
		}
		if !p.lastChat.IsZero() && now.Sub(p.lastChat) < 300*time.Millisecond {
			return ErrRateLimited
		}
		message = strings.TrimSpace(message)
		if message == "" || len(message) > 1024 {
			return fmt.Errorf("game: invalid chat length")
		}
		text := p.Character.Name + ": " + message
		switch channel {
		case 0: // map
			ms := w.maps[p.Character.MapID]
			if ms == nil {
				return ErrNotInGame
			}
			msg := protocol.MapChat{PlayerID: p.ID, Color: 0, Message: text}
			for _, other := range ms.players {
				if other == nil {
					continue
				}
				if err := other.Client.Send(protocol.ServerMapChat, msg); err != nil {
					w.log.Warn("broadcast map chat", "player", other.ID, "err", err)
				}
			}
		case 1: // global
			msg := protocol.ChatMessage{Color: 1, Message: text}
			for _, other := range w.players {
				if other == nil {
					continue
				}
				if err := other.Client.Send(protocol.ServerChat, msg); err != nil {
					w.log.Warn("broadcast global chat", "player", other.ID, "err", err)
				}
			}
		case 2: // party
			party := w.partyByID(p.PartyID)
			if party == nil {
				return ErrNotInParty
			}
			w.partyChat(party, text)
		case 3: // guild
			if p.GuildName == "" {
				return ErrNoGuild
			}
			g := w.guildByName[store.NameKey(p.GuildName)]
			if g == nil {
				return ErrNoGuild
			}
			w.guildChat(g, text)
		case 4: // private
			target := w.playerByName(target)
			if target == nil || target == p {
				return ErrInvalidTarget
			}
			msg := protocol.ChatMessage{Color: 4, Message: text}
			_ = target.Client.Send(protocol.ServerChat, msg)
			_ = p.Client.Send(protocol.ServerChat, msg)
		default:
			return ErrUnsupported
		}
		p.lastChat = now
		w.log.Debug("chat", "player", p.Character.Name, "channel", channel, "text", text)
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
	if int(id) < len(w.players) {
		return w.players[id]
	}
	return nil
}

func (w *World) saveLocked(p *Player) error {
	return w.opts.Store.SaveCharacter(context.Background(), p.Character, p.State)
}

func (w *World) saveAll() {
	for _, p := range w.players {
		if p == nil {
			continue
		}
		if w.persist != nil {
			w.persist.queue(p.Character, p.State)
		} else if err := w.saveLocked(p); err != nil {
			w.log.Error("periodic save", "player", p.ID, "err", err)
		}
		w.saveBankQuietly(p)
	}
}

func (ms *mapState) valid(x, y int) bool {
	return x >= 0 && y >= 0 && x < ms.def.Width && y < ms.def.Height
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
