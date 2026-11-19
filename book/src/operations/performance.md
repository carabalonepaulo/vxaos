# Performance and scaling

## What costs what

At 500 players the old server spent most of its time broadcasting player and
entity updates to every player on the map. That is `O(players²)` per map.

The current model reduces this in three layers:

1. **Map actors** — simulation for different maps runs independently.
2. **Route cache** — player commands do not round-trip through the coordinator
   to find the owning map on every packet.
3. **Area of interest (AOI)** — each player only receives entities inside a
   square window around them.

## Load tool

Build it with `just build`, then create disposable accounts:

```powershell
.\bin\vxaos-load.exe -config config.load.toml -setup -count 500
```

Run:

```powershell
.\bin\vxaos-load.exe -config config.load.toml -addr 127.0.0.1:5050 `
  -count 500 -duration 60s -ramp 10s `
  -move-every 500ms -attack-every 2s `
  -json bin\load.json
```

Flags:

| Flag | Meaning |
|---|---|
| `-count` | number of bots |
| `-duration` | measurement duration |
| `-ramp` | connection ramp |
| `-move-every` | per-bot move interval |
| `-attack-every` | per-bot attack interval |
| `-setup` | create accounts/characters instead of running |
| `-spread` | distribute spawn positions for an AOI workload |
| `-json` | write a JSON report |

The default spawn is a stress test: map 2 puts every bot on the same tile and
map 1 uses a 5x5 block. That is useful for broadcast stress but **not** for AOI.
Use `-spread` and a separate database/config for an AOI measurement.

The load server uses `config.load.toml` on `127.0.0.1:5050`, a separate SQLite
file and 1000 PBKDF2 iterations for bot accounts. Do not point the load tool at
production.

## Measured results

Hardware: Ryzen 9 5900X (12C/24T), Windows 11, loopback, 80 ms tick.

### Map actors and route cache (clustered workload)

500 bots, 60 s, 10 s ramp, move 500 ms, attack 2 s:

| Metric | Pre-parallel | Map actors |
|---|---:|---:|
| EnterWorld OK | 490 | 492 |
| CPU steady | 35.48% | 32.55% |
| CPU peak | 38.2% | 36.46% |
| Memory steady | 53.1 MB | 57.35 MB |
| Received frames | 12.99M | 13.19M |
| `send queue full` | 12 | 0.5 |

### AOI, spread workload

Same 500 bots and rates, positions distributed across 50x30/40x25 maps:

| Metric | No AOI | AOI 9 | Diff |
|---|---:|---:|---:|
| EnterWorld | 500 | 500 | 0 |
| Received frames | 14.12M | 6.93M | -50.9% |
| Received bytes | 114.29 MB | 56.00 MB | -51.0% |
| CPU steady | 33.92% | 19.04% | -43.9% |
| CPU peak | 42.54% | 24.71% | -41.9% |
| Memory | 81.9 MB | 84.6 MB | +3.3% |

The clustered workload cannot show AOI culling because the window contains the
whole cluster. Its numbers are in `server/docs/load-test-aoi.md`.

## Tuning

### `aoi_radius_tiles`

- `9` covers the default 544x416 viewport (17x13 tiles) with margin.
- `0` disables AOI and restores map-wide broadcasts.
- Larger values increase bandwidth and visible-set memory; smaller values may
  clip the screen at map edges unless the camera is also limited.

The window is shifted inward at map edges, so the player sees the whole clamped
camera area. Hysteresis (`aoi_hysteresis_tiles`) prevents enter/leave flapping.

### `tick_ms`

80 ms is a good default. Lower values increase CPU for combat/NPC/event
resolution; movement is rate-limited separately.

### `save_interval_seconds`

900 s is conservative. More frequent saves increase database writes; action
saves already cover important changes.

### `max_connections`

Bounded by memory and send queues. Each session has a 1024-packet send queue.
At 500 sessions this is a few MB, but a slow client can still fill its queue and
be closed by design.

### `map_idle_seconds`

Empty maps stop ticking immediately and are destroyed after this grace. Use 60 s
for normal play; lower it for many short-lived dungeon instances, raise it if
players frequently bounce between two maps.

## Reading a report

- `enter_ok` below `connected` means the client did not receive `EnterWorld`
  in time. Check send queue, AOI snapshot size and server logs.
- `send queue full` means a slow session was closed. It is a symptom, not a
  cause; find the broadcast burst.
- CPU steady excludes the initial ramp; peak includes the join storm.
- Received bytes per second is the best single indicator for AOI efficiency.

## Scaling beyond one process

The current server is one process. The actor model makes future sharding
possible, but there is no distributed coordinator yet. For now:

- run one server per world/database;
- keep SQLite local and fast;
- use a reverse proxy/VPN for network isolation;
- monitor memory as player count and AOI radius grow.
