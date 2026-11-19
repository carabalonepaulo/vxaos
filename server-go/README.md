# VXA-OS server (Go rewrite)

> **Full documentation:** the public mdBook lives in [`../book`](../book)
> (`mdbook serve book`). This README is a development summary and may lag
> behind the book and the changelog.

This directory is the new server. The Ruby server in `../Server` is kept only as
a reference; the Go server never invokes Ruby at runtime.

## Decisions already made

- **Protocol v2 only.** Legacy framing is not supported. The client will be
  patched; the wire format is symmetric and length-prefixed in both directions.
- **Lisp only on the server**, for event command 355 scripts that run on the
  server and for damage formulas. It is not exposed to the client.
- **Formulas are Lisp.** The client overrides `RPG::UsableItem::Damage#eval` to
  return 0 and loses only the cosmetic "base damage" preview; the server is the
  only authority.
- **No `rmc`.** `.rvdata2` is parsed natively in Go. A targeted writer is used
  to update `Scripts.rvdata2`, maps and formulas.
- **New database schema.** The old website/Account Editor schema is not a
  compatibility target. SQLite is the default; migrations are part of the
  project from the start.
- **Auth:** PBKDF2-HMAC-SHA256 challenge-response, no TLS for now. The stored
  verifier is password-equivalent for authentication; a DB leak still requires
  an attacker to use that verifier, and it does not reveal the password.
- **Client changes are allowed**, but this phase keeps them scoped to the
  network layer, auth, generated content and the formula guard.

## Implemented so far

| Package | Status | Notes |
|---|---|---|
| `internal/rvdata` | done | Ruby Marshal 4.8 reader/writer, `Table`, `Color`, `Tone`, `Scripts.rvdata2` (zlib), object links on read, acyclic writer, Ruby-loadable output |
| `internal/rpg` | done | typed loader for Actors, Classes, Skills, Items, Weapons, Armors, Enemies, States, Animations, Tilesets, Common Events, System, MapInfos and Maps/Events/Pages/Commands; note-tag parsing |
| `internal/script` | done | Lisp reader, bytecode compiler, stack VM, lexical closures, variadic functions, native registry, host objects, instruction/depth limits |
| `internal/protocol` | done (core) | v2 symmetric frame codec, raw handshake, direction-specific opcodes, auth/register/select/move/chat/ping messages, fragmentation and limit tests |
| `internal/content` | partial | CSV vocab + TOML quests loaded and rendered into the generated `[VS] Vocab`/`[VS] Quests` entries; `check` validates, `build` rewrites `Scripts.rvdata2` with a `.bak` |
| `cmd/vxaos-content` | done (vocab/quests) | `check` and `build` CLI |
| `cmd/vxaos-formula` | done | migrates `Skills.rvdata2`/`Items.rvdata2` damage formulas to Lisp, keeping `.bak` files |
| `cmd/vxaos-events` | done (helper subset) | migrates event command 355 Ruby helper calls to Lisp in `Map*.rvdata2`/`CommonEvents.rvdata2`, keeping `.bak` files |
| `internal/store` | done (SQLite tested) | new schema, embedded dialect migrations for SQLite/MySQL/Postgres, account/character/state/bank repositories; SQLite E2E tests |
| `internal/auth` | done | PBKDF2-HMAC-SHA256 verifier and challenge proof, RFC/Python vectors |
| `internal/chargen` | done | initial character stats, skills, equips, graphics and position from the RPG data |
| `internal/game` | vertical slice | single-owner world loop, join/leave, tile/event/PvP collision, movement broadcast, map/global chat, periodic/leave save |
| `internal/net` | vertical slice | v2 handshake, PBKDF2 auth, character list/select/create, enter world, move, chat, logout; TCP integration tests |
| `cmd/vxaos-server` | usable | config, store, migrations, world and listener with graceful shutdown |
| `cmd/vxaos-admin` | usable | migrate, account create, character create/list/delete |
| `cmd/vxaos-clientpatch` | usable | replaces/inserts the v2 Ruby scripts in `Client/Data/Scripts.rvdata2`, keeping a `.bak` and reloading the archive |
| `client-patches/` | usable | v2 `Protocol`, `Crypto`, `Buffer`, `Socket`, `Network`, `Send_Data`, `Handle_Data` and the client `Damage#eval` guard |

## Running

```powershell
cd server-go
Copy-Item config.example.toml config.toml
go run ./cmd/vxaos-admin migrate -config config.toml
go run ./cmd/vxaos-admin account create -config config.toml -user admin -pass secret -group 2
go run ./cmd/vxaos-admin character create -config config.toml -user admin -name Hero -class 1 -sex 0
go run ./cmd/vxaos-clientpatch apply
go run ./cmd/vxaos-server -config config.toml
```

The admin command syntax is command-first, then flags. `config.toml` is ignored
by git; `config.example.toml` is the template.

`vxaos-clientpatch` reads `client-patches/manifest.toml`, writes
`Client/Data/Scripts.rvdata2` and keeps the previous archive as
`Scripts.rvdata2.bak`. Run `check` first to validate without writing. Re-running
`apply` is idempotent: existing entries are replaced, missing entries are
inserted at their manifest anchor.

The client `VXAOS.dll` is built from `Extra/VXAOS DLL` with Zig:

```powershell
powershell -File "Extra/VXAOS DLL/build.ps1"
```

It must be a 32-bit (`x86-windows-gnu`) DLL. The script builds directly into
`Client/System/VXAOS.dll` and verifies the crypto/input/socket exports. The
original DLL is kept as `Extra/VXAOS DLL/VXAOS-original.dll`.

The rebuilt DLL also fixes latent native bugs that MinGW happened to hide:

- `Input__key2char` told `ToUnicodeEx` it had two WCHARs while the caller only
  had one, so a typed key could overwrite the adjacent stack slot and corrupt
  the heap (`0xc0000374`). The caller now provides a four-WCHAR buffer and
  `Input__UTF8String` bounds-checks its UTF-8 buffers.
- `SocketLib__connect` freed the `addrinfo` list inside the loop and could keep
  an invalid socket after a failed candidate. It now walks the list safely and
  closes failed sockets.
- `SocketLib__recv` advanced its position by the `recv` return value even on 0
  or `SOCKET_ERROR`, which could loop forever. It now handles peer close and
  `WSAEINTR` explicitly.

On the Ruby side, `Network#update` originally used `while @socket.eof?`; the
helper returns true when `select()` says there is *no* data, so the game could
block in `recv` before parsing the handshake. It is now
`while @socket && !@socket.eof?`, matching the original client semantics and
draining the socket before `parse_stream`.

### Driving the client during tests

`VXAOS.dll` has an optional, test-only input bridge. It is disabled unless the
process starts with `VXAOS_TEST_INPUT` set to a TCP port; it then listens on
`127.0.0.1` only. Keyboard and mouse-button state are merged into
`Input__update`; the injected cursor position is exported as
`VxaosInputTest__mousePos` and the test-only `[VS] Test Input` script (also
gated by `VXAOS_TEST_INPUT`) reopens `Mouse.x`/`Mouse.y` and routes
`Mouse.click?`/`press?` through Input. It never calls `SendInput`, `SetCursorPos` or `SetForegroundWindow`, so the physical
mouse/keyboard and cursor stay with the user. The override expires two seconds
after the last `move`/`click`:

```powershell
cd server-go
go build -o bin/vxaos-input.exe ./cmd/vxaos-input
powershell -File scripts/test-client.ps1 -Port 5001

.\bin\vxaos-input.exe click 476 329
.\bin\vxaos-input.exe type "secret"
.\bin\vxaos-input.exe key enter press 80
.\bin\vxaos-input.exe reset
```

Commands: `key <name|vk> down|up|press [ms]`, `type <text>`, `move <x> <y>`,
`click <x> <y> [left|right|middle]` and `reset`. Coordinates are client
coordinates (the same ones the game uses). Keep the game window visible while
driving it; Windows throttles the frame rate of minimized windows, which can
make the 60 Hz input sampling miss short injected presses.

What the server and patched client do today:

- v2 raw handshake and frame protocol;
- PBKDF2-HMAC-SHA256 challenge-response (unknown users get a fake challenge),
  with SHA-256/HMAC/PBKDF2 added to the 32-bit `VXAOS.dll`;
- character list/select/create, EnterWorld, authoritative tile/event/PvP
  movement, map and global chat, logout;
- server-authoritative equipment, hotbar, creation-point spending, inventory
  deltas and HP/MP/param broadcasts;
- real-time combat against map enemies: basic attacks, target selection,
  damage formulas, enemy AI, death/revive, experience, gold and drops;
- item and skill use with inventory consumption, MP costs and level checks;
- a server-side event interpreter for action/touch map events: show text and
  choices, conditional branches, switches/variables/self switches, waits,
  transfers between maps, gold/items/vitals/skills and client-rendered command
  ranges;
- server-wide switches (ids above the per-player range) with persistence and
  map-level parallel/autorun pages that depend only on them;
- actor graphic changes, erase-event, and pass-through of purely visual
  commands (screen tone/flash/shake, pictures, weather, audio, move routes);
- event-driven teleport windows, shops (buy/sell) and bank (items and gold)
  with server-side validation and persistence;
- quests loaded from `content/quests.toml`: start/turn-in requirements,
  kill and item progress messages, item consumption, gold/exp/item rewards,
  repeatable quests and the client quest UI;
- guilds: event-driven creation with a 64-cell flag, window/members,
  leader and notice management, kick/leave/invite with an accept/decline
  request, guild name tags, guild chat and private chat;
- parties: invite/accept/decline, member panel, party chat, leave/dissolve and
  EXP/gold sharing between members on the same map with the party bonus;
- friends: request/accept/decline with symmetric storage, online-first list,
  add/remove messages and persistence;
- trades: invite/accept, item and gold offers with server-side ownership
  checks, two-step confirmation, atomic-ish exchange and window close on
  cancel/logout;
- states and buffs: state add/remove with state-rate/resist features, buff and
  debuff levels (-2..2) with timed expiry, parameter rates, ex-parameters
  (hit/eva/crit), element rates, attack states, regeneration states, knockout
  on death and client icon sync;
- autonomous map events: random/approach NPC movement, enemy wandering and
  chasing with sight, touch triggers (player/event touch) and `EVENT_MOVE`
  synchronization;
- character deletion: the client re-verifies the password locally and sends a
  login-nonce-bound HMAC proof; the server deletes the character and its guild
  membership/friends (foreign keys cascade);
- SQLite persistence of accounts, characters, the character aggregate and the
  account bank;
- graceful shutdown and periodic saves.

Character creation is validated server-side (slot, name, class for the
account's VIP status and the point budget) and uses `internal/chargen`; the
client's creation window sends the UI parameter order, converted to RPG Maker
parameter order on the wire.

For the current feature set, setup, admin tools, content authoring and the Lisp
reference, see the mdBook in [`../book`](../book). The list above is a
historical development summary; the book and `CHANGELOG.md` are authoritative.

## Tests

```powershell
cd server-go
go test ./...
go vet ./...
```

`internal/rvdata` has a Ruby oracle test. It is skipped when `ruby` is not on
`PATH`; when present it re-loads the rewritten `Map002.rvdata2` and
`Scripts.rvdata2` with Ruby to prove the writer is compatible.

`internal/net` has real TCP integration tests: handshake, auth, character
select, EnterWorld, a move, map chat, logout, character creation (including
slot/class rejection), equip/unequip inventory deltas, hotbar validation,
point spending, item use, combat messages, event text/choice protocol, shops,
bank deposits/withdrawals, guild creation/membership/requests, party
invite/accept/leave, friends and trade offers/confirmation, character deletion
and the event script natives. The patched `Game.exe` has also been validated
end-to-end against the server: typing in the login box, PBKDF2 login, character
select, EnterWorld, movement (persisted on disconnect), map chat, character
creation, a two-hit enemy kill with exp/drop, an NPC Show Text event resolved
with Next, an item-giving event that persisted its inventory, a teleport window
that moved and charged the player, the shop and bank windows opening and closing,
guild creation through the `$Guilda` event plus the guild window, the friend
window opening through the new protocol messages, character deletion through the
password window, and state/buff icons applied and expired through hotbar skills.

## Content pipeline

`content/locales/ptBR.csv` and `content/quests.toml` are the source of truth for
the generated Vocab and Quests modules:

```powershell
cd server-go
go run ./cmd/vxaos-content check   # render + validate, write nothing
go run ./cmd/vxaos-content build   # replace the two entries in Scripts.rvdata2
```

The build keeps `Scripts.rvdata2.bak` and reloads the result with the rvdata
reader before reporting success. `Configs` and `Enums` still come from the
existing client scripts and will be migrated in the next content step.

The server also reads `content/quests.toml` at startup (`content_dir` in the
config) so quest requirements and rewards are authoritative on the server; the
generated `[VS] Quests` module only provides the client UI names and text.

## Damage formulas

Damage formulas are **stored as Lisp in the `.rvdata2` data**, not converted at
server start. The client never evaluates them (`RPG::UsableItem::Damage#eval`
returns 0); they are server-only content:

```powershell
cd server-go
go run ./cmd/vxaos-formula check   # report how many Ruby formulas would change
go run ./cmd/vxaos-formula apply   # rewrite Skills.rvdata2 / Items.rvdata2
```

`apply` keeps `Skills.rvdata2.bak` and `Items.rvdata2.bak`. A Ruby formula such
as `a.atk * 4 - b.def * 2` becomes:

```lisp
(- (* (get a "atk") 4) (* (get b "def") 2))
```

The server compiles and caches these programs with `internal/script` (the same
VM used for event scripts). `a`/`b` expose `hp`, `mp`, `mhp`, `mmp`, `atk`,
`def`, `mat`, `mdf`, `agi`, `luk` and `level`; `v[n]` exposes player variables.
Unsupported Ruby constructs are reported by `check` instead of silently
producing wrong damage.

## Event scripts

Event command 355 helper calls are also stored as Lisp. The migration rewrites
`Map*.rvdata2` and `CommonEvents.rvdata2`:

```powershell
cd server-go
go run ./cmd/vxaos-events check
go run ./cmd/vxaos-events apply
```

For example `start_quest(1)`, `check_point(1, 14, 3)` and
`chat_add('...', Configs::SUCCESS_COLOR)` become `(start-quest 1)`,
`(check-point 1 14 3)` and `(chat-add "..." 5)`. Supported natives include
quest start/finish, revive-point changes, chat lines, VIP days, sex/graphic
changes, parameter reset, teleport windows, the bank window and the shop/guild
hooks (shops open through the 302 command; guild creation still no-ops). `[NG]`
scripts remain client-only; unsupported scripts are reported and left for the
client.

## rvdata notes

- Reader handles Ruby Marshal 4.8 objects, arrays, hashes, symbols and symbol
  links, object links, `I` instance-variable wrappers, `u` user-defined values,
  `U` user marshal, `C` user class, bignums and RGSS-era floats (the short
  binary tail after a NUL is ignored, as Ruby's `strtod` does).
- `Table` payloads are five little-endian `uint32` words (`dims`, x, y, z,
  count) followed by `uint16` cells.
- The writer does not preserve object identity: shared acyclic objects are
  emitted as copies, and cyclic graphs are rejected. RPG Maker documents are
  acyclic.
- Strings are emitted with the Ruby UTF-8 `:E` instance-variable wrapper. Binary
  strings use `rvdata.Bytes` and are emitted without that wrapper; this is
  required for zlib payloads.

## Lisp quick reference

See `docs/lisp.md` for the language and host ABI.

## Next milestone

See the book and `CHANGELOG.md` for the current roadmap and release notes. The
documentation plan is in `../docs/documentation-plan.md` while the book is being
completed.
