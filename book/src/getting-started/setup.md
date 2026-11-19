# Initial setup

This chapter takes a clean checkout to a logged-in character.

## 1. Enter the server directory

```powershell
cd server-go
```

When the repository is reorganized, this becomes `cd server`.

## 2. Create the server configuration

```powershell
Copy-Item config.example.toml config.toml
```

`config.toml` is intentionally ignored by git. `config.example.toml` is the
template. Edit these fields if your paths differ:

```toml
listen = "0.0.0.0:5000"
data_dir = "../Client/Data"
content_dir = "content"
log_level = "info"
```

`data_dir` must contain the RPG Maker data (`Map*.rvdata2`, `Scripts.rvdata2`,
`System.rvdata2`, etc.).

## 3. Migrate the database

```powershell
go run ./cmd/vxaos-admin migrate -config config.toml
```

Expected output:

```text
migrations ok
```

SQLite creates `data/vxaos.db` automatically. Migrations are idempotent; run
this after every server update.

## 4. Create an administrator account

```powershell
go run ./cmd/vxaos-admin account create -config config.toml `
  -user admin -pass secret -group 2
```

Groups:

| Group | Meaning |
|---:|---|
| 0 | normal player |
| 1 | monitor (navigation and mute commands) |
| 2 | administrator (all admin commands) |

Do not use `secret` in production. The password is hashed with
PBKDF2-HMAC-SHA256 using `[auth] iterations`.

## 5. Create the first character

```powershell
go run ./cmd/vxaos-admin character create -config config.toml `
  -user admin -name Hero -class 1 -sex 0
```

This creates slot `0` by default. `-class` refers to the RPG Maker class id in
`Classes.rvdata2`; `-sex 0` is male and `-sex 1` is female.

You can also create characters in the client after logging in, if the account
has free slots and the class is allowed by the shared config.

## 6. Apply the client patch

The client must speak protocol v2 and include the AOI/content handlers:

```powershell
go run ./cmd/vxaos-clientpatch check
go run ./cmd/vxaos-clientpatch apply
```

- `check` validates the patch set without writing.
- `apply` replaces/inserts the scripts in `../Client/Data/Scripts.rvdata2`.
- The previous archive is kept as `Scripts.rvdata2.bak`.
- Re-running `apply` is idempotent.

Close the game before applying the patch.

## 7. Build generated content

```powershell
go run ./cmd/vxaos-content check
go run ./cmd/vxaos-content build
```

`check` renders and validates the TOML/CSV sources without writing. `build`
replaces the generated `[VS] Enums`, `[VS] Configs`, `[VS] Vocab` and
`[VS] Quests` entries inside `Scripts.rvdata2` and keeps a `.bak`.

Run `build` whenever you change:

- `content/configs.toml`
- `content/enums.toml`
- `content/quests.toml`
- `content/locales/*.csv`

Then re-run `vxaos-clientpatch apply` if the generated scripts were replaced and
you also need the non-generated patches. Recommended order after pulling:

```powershell
go run ./cmd/vxaos-clientpatch apply
go run ./cmd/vxaos-content build
```

## 8. Start the server

```powershell
go run ./cmd/vxaos-server -config config.toml
```

Or build first:

```powershell
go build -o bin/vxaos-server.exe ./cmd/vxaos-server
.\bin\vxaos-server.exe -config config.toml
```

Expected log lines:

```text
database ready
game data loaded
quest content loaded
shared config loaded
server started
listening
```

If the server exits immediately, check the database path, `data_dir` and the
`game_version` in `content/configs.toml`.

## 9. Log in

1. Start the patched client.
2. Type the account password (`secret` in this example).
3. Select the character.
4. The loading screen remains until the server sends `EnterWorld`.

If the character list is empty, create a character in the client or with
`vxaos-admin character create`.

## 10. Make yourself an admin in game

The account group comes from the database (`-group 2` above). Reconnect after
changing groups. Admin commands are available through the in-game admin panel;
see [Admin tools](../operations/admin.md).

## `just` shortcuts

If you have `just`:

```powershell
just build       # server, admin and load tools into bin/
just run         # build and start with config.toml
just start       # start the existing binary
just check       # gofmt, vet and tests
just load-setup  # create 500 bot accounts on the load database
just dist        # create a distribution archive
```

## Common first-run problems

| Symptom | Fix |
|---|---|
| `config: ... no such file` | Run from `server-go/` or pass the full `-config` path |
| `game data loaded` fails | `data_dir` does not contain the RPG Maker `.rvdata2` files |
| Client says version mismatch | Regenerate client content and restart both sides |
| Client disconnects after login | Re-apply `vxaos-clientpatch`, then `vxaos-content build` |
| Empty character list | Create a character for the account |
| `account already exists` | Choose another username or delete the account in the DB |
| Port already in use | Stop the other server or change `listen` |

Next: [Distribution](distribution.md).
