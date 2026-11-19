# Load test — parallel map simulation v1

Date: 2026-11-19  
Base commit: `9fc41c9` (on-demand maps) plus the parallel simulation commit  
Comparison: `docs/load-test-baseline.md`

## What changed

This version adds a goroutine per activated map. Each map actor runs the
autonomous simulation for its own map:

- combat/enemy AI/status tick;
- NPC movement;
- ground-drop expiry.

Maps are still activated on demand by `ensureMap`.

Important limitation: **this is not the final queue-owned actor model yet**.

- Player commands (`MOVE`, `ATTACK`, `USE_ITEM`, `USE_SKILL`) still run on the
  coordinator/world goroutine.
- The event interpreter and autoruns still run on the coordinator.
- A global `RWMutex` barrier separates world commands from map ticks.
- Maps tick in parallel with each other, but a world command stops all map ticks
  while it runs.
- Player ownership and attach/detach handoff are not implemented in this
  version.

So the report measures the first parallelization slice, not the full target
architecture.

## Workload

Same as the baseline:

- 500 bots
- 50/50 between map 1 and map 2
- 10 s ramp
- 60 s measurement
- one `MOVE` every 500 ms per bot
- one `ATTACK` every 2 s per bot
- no chat
- SQLite load database, 1000 PBKDF2 iterations for bot accounts
- Windows 11, Ryzen 9 5900X (12C/24T), loopback
- 80 ms tick
- no periodic save inside the window

## Results

Baseline is one run. Post-parallel is the average of two runs.

| Metric | Baseline | Post v1 | Diff |
|---|---:|---:|---:|
| TCP connections | 500 | 500 | 0 |
| Auth OK | 500 | 500 | 0 |
| EnterWorld OK | 490 | 492 | +2 |
| Failed | 10 | 8 | -2 |
| Duration | 60.01 s | 60.01 s | 0 |
| Sent frames | 66,747 | 67,000 | +253 |
| Received frames | 12,992,417 | 13,082,168 | +89,751 |
| Received bytes | 102,858,678 | 103,411,802 | +553,124 |
| Auth p95 | 6.5 ms | 6.7 ms | +0.2 ms |
| Enter p95 | 2.0 ms | 1.9 ms | -0.1 ms |
| Server CPU steady | 35.48% | 33.27% | **-6.2%** |
| Server CPU peak | 38.2% | 36.29% | **-5.0%** |
| Working set steady | 53.1 MB | 54.15 MB | +2.0% |
| `send queue full` | 12 | 10 | -2 |

Post run 1: CPU steady 33.06%, peak 35.65%, enter 491, queue-full 10.  
Post run 2: CPU steady 33.47%, peak 36.92%, enter 493, queue-full 10.

## Analysis

The improvement is real but small: about 6% less CPU in steady state and 5% at
peak. The architecture change only parallelized autonomous simulation.

Why the gain is limited:

1. **Player commands still dominate.**
   At 500 bots, the workload sends ~1,000 moves/s and ~250 attacks/s. Every one
   still passes through the coordinator under the global write barrier, which
   pauses all map actors while it runs.

2. **Only two maps exist.**
   Parallelism helps more when there are many active maps. Here there are only
   two actors, each with ~250 players.

3. **Broadcast cost is unchanged and dominant.**
   Movement and player data are still broadcast to every player on the map,
   around O(players²) per map. Received frames stayed essentially the same.

4. **The actor tick is only part of the CPU.**
   Combat/NPC/drops simulation is now parallel, but command processing, event
   interpretation, chat, social state and socket work remain centralized.

5. **Queue pressure improved slightly.**
   `send queue full` fell from 12 to 10 and EnterWorld success rose from 490 to
   492, but the send queue remains the observable failure mode during the join
   storm.

6. **Memory increased slightly.**
   One goroutine/ticker per active map plus the parallel slices add ~1 MB at this
   scale, which is expected.

## Race validation

A 50-bot run was executed against a `-race` build of the server. No data races
were reported. This validates the global read/write barrier for the current
slice, not the final queue-owned model.

## Conclusion

Parallel map simulation v1 is safe and shows a measurable CPU reduction, but it
does not yet deliver the full architecture. The next meaningful step is moving
player ownership and player commands into the map actor:

- `AttachPlayer` / `DetachPlayer` with snapshots;
- player intents routed to the map mailbox;
- actor-owned event interpreter;
- coordinator only for online/social/persistence;
- remove the global command/tick barrier;
- keep trade restricted to the same map instance and configured maximum range.

That is the version that should produce a larger diff, especially with many
active maps.

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

- `docs/load-test-post-parallel.json`
- `docs/load-test-post-parallel-samples.csv`
- `docs/load-test-baseline.json`
- `docs/load-test-baseline-samples.csv`
