# Vocab, Configs and Enums

Three generated Ruby modules come from `server/content/`:

| Source | Generated script | Used by |
|---|---|---|
| `locales/<locale>.csv` | `[VS] Vocab` | client UI strings |
| `configs.toml` | `[VS] Configs` | client and server shared values |
| `enums.toml` | `[VS] Enums` | client constants; protocol parity |
| `quests.toml` | `[VS] Quests` | client quest UI |

`vxaos-content check` renders and validates all four without writing.
`vxaos-content build` replaces the entries in `Scripts.rvdata2` and keeps a
`.bak`.

## Vocab CSV

Format:

```csv
key,text
Username,Usuário
Password,Senha
```

Rules:

- UTF-8;
- header `key,text`;
- keys are Ruby-like identifiers used by the client;
- `%s` placeholders follow Ruby/Go format conventions;
- duplicate keys are errors.

Select a locale:

```powershell
go run ./cmd/vxaos-content build -locale ptBR
```

The default is `ptBR`.

## Configs

`content/configs.toml` is public. Never put secrets there. It contains:

- network host/port/game version;
- resolutions and fonts;
- limits (items, gold, level, friends, party, guild, bank, trade, drops);
- colors, icons, paperdoll order, key bindings, timings;
- teleport lists;
- projectile definitions.

After editing:

```powershell
go run ./cmd/vxaos-content check
go run ./cmd/vxaos-content build
```

The server also reads this file at startup for limits and teleport definitions.
A malformed file prevents the server from starting.

### TOML ordering

Root keys (`schema_version`, `loading_time`, `start_points`, `teleports`, ...)
must appear **before the first `[table]`**. A key after `[fonts.default]`
belongs to that table. This is standard TOML and a common authoring mistake.

## Enums

`content/enums.toml` contains explicit numeric constants:

```toml
[[enum]]
name = "Group"
values = [
  { name = "STANDARD", value = 0 },
  { name = "MONITOR", value = 1 },
  { name = "ADMIN", value = 2 },
]
```

Rules:

- never renumber an existing value; numbers are wire/save contracts;
- add new values at the end of their group;
- keep `[protocol.client]` / `[protocol.server]` in sync with
  `internal/protocol/opcodes.go` and `client-patches/protocol.rb`;
- the parity test fails if counts/values diverge.

Protocol values are **not** rendered into `[VS] Enums`; they are validated by
tests and consumed by the protocol code.

## Generation and client patches

`vxaos-content build` only replaces the generated entries. It does not replace
the network/handler patches. After pulling changes, run both:

```powershell
go run ./cmd/vxaos-clientpatch apply
go run ./cmd/vxaos-content build
```

Close the game first. Keep `Scripts.rvdata2.bak` until you verify the client
starts and connects.

## Validation

```powershell
go run ./cmd/vxaos-content check
go test ./internal/content ./internal/game ./...
```

The content tests catch invalid limits, duplicate keys, enum/protocol drift and
unreadable generated scripts. The server logs the loaded quest/config counts on
startup.
