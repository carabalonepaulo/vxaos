// Package config loads the server bootstrap configuration.
//
// Shared gameplay/UI values live in content/configs.toml and content/enums.toml
// and are loaded by the content package. This file only contains server-owned
// settings (network listener, database, auth, tick and inventory stacking).
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/kelaia/vxaos/server/internal/auth"
)

// Config is the server configuration.
type Config struct {
	Listen     string `toml:"listen"`
	DataDir    string `toml:"data_dir"`
	ContentDir string `toml:"content_dir"`
	LogLevel   string `toml:"log_level"`
	Server     Server `toml:"server"`
	Game       Game   `toml:"game"`
	Auth       Auth   `toml:"auth"`
	Store      Store  `toml:"store"`
}

// Server contains runtime/session limits.
type Server struct {
	MaxConnections      int `toml:"max_connections"`
	TickMS              int `toml:"tick_ms"`
	AuthTimeoutSeconds  int `toml:"auth_timeout_seconds"`
	InactivitySeconds   int `toml:"inactivity_seconds"`
	SaveIntervalSeconds int `toml:"save_interval_seconds"`
	MapIdleSeconds      int `toml:"map_idle_seconds"`
	AOIRadiusTiles      int `toml:"aoi_radius_tiles"`
	AOIHysteresisTiles  int `toml:"aoi_hysteresis_tiles"`
	AOICellSize         int `toml:"aoi_cell_size"`
}

// Game contains server-only gameplay knobs. Shared limits are in
// content/configs.toml.
type Game struct {
	MaxStack int `toml:"max_stack"`
}

// Auth contains the password KDF parameters.
type Auth struct {
	Iterations int `toml:"iterations"`
}

// Store selects the SQL driver and DSN.
type Store struct {
	Driver       string `toml:"driver"`
	DSN          string `toml:"dsn"`
	MaxOpenConns int    `toml:"max_open_conns"`
	MaxIdleConns int    `toml:"max_idle_conns"`
}

// Default returns a development configuration.
func Default() Config {
	return Config{
		Listen:     "0.0.0.0:5000",
		DataDir:    "../Client/Data",
		ContentDir: "content",
		LogLevel:   "info",
		Server: Server{
			MaxConnections:      500,
			TickMS:              80,
			AuthTimeoutSeconds:  30,
			InactivitySeconds:   300,
			SaveIntervalSeconds: 900,
			MapIdleSeconds:      60,
			AOIRadiusTiles:      9,
			AOIHysteresisTiles:  2,
			AOICellSize:         8,
		},
		Game: Game{MaxStack: 999},
		Auth: Auth{Iterations: auth.DefaultIterations},
		Store: Store{
			Driver: "sqlite",
			DSN:    "data/vxaos.db",
		},
	}
}

// Load reads path. An empty path returns the defaults. Missing files are
// errors; a server should ship an explicit configuration.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, cfg.Validate()
	}
	if _, err := os.Stat(path); err != nil {
		return cfg, fmt.Errorf("config: %s: %w", path, err)
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("config: %s: %w", path, err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Validate checks values that would otherwise fail at runtime.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Listen) == "" {
		return fmt.Errorf("config: listen is empty")
	}
	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("config: data_dir is empty")
	}
	if strings.TrimSpace(c.ContentDir) == "" {
		return fmt.Errorf("config: content_dir is empty")
	}
	if c.Server.MaxConnections <= 0 {
		return fmt.Errorf("config: max_connections must be positive")
	}
	if c.Server.TickMS < 10 {
		return fmt.Errorf("config: tick_ms must be at least 10")
	}
	if c.Server.AuthTimeoutSeconds <= 0 || c.Server.InactivitySeconds <= 0 {
		return fmt.Errorf("config: session timeouts must be positive")
	}
	if c.Server.MapIdleSeconds <= 0 {
		return fmt.Errorf("config: map_idle_seconds must be positive")
	}
	if c.Server.AOIRadiusTiles < 0 {
		return fmt.Errorf("config: aoi_radius_tiles must be non-negative")
	}
	if c.Server.AOIHysteresisTiles < 0 {
		return fmt.Errorf("config: aoi_hysteresis_tiles must be non-negative")
	}
	if c.Server.AOICellSize <= 0 {
		return fmt.Errorf("config: aoi_cell_size must be positive")
	}
	if c.Game.MaxStack <= 0 {
		return fmt.Errorf("config: game.max_stack must be positive")
	}
	if c.Auth.Iterations < 1000 {
		return fmt.Errorf("config: auth.iterations must be at least 1000")
	}
	switch c.Store.Driver {
	case "sqlite", "mysql", "postgres":
	default:
		return fmt.Errorf("config: unsupported store.driver %q", c.Store.Driver)
	}
	if strings.TrimSpace(c.Store.DSN) == "" {
		return fmt.Errorf("config: store.dsn is empty")
	}
	return nil
}
