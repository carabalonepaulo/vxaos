# Maps and events

Maps are loaded from `client/Data/Map*.rvdata2` and `MapInfos.rvdata2`. The
server reads the map size, tileset, PvP flag, events, pages and command lists.
The client renders the same files.

## Map fields used by the server

- width/height and tile data (collision and position validation);
- tileset passage flags;
- `PvP` flag from the map data;
- event list and each event's id, name, position, pages and commands;
- page conditions (global switches, variables, self switches, items, actors);
- page trigger, priority, through, graphic, move type/route/frequency.

The server creates a runtime copy on demand; the template is immutable.

## Event triggers

| Trigger | Value | Behavior |
|---|---:|---|
| Action button | 0 | player faces/interacts with the event |
| Player touch | 1 | player steps into the event tile |
| Event touch | 2 | event moves into the player tile |
| Autorun | 3 | runs automatically, blocks the map |
| Parallel | 4 | runs in parallel, map-level |

Autorun/parallel pages on the server may only depend on **global** switches.
Pages that depend on player state stay client-side or must be triggered by a
player action.

## Supported server behavior

- show text and choices, including number/item input through the choice packet;
- conditional branches, labels/jumps, loops and waits;
- switches, variables, self switches and global switches;
- transfers between maps with direction/position;
- gold, items, weapons, armors, HP/MP/EXP and skills;
- teleport windows (`open-teleport`);
- shops (`302`) and banks (`open-bank`);
- guild creation window (`open-create-guild`);
- actor graphic/name/face and sex changes;
- erase event and purely visual commands passed through to the client;
- common events, including item-triggered common events;
- custom NPC move routes.

Client-only visual scripts marked `[NG]` are not executed by the server. The
client handles them locally; the server leaves them in the event list.

## Command 355 (script)

Server-side command 355 blocks are Lisp. Examples:

```lisp
(start-quest 1)
(finish-quest 1)
(check-point 2 4 20)
(open-teleport 1)
(chat-add "Bem-vindo!" 5)
```

Migration from Ruby helper calls:

```powershell
go run ./cmd/vxaos-events check
go run ./cmd/vxaos-events apply
```

`check` reports what would change; `apply` rewrites `Map*.rvdata2` and
`CommonEvents.rvdata2` with `.bak` files. Unsupported Ruby remains and is
reported; `[NG]` scripts stay untouched.

## Shops

A shop event uses command 302 with goods. The server sends the shop window,
validates buy/sell price/ownership/space, applies inventory/gold changes and
persists. The client opens/closes the window; the server is authoritative.

## Banks

`open-bank` in a 355 block opens the account bank. Deposits/withdrawals validate
item ownership and space, use the account bank state, and persist through the
worker. The bank is account-wide, not character-wide.

## Teleports

Event script:

```lisp
(open-teleport 1)
```

The `1` is a one-based group. Destinations come from `content/configs.toml`
`teleports` and, for server-side use, `config.toml` `[[teleport]]` entries. The
server validates gold, charges it and transfers the player.

## Guild creation

`open-create-guild` opens the client window. The account must not already be in
a guild. Creation validates the name and 64-cell flag, stores the guild, sets
the character's guild id, broadcasts the name and persists.

## Custom move routes

Move type 3 pages use the editor's custom route. The server consumes one route
command per movement tick and synchronizes `EventMove` to nearby players. Route
commands support movement, turns, waits, switches, speed/frequency changes and
through toggles.

## Authoring checklist

1. Map size/tileset/passage flags are correct.
2. Events have unique ids and sensible names.
3. Page conditions are intentional; global-only pages for autoruns.
4. Server scripts use supported Lisp natives.
5. `[NG]` is used for client-only visuals.
6. Shops/banks/teleports reference valid data.
7. Test with one client before opening.
8. Run `go test ./...` after content changes.
