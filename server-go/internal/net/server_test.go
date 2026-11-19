package netserver

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	stdnet "net"

	"github.com/carabalonepaulo/vxaos/server/internal/auth"
	"github.com/carabalonepaulo/vxaos/server/internal/chargen"
	"github.com/carabalonepaulo/vxaos/server/internal/game"
	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
	"github.com/carabalonepaulo/vxaos/server/internal/rpg"
	"github.com/carabalonepaulo/vxaos/server/internal/store"
)

type testServer struct {
	srv  *Server
	ln   stdnet.Listener
	stop chan struct{}
}

func startTestServer(t *testing.T) (*testServer, *store.Store, int64) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	dir := t.TempDir()
	st, err := store.Open(context.Background(), store.Config{Driver: store.SQLite, DSN: filepath.Join(dir, "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	salt, verifier, err := auth.NewVerifier("secret", 1000)
	if err != nil {
		t.Fatal(err)
	}
	acc := &store.Account{
		Username:           "admin",
		PasswordAlgo:       "pbkdf2-sha256",
		PasswordSalt:       salt,
		PasswordIterations: 1000,
		PasswordVerifier:   verifier,
	}
	if err := st.CreateAccount(context.Background(), acc); err != nil {
		t.Fatal(err)
	}
	c := &store.Character{
		AccountID: acc.ID, Slot: 0, Name: "Hero", ClassID: 1, Sex: 0, Level: 1,
		HP: 100, MP: 20, Points: 5, MHP: 100, MMP: 20,
		ATK: 10, DEF: 8, MAT: 4, MDF: 5, AGI: 6, LUK: 7,
		CharacterName: "Charset01", CharacterIndex: 0,
		FaceName: "Charset01", FaceIndex: 0,
		MapID: 1, X: 21, Y: 12, Direction: 2,
		ReviveMapID: 1, ReviveX: 21, ReviveY: 12,
	}
	if err := st.CreateCharacter(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	data, err := rpg.Load(filepath.Join("..", "..", "..", "Client", "Data"))
	if err != nil {
		t.Fatal(err)
	}
	weaponID := 0
	for id, weapon := range data.Weapons {
		if id > 0 && weapon != nil && weapon.ETypeID == 0 {
			weaponID = id
			break
		}
	}
	state := &store.CharacterState{Version: store.StateVersion, Equips: make([]int, 9)}
	state.Equips[0] = weaponID
	if err := st.SaveCharacter(context.Background(), c, state); err != nil {
		t.Fatal(err)
	}
	world, err := game.New(game.Options{Data: data, Store: st, Tick: time.Hour, SaveInterval: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	ln, err := stdnet.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv, err := New(Options{
		World:          world,
		Store:          st,
		GameVersion:    12983,
		AuthIterations: 1000,
		AuthTimeout:    5 * time.Second,
		Inactivity:     30 * time.Second,
		Chargen: chargen.Options{
			Data:               data,
			MaxEquips:          9,
			MaxHotbar:          9,
			MaxPlayerSwitches:  100,
			MaxPlayerVariables: 100,
			StartPoints:        10,
			MaxActors:          3,
			MinCharacters:      3,
			MaxCharacters:      13,
			MaxDefaultClasses:  5,
			MaxVIPClasses:      9,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ts := &testServer{srv: srv, ln: ln, stop: make(chan struct{})}
	go func() {
		defer close(ts.stop)
		_ = srv.Serve(ctx, ln)
	}()
	t.Cleanup(func() {
		cancel()
		<-ts.stop
		wctx, wcancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer wcancel()
		_ = world.Stop(wctx)
		st.Close()
	})
	return ts, st, c.ID
}

func readServerFrame(t *testing.T, conn stdnet.Conn) (protocol.ServerOp, any) {
	t.Helper()
	for {
		op, msg := readServerFrameRaw(t, conn)
		if op == protocol.ServerEventData || op == protocol.ServerGlobalSwitches {
			// Enemy spawn state and the global switch block are streamed
			// after EnterWorld; tests that are not about them skip them.
			continue
		}
		return op, msg
	}
}

func readServerFrameRaw(t *testing.T, conn stdnet.Conn) (protocol.ServerOp, any) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	frame, err := protocol.ReadFrame(conn)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	op := protocol.ServerOp(frame.Op)
	msg, err := protocol.DecodeServer(op, frame.Payload)
	if err != nil {
		t.Fatalf("decode server frame %d: %v", frame.Op, err)
	}
	return op, msg
}

func sendClientFrame(t *testing.T, conn stdnet.Conn, op protocol.ClientOp, msg protocol.ClientMessage) {
	t.Helper()
	payload, err := msg.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if err := protocol.WriteFrame(conn, uint16(op), payload); err != nil {
		t.Fatal(err)
	}
}

func TestLoginSelectMoveChatLogout(t *testing.T) {
	ts, st, characterID := startTestServer(t)
	conn, err := stdnet.Dial("tcp", ts.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	if err := protocol.WriteHello(conn, protocol.Hello{ProtocolVersion: protocol.ProtocolVersion, GameVersion: 12983, Build: "test"}); err != nil {
		t.Fatal(err)
	}
	ack, err := protocol.ReadHelloAck(conn)
	if err != nil {
		t.Fatal(err)
	}
	if ack.Status != protocol.StatusOK {
		t.Fatalf("hello status = %d %s", ack.Status, ack.Message)
	}

	sendClientFrame(t, conn, protocol.ClientAuthBegin, protocol.AuthBegin{Username: "admin"})
	op, msg := readServerFrame(t, conn)
	if op != protocol.ServerAuthChallenge {
		t.Fatalf("auth op = %d msg %#v", op, msg)
	}
	challenge := msg.(protocol.AuthChallenge)
	verifier, err := auth.Derive("secret", challenge.Salt, int(challenge.Iterations), auth.VerifierLen)
	if err != nil {
		t.Fatal(err)
	}
	clientNonce, err := auth.RandomBytes(auth.NonceLen)
	if err != nil {
		t.Fatal(err)
	}
	proof := auth.Proof(verifier, "admin", challenge.Nonce, clientNonce)
	sendClientFrame(t, conn, protocol.ClientAuthProof, protocol.AuthProof{ClientNonce: clientNonce, Proof: proof})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerAuthOK {
		t.Fatalf("auth result op = %d msg %#v", op, msg)
	}
	authOK := msg.(protocol.AuthOK)
	if len(authOK.Actors) != 1 || authOK.Actors[0].Name != "Hero" {
		t.Fatalf("actors = %+v", authOK.Actors)
	}

	sendClientFrame(t, conn, protocol.ClientSelectCharacter, protocol.SelectCharacter{Slot: 0})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerEnterWorld {
		t.Fatalf("enter op = %d msg %#v", op, msg)
	}
	enter := msg.(protocol.EnterWorld)
	if enter.Player.Name != "Hero" || enter.Player.MapID != 1 || enter.Player.X != 21 || enter.Player.Y != 12 {
		t.Fatalf("enter = %+v", enter)
	}

	sendClientFrame(t, conn, protocol.ClientMove, protocol.Move{Direction: 6})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerPlayerMove {
		t.Fatalf("move op = %d msg %#v", op, msg)
	}
	move := msg.(protocol.PlayerMove)
	if move.X != 22 || move.Y != 12 || move.Direction != 6 {
		t.Fatalf("move = %+v", move)
	}

	sendClientFrame(t, conn, protocol.ClientChat, protocol.Chat{Channel: 0, Message: "hello"})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerMapChat {
		t.Fatalf("chat op = %d msg %#v", op, msg)
	}
	chat := msg.(protocol.MapChat)
	if !strings.Contains(chat.Message, "Hero: hello") {
		t.Fatalf("chat = %+v", chat)
	}

	sendClientFrame(t, conn, protocol.ClientLogout, protocol.LogoutRequest{})
	op, _ = readServerFrame(t, conn)
	if op != protocol.ServerLogout {
		t.Fatalf("logout op = %d", op)
	}
	// Leave hands the snapshot to the persistence worker.
	var got *store.Character
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, _, err = st.GetCharacter(context.Background(), characterID)
		if err == nil && got.X == 22 && got.Y == 12 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	if got.X != 22 || got.Y != 12 {
		t.Fatalf("saved position = %d,%d", got.X, got.Y)
	}
}

func login(t *testing.T, addr string) (stdnet.Conn, protocol.AuthOK) {
	t.Helper()
	conn, err := stdnet.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	if err := protocol.WriteHello(conn, protocol.Hello{ProtocolVersion: protocol.ProtocolVersion, GameVersion: 12983, Build: "test"}); err != nil {
		t.Fatal(err)
	}
	ack, err := protocol.ReadHelloAck(conn)
	if err != nil {
		t.Fatal(err)
	}
	if ack.Status != protocol.StatusOK {
		t.Fatalf("hello status = %d %s", ack.Status, ack.Message)
	}
	sendClientFrame(t, conn, protocol.ClientAuthBegin, protocol.AuthBegin{Username: "admin"})
	op, msg := readServerFrame(t, conn)
	if op != protocol.ServerAuthChallenge {
		t.Fatalf("auth op = %d msg %#v", op, msg)
	}
	challenge := msg.(protocol.AuthChallenge)
	verifier, err := auth.Derive("secret", challenge.Salt, int(challenge.Iterations), auth.VerifierLen)
	if err != nil {
		t.Fatal(err)
	}
	clientNonce, err := auth.RandomBytes(auth.NonceLen)
	if err != nil {
		t.Fatal(err)
	}
	proof := auth.Proof(verifier, "admin", challenge.Nonce, clientNonce)
	sendClientFrame(t, conn, protocol.ClientAuthProof, protocol.AuthProof{ClientNonce: clientNonce, Proof: proof})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerAuthOK {
		t.Fatalf("auth result op = %d msg %#v", op, msg)
	}
	return conn, msg.(protocol.AuthOK)
}

func TestCreateCharacter(t *testing.T) {
	ts, st, _ := startTestServer(t)
	conn, _ := login(t, ts.ln.Addr().String())
	defer conn.Close()
	sendClientFrame(t, conn, protocol.ClientCreateCharacter, protocol.CreateCharacter{
		Slot: 1, Name: "arqueira", CharacterIndex: 2, ClassID: 5, Sex: 1,
		Params: [8]uint8{1, 1, 1, 1, 1, 1, 1, 0},
	})
	op, msg := readServerFrame(t, conn)
	if op != protocol.ServerCharacterCreated {
		t.Fatalf("create op = %d msg %#v", op, msg)
	}
	created := msg.(protocol.CharacterCreated)
	if created.Actor.Slot != 1 || created.Actor.Name != "Arqueira" {
		t.Fatalf("created = %+v", created)
	}
	chars, err := st.ListCharacters(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(chars) != 2 {
		t.Fatalf("characters = %d", len(chars))
	}
	// Class 6 is above the non-VIP limit.
	sendClientFrame(t, conn, protocol.ClientCreateCharacter, protocol.CreateCharacter{
		Slot: 0, Name: "Cheater", ClassID: 6, Sex: 0,
	})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerCharacterError {
		t.Fatalf("class limit op = %d msg %#v", op, msg)
	}
	// Slot 1 is taken.
	sendClientFrame(t, conn, protocol.ClientCreateCharacter, protocol.CreateCharacter{
		Slot: 1, Name: "Other", ClassID: 1, Sex: 0,
	})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerCharacterError || msg.(protocol.CharacterError).Code != protocol.CodeCharacterSlot {
		t.Fatalf("slot op = %d msg %#v", op, msg)
	}
}

func TestEquipHotbarPoints(t *testing.T) {
	ts, st, characterID := startTestServer(t)
	conn, _ := login(t, ts.ln.Addr().String())
	defer conn.Close()
	sendClientFrame(t, conn, protocol.ClientSelectCharacter, protocol.SelectCharacter{Slot: 0})
	op, msg := readServerFrame(t, conn)
	if op != protocol.ServerEnterWorld {
		t.Fatalf("enter op = %d msg %#v", op, msg)
	}
	enter := msg.(protocol.EnterWorld)
	weaponID := enter.Player.Equips[0]
	if weaponID == 0 {
		t.Fatal("test character has no weapon")
	}

	// Unequip the weapon: the server returns it to the inventory.
	sendClientFrame(t, conn, protocol.ClientEquip, protocol.EquipItem{Slot: 0, ItemID: 0})
	item, equip := readEquipFrames(t, conn)
	if item.ItemID != weaponID || item.Kind != game.KindWeapon || item.Amount != 1 {
		t.Fatalf("unequip item = %+v", item)
	}
	if equip.Slot != 0 || equip.ItemID != 0 {
		t.Fatalf("unequip equip = %+v", equip)
	}

	// Equip it again from the inventory.
	sendClientFrame(t, conn, protocol.ClientEquip, protocol.EquipItem{Slot: 0, ItemID: weaponID})
	item, equip = readEquipFrames(t, conn)
	if item.Amount != -1 || equip.Slot != 0 || equip.ItemID != weaponID {
		t.Fatalf("equip item=%+v equip=%+v", item, equip)
	}

	// Unknown item ids are rejected without side effects.
	sendClientFrame(t, conn, protocol.ClientEquip, protocol.EquipItem{Slot: 0, ItemID: 60000})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerError {
		t.Fatalf("bad equip op = %d msg %#v", op, msg)
	}

	// Hotbar: type 0 always succeeds; an unowned item is rejected.
	sendClientFrame(t, conn, protocol.ClientSetHotbar, protocol.SetHotbar{Slot: 3, Type: 0})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerHotbar || msg.(protocol.Hotbar).Slot != 3 {
		t.Fatalf("hotbar op = %d msg %#v", op, msg)
	}
	sendClientFrame(t, conn, protocol.ClientSetHotbar, protocol.SetHotbar{Slot: 4, Type: 1, ItemID: 999})
	op, _ = readServerFrame(t, conn)
	if op != protocol.ServerError {
		t.Fatalf("unowned hotbar op = %d", op)
	}

	// Spend one point on HP: Param + Vitals + Points.
	sendClientFrame(t, conn, protocol.ClientSpendPoint, protocol.SpendPoint{ParamID: 0})
	param, vitals, points := readPointFrames(t, conn)
	if param.ParamID != 0 || param.Value != 10 {
		t.Fatalf("param = %+v", param)
	}
	if vitals.HP <= 0 || points.Value != enter.Player.Points-1 {
		t.Fatalf("vitals=%+v points=%+v", vitals, points)
	}
	var savedChar *store.Character
	var err error
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		savedChar, _, err = st.GetCharacter(context.Background(), characterID)
		if err == nil && savedChar.Points == int(enter.Player.Points-1) && savedChar.MHP == int(enter.Player.MHP+10) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	if savedChar.Points != int(enter.Player.Points-1) || savedChar.MHP != int(enter.Player.MHP+10) {
		t.Fatalf("saved points/mhp = %d/%d", savedChar.Points, savedChar.MHP)
	}
}

func TestEnterWorldSendsGlobalSwitches(t *testing.T) {
	ts, _, _ := startTestServer(t)
	conn, _ := login(t, ts.ln.Addr().String())
	defer conn.Close()
	sendClientFrame(t, conn, protocol.ClientSelectCharacter, protocol.SelectCharacter{Slot: 0})
	seenEnter := false
	for i := 0; i < 8; i++ {
		op, msg := readServerFrameRaw(t, conn)
		switch op {
		case protocol.ServerEnterWorld:
			seenEnter = true
		case protocol.ServerGlobalSwitches:
			if !seenEnter {
				t.Fatal("global switches arrived before EnterWorld")
			}
			if len(msg.(protocol.GlobalSwitches).Values) != 100 {
				t.Fatalf("global switch count = %d", len(msg.(protocol.GlobalSwitches).Values))
			}
			return
		}
	}
	t.Fatal("global switches were not sent")
}

func TestPartyInviteAcceptLeave(t *testing.T) {
	ts, st, _ := startTestServer(t)
	ctx := context.Background()
	ally := &store.Character{
		AccountID: 1, Slot: 1, Name: "Ally", ClassID: 1, Sex: 1, Level: 1,
		HP: 100, MP: 20, Points: 5, MHP: 100, MMP: 20,
		ATK: 10, DEF: 8, MAT: 4, MDF: 5, AGI: 6, LUK: 7,
		CharacterName: "Charset02", CharacterIndex: 1,
		FaceName: "Charset02", FaceIndex: 1,
		MapID: 1, X: 22, Y: 12, Direction: 2,
		ReviveMapID: 1, ReviveX: 22, ReviveY: 12,
	}
	if err := st.CreateCharacter(ctx, ally); err != nil {
		t.Fatal(err)
	}

	conn1, _ := login(t, ts.ln.Addr().String())
	defer conn1.Close()
	sendClientFrame(t, conn1, protocol.ClientSelectCharacter, protocol.SelectCharacter{Slot: 0})
	if op, msg := readServerFrame(t, conn1); op != protocol.ServerEnterWorld {
		t.Fatalf("hero enter = %d %#v", op, msg)
	}
	conn2, _ := login(t, ts.ln.Addr().String())
	defer conn2.Close()
	sendClientFrame(t, conn2, protocol.ClientSelectCharacter, protocol.SelectCharacter{Slot: 1})
	if op, msg := readServerFrame(t, conn2); op != protocol.ServerEnterWorld {
		t.Fatalf("ally enter = %d %#v", op, msg)
	}

	// The hero receives the ally's PlayerData and learns its session id.
	var allyID uint16
	for i := 0; i < 4 && allyID == 0; i++ {
		op, msg := readServerFrame(t, conn1)
		if op == protocol.ServerPlayerData {
			allyID = msg.(protocol.PlayerData).Player.ID
		}
	}
	if allyID == 0 {
		t.Fatal("hero did not receive ally PlayerData")
	}

	sendClientFrame(t, conn1, protocol.ClientRequest, protocol.RequestInvite{Type: 3, PlayerID: allyID})
	next := func(conn stdnet.Conn, want protocol.ServerOp) any {
		for i := 0; i < 10; i++ {
			op, msg := readServerFrame(t, conn)
			if op == want {
				return msg
			}
			if op == protocol.ServerPlayerData {
				continue
			}
			if op == protocol.ServerChat {
				continue
			}
			t.Fatalf("want op %d, got %d %#v", want, op, msg)
		}
		t.Fatalf("op %d not received", want)
		return nil
	}
	msg := next(conn2, protocol.ServerRequest)
	if text := msg.(protocol.PlayerRequest).Text; text == "" {
		t.Fatal("empty party invite text")
	}
	sendClientFrame(t, conn2, protocol.ClientRequestAccept, protocol.RequestAccept{})

	// Hero sees Ally join; Ally sees Hero join.
	msg = next(conn1, protocol.ServerPartyJoin)
	if msg.(protocol.PartyJoin).Name != "Ally" {
		t.Fatalf("hero party join = %#v", msg)
	}
	msg = next(conn2, protocol.ServerPartyJoin)
	if msg.(protocol.PartyJoin).Name != "Hero" {
		t.Fatalf("ally party join = %#v", msg)
	}

	sendClientFrame(t, conn2, protocol.ClientPartyLeave, protocol.PartyLeaveRequest{})
	next(conn2, protocol.ServerPartyDissolve)
	next(conn1, protocol.ServerPartyDissolve)
}

func TestDeleteCharacterFlow(t *testing.T) {
	ts, st, characterID := startTestServer(t)
	conn, err := stdnet.Dial("tcp", ts.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	if err := protocol.WriteHello(conn, protocol.Hello{ProtocolVersion: protocol.ProtocolVersion, GameVersion: 12983, Build: "test"}); err != nil {
		t.Fatal(err)
	}
	if _, err := protocol.ReadHelloAck(conn); err != nil {
		t.Fatal(err)
	}
	sendClientFrame(t, conn, protocol.ClientAuthBegin, protocol.AuthBegin{Username: "admin"})
	op, msg := readServerFrame(t, conn)
	if op != protocol.ServerAuthChallenge {
		t.Fatalf("challenge op = %d %#v", op, msg)
	}
	challenge := msg.(protocol.AuthChallenge)
	verifier, err := auth.Derive("secret", challenge.Salt, int(challenge.Iterations), auth.VerifierLen)
	if err != nil {
		t.Fatal(err)
	}
	loginNonce, err := auth.RandomBytes(auth.NonceLen)
	if err != nil {
		t.Fatal(err)
	}
	proof := auth.Proof(verifier, "admin", challenge.Nonce, loginNonce)
	sendClientFrame(t, conn, protocol.ClientAuthProof, protocol.AuthProof{ClientNonce: loginNonce, Proof: proof})
	if op, msg := readServerFrame(t, conn); op != protocol.ServerAuthOK {
		t.Fatalf("auth op = %d %#v", op, msg)
	}

	// Wrong proof is rejected and keeps the character.
	badNonce := make([]byte, auth.NonceLen)
	sendClientFrame(t, conn, protocol.ClientDeleteCharacter, protocol.DeleteCharacter{
		Slot: 0, ClientNonce: badNonce, Proof: make([]byte, auth.ProofLen),
	})
	if op, msg := readServerFrame(t, conn); op != protocol.ServerCharacterError {
		t.Fatalf("bad delete op = %d %#v", op, msg)
	}
	if chars, _ := st.ListCharacters(context.Background(), 1); len(chars) != 1 {
		t.Fatalf("character deleted with a bad proof: %d", len(chars))
	}

	// Valid proof deletes the slot.
	clientNonce := make([]byte, auth.NonceLen)
	deleteProof := auth.DeleteProof(verifier, "admin", challenge.Nonce, clientNonce, "Hero")
	sendClientFrame(t, conn, protocol.ClientDeleteCharacter, protocol.DeleteCharacter{
		Slot: 0, ClientNonce: clientNonce, Proof: deleteProof,
	})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerCharacterDeleted {
		t.Fatalf("delete op = %d %#v", op, msg)
	}
	if deleted := msg.(protocol.CharacterDeleted); deleted.Slot != 0 {
		t.Fatalf("deleted slot = %d", deleted.Slot)
	}
	chars, err := st.ListCharacters(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(chars) != 0 {
		t.Fatalf("characters after delete = %d", len(chars))
	}
	if _, _, err := st.GetCharacter(context.Background(), characterID); err == nil {
		t.Fatal("deleted character still loadable")
	}
}

func TestBankMessagesDispatch(t *testing.T) {
	ts, _, _ := startTestServer(t)
	conn, _ := login(t, ts.ln.Addr().String())
	defer conn.Close()
	sendClientFrame(t, conn, protocol.ClientSelectCharacter, protocol.SelectCharacter{Slot: 0})
	op, msg := readServerFrame(t, conn)
	if op != protocol.ServerEnterWorld {
		t.Fatalf("enter op = %d msg %#v", op, msg)
	}
	for _, tc := range []struct {
		op  protocol.ClientOp
		msg protocol.ClientMessage
	}{
		{protocol.ClientBankItem, protocol.BankItemRequest{ItemID: 1, Kind: 1, Amount: 1}},
		{protocol.ClientBankGold, protocol.BankGoldRequest{Amount: 5}},
	} {
		sendClientFrame(t, conn, tc.op, tc.msg)
		op, msg = readServerFrame(t, conn)
		if op != protocol.ServerError {
			t.Fatalf("bank op %d = %d %#v", tc.op, op, msg)
		}
		if message := msg.(protocol.ErrorMessage).Message; message == "unsupported message" {
			t.Fatalf("bank op %d was not dispatched", tc.op)
		}
	}
}

func readEquipFrames(t *testing.T, conn stdnet.Conn) (protocol.Item, protocol.Equip) {
	t.Helper()
	var item protocol.Item
	var equip protocol.Equip
	for i := 0; i < 2; i++ {
		op, msg := readServerFrame(t, conn)
		switch op {
		case protocol.ServerItem:
			item = msg.(protocol.Item)
		case protocol.ServerEquip:
			equip = msg.(protocol.Equip)
		default:
			t.Fatalf("unexpected op %d msg %#v", op, msg)
		}
	}
	return item, equip
}

func readPointFrames(t *testing.T, conn stdnet.Conn) (protocol.Param, protocol.Vitals, protocol.Points) {
	t.Helper()
	var param protocol.Param
	var vitals protocol.Vitals
	var points protocol.Points
	for i := 0; i < 3; i++ {
		op, msg := readServerFrame(t, conn)
		switch op {
		case protocol.ServerParam:
			param = msg.(protocol.Param)
		case protocol.ServerVitals:
			vitals = msg.(protocol.Vitals)
		case protocol.ServerPoints:
			points = msg.(protocol.Points)
		default:
			t.Fatalf("unexpected op %d msg %#v", op, msg)
		}
	}
	return param, vitals, points
}

func TestCombatMessages(t *testing.T) {
	ts, st, characterID := startTestServer(t)
	data, err := rpg.Load(filepath.Join("..", "..", "..", "Client", "Data"))
	if err != nil {
		t.Fatal(err)
	}
	var enemyEvent *rpg.Event
	for _, ev := range data.Maps[1].Events {
		if ev == nil || len(ev.Pages) == 0 || len(ev.Pages[0].List) == 0 || ev.Pages[0].List[0].Code != 108 {
			continue
		}
		if text, _ := ev.Pages[0].List[0].Parameters[0].(string); strings.HasPrefix(text, "Enemy") {
			enemyEvent = ev
			break
		}
	}
	if enemyEvent == nil {
		t.Fatal("no enemy event on map 1")
	}
	ctx := context.Background()
	c, state, err := st.GetCharacter(ctx, characterID)
	if err != nil {
		t.Fatal(err)
	}
	c.HP = 1
	c.X, c.Y = enemyEvent.X-1, enemyEvent.Y
	c.Direction = 6
	state.Inventory = []store.ItemStack{{Kind: game.KindItem, ItemID: 2, Amount: 2}}
	state.Skills = append(state.Skills, 1)
	if err := st.SaveCharacter(ctx, c, state); err != nil {
		t.Fatal(err)
	}

	conn, _ := login(t, ts.ln.Addr().String())
	defer conn.Close()
	sendClientFrame(t, conn, protocol.ClientSelectCharacter, protocol.SelectCharacter{Slot: 0})
	op, msg := readServerFrame(t, conn)
	if op != protocol.ServerEnterWorld {
		t.Fatalf("enter op = %d msg %#v", op, msg)
	}

	sendClientFrame(t, conn, protocol.ClientUseItem, protocol.UseItem{ItemID: 2})
	gotAttack := false
	gotVitals := false
	gotItem := false
	gotAnimation := false
	for i := 0; i < 4; i++ {
		op, msg = readServerFrame(t, conn)
		switch op {
		case protocol.ServerAttackPlayer:
			gotAttack = true
		case protocol.ServerVitals:
			gotVitals = true
		case protocol.ServerItem:
			gotItem = true
		case protocol.ServerAnimation:
			gotAnimation = true
		default:
			t.Fatalf("item frame op = %d msg %#v", op, msg)
		}
	}
	if !gotAttack || !gotVitals || !gotItem || !gotAnimation {
		t.Fatalf("item frames attack=%v vitals=%v item=%v animation=%v", gotAttack, gotVitals, gotItem, gotAnimation)
	}
	deadline := time.Now().Add(2 * time.Second)
	var saved *store.Character
	for time.Now().Before(deadline) {
		saved, _, err = st.GetCharacter(ctx, characterID)
		if err == nil && saved.HP > 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	if saved.HP <= 1 {
		t.Fatalf("item did not heal, hp=%d", saved.HP)
	}

	time.Sleep(850 * time.Millisecond)
	sendClientFrame(t, conn, protocol.ClientSetTarget, protocol.Target{Type: 2, ID: uint16(enemyEvent.ID)})
	time.Sleep(50 * time.Millisecond)
	sendClientFrame(t, conn, protocol.ClientUseSkill, protocol.UseSkill{SkillID: 1})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerAttackEnemy {
		t.Fatalf("skill op = %d msg %#v", op, msg)
	}
	op, msg = readServerFrame(t, conn)
	for i := 0; i < 5 && op != protocol.ServerVitals; i++ {
		switch op {
		case protocol.ServerExp, protocol.ServerGold, protocol.ServerItem, protocol.ServerDropAdd, protocol.ServerDropRemove:
			op, msg = readServerFrame(t, conn)
		default:
			t.Fatalf("skill vitals op = %d msg %#v", op, msg)
		}
	}
	if op != protocol.ServerVitals {
		t.Fatalf("skill vitals op = %d msg %#v", op, msg)
	}
}

func TestBadPassword(t *testing.T) {
	ts, _, _ := startTestServer(t)
	conn, err := stdnet.Dial("tcp", ts.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	_ = protocol.WriteHello(conn, protocol.Hello{ProtocolVersion: protocol.ProtocolVersion, GameVersion: 12983})
	if _, err := protocol.ReadHelloAck(conn); err != nil {
		t.Fatal(err)
	}
	sendClientFrame(t, conn, protocol.ClientAuthBegin, protocol.AuthBegin{Username: "admin"})
	_, msg := readServerFrame(t, conn)
	challenge := msg.(protocol.AuthChallenge)
	verifier, _ := auth.Derive("wrong", challenge.Salt, int(challenge.Iterations), auth.VerifierLen)
	nonce, _ := auth.RandomBytes(auth.NonceLen)
	proof := auth.Proof(verifier, "admin", challenge.Nonce, nonce)
	sendClientFrame(t, conn, protocol.ClientAuthProof, protocol.AuthProof{ClientNonce: nonce, Proof: proof})
	op, msg := readServerFrame(t, conn)
	if op != protocol.ServerAuthError || msg.(protocol.AuthError).Code != protocol.AuthInvalidCredentials {
		t.Fatalf("auth error = %d %#v", op, msg)
	}
}

func TestUnknownAccountGetsFakeChallenge(t *testing.T) {
	ts, _, _ := startTestServer(t)
	conn, err := stdnet.Dial("tcp", ts.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	_ = protocol.WriteHello(conn, protocol.Hello{ProtocolVersion: protocol.ProtocolVersion, GameVersion: 12983})
	if _, err := protocol.ReadHelloAck(conn); err != nil {
		t.Fatal(err)
	}
	sendClientFrame(t, conn, protocol.ClientAuthBegin, protocol.AuthBegin{Username: "nobody"})
	op, msg := readServerFrame(t, conn)
	if op != protocol.ServerAuthChallenge {
		t.Fatalf("expected fake challenge, got %d %#v", op, msg)
	}
	challenge := msg.(protocol.AuthChallenge)
	if len(challenge.Salt) == 0 || len(challenge.Nonce) == 0 {
		t.Fatalf("fake challenge = %+v", challenge)
	}
	nonce, _ := auth.RandomBytes(auth.NonceLen)
	sendClientFrame(t, conn, protocol.ClientAuthProof, protocol.AuthProof{ClientNonce: nonce, Proof: bytes.Repeat([]byte{1}, auth.ProofLen)})
	op, msg = readServerFrame(t, conn)
	if op != protocol.ServerAuthError {
		t.Fatalf("expected auth error, got %d %#v", op, msg)
	}
}
