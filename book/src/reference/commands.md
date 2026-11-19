# Command reference

All commands assume `server-go/` as the working directory and built binaries in
`bin/`.

## `vxaos-server`

```text
vxaos-server.exe -config config.toml [-listen ADDR] [-data DIR] [-dsn DSN]
```

| Flag | Meaning |
|---|---|
| `-config` | config path (default `config.toml`) |
| `-listen` | override `listen` |
| `-data` | override `data_dir` |
| `-dsn` | override `store.dsn` |

## `vxaos-admin`

```text
vxaos-admin <command> [flags]

migrate
account create -user NAME -pass PASS [-email EMAIL] [-group 0|1|2] [-vip-days N]
character create -user NAME -name NAME [-slot N] [-class ID] [-sex 0|1]
character delete -user NAME (-slot N | -name NAME)
character list -user NAME

global: -config PATH, -data DIR
```

Group `1` allows `GO`, `PULL`, `MUTE`; group `2` allows all admin commands.

## `vxaos-content`

```text
vxaos-content <build|check> [-data DIR] [-content DIR] [-locale NAME]

check   render and validate without writing
build   replace [VS] Enums/Configs/Vocab/Quests in Scripts.rvdata2
```

Defaults: `-data ../Client/Data`, `-content ./content`, `-locale ptBR`.

## `vxaos-clientpatch`

```text
vxaos-clientpatch <check|apply> [-data DIR] [-patches DIR]

check   validate scripts/patches, write nothing
apply   replace/insert entries in Scripts.rvdata2 (keeps .bak)
```

Defaults: `-data ../Client/Data`, `-patches ./client-patches`.

## `vxaos-formula`

```text
vxaos-formula <check|apply> [data-dir]
```

Rewrites `Skills.rvdata2` and `Items.rvdata2` damage formulas to Lisp. Default
data dir `../Client/Data`. `apply` keeps `.bak` files. Invalid formulas cause a
non-zero exit.

## `vxaos-events`

```text
vxaos-events <check|apply> [data-dir]
```

Rewrites command 355 helper calls in `Map*.rvdata2` and
`CommonEvents.rvdata2`. Default data dir `../Client/Data`. `apply` keeps `.bak`
files. Unsupported scripts are reported and left untouched.

## `vxaos-load`

```text
vxaos-load [flags]

-addr           server address (default 127.0.0.1:5050)
-config         config used for setup/content (default config.load.toml)
-count          bots (default 500)
-duration       run duration (default 60s)
-ramp           connection ramp (default 10s)
-move-every     per-bot move interval (default 500ms)
-attack-every   per-bot attack interval (default 2s)
-prefix         account/character prefix (default load)
-pass           bot password (default loadtest)
-setup          create accounts/characters instead of running
-spread         distribute spawns for AOI tests
-json           write JSON report
-iterations     PBKDF2 iterations for created bots (default 1000)
```

Use `-setup` once per database, then run without it.

## `vxaos-input`

```text
vxaos-input [-addr 127.0.0.1:5001] <command>

key <name|vk> down|up|press [ms]
type <text>
move <x> <y>
click <x> <y> [left|right|middle]
reset
```

Requires the client started with `VXAOS_TEST_INPUT` set to the same port. It
does not use the physical mouse/keyboard or change the foreground window. Keep
the game window visible; Windows throttles minimized windows.

## `just`

```text
just build
just run
just start
just test
just vet
just check
just load-setup
just load-server
just load
just dist
```

## mdBook

```text
mdbook build book      # HTML in book/build
mdbook serve book      # local preview
mdbook test book       # only if code fences are marked as tested
```
