package game

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kelaia/vxaos/server/internal/content"
	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

type sentFrame struct {
	op  protocol.ServerOp
	msg protocol.ServerMessage
}

type fakeClient struct {
	mu     sync.Mutex
	frames []sentFrame
	closed bool
}

func (f *fakeClient) Send(op protocol.ServerOp, msg protocol.ServerMessage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.frames = append(f.frames, sentFrame{op: op, msg: msg})
	return nil
}

func (f *fakeClient) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeClient) find(op protocol.ServerOp) []protocol.ServerMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []protocol.ServerMessage
	for _, fr := range f.frames {
		if fr.op == op {
			out = append(out, fr.msg)
		}
	}
	return out
}

func (f *fakeClient) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.frames)
}

func testWorld(t *testing.T) (*World, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(context.Background(), store.Config{Driver: store.SQLite, DSN: filepath.Join(dir, "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := rpg.Load(filepath.Join("..", "..", "..", "Client", "Data"))
	if err != nil {
		t.Fatal(err)
	}
	quests, err := content.LoadQuests(filepath.Join("..", "..", "content", "quests.toml"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := New(Options{
		Data:         data,
		Store:        st,
		Tick:         time.Hour,
		SaveInterval: time.Hour,
		Quests:       quests,
		InlineMaps:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		w.Stop(ctx)
	})
	return w, st
}

func createPlayer(t *testing.T, st *store.Store, name string, x, y int) *Player {
	t.Helper()
	ctx := context.Background()
	acc := &store.Account{
		Username:           name + "user",
		PasswordAlgo:       "pbkdf2-sha256",
		PasswordSalt:       []byte("0123456789abcdef"),
		PasswordIterations: 1000,
		PasswordVerifier:   []byte(strings.Repeat("v", 32)),
	}
	if err := st.CreateAccount(ctx, acc); err != nil {
		t.Fatal(err)
	}
	c := &store.Character{
		AccountID: acc.ID, Slot: 0, Name: name, ClassID: 1, Sex: 0, Level: 1,
		HP: 100, MP: 20, Points: 5, MHP: 100, MMP: 20,
		ATK: 10, DEF: 8, MAT: 4, MDF: 5, AGI: 6, LUK: 7,
		CharacterName: "Charset01", CharacterIndex: 0,
		FaceName: "Charset01", FaceIndex: 0,
		MapID: 1, X: x, Y: y, Direction: 2,
		ReviveMapID: 1, ReviveX: x, ReviveY: y,
		Gold: 10,
	}
	if err := st.CreateCharacter(ctx, c); err != nil {
		t.Fatal(err)
	}
	got, state, err := st.GetCharacter(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	return &Player{Account: acc, Character: got, State: state, Client: &fakeClient{}}
}

func TestJoinMoveChatLeave(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p1 := createPlayer(t, st, "Hero", 21, 12)
	p2 := createPlayer(t, st, "Ally", 21, 12)
	if err := w.Join(ctx, p1); err != nil {
		t.Fatal(err)
	}
	if len(p1.Client.(*fakeClient).find(protocol.ServerEnterWorld)) != 1 {
		t.Fatal("p1 did not receive EnterWorld")
	}
	if err := w.Join(ctx, p2); err != nil {
		t.Fatal(err)
	}
	if len(p2.Client.(*fakeClient).find(protocol.ServerEnterWorld)) != 1 {
		t.Fatal("p2 did not receive EnterWorld")
	}
	if len(p1.Client.(*fakeClient).find(protocol.ServerPlayerData)) != 1 {
		t.Fatal("p1 did not receive p2 PlayerData")
	}
	if len(p2.Client.(*fakeClient).find(protocol.ServerPlayerData)) != 1 {
		t.Fatal("p2 did not receive p1 PlayerData")
	}
	// p1 moves right from 21,12 to 22,12 (passable and empty).
	if err := w.Move(ctx, p1, 6); err != nil {
		t.Fatal(err)
	}
	if p1.Character.X != 22 || p1.Character.Y != 12 || p1.Character.Direction != 6 {
		t.Fatalf("p1 position = %d,%d dir %d", p1.Character.X, p1.Character.Y, p1.Character.Direction)
	}
	if len(p1.Client.(*fakeClient).find(protocol.ServerPlayerMove)) != 1 {
		t.Fatal("p1 did not receive its move")
	}
	if len(p2.Client.(*fakeClient).find(protocol.ServerPlayerMove)) != 1 {
		t.Fatal("p2 did not receive the move broadcast")
	}
	if err := w.Chat(ctx, p1, 0, "hello", ""); err != nil {
		t.Fatal(err)
	}
	if len(p1.Client.(*fakeClient).find(protocol.ServerMapChat)) != 1 {
		t.Fatal("p1 did not receive map chat")
	}
	if len(p2.Client.(*fakeClient).find(protocol.ServerMapChat)) != 1 {
		t.Fatal("p2 did not receive map chat")
	}
	_ = w.do(ctx, func() error {
		w.online[p1.ID].LastChat = time.Time{}
		return nil
	})
	if err := w.Chat(ctx, p1, 1, "global", ""); err != nil {
		t.Fatal(err)
	}
	if len(p2.Client.(*fakeClient).find(protocol.ServerChat)) != 1 {
		t.Fatal("p2 did not receive global chat")
	}
	// Leave p1; it must be saved and removed, and p2 gets PlayerLeft.
	if err := w.Leave(ctx, p1, true); err != nil {
		t.Fatal(err)
	}
	if len(p2.Client.(*fakeClient).find(protocol.ServerPlayerLeft)) != 1 {
		t.Fatal("p2 did not receive PlayerLeft")
	}
	got, _, err := st.GetCharacter(ctx, p1.Character.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.X != 22 || got.Y != 12 || got.Direction != 6 {
		t.Fatalf("saved position = %d,%d dir %d", got.X, got.Y, got.Direction)
	}
}

func TestRateLimits(t *testing.T) {
	w, st := testWorld(t)
	ctx := context.Background()
	p := createPlayer(t, st, "Hero", 21, 12)
	if err := w.Join(ctx, p); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, p, 6); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(ctx, p, 6); err != ErrRateLimited {
		t.Fatalf("second move err = %v", err)
	}
	if err := w.Chat(ctx, p, 0, "one", ""); err != nil {
		t.Fatal(err)
	}
	if err := w.Chat(ctx, p, 0, "two", ""); err != ErrRateLimited {
		t.Fatalf("second chat err = %v", err)
	}
}
