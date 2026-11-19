# Load test — map actors (final)

Date: 2026-11-19  
Comparison: `docs/load-test-baseline.md`  
Workload: identical to the baseline (500 bots, 50/50 maps 1/2, 10 s ramp,
60 s measurement, one move every 500 ms, one attack every 2 s, no chat).

## What changed

This is the completed map-actor model, not the v1 hybrid.

- One actor goroutine per active map with its own mailbox.
- The actor owns the map runtime and the full player runtime while the player
  is on that map: position, HP/MP/EXP/gold, inventory, equips, status, event,
  shop, bank window, trade and target.
- Player commands (`MOVE`, `ATTACK`, `USE_SKILL`, `USE_ITEM`, `EQUIP`,
  `BANK`, `SHOP`, `DROP`, `PICKUP`, events, teleport, trade, guild creation)
  execute on the owning map actor.
- The coordinator owns the online directory, guilds, parties, friends,
  requests, global switches, MOTD, map/instance lifecycle and persistence.
- Player transfer uses detach-by-snapshot and attach-by-snapshot with a new
  route epoch. No map goroutine shares a mutable `*Player` with another.
- The global `RWMutex` barrier between commands and ticks was removed.
- Events, autoruns, NPC touch, player death/respawn and drop expiry run inside
  the map tick.
- Empty maps stop ticking immediately and are destroyed after
  `map_idle_seconds`.
- `MapKey{TemplateID, InstanceID}` is used for every route and map registry
  entry; instance `0` is the open world.
- Periodic and detach saves are snapshots handed to the persistence worker.
  Bank saves also go through the same worker.
- Each session caches its route by player id; the cache stores
  `{MapKey, epoch, actor}` and is refreshed on attach/transfer. Stale epochs
  are rejected and retried once.

Trade remains restricted to the same map instance and the configured maximum
distance.

## Results

Baseline is one run. Post is the average of two runs of the final binary.

| Metric | Baseline | Post | Diff |
|---|---:|---:|---:|
| TCP connections | 500 | 500 | 0 |
| Auth OK | 500 | 500 | 0 |
| EnterWorld OK | 490 | 496 | **+6** |
| Failed | 10 | 4 | **-6** |
| Duration | 60.01 s | 60.01 s | 0 |
| Sent frames | 66,747 | 67,246 | +499 |
| Received frames | 12,992,417 | 13,187,286 | +1.5% |
| Received bytes | 102,858,678 | 104,370,890 | +1.5% |
| Auth p95 | 6.5 ms | 7.0 ms | +0.5 ms |
| Enter p95 | 2.0 ms | 1.85 ms | -0.15 ms |
| Server CPU steady | 35.48% | 32.55% | **-8.3%** |
| Server CPU peak | 38.2% | 36.46% | **-4.6%** |
| Working set steady | 53.1 MB | 57.35 MB | +8.0% |
| `send queue full` | 12 | 0.5 | **-11.5** |

Per run:

| Run | CPU steady | CPU peak | Memory | EnterWorld | Failed | Queue full |
|---|---:|---:|---:|---:|---:|---:|
| post run 1 | 33.79% | 38.58% | 56.7 MB | 496 | 4 | 0 |
| post run 2 | 31.31% | 34.33% | 58.0 MB | 496 | 4 | 1 |

## Analysis

1. **CPU is lower and the join storm is much healthier.**
   Steady CPU fell about 8% and the send-queue failures dropped from 12 to
   0–1. EnterWorld success rose from 490 to 496. The clients that used to be
   dropped by a full send queue now survive the join burst.

2. **The global barrier is gone.**
   In the v1 hybrid every player command took a world write lock and paused
   all map ticks. Now a command is posted to the owning map mailbox and only
   that actor is affected. The route cache removed the coordinator round trip
   from the command hot path; without it the same workload measured 36.5–37.1%
   CPU, which shows the cache is worth keeping.

3. **Memory is slightly higher.**
   The extra ~4 MB comes from one goroutine/ticker/mailbox per active map,
   per-player snapshot materialization on attach, and the route cache. With
   only two active maps this is the expected fixed cost, not a leak.

4. **Frames/bytes are slightly higher.**
   The post runs completed more EnterWorld handshakes (496 vs 490), so they
   legitimately sent and received more map/player data during the ramp. The
   per-player steady-state broadcast cost did not change.

5. **Variance is still visible.**
   The two post runs differ by ~2.5 CPU points in steady state and ~4 points at
   peak. The baseline is a single run, so the comparison should be read as
   "same or better CPU with a much lower failure rate", not as a precise
   instruction-level result.

6. **Only two maps.**
   The actor model scales with the number of active maps. The workload uses two
   maps with ~250 players each, so the win here comes from removing the global
   barrier and improving join behavior, not from spreading many small actors.

## Race and correctness validation

- `go test ./... -race` passes.
- A 50-bot run against a `-race` server reports 0 data races and 0 panics.
- Actor tests cover:
  - join/move and event-driven transfer across maps;
  - idle map unload and recreation;
  - party invite/accept/chat/leave;
  - guild create/invite/accept/leave;
  - trade with same-map offers and persisted result.
- A real client smoke test authenticated and entered the world on the actor
  server.

## Remaining boundaries (intentional)

- Map runtime state is not persisted; only player/bank state is.
- `InstanceID` is 0 everywhere because no content creates dungeon instances
  yet. The routing and registry already support non-zero values.
- Party join packets use a coordinator-side view refreshed on join/transfer;
  live HP/EXP changes are not mirrored into that view between joins.

## Reproduction

```powershell
cd server-go
go build -o bin/vxaos-server.exe ./cmd/vxaos-server
go build -o bin/vxaos-load.exe ./cmd/vxaos-load

# terminal 1
.\bin\vxaos-server.exe -config config.load.toml

# terminal 2
.\bin\vxaos-load.exe -config config.load.toml -addr 127.0.0.1:5050 `
  -count 500 -duration 60s -ramp 10s `
  -move-every 500ms -attack-every 2s `
  -json bin\load-post.json
```

Raw data:

- `docs/load-test-post-parallel.json` (run 1)
- `docs/load-test-post-parallel-samples.csv`
- `docs/load-test-post-parallel-run2.json` (run 2)
- `docs/load-test-post-parallel-run2-samples.csv`
- `docs/load-test-baseline.json`
- `docs/load-test-baseline-samples.csv`
