# Load test — area of interest

Date: 2026-11-19  
Feature: server-authoritative area of interest (AOI) on top of the map-actor
model.  
Comparison reports:

- `docs/load-test-post-parallel.md` — map actors, no AOI
- `docs/load-test-baseline.md` — coordinator model, no AOI

## What was implemented

- AOI is local to each map actor. The coordinator never sees entity positions.
- Each player has a square window of `2*aoi_radius_tiles+1` tiles. The window is
  shifted inward at map edges, matching the client camera clamp. In the interior
  it is the player-centered square requested; at edges the player stays inside
  the window and the whole clamped camera area is covered.
- `aoi_radius_tiles = 9`, `aoi_hysteresis_tiles = 2`, `aoi_cell_size = 8`.
  The radius covers the default VX Ace viewport (544x416 at 32 px = 17x13
  tiles) with a small margin.
- Per-player `visible` sets and a uniform grid index players, enemies, NPC/event
  sprites and drops.
- Movement, NPC/enemy movement, spawn/revive/death, drops, animations, vitals,
  equipment and guild-name updates are all scoped to the viewers whose window
  contains the position.
- The protocol/client gained:
  - `MapReset` — clears client-side AOI state before a new snapshot;
  - `EventRemove` — removes an NPC/enemy sprite that left the window or died;
  - `MapPlayerEnter` / `MapPlayerLeave` — logical map membership, separate from
    AOI sprite add/remove (`PlayerData` / `PlayerLeft`);
  - stable drop IDs (`uint32`) carried by `DropAdd`, `DropRemove` and
    `PickupDrop`.
- Drops use a generation-checked slab: O(1) allocate/lookup/release, no map
  allocation per drop, and stale client IDs cannot pick up a recycled slot.
- `PlayerLeft` no longer refreshes the party HUD in the client; party state is
  driven by party packets.
- The session send queue was raised from 256 to 1024. The initial AOI snapshot
  can legitimately send more packets than the old map-wide introduction in a
  very short burst; 256 caused client sessions to be closed during the join
  storm.
- AOI is disabled with `aoi_radius_tiles = 0`, which keeps the pre-AOI behavior
  for A/B tests.

Map chat remains map-wide. Party/guild/global chat remains coordinator-driven.

## Workloads

Both use 500 bots, 60 s, 10 s ramp, one move every 500 ms and one attack every
2 s. The only difference is spawn distribution.

- **Clustered** is the original baseline workload:
  - map 2: all bots spawn on the same tile `(4,20)`;
  - map 1: bots spawn in a 5x5 block around `(21,12)`.
- **Spread** uses the new `-spread` load flag:
  - map 1: positions distributed over 50x30;
  - map 2: positions distributed over 40x25.

The clustered workload is a broadcast stress test, not an AOI test: a 19x19
window contains the whole cluster, so AOI cannot cull anything.

## Spread workload (AOI A/B)

| Metric | No AOI | AOI 9 | Diff |
|---|---:|---:|---:|
| TCP connections | 500 | 500 | 0 |
| Auth OK | 500 | 500 | 0 |
| EnterWorld OK | 500 | 500 | 0 |
| Failed | 0 | 0 | 0 |
| Duration | 60.01 s | 60.01 s | 0 |
| Sent frames | 68,250 | 68,250 | 0 |
| Received frames | 14,124,224 | 6,929,088 | **-50.9%** |
| Received bytes | 114,286,223 | 55,998,516 | **-51.0%** |
| Auth p95 | 7.1 ms | 4.2 ms | -2.9 ms |
| Enter p95 | 1.7 ms | 1.2 ms | -0.5 ms |
| Server CPU steady | 33.92% | 19.04% | **-43.9%** |
| Server CPU peak | 42.54% | 24.71% | **-41.9%** |
| Working set steady | 81.9 MB | 84.6 MB | +3.3% |
| `send queue full` | 0 | 0 | 0 |

This is the representative AOI result: about half the bandwidth and 44% less
steady CPU, with a small memory increase.

## Clustered workload (original stress test)

| Metric | No AOI | AOI 9 | Diff |
|---|---:|---:|---:|
| EnterWorld OK | 500 | 500 | 0 |
| Failed | 0 | 0 | 0 |
| Received frames | 13,976,982 | 13,441,584 | -3.8% |
| Received bytes | 112,803,568 | 106,966,104 | -5.2% |
| Server CPU steady | 35.33% | 36.47% | +3.2% |
| Server CPU peak | 43.24% | 39.28% | -9.2% |
| Working set steady | 82.8 MB | 89.2 MB | +7.7% |
| `send queue full` | 0 | 0 | 0 |

The clustered result shows the fixed AOI cost without a culling benefit. It is
kept to make the trade-off explicit and to avoid claiming a win that the
original workload cannot produce.

## Analysis

1. **AOI is the largest bandwidth win so far on a realistic distribution.**
   Frames and bytes both fall by about half. Most of the old traffic was
   broadcasting every player move to every player on the map.

2. **CPU falls with the traffic.**
   Sending is a significant part of the map tick; with half the recipients, the
   steady CPU drops from 33.9% to 19.0%. The AOI diff itself (grid lookups,
   visible sets) is cheaper than the removed sends for this distribution.

3. **Memory rises slightly.**
   Each player keeps a visible set and a view window; the session send queue is
   also four times larger. At 500 players this costs a few MB, not a leak.

4. **The original workload cannot show the win.**
   Stacking 250 bots on one tile means their windows all contain the same
   entities. The result is AOI overhead with no culling. This is expected and
   documented rather than hidden.

5. **Join bursts need headroom.**
   The initial AOI snapshot is more bursty than the old map-wide introduction.
   The 1024 send queue removed all EnterWorld failures at 500 bots; with 256 the
   same run failed 57 enters because sessions closed on a full queue.

6. **The client is authoritative only for rendering input, never for visibility.**
   There is no client-reported viewport. The server window is fixed by config.

## Validation

- `go test ./...` and `go test -race ./...` pass.
- 50 bots against a `-race` AOI server: 0 data races, 0 panics.
- Actor tests cover AOI player enter/leave and drop enter/leave.
- Real client smoke test with the patched `Scripts.rvdata2`:
  - authenticated, entered the world;
  - received `MapReset` (op 78) and AOI `EventData` (op 23);
  - no Ruby/network errors in `Client/System/network.log`.

## Files

- `docs/load-test-aoi-spread.json` / `-samples.csv`
- `docs/load-test-aoi-spread-noaoi.json` / `-samples.csv`
- `docs/load-test-aoi-stacked.json` / `-samples.csv`
- `docs/load-test-aoi-stacked-noaoi.json` / `-samples.csv`

## Reproduction

```powershell
cd server-go
go build -o bin/vxaos-server.exe ./cmd/vxaos-server
go build -o bin/vxaos-load.exe ./cmd/vxaos-load

# use a separate database so -spread creates fresh positions
Copy-Item config.load.toml config.load.spread.toml
# edit config.load.spread.toml: dsn = "data/vxaos-load-spread.db"

# create a spread dataset
.\bin\vxaos-load.exe -config config.load.spread.toml -setup -spread -count 500

# terminal 1
.\bin\vxaos-server.exe -config config.load.spread.toml

# terminal 2
.\bin\vxaos-load.exe -config config.load.spread.toml -addr 127.0.0.1:5050 `
  -count 500 -duration 60s -ramp 10s `
  -move-every 500ms -attack-every 2s `
  -json bin\load-aoi-spread.json
```

For the no-AOI A/B, set `aoi_radius_tiles = 0` in a copy of
`config.load.spread.toml` and use the same spread database.
