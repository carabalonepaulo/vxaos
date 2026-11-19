// Command vxaos-server runs the authoritative game server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kelaia/vxaos/server/internal/chargen"
	"github.com/kelaia/vxaos/server/internal/config"
	"github.com/kelaia/vxaos/server/internal/content"
	"github.com/kelaia/vxaos/server/internal/game"
	netserver "github.com/kelaia/vxaos/server/internal/net"
	"github.com/kelaia/vxaos/server/internal/protocol"
	"github.com/kelaia/vxaos/server/internal/rpg"
	"github.com/kelaia/vxaos/server/internal/store"
)

func main() {
	configPath := flag.String("config", "config.toml", "server config file")
	listen := flag.String("listen", "", "override listen address")
	dataDir := flag.String("data", "", "override data_dir")
	dsn := flag.String("dsn", "", "override store.dsn")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fatal(err)
	}
	if *listen != "" {
		cfg.Listen = *listen
	}
	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *dsn != "" {
		cfg.Store.DSN = *dsn
	}
	if err := cfg.Validate(); err != nil {
		fatal(err)
	}
	log := newLogger(cfg.LogLevel)
	slog.SetDefault(log)

	if cfg.Store.Driver == "sqlite" && cfg.Store.DSN != ":memory:" && !strings.HasPrefix(cfg.Store.DSN, "file:") {
		if dir := filepath.Dir(cfg.Store.DSN); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				fatal(err)
			}
		}
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, store.Config{
		Driver:       store.Driver(cfg.Store.Driver),
		DSN:          cfg.Store.DSN,
		MaxOpenConns: cfg.Store.MaxOpenConns,
		MaxIdleConns: cfg.Store.MaxIdleConns,
	})
	if err != nil {
		fatal(err)
	}
	defer st.Close()
	if err := st.Migrate(ctx); err != nil {
		fatal(err)
	}
	log.Info("database ready", "driver", cfg.Store.Driver)

	data, err := rpg.Load(cfg.DataDir)
	if err != nil {
		fatal(err)
	}
	mapCount := 0
	for _, m := range data.Maps {
		if m != nil {
			mapCount++
		}
	}
	log.Info("game data loaded", "maps", mapCount, "actors", len(data.Actors), "items", len(data.Items))

	quests, err := content.LoadQuests(filepath.Join(cfg.ContentDir, "quests.toml"))
	if err != nil {
		fatal(err)
	}
	log.Info("quest content loaded", "quests", len(quests))
	shared, err := content.LoadConfigs(filepath.Join(cfg.ContentDir, "configs.toml"))
	if err != nil {
		fatal(err)
	}
	log.Info("shared config loaded", "port", shared.Network.Port, "game_version", shared.Network.GameVersion)

	world, err := game.New(game.Options{
		Data:               data,
		Store:              st,
		Tick:               time.Duration(cfg.Server.TickMS) * time.Millisecond,
		SaveInterval:       time.Duration(cfg.Server.SaveIntervalSeconds) * time.Second,
		MapIdle:            time.Duration(cfg.Server.MapIdleSeconds) * time.Second,
		AOIRadius:          cfg.Server.AOIRadiusTiles,
		AOIHysteresis:      cfg.Server.AOIHysteresisTiles,
		AOICellSize:        cfg.Server.AOICellSize,
		MaxEquips:          shared.Limits.MaxEquips,
		MaxHotbar:          shared.Limits.MaxHotbar,
		MaxItems:           shared.Limits.MaxPlayerItems,
		MaxBankItems:       shared.Limits.MaxBankItems,
		MaxStack:           cfg.Game.MaxStack,
		MaxGold:            shared.Limits.MaxGold,
		MaxLevel:           shared.Limits.MaxLevel,
		MaxPlayerSwitches:  shared.Limits.MaxPlayerSwitches,
		MaxPlayerVariables: shared.Limits.MaxPlayerVariables,
		MaxGuildMembers:    shared.Limits.MaxGuildMembers,
		MaxPartyMembers:    shared.Limits.MaxPartyMembers,
		MaxFriends:         shared.Limits.MaxFriends,
		MaxMapDrops:        shared.Limits.MaxMapDrops,
		DropDespawnSeconds: shared.DropDespawnSeconds,
		DropPickupSeconds:  shared.DropPickupSeconds,
		ProjectileSkills:   shared.ProjectileSkillIDs(),
		Teleports:          teleports(shared.Teleports),
		Quests:             quests,
		Logger:             log,
	})
	if err != nil {
		fatal(err)
	}
	srv, err := netserver.New(netserver.Options{
		Listen:         cfg.Listen,
		World:          world,
		Store:          st,
		GameVersion:    shared.Network.GameVersion,
		MaxActors:      shared.Limits.MaxActors,
		MaxFrame:       protocol.MaxFrameSize,
		AuthIterations: cfg.Auth.Iterations,
		AuthTimeout:    time.Duration(cfg.Server.AuthTimeoutSeconds) * time.Second,
		Inactivity:     time.Duration(cfg.Server.InactivitySeconds) * time.Second,
		MaxConnections: cfg.Server.MaxConnections,
		Chargen: chargen.Options{
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
		},
		Logger: log,
	})
	if err != nil {
		fatal(err)
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.ListenAndServe(ctx) }()
	log.Info("server started", "listen", cfg.Listen, "version", shared.Network.GameVersion)

	serverDone := false
	select {
	case <-ctx.Done():
		log.Info("shutdown requested")
	case err := <-serveErr:
		serverDone = true
		if err != nil {
			log.Error("server stopped", "err", err)
			stop()
		}
	}
	if !serverDone {
		select {
		case <-serveErr:
		case <-time.After(5 * time.Second):
			log.Warn("server did not stop in time")
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := world.Stop(shutdownCtx); err != nil {
		log.Error("world shutdown", "err", err)
	}
	log.Info("server stopped")
}

func teleports(groups [][]content.TeleportStop) []game.Teleport {
	out := make([]game.Teleport, 0)
	for group, stops := range groups {
		for _, stop := range stops {
			out = append(out, game.Teleport{
				Group: group, MapID: stop.MapID, X: stop.X, Y: stop.Y, Gold: stop.Gold,
			})
		}
	}
	return out
}

func newLogger(level string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lv}))
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "vxaos-server: %v\n", err)
	os.Exit(1)
}
