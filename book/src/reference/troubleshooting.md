# Troubleshooting

## The client cannot connect

Symptoms: connection refused, handshake timeout, immediate disconnect.

Check:

1. Server is running and logs `listening`.
2. `listen` and the client `HOST`/`PORT` match.
3. Firewall allows TCP 5000.
4. The client is patched with protocol v2.
5. `network.log` in `Client/System` for the last sent/received opcode.

A wrong magic/version closes the connection immediately; the server log shows a
handshake failure.

## Version mismatch

The server and client `GAME_VERSION` must match.

```powershell
go run ./cmd/vxaos-content check
go run ./cmd/vxaos-content build
```

Then restart the server and client. `game_version` lives in
`content/configs.toml` `[network]` and is rendered into the client.

## Authentication fails

- Incorrect password or unknown account.
- Account banned (account or IP).
- Client content out of date (challenge/KDF mismatch).
- `[auth] iterations` changed but the account uses its stored iterations; that
  is fine, the challenge carries the per-account value.
- Registration is disabled; create accounts with `vxaos-admin`.

Unknown usernames intentionally receive a fake challenge; the error is the same
as a wrong password.

## Character already online

The online directory still has the character. Causes:

- the previous session did not disconnect cleanly;
- two clients use the same character;
- a crash left the process running.

Check the server process and logs. Restarting the server clears the in-memory
directory; the database is not changed.

## Empty character list

Create a character:

```powershell
vxaos-admin.exe character create -config config.toml `
  -user admin -name Hero -class 1 -sex 0
```

or use the client's creation window.

## Client closes during EnterWorld / join storm

- The initial AOI snapshot is bursty; a slow client can fill its 1024-packet
  send queue and be closed (`send queue full`).
- Reduce `aoi_radius_tiles` or increase client/network performance.
- Check for many players/entities in a small area.
- Check server CPU and DB latency.
- Re-apply `vxaos-clientpatch` after protocol changes.

## Map is empty or sprites are missing

- AOI only sends entities inside the window; that is expected.
- `MapReset` clears old sprites; if it is not handled, the client may show stale
  sprites. Re-apply the client patch.
- A player/NPC outside AOI has no chat balloon by design.
- The minimap shows only AOI entities.
- If the whole map is empty, check `MapReset`/`EventData` in `network.log` and
  server logs for `attach failed`.

## Drops cannot be picked up

- The drop id is stale/expired; refresh by moving.
- You are not on the drop tile.
- Another owner's pickup delay has not elapsed.
- The client patch predates stable drop ids; re-apply it.

## Trade fails

- Players are not on the same map instance.
- Distance is greater than 10 tiles.
- One side is in a shop/bank/trade.
- An offered item is soulbound or no longer owned.
- The window was closed by logout/cancel.

## Shop/bank does nothing

- The event must use command 302 (shop) or `open-bank` (bank).
- The server is authoritative; check the error message (`not in game`, full
  inventory, not enough gold, invalid target).
- The bank is account-wide; reopening should show the persisted state.

## Quest does not start/finish

- Scripts use one-based ids; TOML is zero-based.
- The quest is already in the state (finished non-repeatable).
- Requirements (switch/variable/item/kills) are not met.
- `vxaos-content build` was not run after editing `quests.toml`.
- The server was not restarted after editing `quests.toml`.

## Lisp error

The server logs the script name and error. Common causes:

- Ruby syntax left in a 355 script (`a.atk`, `start_quest(1)`);
- unknown native;
- unquoted list;
- formula returns a non-number;
- script was `[NG]` but expected to run on the server.

Run `vxaos-formula check` / `vxaos-events check` and read the reported lines.

## Database is locked (SQLite)

- Another process has the DB open (admin tool, backup, antivirus).
- The DB is on a network share; keep it on local disk.
- A long transaction from a manual SQL client.
- Stop the server, close other tools, retry.

Back up the DB before manual changes.

## Server exits at startup

Read the first error line. Typical:

| Error | Cause |
|---|---|
| `config: ... no such file` | wrong `-config` path |
| `data_dir is empty` | missing `data_dir` |
| `game data loaded` fails | wrong `data_dir` |
| `unsupported store.driver` | typo in `[store] driver` |
| `map_idle_seconds`/`aoi_*` invalid | values out of range |
| content load error | invalid `configs.toml`/`enums.toml`/`quests.toml` |

## Performance regressions

- Check `send queue full` and received bytes/s.
- Measure with `vxaos-load -spread` for AOI.
- Use `go test -race` and `go vet`.
- Check DB latency and map actor count.
- Do not run the load tool against production.

## Client input tests

- Set `VXAOS_TEST_INPUT` before starting the client.
- Use `vxaos-input -addr 127.0.0.1:<port>`.
- Keep the game window visible; minimized windows throttle frames.
- `reset` releases injected keys/buttons.
