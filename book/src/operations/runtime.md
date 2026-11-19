# Runtime and persistence

## Processes and goroutines

The server is one process with:

- one TCP acceptor and one reader/writer pair per session;
- one **coordinator** goroutine owning the online directory, social state,
  global switches, MOTD, map lifecycle and persistence;
- one **map actor** goroutine per active map, owning map runtime and the players
  currently on it;
- one **persistence worker** goroutine consuming character/bank snapshots.

## Map lifecycle

1. First player enters a map → `ensureMap` creates the runtime from the
   immutable `rpg.Map` template and starts the actor.
2. The actor ticks combat, events, autoruns, NPCs and drops.
3. When the map has no players, it stops simulating.
4. After `map_idle_seconds`, the coordinator removes the actor and frees its
   runtime.

Map runtime state is **not** persisted. Re-entering a map rebuilds NPC/enemy
positions, drops and autorun state from the template.

## Persistence model

- Character and bank state are persisted.
- Periodic saves take a deep snapshot on the map actor and enqueue it to the
  persistence worker.
- Action saves (`saveQuietly`) enqueue a snapshot; the map actor does not wait
  for the database.
- Disconnect/leave enqueues a snapshot before the player is removed.
- `World.Stop` flushes all actors and then waits for the worker queue.
- The worker clones again before writing, so map state can never be mutated by
  the database path.

Snapshots include character stats/position, inventory, equips, skills, quests,
hotbar, switches, variables, self switches, status, bank and the player's
runtime social caches.

## Graceful shutdown

Send Ctrl+C/SIGTERM once. The server:

1. stops accepting commands;
2. tells every map actor to flush its players;
3. waits for actors to finish;
4. closes and flushes the persistence worker.

Do not kill the process during the join/leave path if you can avoid it. If you
must, the last periodic save is the recovery point.

## Backups

SQLite default:

```text
data/vxaos.db
data/vxaos.db-wal   (file SQLite enables WAL)
data/vxaos.db-shm
```

Recommended:

1. Stop the server gracefully.
2. Copy all three files to a timestamped backup directory.
3. Keep at least one backup outside the machine.
4. Test restoration on a copy before relying on it.

For MySQL/PostgreSQL use the normal dump tools. The schema migrations are
embedded in the server binary; restore the database, then run
`vxaos-admin migrate`.

## Logs

The server logs to stdout/stderr as structured `slog` text. Set
`log_level = "debug"` only while diagnosing; debug logs include per-message
network activity and can be large.

Useful events:

- `server started`, `listening`, `database ready`;
- `map activated`, `map unloaded`;
- `player joining`, `player joined`, `player left`;
- `auth ok`, `handshake ok`, `attach failed`;
- `send queue full`, `persist queue full`, `save character`.

Collect stdout/stderr with the process manager of your choice. On Windows,
`start-server.bat` keeps the console open; redirect to a file for a service.

## Database drivers

- **SQLite** (default): single file, no separate service. Good for small and
  medium servers. Keep the DB on local disk, not a network share.
- **MySQL/PostgreSQL**: dialect migrations are included but not part of the
  release test matrix. Use `max_open_conns`/`max_idle_conns` to bound the pool.

The schema is not compatible with the old website or Account Editor. A fresh
schema is expected.

## Security posture

- Passwords are PBKDF2-HMAC-SHA256 verifiers; the server never stores plaintext.
- Unknown accounts receive fake challenges, so login does not enumerate users.
- Bans are checked at authentication for account and IP.
- Admin permissions are enforced server-side.
- Damage, drops, inventory, trade, shops, banks, movement and visibility are
  server-authoritative.
- There is **no TLS**. Use a VPN or a TLS-terminating proxy for untrusted
  networks. The stored verifier is password-equivalent for authentication; a DB
  leak does not reveal the password but can be used to log in.

## Maintenance windows

1. Set a MOTD warning with the admin panel.
2. Stop the server gracefully.
3. Back up the database.
4. Replace binaries/content.
5. Run migrations.
6. Regenerate client content if shared content changed.
7. Start the server and verify `listening`.
8. Log in with a test account before reopening.
