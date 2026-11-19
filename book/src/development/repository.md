# Repository and workflow

## Layout

Current working tree:

```text
vxaos/
  Client/            game, data, scripts, DLL
  Server/            old Ruby server (reference only)
  Extra/             DLL sources, website, tools
  server-go/         Go server (moves to server/)
  book/              this mdBook
  docs/              temporary documentation plan
  CHANGELOG.md
```

Server module:

```text
server-go/
  cmd/               server, admin, content, clientpatch, formula, events, load, input
  internal/          rvdata, rpg, content, store, auth, chargen, protocol, script, game, net
  client-patches/    protocol/crypto/buffer/socket/network/handlers
  content/           configs.toml, enums.toml, quests.toml, locales, scripts
  packaging/         distribution config/readme/start script
  scripts/           build-dist, test-client
  docs/              protocol, lisp, load reports
  Justfile
  config.example.toml
  config.load.toml
```

The Go module is `github.com/carabalonepaulo/vxaos/server`. It assumes the
server lives in `server/` at the repository root; the local folder rename is
pending.

## Build and test

```powershell
cd server-go
go build ./...
go test ./...
go vet ./...
```

Race tests:

```powershell
go test -race ./...
```

The net tests start real TCP listeners on loopback; the game tests use an
in-memory-style SQLite temp database and `InlineMaps` for deterministic
single-goroutine tests. Actor tests use real map goroutines and AOI.

`just` shortcuts:

```powershell
just build
just test
just vet
just check
just run
just start
just load-setup
just load-server
just load
just dist
```

## Client workflow

After changing `client-patches/`:

```powershell
go run ./cmd/vxaos-clientpatch check
go run ./cmd/vxaos-clientpatch apply
```

After changing `content/configs.toml`, `enums.toml`, `quests.toml` or locales:

```powershell
go run ./cmd/vxaos-content check
go run ./cmd/vxaos-content build
```

Close the game first. Keep the `.bak` files until the client is verified.

## Content migration tools

```powershell
go run ./cmd/vxaos-formula check
go run ./cmd/vxaos-formula apply
go run ./cmd/vxaos-events check
go run ./cmd/vxaos-events apply
```

They rewrite `.rvdata2` files and keep backups. Run `check` first.

## Load tests

```powershell
# isolated server
.\bin\vxaos-server.exe -config config.load.toml

# setup (once)
.\bin\vxaos-load.exe -config config.load.toml -setup -count 500

# run
.\bin\vxaos-load.exe -config config.load.toml -addr 127.0.0.1:5050 `
  -count 500 -duration 60s -ramp 10s `
  -move-every 500ms -attack-every 2s `
  -json bin\load.json
```

For AOI measurements use `-spread` with a separate database/config, as in
`server-go/docs/load-test-aoi.md`.

## Distribution

```powershell
just dist
```

Requires `7z` on `PATH`. See [Distribution](../getting-started/distribution.md).

## Code style

- `gofmt`; no manual formatting.
- Keep map state actor-owned; no external mutex access.
- Coordinator→map messages are fire-and-forget; map→coordinator may wait.
- Snapshot, don't share mutable player state across maps.
- Add tests for protocol encoding, map-local commands, transfers and AOI.
- Never log secrets or plaintext passwords.
- Keep content changes paired with `vxaos-content check`.

## Commit checklist

1. `gofmt -w .`
2. `go vet ./...`
3. `go test ./...`
4. `go test -race ./...` for concurrency changes
5. `vxaos-content check` for content changes
6. `vxaos-clientpatch check` for client patch changes
7. Client smoke test for protocol/UI changes
8. Update load report if the change is performance-related
