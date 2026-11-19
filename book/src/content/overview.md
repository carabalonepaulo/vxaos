# Content overview

VXA-OS content lives in two places.

## RPG Maker data (`Client/Data/*.rvdata2`)

The server reads the same files the editor writes:

```text
Actors.rvdata2
Classes.rvdata2
Skills.rvdata2
Items.rvdata2
Weapons.rvdata2
Armors.rvdata2
Enemies.rvdata2
States.rvdata2
Animations.rvdata2
Tilesets.rvdata2
CommonEvents.rvdata2
System.rvdata2
MapInfos.rvdata2
Map001.rvdata2, Map002.rvdata2, ...
```

`internal/rpg` loads them into typed structs. The server reads the fields it
needs for gameplay; the client still renders everything from the same data.

The server never mutates maps or the database at runtime. The migration tools
(`vxaos-content`, `vxaos-formula`, `vxaos-events`) are the only things that
rewrite `.rvdata2` files, and they keep `.bak` copies.

## Shared sources (`server-go/content/`)

```text
configs.toml           shared gameplay/UI values -> [VS] Configs
enums.toml             constants/opcodes        -> [VS] Enums
quests.toml            quest definitions        -> [VS] Quests + server
locales/ptBR.csv       UI strings               -> [VS] Vocab
scripts/*.lisp         optional server prelude (user functions)
```

`vxaos-content build` renders the four generated Ruby entries into
`Scripts.rvdata2`. The server also reads `configs.toml` and `quests.toml`
directly at startup; the generated client scripts are only for display.

## Pipeline

```powershell
# validate without writing
go run ./cmd/vxaos-content check

# rewrite generated scripts in Scripts.rvdata2 (keeps .bak)
go run ./cmd/vxaos-content build

# one-time formula migration (Ruby -> Lisp)
go run ./cmd/vxaos-formula check
go run ./cmd/vxaos-formula apply

# one-time event helper migration (Ruby -> Lisp)
go run ./cmd/vxaos-events check
go run ./cmd/vxaos-events apply

# apply the protocol/client scripts
go run ./cmd/vxaos-clientpatch check
go run ./cmd/vxaos-clientpatch apply
```

Recommended order after changing content:

1. `vxaos-content check`
2. `vxaos-content build`
3. `vxaos-clientpatch apply` if the patch files changed
4. restart the server and reconnect the client

## What the server actually reads

| Data | Used for |
|---|---|
| Actors, Classes | initial stats, graphics, level curves, learnings, class graphics notes |
| Skills, Items | use, damage, effects, scope, range/AOE, MP/TP, animation, note flags |
| Weapons, Armors | equips, params, paperdoll, VIP/soulbound/level notes |
| Enemies | params, exp/gold, drops, actions, sight/revive/disable notes |
| States | restrictions, removal/chance, turns, features |
| Tilesets | passage flags |
| CommonEvents | item-triggered and shared event commands |
| Maps | size, tileset, events, pages, commands, PvP flag |
| System | start map/position, terms, audio, battle settings |
| Animations | animation frame metadata for timing |

## Validation rules

The loader rejects malformed data early. Common failures:

- a map referenced by `MapInfos` is missing;
- an event references an invalid page/command;
- a class/skill/item id is out of range;
- a Lisp formula fails to compile or references an unknown native;
- `content/configs.toml` has a root key after a table header (TOML rule) or a
  limit below 1;
- `content/enums.toml` no longer matches the Go/Ruby protocol tables.

Run `go test ./...` after content changes; the content and enum parity tests
catch many mistakes before a client reconnect.
