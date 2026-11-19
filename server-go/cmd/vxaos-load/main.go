// Command vxaos-load creates disposable bot accounts and drives protocol v2
// load tests against a running server.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kelaia/vxaos/server/internal/auth"
	"github.com/kelaia/vxaos/server/internal/chargen"
	"github.com/kelaia/vxaos/server/internal/config"
	"github.com/kelaia/vxaos/server/internal/content"
	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

type options struct {
	configPath  string
	addr        string
	count       int
	duration    time.Duration
	ramp        time.Duration
	moveEvery   time.Duration
	attackEvery time.Duration
	prefix      string
	password    string
	setup       bool
	spread      bool
	jsonPath    string
	iterations  int
}

type report struct {
	StartedAt       string         `json:"started_at"`
	Addr            string         `json:"addr"`
	Count           int            `json:"count"`
	DurationSeconds float64        `json:"duration_seconds"`
	RampSeconds     float64        `json:"ramp_seconds"`
	MoveEveryMs     int64          `json:"move_every_ms"`
	AttackEveryMs   int64          `json:"attack_every_ms"`
	Connected       int64          `json:"connected"`
	AuthOK          int64          `json:"auth_ok"`
	EnterOK         int64          `json:"enter_ok"`
	Failed          int64          `json:"failed"`
	SentFrames      int64          `json:"sent_frames"`
	RecvFrames      int64          `json:"recv_frames"`
	RecvBytes       int64          `json:"recv_bytes"`
	AuthP50Ms       float64        `json:"auth_p50_ms"`
	AuthP95Ms       float64        `json:"auth_p95_ms"`
	EnterP50Ms      float64        `json:"enter_p50_ms"`
	EnterP95Ms      float64        `json:"enter_p95_ms"`
	Errors          map[string]int `json:"errors"`
}

var (
	connected   atomic.Int64
	authOK      atomic.Int64
	enterOK     atomic.Int64
	failed      atomic.Int64
	sentFrames  atomic.Int64
	recvFrames  atomic.Int64
	recvBytes   atomic.Int64
	metricsMu   sync.Mutex
	authTimes   []float64
	enterTimes  []float64
	errorCounts = make(map[string]int)
)

func main() {
	opts := parseFlags()
	if opts.setup {
		if err := setup(opts); err != nil {
			fail(err)
		}
		return
	}
	if err := run(opts); err != nil {
		fail(err)
	}
}

func parseFlags() options {
	var opts options
	fs := flag.NewFlagSet("vxaos-load", flag.ExitOnError)
	fs.StringVar(&opts.configPath, "config", "config.load.toml", "server config used for setup")
	fs.StringVar(&opts.addr, "addr", "127.0.0.1:5050", "server address")
	fs.IntVar(&opts.count, "count", 500, "number of bots")
	fs.DurationVar(&opts.duration, "duration", 60*time.Second, "run duration")
	fs.DurationVar(&opts.ramp, "ramp", 10*time.Second, "connection ramp-up time")
	fs.DurationVar(&opts.moveEvery, "move-every", 500*time.Millisecond, "movement interval per bot")
	fs.DurationVar(&opts.attackEvery, "attack-every", 2*time.Second, "attack interval per bot")
	fs.StringVar(&opts.prefix, "prefix", "load", "account/character name prefix")
	fs.StringVar(&opts.password, "pass", "loadtest", "bot password")
	fs.BoolVar(&opts.setup, "setup", false, "create the bot accounts and characters instead of running")
	fs.BoolVar(&opts.spread, "spread", false, "spread bot spawn positions across the map (AOI workload)")
	fs.StringVar(&opts.jsonPath, "json", "", "write the JSON report to this path")
	fs.IntVar(&opts.iterations, "iterations", 1000, "PBKDF2 iterations for created bot accounts")
	fs.Parse(os.Args[1:])
	if opts.count <= 0 {
		fail(fmt.Errorf("count must be positive"))
	}
	if opts.duration <= 0 {
		fail(fmt.Errorf("duration must be positive"))
	}
	if opts.moveEvery <= 0 || opts.attackEvery <= 0 {
		fail(fmt.Errorf("move/attack intervals must be positive"))
	}
	return opts
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "vxaos-load: %v\n", err)
	os.Exit(1)
}

func setup(opts options) error {
	cfg, err := config.Load(opts.configPath)
	if err != nil {
		return err
	}
	shared, err := content.LoadConfigs(filepath.Join(cfg.ContentDir, "configs.toml"))
	if err != nil {
		return err
	}
	data, err := rpg.Load(cfg.DataDir)
	if err != nil {
		return err
	}
	ctx := context.Background()
	st, err := store.Open(ctx, store.Config{
		Driver: store.Driver(cfg.Store.Driver), DSN: cfg.Store.DSN,
		MaxOpenConns: cfg.Store.MaxOpenConns, MaxIdleConns: cfg.Store.MaxIdleConns,
	})
	if err != nil {
		return err
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		return err
	}
	chargenOpts := chargen.Options{
		Data:               data,
		MaxEquips:          shared.Limits.MaxEquips,
		MaxHotbar:          shared.Limits.MaxHotbar,
		MaxPlayerSwitches:  shared.Limits.MaxPlayerSwitches,
		MaxPlayerVariables: shared.Limits.MaxPlayerVariables,
		StartPoints:        shared.StartPoints,
		MaxActors:          shared.Limits.MaxActors,
		MinCharacters:      shared.Limits.MinCharacters,
		MaxCharacters:      shared.Limits.MaxCharacters,
		MaxDefaultClasses:  shared.Limits.MaxDefaultClasses,
		MaxVIPClasses:      shared.Limits.MaxVIPClasses,
	}
	created := 0
	for i := 1; i <= opts.count; i++ {
		username := fmt.Sprintf("%s%04d", opts.prefix, i)
		name := fmt.Sprintf("Bot%04d", i)
		acc, err := st.GetAccountByUsername(ctx, username)
		if err != nil {
			if !isNoRows(err) {
				return err
			}
			salt, verifier, err := auth.NewVerifier(opts.password, opts.iterations)
			if err != nil {
				return err
			}
			acc = &store.Account{
				Username:           username,
				UsernameKey:        store.NameKey(username),
				Email:              username + "@load.invalid",
				PasswordAlgo:       "pbkdf2-sha256",
				PasswordSalt:       salt,
				PasswordIterations: opts.iterations,
				PasswordVerifier:   verifier,
			}
			if err := st.CreateAccount(ctx, acc); err != nil {
				return err
			}
			created++
		}
		if _, err := st.GetCharacterByName(ctx, name); err == nil {
			continue
		} else if !isNoRows(err) {
			return err
		}
		c, state, err := chargen.Create(chargenOpts, acc.ID, 0, name, 1, 0, 0, [8]int{})
		if err != nil {
			return err
		}
		mapID, x, y := spawn(i, opts.spread)
		c.MapID, c.X, c.Y = mapID, x, y
		c.Direction = 2
		c.ReviveMapID, c.ReviveX, c.ReviveY = mapID, x, y
		if err := st.CreateCharacter(ctx, c); err != nil {
			return err
		}
		if err := st.SaveCharacter(ctx, c, state); err != nil {
			return err
		}
	}
	fmt.Printf("setup ok: %d accounts ensured, %d new\n", opts.count, created)
	return nil
}

func isNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

func spawn(i int, spread bool) (int, int, int) {
	if spread {
		if i%2 == 0 {
			return 2, 3 + (i*7)%34, 3 + (i*11)%19
		}
		return 1, 3 + (i*7)%44, 3 + (i*11)%24
	}
	if i%2 == 0 {
		return 2, 4, 20
	}
	x := 21 + (i%5 - 2)
	y := 12 + ((i/5)%5 - 2)
	return 1, x, y
}

func run(opts options) error {
	cfg, err := config.Load(opts.configPath)
	if err != nil {
		return err
	}
	shared, err := content.LoadConfigs(filepath.Join(cfg.ContentDir, "configs.toml"))
	if err != nil {
		return err
	}
	start := time.Now()
	stop := make(chan struct{})
	time.AfterFunc(opts.duration, func() { close(stop) })
	var wg sync.WaitGroup
	for i := 1; i <= opts.count; i++ {
		wg.Add(1)
		delay := time.Duration(0)
		if opts.count > 1 && opts.ramp > 0 {
			delay = time.Duration(int64(opts.ramp) * int64(i-1) / int64(opts.count))
		}
		go func(i int) {
			defer wg.Done()
			if delay > 0 {
				select {
				case <-time.After(delay):
				case <-stop:
					return
				}
			}
			runBot(opts, uint16(shared.Network.GameVersion), i, stop)
		}(i)
	}
	wg.Wait()
	elapsed := time.Since(start)
	rep := report{
		StartedAt:       start.Format(time.RFC3339),
		Addr:            opts.addr,
		Count:           opts.count,
		DurationSeconds: elapsed.Seconds(),
		RampSeconds:     opts.ramp.Seconds(),
		MoveEveryMs:     opts.moveEvery.Milliseconds(),
		AttackEveryMs:   opts.attackEvery.Milliseconds(),
		Connected:       connected.Load(),
		AuthOK:          authOK.Load(),
		EnterOK:         enterOK.Load(),
		Failed:          failed.Load(),
		SentFrames:      sentFrames.Load(),
		RecvFrames:      recvFrames.Load(),
		RecvBytes:       recvBytes.Load(),
		Errors:          snapshotErrors(),
	}
	rep.AuthP50Ms, rep.AuthP95Ms = percentile(authTimes)
	rep.EnterP50Ms, rep.EnterP95Ms = percentile(enterTimes)
	if opts.jsonPath != "" {
		data, err := json.MarshalIndent(rep, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(opts.jsonPath, data, 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("bots=%d connected=%d auth=%d enter=%d failed=%d sent=%d recv=%d bytes=%d\n",
		opts.count, rep.Connected, rep.AuthOK, rep.EnterOK, rep.Failed, rep.SentFrames, rep.RecvFrames, rep.RecvBytes)
	fmt.Printf("auth p50=%.1fms p95=%.1fms enter p50=%.1fms p95=%.1fms\n",
		rep.AuthP50Ms, rep.AuthP95Ms, rep.EnterP50Ms, rep.EnterP95Ms)
	return nil
}

func runBot(opts options, gameVersion uint16, i int, stop <-chan struct{}) {
	username := fmt.Sprintf("%s%04d", opts.prefix, i)
	conn, err := net.DialTimeout("tcp", opts.addr, 10*time.Second)
	if err != nil {
		recordError("dial")
		return
	}
	connected.Add(1)
	defer conn.Close()

	authStart := time.Now()
	if err := authenticate(conn, username, opts.password, gameVersion); err != nil {
		recordError("auth")
		return
	}
	authOK.Add(1)
	recordTime(&authTimes, time.Since(authStart))

	enterStart := time.Now()
	if err := enterWorld(conn); err != nil {
		recordError("enter")
		return
	}
	enterOK.Add(1)
	recordTime(&enterTimes, time.Since(enterStart))
	_ = conn.SetReadDeadline(time.Time{})

	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			frame, err := protocol.ReadFrame(conn)
			if err != nil {
				return
			}
			recvFrames.Add(1)
			recvBytes.Add(int64(len(frame.Payload)))
			if protocol.ServerOp(frame.Op) == protocol.ServerError {
				recordError("server_error")
			}
		}
	}()

	move := time.NewTicker(opts.moveEvery)
	attack := time.NewTicker(opts.attackEvery)
	defer move.Stop()
	defer attack.Stop()
	directions := []uint8{2, 6, 4, 8}
	n := 0
	for {
		select {
		case <-stop:
			return
		case <-readerDone:
			return
		case <-move.C:
			if send(conn, protocol.ClientMove, protocol.Move{Direction: directions[n%len(directions)]}) == nil {
				sentFrames.Add(1)
			} else {
				recordError("move_write")
				return
			}
			n++
		case <-attack.C:
			if send(conn, protocol.ClientAttack, protocol.Attack{}) == nil {
				sentFrames.Add(1)
			} else {
				recordError("attack_write")
				return
			}
		}
	}
}

func authenticate(conn net.Conn, username, password string, gameVersion uint16) error {
	if err := protocol.WriteHello(conn, protocol.Hello{
		ProtocolVersion: protocol.ProtocolVersion, GameVersion: gameVersion, Build: "load",
	}); err != nil {
		return err
	}
	ack, err := protocol.ReadHelloAck(conn)
	if err != nil {
		return err
	}
	if ack.Status != protocol.StatusOK {
		return fmt.Errorf("handshake status %d", ack.Status)
	}
	if err := send(conn, protocol.ClientAuthBegin, protocol.AuthBegin{Username: username}); err != nil {
		return err
	}
	challenge, err := readChallenge(conn)
	if err != nil {
		return err
	}
	verifier, err := auth.Derive(password, challenge.Salt, int(challenge.Iterations), auth.VerifierLen)
	if err != nil {
		return err
	}
	clientNonce, err := auth.RandomBytes(auth.NonceLen)
	if err != nil {
		return err
	}
	proof := auth.Proof(verifier, username, challenge.Nonce, clientNonce)
	if err := send(conn, protocol.ClientAuthProof, protocol.AuthProof{ClientNonce: clientNonce, Proof: proof}); err != nil {
		return err
	}
	for {
		op, msg, err := readServer(conn)
		if err != nil {
			return err
		}
		if op == protocol.ServerAuthOK {
			return nil
		}
		if op == protocol.ServerAuthError {
			if e, ok := msg.(protocol.AuthError); ok {
				return fmt.Errorf("auth error %d: %s", e.Code, e.Message)
			}
			return fmt.Errorf("auth error")
		}
	}
}

func readChallenge(conn net.Conn) (protocol.AuthChallenge, error) {
	for {
		op, msg, err := readServer(conn)
		if err != nil {
			return protocol.AuthChallenge{}, err
		}
		if op == protocol.ServerAuthChallenge {
			if c, ok := msg.(protocol.AuthChallenge); ok {
				return c, nil
			}
			return protocol.AuthChallenge{}, fmt.Errorf("bad challenge")
		}
		if op == protocol.ServerAuthError {
			return protocol.AuthChallenge{}, fmt.Errorf("auth rejected")
		}
	}
}

func enterWorld(conn net.Conn) error {
	if err := send(conn, protocol.ClientSelectCharacter, protocol.SelectCharacter{Slot: 0}); err != nil {
		return err
	}
	for {
		op, msg, err := readServer(conn)
		if err != nil {
			return err
		}
		if op == protocol.ServerEnterWorld {
			return nil
		}
		if op == protocol.ServerError {
			if e, ok := msg.(protocol.ErrorMessage); ok {
				return fmt.Errorf("server error %d: %s", e.Code, e.Message)
			}
			return fmt.Errorf("server error")
		}
	}
}

func send(conn net.Conn, op protocol.ClientOp, msg protocol.ClientMessage) error {
	payload, err := msg.Encode()
	if err != nil {
		return err
	}
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return protocol.WriteFrame(conn, uint16(op), payload)
}

func readServer(conn net.Conn) (protocol.ServerOp, any, error) {
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	frame, err := protocol.ReadFrame(conn)
	if err != nil {
		return 0, nil, err
	}
	op := protocol.ServerOp(frame.Op)
	msg, err := protocol.DecodeServer(op, frame.Payload)
	if err != nil {
		return op, nil, err
	}
	return op, msg, nil
}

func recordError(kind string) {
	failed.Add(1)
	metricsMu.Lock()
	errorCounts[kind]++
	metricsMu.Unlock()
}

func recordTime(list *[]float64, d time.Duration) {
	metricsMu.Lock()
	*list = append(*list, float64(d)/float64(time.Millisecond))
	metricsMu.Unlock()
}

func snapshotErrors() map[string]int {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	out := make(map[string]int, len(errorCounts))
	for k, v := range errorCounts {
		out[k] = v
	}
	return out
}

func percentile(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}
	sort.Float64s(values)
	p50 := values[(len(values)-1)*50/100]
	p95 := values[(len(values)-1)*95/100]
	return p50, p95
}
