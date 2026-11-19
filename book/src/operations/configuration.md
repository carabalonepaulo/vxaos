# Configuration

There are two configuration layers:

1. **Server bootstrap** — `config.toml`, owned by the Go server. It contains
   the listener, database, auth KDF and runtime limits. Never put secrets in
   shared content.
2. **Shared content** — `content/configs.toml` and `content/enums.toml`. These
   are public: they are rendered into the client's `[VS] Configs` and
   `[VS] Enums` scripts and are also validated against the Go protocol.

`content/quests.toml` and `content/locales/*.csv` are content sources, not
server bootstrap.

## `config.toml`

### Root

| Key | Default | Meaning |
|---|---|---|
| `listen` | `0.0.0.0:5000` | TCP address for the game client |
| `data_dir` | `../client/Data` | RPG Maker `.rvdata2` directory |
| `content_dir` | `content` | TOML/CSV content directory |
| `log_level` | `info` | `debug`, `info`, `warn` or `error` |

### `[server]`

| Key | Default | Meaning |
|---|---:|---|
| `max_connections` | `500` | Maximum simultaneous TCP sessions |
| `tick_ms` | `80` | Map actor tick in milliseconds; minimum 10 |
| `auth_timeout_seconds` | `30` | Time allowed to complete authentication |
| `inactivity_seconds` | `300` | Read timeout for an authenticated idle session |
| `save_interval_seconds` | `900` | Periodic character snapshot interval |
| `map_idle_seconds` | `60` | Empty-map grace before the actor is destroyed |
| `aoi_radius_tiles` | `9` | AOI square half-size; `0` disables AOI |
| `aoi_hysteresis_tiles` | `2` | Extra tiles kept before removing a known entity |
| `aoi_cell_size` | `8` | Spatial grid cell size in tiles |

`tick_ms` is simulation, not network tick. Movement rate limiting is separate.

### `[game]`

| Key | Default | Meaning |
|---|---:|---|
| `max_stack` | `999` | Maximum stack size for items in inventory/bank |

### `[auth]`

| Key | Default | Meaning |
|---|---:|---|
| `iterations` | `210000` | PBKDF2-HMAC-SHA256 iterations for new accounts |

The client receives the iteration count in the auth challenge, so it does not
need to be configured on the client. Values below 1000 are rejected. Changing
this value does not rehash existing accounts; new accounts use the new count.

### `[store]`

| Key | Default | Meaning |
|---|---|---|
| `driver` | `sqlite` | `sqlite`, `mysql` or `postgres` |
| `dsn` | `data/vxaos.db` | Driver-specific data source name |
| `max_open_conns` | driver default | Connection pool limit |
| `max_idle_conns` | driver default | Idle pool limit |

SQLite is the tested dialect. The server creates the parent directory for a
file DSN automatically.

### `[[teleport]]`

Event-driven teleport destinations, used by `open-teleport` scripts and the
client teleport window:

```toml
[[teleport]]
group = 0
map_id = 1
x = 21
y = 12
gold = 10
```

The event script opens a group with `(open-teleport 1)` (one-based). Each entry
is a destination; `gold` is charged when selected. `group` in the server config
is zero-based, matching the client.

## `content/configs.toml`

This file is the single source of truth for the generated `[VS] Configs`
module and for shared server limits. Root keys must appear before the first
table header.

Important groups:

- root: `start_points`, `drop_despawn_seconds`, `drop_pickup_seconds`,
  `teleports`, `forbidden_names`;
- `[network]`: `host`, `port`, `game_version`, `shop_website`;
- `[limits]`: characters, actors, classes, items, gold, level, switches,
  variables, friends, guild, party, bank, trade, equips, map drops;
- `[fonts.*]`, `[colors.*]`, `[icons.*]`, `[paperdoll]`, `[timing]`,
  `[keys]`, `[[resolution]]` — client presentation and input;
- `[range_weapons]` / `[range_skills]` — projectile definitions.

After editing:

```powershell
go run ./cmd/vxaos-content check
go run ./cmd/vxaos-content build
```

`check` renders and validates without writing; `build` updates
`Scripts.rvdata2` and keeps a `.bak`.

### Values that must agree

`[network] game_version` must match the server's game version and the client's
generated `GAME_VERSION`. `[network] port` should match `listen` when the
client uses the generated value. A mismatch is the most common cause of the
client closing at the handshake.

## `content/enums.toml`

`content/enums.toml` is the source of truth for the client's `[VS] Enums`
module. It also contains the protocol opcode table used by the parity tests:

- `[protocol.client]` and `[protocol.server]` must match
  `server/internal/protocol/opcodes.go`;
- the Ruby `client-patches/protocol.rb` must match the same table;
- `internal/content/enums_test.go` validates all three.

Do not renumber existing opcodes in a released major line. Add new opcodes at
the end and update the TOML, Ruby and Go tables together.

## `content/quests.toml`

Quests are ordered; the first entry is quest id `0`. Event scripts address them
one-based: `(start-quest 1)` starts the first entry. See
[Quests](../content/quests.md) for the schema.

## `content/locales/*.csv`

The locale CSV (`ptBR.csv` by default) provides the `[VS] Vocab` module.
`vxaos-content check/build -locale <name>` selects it. Keep keys stable; the
generated script is consumed by the client UI.

## Environment overrides

The server command accepts:

```powershell
vxaos-server.exe -config config.toml -listen 0.0.0.0:6000 `
  -data ..\client\Data -dsn data\other.db
```

| Flag | Overrides |
|---|---|
| `-config` | config file path |
| `-listen` | `listen` |
| `-data` | `data_dir` |
| `-dsn` | `store.dsn` |

Use overrides for temporary tests; keep the real values in `config.toml`.

## Reloading

The server does not hot-reload configuration. Restart it after changing
`config.toml`. Content TOML/CSV changes require `vxaos-content build` and a
client reconnect (and often a client restart) to pick up the generated scripts.
