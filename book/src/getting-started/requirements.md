# Requirements

## Supported platform

The server is written in Go and runs on Windows, Linux and macOS. The supported
production target is **Windows 64-bit** because the client is a 32-bit RPG Maker
VX Ace game.

| Component | Requirement |
|---|---|
| Client | RPG Maker VX Ace game with this project's `VXAOS.dll` and patched scripts |
| Go | The version in `server-go/go.mod` (Go 1.27 or newer) |
| Git | To clone and update the repository |
| Database | SQLite by default; MySQL or PostgreSQL optional |
| `just` | Optional command runner used by the recipes |
| Zig | Optional; only to rebuild `VXAOS.dll` |
| 7-Zip | Optional; only for `just dist` |

You do **not** need Ruby, RPG Maker, `rmc` or the old Ruby server to run the Go
server.

## Repository contents

You need both the server and the client:

```text
client data       ../Client/Data (maps, scripts, database .rvdata2 files)
server            server-go/
content           server-go/content (shared configs, enums, quests, locales)
```

`data_dir` in `config.toml` must point to the client `Data` folder. The example
configuration uses `../Client/Data` relative to `server-go/`.

The `content_dir` must contain at least:

```text
content/configs.toml
content/enums.toml
content/quests.toml
content/locales/ptBR.csv
```

## Network

Default ports:

| Port | Use |
|---|---|
| 5000 | game client |
| 5050 | isolated load-test server (`config.load.toml`) |

The server listens on `0.0.0.0:5000` by default. If the client is on another
machine, allow TCP 5000 in Windows Firewall and set the client's `HOST` to the
server's LAN/public IP.

There is **no TLS** in this release. If the server is exposed to the internet,
put it behind a VPN or a TLS-terminating TCP proxy. Do not expose the plaintext
port directly to untrusted networks.

## Versions that must match

- `content/configs.toml` → `[network] game_version` must equal the value in
  `config.toml`/`config.example.toml` startup validation and the client's
  generated `Configs::GAME_VERSION`.
- The client must use protocol v2. `vxaos-clientpatch` installs it.
- The server's `max frame` is 1 MiB; the client uses the same limit from the
  handshake.

## Optional tools

- `vxaos-admin` is built with the server and is required for the first account.
- `vxaos-clientpatch` is required after cloning and after pulling client patch
  changes.
- `vxaos-content` is required after changing TOML/CSV content.
- `vxaos-formula` and `vxaos-events` are one-time migration tools for turning
  Ruby formulas/helpers into Lisp.
- `vxaos-load` and `vxaos-input` are for tests, not production.

## Before you start

1. The client runs and reaches the title screen.
2. You have a writable copy of `Client/Data`.
3. The server can read `Client/Data` and write `server-go/data/`.
4. Port 5000 is free.

Next: [Initial setup](setup.md).
