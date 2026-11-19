package content

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/carabalonepaulo/vxaos/server/internal/protocol"
)

func TestLoadEnums(t *testing.T) {
	file, err := LoadEnums("../../content/enums.toml")
	if err != nil {
		t.Fatal(err)
	}
	if len(file.Enums) < 15 {
		t.Fatalf("enum groups = %d", len(file.Enums))
	}
	check(t, file, "Sex", "MALE", 0)
	check(t, file, "Sex", "FEMALE", 1)
	check(t, file, "Group", "ADMIN", 2)
	check(t, file, "Dir", "RIGHT", 6)
	check(t, file, "Command", "KICK", 0)
	check(t, file, "Command", "MSG", 14)
	check(t, file, "Target", "ENEMY", 2)
	check(t, file, "Projectile", "SKILL", 1)
	if file.Protocol.Client["ADMIN_COMMAND"] != 47 {
		t.Fatalf("client admin opcode = %d", file.Protocol.Client["ADMIN_COMMAND"])
	}
	if file.Protocol.Server["GUILD_CREATE_OPEN"] != 77 {
		t.Fatalf("server guild open opcode = %d", file.Protocol.Server["GUILD_CREATE_OPEN"])
	}
}

func TestRenderEnumsDeterministic(t *testing.T) {
	file, err := LoadEnums("../../content/enums.toml")
	if err != nil {
		t.Fatal(err)
	}
	first, err := RenderEnums(file)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderEnums(file)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("render is not deterministic")
	}
	source := string(first)
	if !strings.Contains(source, "Sex = Module.new") || !strings.Contains(source, "Sex.const_set(:FEMALE, 1)") {
		t.Fatalf("rendered enums missing constants:\n%s", source)
	}
	if !strings.Contains(source, "Dir.const_set(:RIGHT, 6)") {
		t.Fatalf("rendered enums missing Dir gap:\n%s", source)
	}
	if strings.Contains(source, "Packet") {
		t.Fatal("legacy Packet enum should not be generated")
	}
}

func TestEnumsRejectBadValues(t *testing.T) {
	base := &EnumsFile{
		SchemaVersion: EnumsSchemaVersion,
		Enums:         []EnumGroup{{Name: "Sex", Values: []EnumValue{{Name: "MALE", Value: 0}}}},
		Protocol: ProtocolEnums{
			Client: map[string]int{"PING": 1},
			Server: map[string]int{"PONG": 1},
		},
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid file rejected: %v", err)
	}
	bad := *base
	bad.Enums = []EnumGroup{{Name: "Sex", Values: []EnumValue{{Name: "MALE", Value: 0}, {Name: "MALE", Value: 1}}}}
	if err := bad.Validate(); err == nil {
		t.Fatal("duplicate constant accepted")
	}
	bad = *base
	bad.Protocol.Client = map[string]int{"PING": 1, "MOVE": 1}
	if err := bad.Validate(); err == nil {
		t.Fatal("duplicate protocol value accepted")
	}
	bad = *base
	bad.SchemaVersion = 99
	if err := bad.Validate(); err == nil {
		t.Fatal("unsupported schema accepted")
	}
}

func TestProtocolTomlMatchesGo(t *testing.T) {
	file, err := LoadEnums("../../content/enums.toml")
	if err != nil {
		t.Fatal(err)
	}
	client := []struct {
		name  string
		value protocol.ClientOp
	}{
		{"PING", protocol.ClientPing},
		{"AUTH_BEGIN", protocol.ClientAuthBegin},
		{"AUTH_PROOF", protocol.ClientAuthProof},
		{"REGISTER_BEGIN", protocol.ClientRegisterBegin},
		{"REGISTER_PROOF", protocol.ClientRegisterProof},
		{"SELECT_CHARACTER", protocol.ClientSelectCharacter},
		{"CREATE_CHARACTER", protocol.ClientCreateCharacter},
		{"DELETE_CHARACTER", protocol.ClientDeleteCharacter},
		{"MOVE", protocol.ClientMove},
		{"CHAT", protocol.ClientChat},
		{"ATTACK", protocol.ClientAttack},
		{"USE_ITEM", protocol.ClientUseItem},
		{"USE_SKILL", protocol.ClientUseSkill},
		{"USE_HOTBAR", protocol.ClientUseHotbar},
		{"BALLOON", protocol.ClientBalloon},
		{"DROP_ITEM", protocol.ClientDropItem},
		{"PICKUP_DROP", protocol.ClientPickupDrop},
		{"SPEND_POINT", protocol.ClientSpendPoint},
		{"EQUIP", protocol.ClientEquip},
		{"SET_HOTBAR", protocol.ClientSetHotbar},
		{"SET_TARGET", protocol.ClientSetTarget},
		{"OPEN_FRIENDS", protocol.ClientOpenFriends},
		{"REMOVE_FRIEND", protocol.ClientRemoveFriend},
		{"GUILD_CREATE", protocol.ClientGuildCreate},
		{"GUILD_OPEN", protocol.ClientGuildOpen},
		{"GUILD_SET_LEADER", protocol.ClientGuildSetLeader},
		{"GUILD_SET_NOTICE", protocol.ClientGuildSetNotice},
		{"GUILD_KICK", protocol.ClientGuildKick},
		{"GUILD_INVITE", protocol.ClientGuildInvite},
		{"GUILD_LEAVE", protocol.ClientGuildLeave},
		{"PARTY_LEAVE", protocol.ClientPartyLeave},
		{"CHOICE", protocol.ClientChoice},
		{"BANK_ITEM", protocol.ClientBankItem},
		{"BANK_GOLD", protocol.ClientBankGold},
		{"CLOSE_WINDOW", protocol.ClientCloseWindow},
		{"SHOP_BUY", protocol.ClientShopBuy},
		{"SHOP_SELL", protocol.ClientShopSell},
		{"CHOICE_TELEPORT", protocol.ClientChoiceTeleport},
		{"EVENT_NEXT", protocol.ClientEventNext},
		{"EVENT_CHOICE", protocol.ClientEventChoice},
		{"REQUEST", protocol.ClientRequest},
		{"REQUEST_ACCEPT", protocol.ClientRequestAccept},
		{"REQUEST_DECLINE", protocol.ClientRequestDecline},
		{"TRADE_ITEM", protocol.ClientTradeItem},
		{"TRADE_GOLD", protocol.ClientTradeGold},
		{"LOGOUT", protocol.ClientLogout},
		{"ADMIN_COMMAND", protocol.ClientAdminCommand},
	}
	server := []struct {
		name  string
		value protocol.ServerOp
	}{
		{"PONG", protocol.ServerPong},
		{"AUTH_CHALLENGE", protocol.ServerAuthChallenge},
		{"AUTH_OK", protocol.ServerAuthOK},
		{"AUTH_ERROR", protocol.ServerAuthError},
		{"REGISTER_CHALLENGE", protocol.ServerRegisterChallenge},
		{"REGISTER_OK", protocol.ServerRegisterOK},
		{"REGISTER_ERROR", protocol.ServerRegisterError},
		{"CHARACTER_CREATED", protocol.ServerCharacterCreated},
		{"CHARACTER_DELETED", protocol.ServerCharacterDeleted},
		{"CHARACTER_ERROR", protocol.ServerCharacterError},
		{"ENTER_WORLD", protocol.ServerEnterWorld},
		{"PLAYER_DATA", protocol.ServerPlayerData},
		{"PLAYER_LEFT", protocol.ServerPlayerLeft},
		{"PLAYER_MOVE", protocol.ServerPlayerMove},
		{"MAP_CHAT", protocol.ServerMapChat},
		{"CHAT", protocol.ServerChat},
		{"ALERT", protocol.ServerAlert},
		{"ATTACK_PLAYER", protocol.ServerAttackPlayer},
		{"ATTACK_ENEMY", protocol.ServerAttackEnemy},
		{"ANIMATION", protocol.ServerAnimation},
		{"BALLOON", protocol.ServerBalloon},
		{"ENEMY_REVIVE", protocol.ServerEnemyRevive},
		{"EVENT_DATA", protocol.ServerEventData},
		{"EVENT_MOVE", protocol.ServerEventMove},
		{"DROP_ADD", protocol.ServerDropAdd},
		{"DROP_REMOVE", protocol.ServerDropRemove},
		{"PROJECTILE", protocol.ServerProjectile},
		{"VITALS", protocol.ServerVitals},
		{"EXP", protocol.ServerExp},
		{"STATE", protocol.ServerState},
		{"BUFF", protocol.ServerBuff},
		{"ITEM", protocol.ServerItem},
		{"GOLD", protocol.ServerGold},
		{"PARAM", protocol.ServerParam},
		{"EQUIP", protocol.ServerEquip},
		{"SKILL", protocol.ServerSkill},
		{"CLASS", protocol.ServerClass},
		{"SEX", protocol.ServerSex},
		{"GRAPHIC", protocol.ServerGraphic},
		{"POINTS", protocol.ServerPoints},
		{"HOTBAR", protocol.ServerHotbar},
		{"TARGET", protocol.ServerTarget},
		{"TRANSFER", protocol.ServerTransfer},
		{"FRIENDS", protocol.ServerFriends},
		{"FRIEND_ADD", protocol.ServerFriendAdd},
		{"FRIEND_REMOVE", protocol.ServerFriendRemove},
		{"GUILD_OPEN", protocol.ServerGuildOpen},
		{"GUILD_NAME", protocol.ServerGuildName},
		{"GUILD_LEADER", protocol.ServerGuildLeader},
		{"GUILD_NOTICE", protocol.ServerGuildNotice},
		{"GUILD_MEMBER_REMOVE", protocol.ServerGuildMemberRemove},
		{"PARTY_JOIN", protocol.ServerPartyJoin},
		{"PARTY_LEAVE", protocol.ServerPartyLeave},
		{"PARTY_DISSOLVE", protocol.ServerPartyDissolve},
		{"BANK_OPEN", protocol.ServerBankOpen},
		{"BANK_ITEM", protocol.ServerBankItem},
		{"BANK_GOLD", protocol.ServerBankGold},
		{"WINDOW_CLOSE", protocol.ServerWindowClose},
		{"SHOP_OPEN", protocol.ServerShopOpen},
		{"TELEPORT_OPEN", protocol.ServerTeleportOpen},
		{"EVENT_COMMAND", protocol.ServerEventCommand},
		{"REQUEST", protocol.ServerRequest},
		{"REQUEST_ACCEPT", protocol.ServerRequestAccept},
		{"TRADE_ITEM", protocol.ServerTradeItem},
		{"TRADE_GOLD", protocol.ServerTradeGold},
		{"QUEST_ADD", protocol.ServerQuestAdd},
		{"QUEST_FINISH", protocol.ServerQuestFinish},
		{"VIP_DAYS", protocol.ServerVIPDays},
		{"LOGOUT", protocol.ServerLogout},
		{"ADMIN_COMMAND", protocol.ServerAdminCommand},
		{"SWITCH", protocol.ServerSwitch},
		{"VARIABLE", protocol.ServerVariable},
		{"SELF_SWITCH", protocol.ServerSelfSwitch},
		{"GLOBAL_SWITCHES", protocol.ServerGlobalSwitches},
		{"ERROR", protocol.ServerError},
		{"DISCONNECT", protocol.ServerDisconnect},
		{"GUILD_CREATE_OPEN", protocol.ServerGuildCreateOpen},
		{"MAP_RESET", protocol.ServerMapReset},
		{"EVENT_REMOVE", protocol.ServerEventRemove},
		{"MAP_PLAYER_ENTER", protocol.ServerMapPlayerEnter},
		{"MAP_PLAYER_LEAVE", protocol.ServerMapPlayerLeave},
	}
	for _, tc := range client {
		if got := file.Protocol.Client[tc.name]; got != int(tc.value) {
			t.Fatalf("client %s = %d, Go = %d", tc.name, got, tc.value)
		}
	}
	for _, tc := range server {
		if got := file.Protocol.Server[tc.name]; got != int(tc.value) {
			t.Fatalf("server %s = %d, Go = %d", tc.name, got, tc.value)
		}
	}
	if len(client) != len(file.Protocol.Client) {
		t.Fatalf("client opcode count = %d, toml = %d", len(client), len(file.Protocol.Client))
	}
	if len(server) != len(file.Protocol.Server) {
		t.Fatalf("server opcode count = %d, toml = %d", len(server), len(file.Protocol.Server))
	}
}

var rubyConstRE = regexp.MustCompile(`^\s*([A-Z][A-Z0-9_]*)\s*=\s*(\d+)\s*$`)

func TestProtocolTomlMatchesClientRuby(t *testing.T) {
	file, err := LoadEnums("../../content/enums.toml")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../client-patches/protocol.rb")
	if err != nil {
		t.Fatal(err)
	}
	client := readRubyModule(t, string(data), "Client")
	server := readRubyModule(t, string(data), "Server")
	for name, want := range file.Protocol.Client {
		if got := client[name]; got != want {
			t.Fatalf("protocol.rb client %s = %d, toml = %d", name, got, want)
		}
	}
	for name, want := range file.Protocol.Server {
		if got := server[name]; got != want {
			t.Fatalf("protocol.rb server %s = %d, toml = %d", name, got, want)
		}
	}
	if len(client) != len(file.Protocol.Client) || len(server) != len(file.Protocol.Server) {
		t.Fatalf("protocol.rb counts client=%d/%d server=%d/%d", len(client), len(file.Protocol.Client), len(server), len(file.Protocol.Server))
	}
}

func readRubyModule(t *testing.T, source, module string) map[string]int {
	t.Helper()
	lines := strings.Split(source, "\n")
	inside := false
	out := make(map[string]int)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "module "+module {
			inside = true
			continue
		}
		if inside && trimmed == "end" {
			break
		}
		if !inside {
			continue
		}
		match := rubyConstRE.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		value, err := strconv.Atoi(match[2])
		if err != nil {
			t.Fatalf("bad protocol constant %q", line)
		}
		out[match[1]] = value
	}
	return out
}

func check(t *testing.T, file *EnumsFile, group, name string, want int) {
	t.Helper()
	for _, g := range file.Enums {
		if g.Name != group {
			continue
		}
		for _, v := range g.Values {
			if v.Name == name {
				if v.Value != want {
					t.Fatalf("%s::%s = %d, want %d", group, name, v.Value, want)
				}
				return
			}
		}
		t.Fatalf("%s has no %s", group, name)
	}
	t.Fatalf("missing enum group %s", group)
}
