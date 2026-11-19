# Introduction

VXA-OS is a server-authoritative online layer for RPG Maker VX Ace. The client
keeps the editor, maps, database and rendering; the server owns authentication,
characters, movement, combat, inventory, quests, events, social state,
persistence and visibility.

The project has three parts:

| Part | What it is |
|---|---|
| Client | A patched VX Ace game (`Game.exe`, `RGSS300.dll`, `VXAOS.dll`, patched Ruby scripts) |
| Server | A Go server in `server/` |
| Content | RPG Maker `.rvdata2` data plus TOML/CSV sources under `content/` |

The client is not optional and is not a generic VX Ace client: it must be
patched with `vxaos-clientpatch` so it speaks protocol v2, performs the
challenge-response login and renders server-driven content.

## Current version

This release is **3.0.0**. Version 3 starts a new major line over the old v2
server. Breaking changes to the protocol, data layout, content pipeline and
runtime architecture are expected during the 3.x line.

The wire protocol remains **protocol v2**; “v2” in this book always means the
wire protocol, not the product version.

## Design decisions

- The Go server never invokes Ruby at runtime.
- Protocol v2 only. There is no compatibility layer for the old framing.
- Authentication is PBKDF2-HMAC-SHA256 challenge-response. There is no TLS in
  this release; run the server on a trusted network, VPN or a TLS-terminating
  proxy.
- SQLite is the default database. Migrations for MySQL and PostgreSQL are
  included, but SQLite is the tested dialect.
- Lisp is server-only. It is used for damage formulas and event command 355
  scripts that must run on the server. Client-only visual scripts stay RGSS
  Ruby and are marked `[NG]`.
- The server is authoritative for damage, drops, inventory, trade, shops,
  banks, movement and visibility.
- Map runtime state is not persisted. Player and bank state are.

## Runtime model in one page

- The **coordinator** owns the online directory, guilds, parties, friends,
  requests, global switches, MOTD, map/instance lifecycle and persistence.
- Each active **map actor** owns its map runtime and the full runtime of the
  players currently on that map: position, HP/MP/EXP/gold, inventory, equips,
  status, event state, shop/bank window, trade and target.
- A player command is routed by `playerID + MapKey + epoch` to the actor that
  currently owns the player.
- Transfers use `detach` by snapshot and `attach` by snapshot. No map shares a
  mutable player object with another map.
- Empty maps stop ticking and are destroyed after `map_idle_seconds`.
- The **area of interest (AOI)** is a server-authoritative square window around
  each player. Only entities inside the window are sent to that client.

This model is why map-local gameplay never needs a global mutex and why the
coordinator never waits for a map.

## Repository layout

```text
/client    patched game and data
/server    Go server and tools (module github.com/carabalonepaulo/vxaos/server)
/server-rb legacy Ruby server (historical reference)
/extra     DLL sources, website, converters and historical assets
/book      this documentation
```

## Who this book is for

- **Operators** who install, configure, back up and monitor a server.
- **Content authors** who create items, quests, maps, events and Lisp scripts.
- **Developers** who extend the protocol, gameplay, persistence or tools.

Read Part I first if you only want a working server. Content authors can start
at Part III and read Part IV before writing formulas or event scripts.
Developers should read Part V before changing the runtime.

## What is intentionally not here

- A guide to installing RPG Maker VX Ace itself.
- A Ruby/RGSS language tutorial. Client scripts are patched, not rewritten from
  scratch.
- A promise of TLS, horizontal multi-process sharding or map-state persistence.
  Those are future work.
- Screenshots. Commands and expected output are documented as text.
