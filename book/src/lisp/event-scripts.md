# Event scripts

Event command 355 blocks that must run on the server are stored as Lisp.
Client-only visual scripts stay RGSS Ruby and are marked `[NG]`; the server does
not execute them.

## Migration

```powershell
cd server-go
go run ./cmd/vxaos-events check
go run ./cmd/vxaos-events apply
```

`check` reports how many Ruby helper calls would change. `apply` rewrites
`Map*.rvdata2` and `CommonEvents.rvdata2`, keeping `.bak` files. Unsupported
scripts are reported and left in place.

Examples:

```ruby
start_quest(1)
check_point(1, 14, 3)
chat_add('...', Configs::SUCCESS_COLOR)
```

become:

```lisp
(start-quest 1)
(check-point 1 14 3)
(chat-add "..." 5)
```

## Natives

| Native | Args | Effect |
|---|---|---|
| `start-quest` | quest id (one-based) | starts a quest |
| `finish-quest` | quest id (one-based) | validates and turns in |
| `check-point` | map, x, y | sets the revive point |
| `chat-add` | text, color | sends a chat line to the player |
| `add-vip-days` | days | extends VIP through the coordinator/DB |
| `change-sex` | none | toggles sex and broadcasts |
| `save-original-graphic` | none | caches the current character/face |
| `load-original-graphic` | none | restores the cached graphic |
| `send-player-graphic` | none | broadcasts the current graphic |
| `send-player-sex` | none | broadcasts the current sex |
| `reset-parameters` | none | resets class base params |
| `erase-buff` | parameter id | removes a buff |
| `open-teleport` | group (one-based) | opens the teleport window |
| `open-bank` | none | opens the account bank |
| `open-create-guild` | none | opens guild creation |
| `open-shop` | none/optional | shop is opened by command 302 |

Native behavior is implemented in `internal/game/eventscript.go`. Host objects
such as `client`, `event`, `map` and `interpreter` are documented in the
low-level `server-go/docs/lisp.md`.

## Examples

Quest start and checkpoint:

```lisp
(start-quest 2)
(check-point 2 4 20)
```

Turn in and reward:

```lisp
(finish-quest 2)
```

Teleport window:

```lisp
(open-teleport 1)
```

Bank:

```lisp
(open-bank)
```

Guild creation:

```lisp
(open-create-guild)
```

Chat feedback:

```lisp
(chat-add "Você recebeu a recompensa." 5)
```

## Execution model

- Command 355 executes inside the player's event interpreter on the map actor.
- It may block the event (for example, opening a bank) like a normal command.
- It may call coordinator APIs (quests, VIP, guild) synchronously; the
  coordinator never waits for the map.
- An error aborts the event and is logged; the map continues.
- Instructions and call depth are limited.

## `[NG]`

`[NG]` scripts are client-only. They stay Ruby in the event data and are skipped
by the server migration. Use them for visual effects, pictures, weather, screen
tone/flash/shake and other things the server does not need to know.

## Authoring checklist

1. Use server Lisp for state-changing logic.
2. Use `[NG]` for client visuals.
3. Use one-based quest ids in scripts.
4. Test the event with one client and check the server log for Lisp errors.
5. Re-run `vxaos-events check` after editing maps.
6. Keep `.bak` files until the map is verified.
