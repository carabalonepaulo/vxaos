# Architecture

## High level

```text
TCP sessions ──intents──▶ World coordinator ──directory/route──▶ map actor
     ▲                         │                                   │
     │                         ├── guilds/parties/friends            ├── mapState
     │                         ├── global switches/MOTD              ├── players
     │                         └── persistence worker              ├── enemies/NPCs/drops
     └──────── packets ──────── clients ◀──── AOI-scoped packets ───┘
```

- **Sessions** own the socket, auth state and the player id. They never read
  map/player runtime fields.
- **Coordinator** owns the online directory, social state, global switches,
  MOTD, map/instance lifecycle and persistence.
- **Map actors** own map runtime and the full runtime of the players on that
  map.
- **Persistence worker** consumes immutable snapshots.

## Ownership table

| State | Owner |
|---|---|
| online directory, character ids | coordinator |
| guilds, parties, friends, requests | coordinator |
| global switches, MOTD | coordinator |
| map/actor registry, next player id/epoch | coordinator |
| mapState (npcs, enemies, drops, autorun) | map actor |
| player character/state/status/event/shop/bank/trade/target | map actor |
| map AOI windows/visible sets | map actor |
| character/bank persistence queue | worker |

The coordinator never dereferences a map-owned `*Player`. Social decisions use
`onlineEntry` and IDs; map actions use map commands.

## Routing

A player command resolves to `playerRoute{key, epoch, actor}`:

1. The route cache is checked.
2. On a miss, the coordinator returns the current `onlineEntry`.
3. The command is posted to the actor inbox with `playerID + epoch`.
4. The actor rejects stale epochs; the caller refreshes and retries once.

A transfer:

1. origin actor detaches and creates a `playerSnapshot`;
2. coordinator updates the directory and increments the epoch;
3. destination actor materializes a new `Player` and sends `Transfer` + AOI
   snapshot.

No map holds the same mutable player object as another map.

## Concurrency rules

- Map state is single-owner: only the actor goroutine touches `ms`/players.
- The coordinator never waits for a map. Coordinator→map messages are
  fire-and-forget.
- Maps may wait for the coordinator (social/global operations) because the
  coordinator always drains its queue.
- Mailboxes are mutex-protected unbounded queues with a `wake` channel. This
  avoids blocking the coordinator on a full channel.
- Persistence snapshots are deep clones; the worker never sees live state.
- Stop flushes actors before closing the worker.

## AOI

Each player has a square window of `2*radius+1` tiles, shifted inward at map
edges to cover the clamped camera. A per-map grid indexes players, enemies,
NPC/event sprites and drops. A visible set per viewer is diffed on movement:

- enter when inside the window;
- keep until outside window + hysteresis;
- remove with `PlayerLeft`/`EventRemove`/`DropRemove`.

Initial snapshots are sent after `MapReset`, which clears the client's previous
sprite/drop state.

## Tick

Each actor tick runs, in order:

1. combat, status, respawn, recovery, enemy AI;
2. event timers;
3. autoruns;
4. NPC movement;
5. drop expiry;
6. periodic save snapshots.

Empty maps skip all of it and are unloaded after `map_idle_seconds`.

## Packages

| Package | Responsibility |
|---|---|
| `internal/rvdata` | Ruby Marshal 4.8 reader/writer, Tables, Scripts |
| `internal/rpg` | typed RPG Maker data loader |
| `internal/content` | configs/enums/quests/vocab generation and validation |
| `internal/store` | schema/migrations/repositories |
| `internal/auth` | PBKDF2 verifier/proof |
| `internal/chargen` | initial character creation |
| `internal/protocol` | v2 framing, opcodes, messages |
| `internal/script` | Lisp reader/compiler/VM |
| `internal/formula` | formula migration/validation |
| `internal/eventscript` | event Lisp migration |
| `internal/game` | coordinator, actors, gameplay, AOI, persistence |
| `internal/net` | TCP sessions, handshake, auth, command dispatch |
| `cmd/*` | server/admin/content/clientpatch/formula/events/load/input |

## Design constraints

- No Ruby at server runtime.
- No per-map network reactor.
- No map-state persistence.
- No global mutex around gameplay.
- Server authority for all gameplay outcomes.
- Protocol v2 only; client must be patched.
