package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
	"testing"
	"testing/iotest"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, 42, []byte{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 4+2+3 {
		t.Fatalf("frame length = %d", buf.Len())
	}
	f, err := ReadFrame(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if f.Op != 42 || !bytes.Equal(f.Payload, []byte{1, 2, 3}) {
		t.Fatalf("frame = %+v", f)
	}
}

func TestFrameFragmented(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, 7, []byte("hello fragmented world")); err != nil {
		t.Fatal(err)
	}
	r := iotest.OneByteReader(bytes.NewReader(buf.Bytes()))
	f, err := ReadFrame(r)
	if err != nil {
		t.Fatal(err)
	}
	if f.Op != 7 || string(f.Payload) != "hello fragmented world" {
		t.Fatalf("frame = %+v", f)
	}
}

func TestFrameRejectsOversize(t *testing.T) {
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], MaxFrameSize+1)
	if _, err := ReadFrame(bytes.NewReader(header[:])); !errors.Is(err, ErrFrameTooBig) {
		t.Fatalf("err = %v, want ErrFrameTooBig", err)
	}
	if err := WriteFrame(&bytes.Buffer{}, 1, make([]byte, MaxFrameSize)); !errors.Is(err, ErrFrameTooBig) {
		t.Fatalf("write err = %v", err)
	}
}

func TestFrameRejectsShort(t *testing.T) {
	var header [4]byte
	binary.LittleEndian.PutUint32(header[:], 1)
	if _, err := ReadFrame(bytes.NewReader(header[:])); !errors.Is(err, ErrFrameTooShort) {
		t.Fatalf("err = %v, want ErrFrameTooShort", err)
	}
	if _, err := ReadFrame(bytes.NewReader([]byte{1, 2})); err == nil {
		t.Fatal("expected truncated frame error")
	}
}

func TestHelloRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	want := Hello{ProtocolVersion: ProtocolVersion, GameVersion: 12983, Build: "dev"}
	if err := WriteHello(&buf, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHello(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("hello = %+v, want %+v", got, want)
	}
}

func TestHelloAckRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	want := HelloAck{
		ProtocolVersion: ProtocolVersion,
		GameVersion:     12983,
		ServerTime:      1700000000,
		MaxFrameSize:    MaxFrameSize,
		Status:          StatusOK,
		Message:         "ok",
	}
	if err := WriteHelloAck(&buf, want); err != nil {
		t.Fatal(err)
	}
	got, err := ReadHelloAck(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("ack = %+v, want %+v", got, want)
	}
}

func TestHelloBadMagic(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("NOTVXAOS")
	if _, err := ReadHello(&buf); !errors.Is(err, ErrMagic) {
		t.Fatalf("err = %v, want ErrMagic", err)
	}
}

func TestAuthMessagesRoundTrip(t *testing.T) {
	t.Run("auth-begin", func(t *testing.T) {
		m := AuthBegin{Username: "admin"}
		payload, err := m.Encode()
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeClient(ClientAuthBegin, payload)
		if err != nil {
			t.Fatal(err)
		}
		if got.(AuthBegin) != m {
			t.Fatalf("got %+v", got)
		}
	})
	t.Run("auth-challenge", func(t *testing.T) {
		m := AuthChallenge{Salt: []byte("salt"), KDF: 1, Iterations: 600000, Nonce: []byte("nonce"), ServerTime: 123}
		payload, err := m.Encode()
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeServer(ServerAuthChallenge, payload)
		if err != nil {
			t.Fatal(err)
		}
		g := got.(AuthChallenge)
		if !bytes.Equal(g.Salt, m.Salt) || !bytes.Equal(g.Nonce, m.Nonce) || g.KDF != m.KDF || g.Iterations != m.Iterations || g.ServerTime != m.ServerTime {
			t.Fatalf("got %+v", g)
		}
	})
	t.Run("auth-proof", func(t *testing.T) {
		m := AuthProof{ClientNonce: make([]byte, 16), Proof: make([]byte, 32)}
		payload, err := m.Encode()
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeClient(ClientAuthProof, payload)
		if err != nil {
			t.Fatal(err)
		}
		g := got.(AuthProof)
		if !bytes.Equal(g.ClientNonce, m.ClientNonce) || !bytes.Equal(g.Proof, m.Proof) {
			t.Fatalf("got %+v", g)
		}
	})
	t.Run("auth-ok", func(t *testing.T) {
		m := AuthOK{Group: 2, VIPUntil: 123456, Actors: []ActorSummary{
			{Slot: 0, Name: "Admin", CharacterName: "Charset01", CharacterIndex: 0, FaceName: "Charset01", FaceIndex: 0, Sex: 0, Equips: [MaxEquips]uint16{1, 46, 0, 1, 0}},
			{Slot: 1, Name: "Lola", CharacterName: "Charset01", CharacterIndex: 1, Sex: 1},
		}}
		payload, err := m.Encode()
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeServer(ServerAuthOK, payload)
		if err != nil {
			t.Fatal(err)
		}
		g := got.(AuthOK)
		if g.Group != m.Group || g.VIPUntil != m.VIPUntil || len(g.Actors) != 2 || g.Actors[0].Name != "Admin" || g.Actors[1].Sex != 1 {
			t.Fatalf("got %+v", g)
		}
	})
}

func TestOtherMessagesRoundTrip(t *testing.T) {
	cases := []struct {
		op      ClientOp
		message ClientMessage
	}{
		{ClientSelectCharacter, SelectCharacter{Slot: 2}},
		{ClientCreateCharacter, CreateCharacter{Slot: 1, Name: "New", CharacterIndex: 0, ClassID: 1, Sex: 0, Params: [8]uint8{1, 2, 0, 3, 0, 0, 4, 0}}},
		{ClientMove, Move{Direction: 6}},
		{ClientChat, Chat{Channel: 1, Message: "olÃ¡", Target: ""}},
		{ClientEquip, EquipItem{Slot: 0, ItemID: 1}},
		{ClientSetHotbar, SetHotbar{Slot: 2, Type: 1, ItemID: 9}},
		{ClientSpendPoint, SpendPoint{ParamID: 3}},
		{ClientAttack, Attack{}},
		{ClientSetTarget, Target{Type: 2, ID: 5}},
		{ClientUseItem, UseItem{ItemID: 1}},
		{ClientUseSkill, UseSkill{SkillID: 3}},
		{ClientUseHotbar, UseHotbar{Slot: 2}},
		{ClientPickupDrop, PickupDrop{DropID: 70000}},
		{ClientChoice, Choice{Index: 3}},
		{ClientChoiceTeleport, ChoiceTeleport{Index: 1}},
		{ClientShopBuy, ShopBuy{Index: 2, Amount: 3}},
		{ClientShopSell, ShopSell{ItemID: 4, Kind: 1, Amount: 2}},
		{ClientBankItem, BankItemRequest{ItemID: 7, Kind: 2, Amount: -12}},
		{ClientBankGold, BankGoldRequest{Amount: -250}},
		{ClientGuildCreate, GuildCreate{Name: "Teste", Flag: make([]byte, 64)}},
		{ClientGuildOpen, GuildOpenRequest{}},
		{ClientGuildSetLeader, GuildSetLeader{Name: "Hero"}},
		{ClientGuildSetNotice, GuildSetNotice{Notice: "aviso"}},
		{ClientGuildKick, GuildKick{Name: "Hero"}},
		{ClientGuildInvite, GuildInvite{Name: "Hero"}},
		{ClientGuildLeave, GuildLeave{}},
		{ClientRequestAccept, RequestAccept{}},
		{ClientRequestDecline, RequestDecline{}},
		{ClientRequest, RequestInvite{Type: 3, PlayerID: 2}},
		{ClientPartyLeave, PartyLeaveRequest{}},
		{ClientDeleteCharacter, DeleteCharacter{Slot: 1, ClientNonce: make([]byte, 16), Proof: make([]byte, 32)}},
		{ClientOpenFriends, OpenFriends{}},
		{ClientRemoveFriend, RemoveFriend{Index: 2}},
		{ClientTradeItem, TradeItemOffer{ItemID: 4, Kind: 1, Amount: -2}},
		{ClientTradeGold, TradeGoldOffer{Amount: 25}},
		{ClientCloseWindow, CloseWindow{}},
		{ClientEventNext, EventNext{}},
		{ClientPing, Ping{Timestamp: 999}},
		{ClientRegisterBegin, RegisterBegin{Username: "new", Email: "a@b.c"}},
		{ClientRegisterProof, RegisterProof{ClientNonce: make([]byte, 16), Verifier: make([]byte, 32), Proof: make([]byte, 32)}},
	}
	for _, tc := range cases {
		payload, err := tc.message.Encode()
		if err != nil {
			t.Fatalf("%T encode: %v", tc.message, err)
		}
		got, err := DecodeClient(tc.op, payload)
		if err != nil {
			t.Fatalf("%T decode: %v", tc.message, err)
		}
		if got == nil {
			t.Fatalf("%T decoded nil", tc.message)
		}
	}

	serverCases := []struct {
		op      ServerOp
		message ServerMessage
	}{
		{ServerPong, Pong{Timestamp: 999}},
		{ServerError, ErrorMessage{Code: CodeBadRequest, Message: "bad"}},
		{ServerDisconnect, Disconnect{Code: CodeInternal, Message: "bye"}},
		{ServerRegisterOK, RegisterOK{}},
		{ServerRegisterError, RegisterError{Code: AuthAccountExists, Message: "exists"}},
		{ServerCharacterCreated, CharacterCreated{Actor: ActorSummary{Slot: 1, Name: "New", CharacterName: "Charset01", CharacterIndex: 2, FaceName: "Charset01", FaceIndex: 2, Sex: 1, Equips: [MaxEquips]uint16{1, 2, 3}}}},
		{ServerCharacterError, CharacterError{Code: AuthAccountExists, Message: "name exists"}},
		{ServerEnterWorld, EnterWorld{Player: PlayerSnapshot{ID: 3, Name: "Hero", CharacterName: "Charset01", MapID: 1, X: 10, Y: 20, Direction: 2, Level: 5, HP: 50, MP: 10, MHP: 60, MMP: 12, EXP: 99, Gold: 1234}}},
		{ServerPlayerData, PlayerData{Player: PlayerSnapshot{ID: 4, Name: "Other", MapID: 2, X: 1, Y: 2, Direction: 4}}},
		{ServerPlayerMove, PlayerMove{ID: 4, X: 2, Y: 2, Direction: 6}},
		{ServerPlayerLeft, PlayerLeft{ID: 4}},
		{ServerMapChat, MapChat{PlayerID: 3, Color: 1, Message: "hi"}},
		{ServerChat, ChatMessage{Color: 1, Message: "global"}},
		{ServerEquip, Equip{PlayerID: 3, Slot: 1, ItemID: 46}},
		{ServerVitals, Vitals{PlayerID: 3, HP: 70, MP: 30}},
		{ServerHotbar, Hotbar{Slot: 2, Type: 2, ItemID: 7}},
		{ServerParam, Param{PlayerID: 3, ParamID: 1, Value: 10}},
		{ServerPoints, Points{Value: 4}},
		{ServerItem, Item{ItemID: 9, Kind: 1, Amount: -1, DropSound: true, Popup: false}},
		{ServerGold, Gold{Amount: -250, ShopSound: true, Popup: true}},
		{ServerSkill, Skill{SkillID: 5, Learned: true}},
		{ServerAttackEnemy, AttackEnemy{AttackerID: 3, AttackerType: 0, AniIndex: 1, EventID: 8, HPDamage: -12, MPDamage: 0, Critical: true, AnimationID: 7}},
		{ServerAttackPlayer, AttackPlayer{AttackerID: 8, AttackerType: 1, AniIndex: 0, PlayerID: 3, HPDamage: -4, Critical: false, AnimationID: 1, NotShowMissed: true}},
		{ServerEventData, EventData{EventID: 8, X: 10, Y: 12, Direction: 4, HP: 40}},
		{ServerEventMove, EventMove{EventID: 8, X: 11, Y: 12, Direction: 6}},
		{ServerEventRemove, EventRemove{EventID: 8}},
		{ServerMapReset, MapReset{}},
		{ServerMapPlayerEnter, MapPlayerEnter{ID: 4, Name: "Ally"}},
		{ServerMapPlayerLeave, MapPlayerLeave{ID: 4, Name: "Ally"}},
		{ServerDropAdd, DropAdd{DropID: 70000, ItemID: 9, Kind: 1, Amount: 2, X: 10, Y: 12}},
		{ServerDropRemove, DropRemove{DropID: 70000}},
		{ServerEnemyRevive, EnemyRevive{EventID: 8}},
		{ServerAnimation, Animation{AttackerID: 3, AttackerType: 0, AniIndex: 0, CharacterID: 8, CharacterType: 2, AnimationID: 7}},
		{ServerExp, Exp{PlayerID: 3, Amount: 12}},
		{ServerTransfer, Transfer{MapID: 2, X: 4, Y: 5, Direction: 2}},
		{ServerEventCommand, EventCommand{EventID: -3, InitialIndex: 1, FinalIndex: 4}},
		{ServerSwitch, Switch{ID: 2, Value: true}},
		{ServerVariable, Variable{ID: 5, Value: 42}},
		{ServerSelfSwitch, SelfSwitch{MapID: 1, EventID: 8, Ch: "A", Value: true}},
		{ServerQuestAdd, QuestAdd{QuestID: 2}},
		{ServerQuestFinish, QuestFinish{QuestID: 2}},
		{ServerVIPDays, VIPDays{Until: 123456}},
		{ServerSex, Sex{PlayerID: 3, Sex: 1}},
		{ServerGraphic, Graphic{PlayerID: 3, CharacterName: "Charset02", CharacterIndex: 1, FaceName: "Face02", FaceIndex: 1}},
		{ServerTeleportOpen, TeleportOpen{ID: 2}},
		{ServerShopOpen, ShopOpen{EventID: 15, Index: 0}},
		{ServerBankOpen, BankOpen{Gold: 500, Stacks: []BankStack{
			{Kind: 1, ItemID: 7, Amount: 2},
			{Kind: 2, ItemID: 13, Amount: 1},
			{Kind: 3, ItemID: 26, Amount: 5},
		}}},
		{ServerBankItem, BankItem{ItemID: 9, Kind: 1, Amount: -3}},
		{ServerBankGold, BankGold{Amount: 99}},
		{ServerGuildCreateOpen, GuildCreateOpen{}},
		{ServerGuildOpen, GuildOpen{Leader: "Hero", Notice: "oi", Flag: make([]byte, 64), Members: []string{"Hero", "Ally"}, Online: 2}},
		{ServerGuildName, GuildName{PlayerID: 3, Name: "Teste"}},
		{ServerGuildLeader, GuildLeader{Name: "Hero"}},
		{ServerGuildNotice, GuildNotice{Notice: "aviso"}},
		{ServerGuildMemberRemove, GuildMemberRemove{Name: "Ally"}},
		{ServerRequest, PlayerRequest{Type: 5, FromID: 3, Text: "convite"}},
		{ServerPartyJoin, PartyJoin{PlayerID: 2, Name: "Ally", CharacterName: "Charset02", CharacterIndex: 1, Sex: 1, MHP: 100, HP: 90, EXP: 5}},
		{ServerPartyLeave, PartyLeave{PlayerID: 2}},
		{ServerPartyDissolve, PartyDissolve{}},
		{ServerFriends, Friends{Online: 1, Names: []string{"Hero"}}},
		{ServerFriendAdd, FriendAdd{Name: "Hero"}},
		{ServerFriendRemove, FriendRemove{Index: 1}},
		{ServerRequestAccept, RequestAccepted{Type: 1}},
		{ServerWindowClose, WindowClose{}},
		{ServerTradeItem, TradeItem{PlayerID: 2, ItemID: 3, Kind: 1, Amount: -1}},
		{ServerTradeGold, TradeGold{PlayerID: 2, Amount: 50}},
		{ServerCharacterDeleted, CharacterDeleted{Slot: 1}},
		{ServerState, State{StateID: 2, Add: true}},
		{ServerBuff, Buff{ParamID: 2, Level: -1}},
		{ServerLogout, Logout{}},
		{ServerGlobalSwitches, GlobalSwitches{Values: []bool{true, false, true}}},
	}
	for _, tc := range serverCases {
		payload, err := tc.message.Encode()
		if err != nil {
			t.Fatalf("%T encode: %v", tc.message, err)
		}
		if _, err := DecodeServer(tc.op, payload); err != nil {
			t.Fatalf("%T decode: %v", tc.message, err)
		}
	}
}

func TestDecodeValidation(t *testing.T) {
	if _, err := DecodeClient(ClientMove, []byte{5}); err == nil {
		t.Fatal("direction 5 accepted")
	}
	if _, err := DecodeClient(ClientChat, []byte{1, 1, 0, 'x', 0, 0}); err != nil {
		t.Fatalf("valid chat rejected: %v", err)
	}
	// Invalid UTF-8 in a length-1 string.
	if _, err := DecodeClient(ClientAuthBegin, []byte{0, 1, 0xff}); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
	if _, err := DecodeClient(ClientOp(9999), nil); !errors.Is(err, ErrUnknownOpcode) {
		t.Fatalf("unknown opcode err = %v", err)
	}
	if _, err := DecodeClient(ClientEquip, []byte{MaxEquips, 0, 0}); err == nil {
		t.Fatal("equip slot out of range accepted")
	}
	if _, err := DecodeClient(ClientSetHotbar, []byte{0, 3, 0, 0}); err == nil {
		t.Fatal("hotbar type out of range accepted")
	}
	if _, err := DecodeClient(ClientSpendPoint, []byte{8}); err == nil {
		t.Fatal("param id out of range accepted")
	}
}

func TestActorCountLimit(t *testing.T) {
	w := NewWriter()
	w.U8(0)
	w.I64(0)
	w.Count(MaxActorsInList + 1)
	payload, _ := w.Done()
	if _, err := DecodeServer(ServerAuthOK, payload); err == nil {
		t.Fatal("oversized actor list accepted")
	}
}

func TestChatLimit(t *testing.T) {
	tooLong := strings.Repeat("x", MaxChatBytes+1)
	payload, err := (Chat{Channel: 1, Message: tooLong}).Encode()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeClient(ClientChat, payload); err == nil {
		t.Fatal("oversized chat accepted")
	}
}
