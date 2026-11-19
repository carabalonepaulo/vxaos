// Command vxaos-admin performs development/operations tasks: migrate the
// database, create accounts and create initial characters.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelaia/vxaos/server/internal/auth"
	"github.com/kelaia/vxaos/server/internal/chargen"
	"github.com/kelaia/vxaos/server/internal/config"
	"github.com/kelaia/vxaos/server/internal/content"
	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	switch cmd {
	case "migrate":
		runMigrate(os.Args[2:])
	case "account":
		if len(os.Args) < 3 || os.Args[2] != "create" {
			usage()
			os.Exit(2)
		}
		runAccountCreate(os.Args[3:])
	case "character":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		switch os.Args[2] {
		case "create":
			runCharacterCreate(os.Args[3:])
		case "list":
			runCharacterList(os.Args[3:])
		case "delete":
			runCharacterDelete(os.Args[3:])
		default:
			usage()
			os.Exit(2)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func baseFlags(fs *flag.FlagSet) (*string, *string) {
	configPath := fs.String("config", "config.toml", "server config file")
	dataDir := fs.String("data", "", "override data_dir")
	return configPath, dataDir
}

func loadConfig(path, dataOverride string) config.Config {
	cfg, err := config.Load(path)
	if err != nil {
		fail(err)
	}
	if dataOverride != "" {
		cfg.DataDir = dataOverride
	}
	return cfg
}

func openStore(cfg config.Config) *store.Store {
	if cfg.Store.Driver == "sqlite" {
		dsn := cfg.Store.DSN
		if dsn != ":memory:" && dsn != "" && !strings.HasPrefix(dsn, "file:") {
			if dir := filepath.Dir(dsn); dir != "." && dir != "" {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					fail(err)
				}
			}
		}
	}
	st, err := store.Open(context.Background(), store.Config{
		Driver:       store.Driver(cfg.Store.Driver),
		DSN:          cfg.Store.DSN,
		MaxOpenConns: cfg.Store.MaxOpenConns,
		MaxIdleConns: cfg.Store.MaxIdleConns,
	})
	if err != nil {
		fail(err)
	}
	if err := st.Migrate(context.Background()); err != nil {
		st.Close()
		fail(err)
	}
	return st
}

func runMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	cfgPath, dataDir := baseFlags(fs)
	fs.Parse(args)
	cfg := loadConfig(*cfgPath, *dataDir)
	st := openStore(cfg)
	defer st.Close()
	fmt.Println("migrations ok")
}

func runAccountCreate(args []string) {
	fs := flag.NewFlagSet("account create", flag.ExitOnError)
	cfgPath, dataDir := baseFlags(fs)
	user := fs.String("user", "", "username")
	pass := fs.String("pass", "", "plain password (dev tool)")
	email := fs.String("email", "", "email")
	group := fs.Int("group", 0, "0 standard, 1 monitor, 2 admin")
	vipDays := fs.Int("vip-days", 0, "VIP days")
	fs.Parse(args)
	if *user == "" || *pass == "" {
		fail(fmt.Errorf("-user and -pass are required"))
	}
	cfg := loadConfig(*cfgPath, *dataDir)
	st := openStore(cfg)
	defer st.Close()
	ctx := context.Background()
	exists, err := st.AccountExists(ctx, *user)
	if err != nil {
		fail(err)
	}
	if exists {
		fail(fmt.Errorf("account %q already exists", *user))
	}
	salt, verifier, err := auth.NewVerifier(*pass, cfg.Auth.Iterations)
	if err != nil {
		fail(err)
	}
	vipUntil := time.Now().Unix()
	if *vipDays > 0 {
		vipUntil += int64(*vipDays) * 86400
	}
	acc := &store.Account{
		Username:           *user,
		Email:              *email,
		PasswordAlgo:       "pbkdf2-sha256",
		PasswordSalt:       salt,
		PasswordIterations: cfg.Auth.Iterations,
		PasswordVerifier:   verifier,
		GroupID:            *group,
		VIPUntil:           vipUntil,
	}
	if err := st.CreateAccount(ctx, acc); err != nil {
		fail(err)
	}
	fmt.Printf("account %s created (id=%d)\n", acc.Username, acc.ID)
}

func runCharacterCreate(args []string) {
	fs := flag.NewFlagSet("character create", flag.ExitOnError)
	cfgPath, dataDir := baseFlags(fs)
	user := fs.String("user", "", "account username")
	slot := fs.Int("slot", 0, "character slot")
	name := fs.String("name", "", "character name")
	classID := fs.Int("class", 1, "class id")
	sex := fs.Int("sex", 0, "0 male, 1 female")
	fs.Parse(args)
	if *user == "" || *name == "" {
		fail(fmt.Errorf("-user and -name are required"))
	}
	cfg := loadConfig(*cfgPath, *dataDir)
	data, err := rpg.Load(cfg.DataDir)
	if err != nil {
		fail(err)
	}
	shared, err := content.LoadConfigs(filepath.Join(cfg.ContentDir, "configs.toml"))
	if err != nil {
		fail(err)
	}
	st := openStore(cfg)
	defer st.Close()
	ctx := context.Background()
	acc, err := st.GetAccountByUsername(ctx, *user)
	if err != nil {
		fail(err)
	}
	c, state, err := chargen.Create(chargen.Options{
		Data:               data,
		MaxEquips:          shared.Limits.MaxEquips,
		MaxHotbar:          shared.Limits.MaxHotbar,
		MaxPlayerSwitches:  shared.Limits.MaxPlayerSwitches,
		MaxPlayerVariables: shared.Limits.MaxPlayerVariables,
		StartPoints:        shared.StartPoints,
		MaxActors:          shared.Limits.MaxActors,
		MinCharacters:      shared.Limits.MinCharacters,
		MaxCharacters:      shared.Limits.MaxCharacters,
	}, acc.ID, *slot, *name, *classID, *sex, 0, [8]int{})
	if err != nil {
		fail(err)
	}
	if err := st.CreateCharacter(ctx, c); err != nil {
		fail(err)
	}
	if err := st.SaveCharacter(ctx, c, state); err != nil {
		fail(err)
	}
	fmt.Printf("character %s created (id=%d, slot=%d, class=%d)\n", c.Name, c.ID, c.Slot, c.ClassID)
}

func runCharacterDelete(args []string) {
	fs := flag.NewFlagSet("character delete", flag.ExitOnError)
	cfgPath, dataDir := baseFlags(fs)
	user := fs.String("user", "", "account username")
	slot := fs.Int("slot", -1, "character slot (-1 uses -name)")
	name := fs.String("name", "", "character name")
	fs.Parse(args)
	if *user == "" || (*slot < 0 && *name == "") {
		fail(fmt.Errorf("-user and (-slot or -name) are required"))
	}
	cfg := loadConfig(*cfgPath, *dataDir)
	st := openStore(cfg)
	defer st.Close()
	ctx := context.Background()
	acc, err := st.GetAccountByUsername(ctx, *user)
	if err != nil {
		fail(err)
	}
	chars, err := st.ListCharacters(ctx, acc.ID)
	if err != nil {
		fail(err)
	}
	for _, c := range chars {
		if (*slot >= 0 && c.Slot == *slot) || (*name != "" && strings.EqualFold(c.Name, *name)) {
			if err := st.DeleteCharacter(ctx, c.ID); err != nil {
				fail(err)
			}
			fmt.Printf("character %s deleted (id=%d, slot=%d)\n", c.Name, c.ID, c.Slot)
			return
		}
	}
	fail(fmt.Errorf("character not found"))
}

func runCharacterList(args []string) {
	fs := flag.NewFlagSet("character list", flag.ExitOnError)
	cfgPath, dataDir := baseFlags(fs)
	user := fs.String("user", "", "account username")
	fs.Parse(args)
	if *user == "" {
		fail(fmt.Errorf("-user is required"))
	}
	cfg := loadConfig(*cfgPath, *dataDir)
	st := openStore(cfg)
	defer st.Close()
	acc, err := st.GetAccountByUsername(context.Background(), *user)
	if err != nil {
		fail(err)
	}
	chars, err := st.ListCharacters(context.Background(), acc.ID)
	if err != nil {
		fail(err)
	}
	for _, c := range chars {
		fmt.Printf("slot=%d id=%d name=%s class=%d level=%d map=%d (%d,%d)\n", c.Slot, c.ID, c.Name, c.ClassID, c.Level, c.MapID, c.X, c.Y)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "vxaos-admin: %v\n", err)
	os.Exit(1)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: vxaos-admin <command> [flags]

  migrate
  account create -user NAME -pass PASS [-email EMAIL] [-group 0|1|2] [-vip-days N]
  character create -user NAME -name NAME [-slot N] [-class ID] [-sex 0|1]
  character delete -user NAME (-slot N | -name NAME)
  character list -user NAME

global flags:
  -config  server config file (default config.toml)
  -data    override data_dir`)
}
