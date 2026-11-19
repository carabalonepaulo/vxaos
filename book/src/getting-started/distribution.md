# Distribution

The distribution archive contains the server, the admin tool, game data,
content sources and a start script. It does **not** contain the client.

## Build the binaries

```powershell
cd server-go
just build
```

This builds:

```text
bin/vxaos-server.exe
bin/vxaos-admin.exe
bin/vxaos-load.exe
```

## Create the archive

```powershell
just dist
```

Requirements:

- `bin/vxaos-server.exe` and `bin/vxaos-admin.exe` must exist;
- `7z.exe` must be on `PATH`.

The script `scripts/build-dist.ps1` creates:

```text
dist/vxaos-server-<timestamp>/
dist/vxaos-server-<timestamp>.7z
```

You can pass a version explicitly:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass `
  -File scripts/build-dist.ps1 -Version 3.0.0
```

## Archive contents

```text
vxaos-server-<version>/
  vxaos-server.exe
  vxaos-admin.exe
  config.toml
  README.txt
  start-server.bat
  content/
  game-data/        RPG Maker .rvdata2 files
  data/
    README.txt      the database is created here on first run
```

`packaging/config.toml` uses `data_dir = "game-data"`, so the archive does not
depend on `../Client/Data`.

## First run on a new machine

```powershell
cd vxaos-server-3.0.0
.\vxaos-admin.exe migrate -config config.toml
.\vxaos-admin.exe account create -config config.toml -user admin -pass secret -group 2
.\vxaos-admin.exe character create -config config.toml -user admin -name Hero -class 1 -sex 0
.\start-server.bat
```

The server creates `data/vxaos.db` automatically.

## Updating a live server

1. Stop the server gracefully (Ctrl+C) so the persistence worker flushes.
2. Back up `data/vxaos.db` (and `data/vxaos.db-wal`/`-shm` if present).
3. Replace `vxaos-server.exe`, `vxaos-admin.exe` and `content/`.
4. Run `vxaos-admin migrate -config config.toml`.
5. Rebuild generated client content if `content/` changed.
6. Start the server and check the logs.

Do not copy a development `data/vxaos.db` into a production archive. Keep the
database outside the release folder or in a backed-up location.

## Client distribution

The client is distributed separately. It must include:

- the game and `RGSS300.dll`;
- the rebuilt `VXAOS.dll`;
- `Data/Scripts.rvdata2` after `vxaos-clientpatch apply` and
  `vxaos-content build`;
- `Data/Map*.rvdata2`, `System.rvdata2`, `Classes.rvdata2`, etc.

The client's `content/configs.toml` values are compiled into the generated
`[VS] Configs` script. Changing the server address or game version requires
regenerating the client content.

## Server configuration in the archive

`packaging/config.toml` is deliberately minimal:

```toml
listen = "0.0.0.0:5000"
data_dir = "game-data"
content_dir = "content"
log_level = "info"

[server]
max_connections = 500
tick_ms = 80
auth_timeout_seconds = 30
inactivity_seconds = 300
save_interval_seconds = 900

[game]
max_stack = 999

[auth]
iterations = 210000

[store]
driver = "sqlite"
dsn = "data/vxaos.db"
```

Newer server versions add fields such as `map_idle_seconds` and AOI settings
with safe defaults, so an older minimal config continues to load. Add the new
keys explicitly if you want to tune them; see
[Configuration](../operations/configuration.md).

## Checking an archive before shipping

```powershell
# extract to a temporary folder
7z x dist\vxaos-server-3.0.0.7z -oTemp\vxaos-dist

cd Temp\vxaos-dist\vxaos-server-3.0.0
.\vxaos-admin.exe migrate -config config.toml
.\vxaos-server.exe -config config.toml
```

The server should reach `listening` and create `data/vxaos.db`.
