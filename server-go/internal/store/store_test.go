package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func openTest(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dsn := filepath.Join(dir, "test.db")
	s, err := Open(context.Background(), Config{Driver: SQLite, DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}

func testAccount(user string) *Account {
	return &Account{
		Username:           user,
		Email:              user + "@example.test",
		PasswordAlgo:       "pbkdf2-sha256",
		PasswordSalt:       []byte("0123456789abcdef"),
		PasswordIterations: 1000,
		PasswordVerifier:   []byte(strings.Repeat("v", 32)),
	}
}

func TestMigrateIdempotent(t *testing.T) {
	s := openTest(t)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"accounts", "characters", "character_state", "bank_state", "guilds", "bans", "server_state"} {
		var name string
		err := s.DB().QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s: %v", table, err)
		}
	}
}

func TestAccountCRUD(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	a := testAccount("Admin")
	if err := s.CreateAccount(ctx, a); err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 {
		t.Fatal("account id not set")
	}
	got, err := s.GetAccountByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != a.ID || got.Username != "Admin" || got.UsernameKey != "admin" {
		t.Fatalf("got %+v", got)
	}
	if !strings.EqualFold(string(got.PasswordVerifier), string(a.PasswordVerifier)) {
		t.Fatalf("verifier mismatch")
	}
	ok, err := s.AccountExists(ctx, "ADMIN")
	if err != nil || !ok {
		t.Fatalf("exists = %v, %v", ok, err)
	}
	if err := s.UpdateAccountLogin(ctx, a.ID, 123); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateAccountVIP(ctx, a.ID, 456); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateAccountPassword(ctx, a.ID, "pbkdf2-sha256", []byte("newsalt"), 2000, []byte("newverifier")); err != nil {
		t.Fatal(err)
	}
	got, err = s.GetAccountByID(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.LastLoginAt != 123 || got.VIPUntil != 456 || got.PasswordIterations != 2000 || string(got.PasswordSalt) != "newsalt" {
		t.Fatalf("updated account = %+v", got)
	}
	if _, err := s.GetAccountByUsername(ctx, "missing"); err != sql.ErrNoRows {
		t.Fatalf("missing account err = %v", err)
	}
}

func TestAccountUniqueness(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	a := testAccount("Admin")
	if err := s.CreateAccount(ctx, a); err != nil {
		t.Fatal(err)
	}
	b := testAccount("admin")
	if err := s.CreateAccount(ctx, b); err == nil {
		t.Fatal("duplicate username_key accepted")
	}
}

func testCharacter(accountID int64, slot int, name string) *Character {
	return &Character{
		AccountID:      accountID,
		Slot:           slot,
		Name:           name,
		ClassID:        1,
		Sex:            0,
		Level:          1,
		Exp:            0,
		HP:             100,
		MP:             20,
		Points:         5,
		MHP:            100,
		MMP:            20,
		ATK:            10,
		DEF:            8,
		MAT:            4,
		MDF:            5,
		AGI:            6,
		LUK:            7,
		CharacterName:  "Charset01",
		CharacterIndex: 0,
		FaceName:       "Charset01",
		FaceIndex:      0,
		MapID:          1,
		X:              10,
		Y:              20,
		Direction:      2,
		ReviveMapID:    1,
		ReviveX:        10,
		ReviveY:        20,
		Gold:           123,
	}
}

func TestCharacterCRUD(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	a := testAccount("admin")
	if err := s.CreateAccount(ctx, a); err != nil {
		t.Fatal(err)
	}
	c := testCharacter(a.ID, 0, "Hero")
	if err := s.CreateCharacter(ctx, c); err != nil {
		t.Fatal(err)
	}
	if c.ID == 0 {
		t.Fatal("character id not set")
	}
	list, err := s.ListCharacters(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "Hero" {
		t.Fatalf("list = %+v", list)
	}
	got, st, err := s.GetCharacter(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Hero" || got.Gold != 123 || st.Version != StateVersion {
		t.Fatalf("got %+v state %+v", got, st)
	}
	st.Equips = []int{1, 46, 0, 1, 0}
	st.Inventory = []ItemStack{{Kind: 1, ItemID: 5, Amount: 3}}
	st.Skills = []int{80}
	st.Quests = []QuestState{{QuestID: 0, State: 1, Kills: 0}}
	st.Hotbar = []HotbarSlot{{Type: 1, ItemID: 5}}
	st.Switches = []bool{true, false}
	st.Variables = []int32{7, 8}
	st.SelfSwitches = []SelfSwitchState{{MapID: 1, EventID: 17, Ch: "A", Value: true}}
	got.Gold = 500
	got.Level = 2
	if err := s.SaveCharacter(ctx, got, st); err != nil {
		t.Fatal(err)
	}
	// Saving unchanged values must not fail on dialects that report changed
	// rows instead of matched rows.
	if err := s.SaveCharacter(ctx, got, st); err != nil {
		t.Fatal(err)
	}
	got2, st2, err := s.GetCharacter(ctx, c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Gold != 500 || got2.Level != 2 {
		t.Fatalf("core = %+v", got2)
	}
	if len(st2.Equips) != 5 || st2.Equips[1] != 46 || st2.Variables[1] != 8 || !st2.Switches[0] {
		t.Fatalf("state = %+v", st2)
	}
	if byName, err := s.GetCharacterByName(ctx, "hero"); err != nil || byName.ID != c.ID {
		t.Fatalf("by name = %+v, %v", byName, err)
	}
	if ok, err := s.CharacterExists(ctx, "HERO"); err != nil || !ok {
		t.Fatalf("exists = %v, %v", ok, err)
	}
	if err := s.DeleteCharacter(ctx, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.GetCharacter(ctx, c.ID); err != sql.ErrNoRows {
		t.Fatalf("deleted character err = %v", err)
	}
	var count int
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM character_state WHERE character_id=?`, c.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("state rows after delete = %d", count)
	}
}

func TestCharacterUniqueness(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	a := testAccount("admin")
	if err := s.CreateAccount(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateCharacter(ctx, testCharacter(a.ID, 0, "Hero")); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateCharacter(ctx, testCharacter(a.ID, 0, "Other")); err == nil {
		t.Fatal("duplicate slot accepted")
	}
	if err := s.CreateCharacter(ctx, testCharacter(a.ID, 1, "hero")); err == nil {
		t.Fatal("duplicate name accepted")
	}
}

func TestBank(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	a := testAccount("admin")
	if err := s.CreateAccount(ctx, a); err != nil {
		t.Fatal(err)
	}
	b, err := s.GetBank(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Version != StateVersion || b.Gold != 0 {
		t.Fatalf("default bank = %+v", b)
	}
	b.Gold = 999
	b.Items = []ItemStack{{Kind: 1, ItemID: 5, Amount: 2}}
	if err := s.SaveBank(ctx, a.ID, b); err != nil {
		t.Fatal(err)
	}
	b2, err := s.GetBank(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b2.Gold != 999 || len(b2.Items) != 1 || b2.Items[0].Amount != 2 {
		t.Fatalf("bank = %+v", b2)
	}
}

func TestGuildStore(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	a := testAccount("guildmaster")
	if err := s.CreateAccount(ctx, a); err != nil {
		t.Fatal(err)
	}
	g := &Guild{Name: "Guardians", LeaderCharacterID: 1, Flag: []byte{1, 2, 3}}
	if err := s.CreateGuild(ctx, g); err != nil {
		t.Fatal(err)
	}
	if g.ID == 0 {
		t.Fatal("guild id was not assigned")
	}
	got, err := s.GetGuildByName(ctx, "guardians")
	if err != nil || got.Name != "Guardians" {
		t.Fatalf("guild = %+v, %v", got, err)
	}
	c := testCharacter(a.ID, 0, "Leader")
	c.GuildID = g.ID
	if err := s.CreateCharacter(ctx, c); err != nil {
		t.Fatal(err)
	}
	members, err := s.GuildMembers(ctx, g.ID)
	if err != nil || len(members) != 1 || members[0].Name != "Leader" {
		t.Fatalf("members = %+v, %v", members, err)
	}
	got.LeaderCharacterID = c.ID
	got.Notice = "hello"
	if err := s.SaveGuild(ctx, got); err != nil {
		t.Fatal(err)
	}
	again, err := s.GetGuildByName(ctx, "Guardians")
	if err != nil || again.LeaderCharacterID != c.ID || again.Notice != "hello" {
		t.Fatalf("saved guild = %+v, %v", again, err)
	}
	if err := s.DeleteGuild(ctx, g.ID); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListGuilds(ctx)
	if err != nil || len(list) != 0 {
		t.Fatalf("guilds after delete = %+v, %v", list, err)
	}
	members, err = s.GuildMembers(ctx, g.ID)
	if err != nil || len(members) != 0 {
		t.Fatalf("members after delete = %+v, %v", members, err)
	}
}

func TestServerState(t *testing.T) {
	s := openTest(t)
	ctx := context.Background()
	value, err := s.GetServerState(ctx, "global_switches")
	if err != nil || value != "" {
		t.Fatalf("missing state = %q, %v", value, err)
	}
	if err := s.SetServerState(ctx, "global_switches", `[true,false]`); err != nil {
		t.Fatal(err)
	}
	value, err = s.GetServerState(ctx, "global_switches")
	if err != nil || value != `[true,false]` {
		t.Fatalf("state = %q, %v", value, err)
	}
	if err := s.SetServerState(ctx, "global_switches", `[false,true]`); err != nil {
		t.Fatal(err)
	}
	value, err = s.GetServerState(ctx, "global_switches")
	if err != nil || value != `[false,true]` {
		t.Fatalf("updated state = %q, %v", value, err)
	}
}

func TestStateVersion(t *testing.T) {
	if _, err := DecodeState(`{"version":999}`); err == nil {
		t.Fatal("future version accepted")
	}
	st, err := DecodeState("")
	if err != nil || st.Version != StateVersion {
		t.Fatalf("empty state = %+v, %v", st, err)
	}
}

func TestPostgresRebind(t *testing.T) {
	s := &Store{driver: Postgres}
	got := s.q(`SELECT * FROM x WHERE a = ? AND b IN (?, ?) AND c = '?'`)
	want := `SELECT * FROM x WHERE a = $1 AND b IN ($2, $3) AND c = '?'`
	if got != want {
		t.Fatalf("rebind = %q", got)
	}
}

func TestSplitSQL(t *testing.T) {
	src := `
-- a comment; with a semicolon
CREATE TABLE a (x TEXT DEFAULT 'a;b');
/* block ; comment */
CREATE TABLE b (y INT);
`
	stmts, err := splitSQL(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(stmts) != 2 {
		t.Fatalf("statements = %#v", stmts)
	}
	if !strings.Contains(stmts[0], "'a;b'") {
		t.Fatalf("quote handling = %q", stmts[0])
	}
}
