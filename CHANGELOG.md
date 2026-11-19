# Changelog

All notable changes to VXA-OS are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and this project follows [Semantic Versioning](https://semver.org/).

## [0.1.0] - 2026-11-19

Initial public release. This is the Go server rewrite with a patched VX Ace
client, protocol v2, server-authoritative gameplay, map actors and area of
interest.

### Added

#### Server runtime

- Go game server with TCP listener, graceful shutdown and structured logging.
- Protocol v2: symmetric length-prefixed framing, fragmentation, per-direction
  opcodes, handshake, ping/pong and structured errors.
- PBKDF2-HMAC-SHA256 challenge-response authentication, including fake
  challenges for unknown accounts and login-time ban checks.
- New database schema with embedded migrations; SQLite is the default and the
  tested dialect. MySQL and PostgreSQL migrations are included.
- Account, character, character-state and account-bank persistence.
- Character creation with server-side slot, name, class, VIP and point-budget
  validation.
- Character deletion with a login-nonce-bound password proof and cascade
  cleanup.
- Dedicated persistence worker for periodic/action/leave saves; map and world
  goroutines never write the database directly.
- On-demand map runtime: map state is created when the first player enters and
  destroyed after an idle grace period.
- Map actors: each active map owns its runtime and player state, has an inbox
  and ticks independently. Player commands are routed by
  `playerID + MapKey + epoch`.
- Snapshot handoff between maps on transfer; no map goroutine shares a mutable
  player with another.
- `MapKey{TemplateID, InstanceID}` routing, ready for dungeon instances.
- Route cache for player commands, refreshed on attach/transfer.
- Area of interest: server-authoritative square window per player, uniform grid,
  visibility diffs, hysteresis, map-edge clamping and configurable radius.
- Session send queue raised to 1024 to absorb the initial AOI snapshot burst.

#### Gameplay

- Authoritative tile movement, event collision, PvP collision and turn/step
  echo.
- Map, global, party, guild and private chat.
- Real-time combat with basic attacks, target selection, damage formulas,
  enemy AI, death/revive, experience, gold and drops.
- Skills and items: MP costs, level/ownership checks, inventory consumption,
  area effects, projectiles, animations and hotbar use.
- States and buffs: state chance/resist, parameters, ex-parameters, elements,
  attack states, regeneration, timed expiry and client icon sync.
- NPC/enemy autonomous movement: random, approach, custom move routes, sight,
  chase, escape, touch triggers and `EVENT_MOVE` synchronization.
- Server-side event interpreter: text, choices, conditional branches,
  switches/variables/self switches, waits, transfers, gold, items, vitals,
  skills, teleports, shops and banks.
- Autorun/parallel map pages driven by global switches, with persistence.
- Common events triggered by items.
- Quests loaded from `content/quests.toml`: start/turn-in requirements, kill and
  item progress, item consumption, gold/exp/item rewards, repeatable quests and
  client UI text.
- Ground drops with ownership/party rules, pickup delay, despawn and stable
  generation-checked drop IDs.
- Shops with buy/sell validation and bank deposits/withdrawals for items and
  gold.
- Guilds: event-driven creation with a 64-cell flag, member window, leader and
  notice management, invite/accept/decline, kick, leave, deletion, name tags and
  guild chat.
- Parties: invite/accept/decline, member panel, chat, leave/dissolve and
  same-map EXP/gold sharing with the party bonus.
- Friends: request/accept/decline, symmetric storage, online-first list,
  add/remove messages and persistence.
- Trades: invite/accept, item/gold offers, ownership checks, two-step
  confirmation, same-map/max-distance restriction and window close on
  cancel/logout.
- In-game admin panel: kick, teleport, go, pull, give item/weapon/armor/gold,
  ban IP/account, unban, global switch, MOTD, mute and global message, with
  permission levels.
- Guild, party, friend and trade state is coordinator-owned; map-local gameplay
  is actor-owned.
- Map chat is map-wide for the chat window, while chat balloons are bound to the
  local AOI sprite and never appear without a visible player.

#### Content and tools

- Generated client content: `Configs`, `Enums`, `Vocab` and `Quests` from
  `content/configs.toml`, `content/enums.toml`, `content/locales/*.csv` and
  `content/quests.toml`.
- Lisp VM on the server for damage formulas and event command 355 scripts:
  reader, compiler, stack VM, closures, variadics, native functions, host
  objects and resource limits.
- `vxaos-formula` to migrate Ruby damage formulas to Lisp.
- `vxaos-events` to migrate Ruby event helper calls to Lisp.
- `vxaos-content` to validate and generate client content into
  `Scripts.rvdata2`.
- `vxaos-clientpatch` to apply/replace the v2 Ruby client scripts with a backup
  and readability validation.
- `vxaos-admin` for migrations, accounts and characters.
- `vxaos-load` for disposable bot accounts, setup/run modes, clustered or
  spread spawns and JSON reports.
- `vxaos-input` test-only client input bridge for automated client tests.
- `just` recipes for build, run, test, vet, load setup/run and distribution.
- Distribution packaging to `dist/vxaos-server-<timestamp>.7z`.
- Native `VXAOS.dll` build script and fixes for latent input/socket bugs.

#### Client

- Patched VX Ace scripts for protocol v2, crypto, buffering, sockets, network,
  send/receive handling, trading, guild UI and server authority.
- Loading screen no longer has an artificial delay or progress bar; it waits for
  `EnterWorld`.
- Client-side formula evaluation returns 0; the server is the only damage
  authority.
- AOI support: `MapReset`, `EventRemove`, stable drop IDs, logical map
  membership packets and sprite-bound chat balloons.

### Changed

- RPG data maps and events, runtime maps/players/enemies/NPCs and status states
  use ID-indexed slices.
- Map simulation moved from the coordinator to per-map actors.
- Player runtime (position, combat, status, event, inventory, equipment, bank
  window, trade and target) is owned by the current map actor.
- Guild/party/friends/requests/global switches/MOTD/online directory and
  persistence are owned by the coordinator.
- Map-wide player/entity broadcasts were replaced by AOI-scoped broadcasts for
  movement, combat, drops, vitals, equipment, animations and guild names.
- Trade is restricted to players on the same map instance and within the
  configured maximum distance.
- Periodic saves are deep snapshots handed to a worker; action saves no longer
  block the map actor on the database.
- The loading flow waits for `EnterWorld` without an artificial client delay.

### Fixed

- Client heap corruption in `Input__key2char`/UTF-8 handling.
- Socket connect/recv edge cases in the native DLL.
- Client `Network#update` blocking before the handshake could be parsed.
- Out-of-bounds external IDs in slice-based RPG data.
- Loading screen progress that advanced independently of the real server
  handshake.
- Party HUD no longer reacts to an AOI sprite removal as if the player had left
  the party.
- Anonymous map-chat balloons no longer appear for players outside the local
  AOI.

### Performance

Measured with 500 bots, 60 s, 10 s ramp, one move every 500 ms and one attack
every 2 s:

- Pre-parallel coordinator baseline: 35.48% steady CPU, 12.99M received frames,
  102.9 MB received, 12 `send queue full`.
- Map actors and route cache: 32.55% steady CPU, 13.19M received frames, 104.4 MB
  received, 0.5 `send queue full`.
- Area of interest on a spread 500-bot workload: 19.04% steady CPU and 56.0 MB
  received, about **-44% CPU** and **-51% bytes/frames** versus the same workload
  without AOI.
- The original clustered workload stacks bots on the same tiles, so it cannot
  show AOI culling; its numbers are documented in
  `server-go/docs/load-test-aoi.md`.

### Security

- PBKDF2-HMAC-SHA256 verifiers; the server never stores plaintext passwords.
- Challenge-response auth with fake challenges for unknown users.
- Account/IP bans and admin permission levels.
- Server authority for damage, drops, inventory, trade, shops, banks, movement
  and visibility.
- Lisp VM instruction/depth limits.
- No TLS in this release. Run the server behind a trusted network, VPN or a
  TLS-terminating proxy until native TLS is added.

### Compatibility

- Protocol v2 only; the client must be patched with `vxaos-clientpatch`.
- Fresh database schema. The legacy website/Account Editor schema is not a
  migration target.
- The Go server does not invoke Ruby at runtime.
- Map runtime state is not persisted; player and bank state are.
- `InstanceID` is currently 0 for all content; routing supports non-zero
  instances for future dungeon support.
- SQLite is the default and tested database; MySQL/PostgreSQL migrations are
  present but not part of the release test matrix.
- The minimap shows only entities inside the local AOI by design.

### Documentation

- Protocol v2 reference: `server-go/docs/protocol-v2.md`.
- Lisp language and host ABI: `server-go/docs/lisp.md`.
- Load test reports: `server-go/docs/load-test-baseline.md`,
  `load-test-post-parallel.md`, `load-test-aoi.md`.
- Full setup, admin, content and Lisp guide: the mdBook in `book/` (in
  progress).

[0.1.0]: https://github.com/kelaia/vxaos/releases/tag/v0.1.0
