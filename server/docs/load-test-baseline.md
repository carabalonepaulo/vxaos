# Load test baseline — single-world server

Date: 2026-11-19  
Server revision: `2aad2f2` (before the per-map actor architecture)  
Status: baseline for comparison after map parallelization

## Environment

- OS: Windows 11 Pro 10.0.26100
- CPU: AMD Ryzen 9 5900X, 12 cores / 24 logical processors
- Server and bots on the same machine, over loopback (`127.0.0.1`)
- SQLite database: `data/vxaos-load.db`, disposable
- Server log level: `warn`
- Server tick: 80 ms
- Save interval: 900 s
- Server build: `vxaos-server.exe` from `cmd/vxaos-server`
- Bot build: `vxaos-load.exe` from `cmd/vxaos-load`

## Architecture under test

At this revision:

- one goroutine owns the whole world;
- all maps are ticked sequentially in that goroutine;
- there are no per-map goroutines or parallel map ticks;
- player/map/enemy/NPC/status storage uses id-indexed slices;
- network sessions have their own reader/writer goroutines, but all game
  mutations are serialized through the world loop;
- database saves and joins happen inside the world goroutine;
- no map runtime state is persisted.

This is the reference point before moving maps to on-demand actors.

## Workload

- 500 bot accounts, each with one character
- Bots split 50/50 between map 1 and map 2
- Connection ramp: 10 seconds
- Measurement duration: 60 seconds
- Movement: one `MOVE` per bot every 500 ms
- Combat: one `ATTACK` per bot every 2 seconds
- No chat traffic
- Bot accounts created with PBKDF2 `1000` iterations to keep the test focused
  on runtime load; production accounts use `210000`
- No periodic character save occurs inside the 60 s window (900 s interval)
- Bot setup: 500 accounts + 500 characters in 1.14 s

## Results

| Metric | Value |
|---|---:|
| Bots attempted | 500 |
| TCP connections | 500 |
| Auth success | 500 |
| EnterWorld success | 490 |
| Failed | 10 |
| Measured duration | 60.01 s |
| Auth latency p50 | 2.1 ms |
| Auth latency p95 | 6.5 ms |
| EnterWorld latency p50 | 0.5 ms |
| EnterWorld latency p95 | 2.0 ms |
| Frames sent by bots | 66,747 (~1,112/s) |
| Frames received by bots | 12,992,417 (~216,540/s) |
| Payload received by bots | 102,858,678 bytes (~1.71 MB/s) |
| Server CPU, steady state | 35.48% across 24 logical CPUs |
| Server CPU, peak | 38.2% across 24 logical CPUs |
| Server working set, steady avg | 53.1 MB |
| Server working set, peak | 55.2 MB |
| Server idle working set | 17.0 MB |
| Send queue full events | 12 |

CPU percentage is relative to all 24 logical processors. The steady-state
average of 35.48% corresponds to roughly 8.5 logical CPUs busy.

The 10 missing EnterWorld responses occurred during the connection ramp. The
server logged `netserver: send queue full` for 12 sessions; the session send
queue is bounded at 256 frames and closes the session when it overflows.

## Observations

1. **The single world goroutine is the throughput ceiling.**
   All map ticks, combat, broadcasts, entrances and DB calls run in one
   sequence. At 490 online bots the server already shows a sustained CPU cost
   around 35% of the machine.

2. **Movement broadcast is O(players per map).**
   With 500 bots split across two maps, every move is broadcast to roughly 250
   players. This is the main reason for the 216k received frames/s and the
   send-queue overflows.

3. **The send queue is a deliberate backpressure boundary.**
   A 256-frame queue is enough for normal play but overflows during a join storm
   where hundreds of players receive player-data and movement broadcasts at
   once. Ten bots failed to enter in this workload.

4. **Latency is still low on loopback.**
   Auth and EnterWorld p95 are in single-digit milliseconds. The bottleneck is
   CPU/broadcast volume, not handshake or connection setup.

5. **Memory is not a concern at this scale.**
   The server stays around 53–55 MB with 490 online players.

6. **Database persistence was not stressed.**
   With a 900 s save interval and a 60 s run, there were no periodic saves.
   The test does not represent slow-disk or save-contention behavior.

7. **Auth time is not production-representative.**
   Bot accounts used 1000 PBKDF2 iterations. Production uses 210000; real auth
   CPU will be higher, but auth is outside the steady-state tick loop.

## Raw data

- `docs/load-test-baseline.json`
- `docs/load-test-baseline-samples.csv`

## Reproduction

```powershell
cd server
go build -o bin/vxaos-server.exe ./cmd/vxaos-server
go build -o bin/vxaos-load.exe ./cmd/vxaos-load

# first time only: create disposable bots
.\bin\vxaos-load.exe -setup -config config.load.toml -count 500

# terminal 1: server
.\bin\vxaos-server.exe -config config.load.toml

# terminal 2: load
.\bin\vxaos-load.exe -config config.load.toml -addr 127.0.0.1:5050 `
  -count 500 -duration 60s -ramp 10s `
  -move-every 500ms -attack-every 2s `
  -json bin\load-test-baseline.json
```

The server process must be running with `config.load.toml`; the dev
`config.toml` uses port 5000 and the live database.

## Comparison rules for the parallel-map version

To keep the comparison meaningful, preserve:

- same 500 bots, same 50/50 map split;
- same 10 s ramp and 60 s duration;
- same 500 ms movement and 2 s attack intervals;
- same 80 ms tick;
- same SQLite driver and no periodic save inside the window;
- same machine and loopback setup;
- same `vxaos-load` JSON output.

Diff the following primarily:

- EnterWorld success count and send-queue-full events;
- server steady-state CPU and peak CPU;
- server working set;
- sent/received frames and bytes;
- auth and enter latency percentiles.

Expected architectural differences to evaluate later:

- map CPU should distribute across map actors;
- one busy map should not block an idle map;
- cross-map handoffs add latency and need their own percentiles;
- the coordinator/global state may become the new contention point.
