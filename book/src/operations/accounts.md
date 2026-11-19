# Accounts and characters

`vxaos-admin` manages migrations, accounts and characters from the command
line. It opens the same database as the server. SQLite uses WAL for file
databases, so stop the server for admin changes when possible.

Build it with `just build` or:

```powershell
go build -o bin/vxaos-admin.exe ./cmd/vxaos-admin
```

All commands accept:

```text
-config <path>   server config file (default config.toml)
-data <path>     override data_dir
```

## Migrations

```powershell
.\bin\vxaos-admin.exe migrate -config config.toml
```

Run after every server update. Migrations are idempotent.

## Account creation

```powershell
.\bin\vxaos-admin.exe account create -config config.toml `
  -user admin -pass secret -group 2
```

Flags:

| Flag | Required | Meaning |
|---|---|---|
| `-user` | yes | Unique username |
| `-pass` | yes | Plain password; hashed before storage |
| `-email` | no | Email field |
| `-group` | no | `0` player, `1` monitor, `2` administrator |
| `-vip-days` | no | VIP duration in days from now |

The password is hashed with PBKDF2-HMAC-SHA256 using the `[auth] iterations` in
`config.toml`. The account is rejected if the username already exists.

Groups:

| Group | In-game permissions |
|---:|---|
| 0 | normal player |
| 1 | navigation (`GO`, `PULL`) and `MUTE` |
| 2 | all admin commands |

Changing the group of an existing account is not exposed by the CLI; update the
`accounts.group_id` column or extend the tool. The change takes effect on the
next login.

## Character creation

```powershell
.\bin\vxaos-admin.exe character create -config config.toml `
  -user admin -name Hero -class 1 -sex 0
```

Flags:

| Flag | Default | Meaning |
|---|---|---|
| `-user` | required | Account username |
| `-name` | required | Character name; unique |
| `-slot` | `0` | Character slot |
| `-class` | `1` | RPG Maker class id |
| `-sex` | `0` | `0` male, `1` female |

The tool uses `internal/chargen` and the shared `content/configs.toml` limits
to compute initial stats, skills, equipment, graphics, map position and
switches/variables. It also validates the class against the account's VIP
status (the CLI uses the shared `max_default_classes` and `max_vip_classes`).

## Character list

```powershell
.\bin\vxaos-admin.exe character list -config config.toml -user admin
```

Output example:

```text
slot=0 id=1 name=Hero class=1 level=1 map=1 (21,12)
```

## Character deletion

```powershell
.\bin\vxaos-admin.exe character delete -config config.toml `
  -user admin -slot 0
```

or by name:

```powershell
.\bin\vxaos-admin.exe character delete -config config.toml `
  -user admin -name Hero
```

Deletion cascades to the character state, bank, friends and guild membership
depending on the schema. The in-game deletion flow is safer for players because
it re-verifies the password and sends a login-nonce-bound proof.

## In-game character creation

Players can create characters in the client if:

- the account has a free slot within `min_characters`/`max_characters`;
- the requested class is allowed by the account's VIP status;
- the name is not forbidden and does not collide.

The server validates slot, name, class, sex, character index and the point
budget. The client sends the UI parameter order; the server stores RPG Maker
parameter order.

## Resetting a password

There is no CLI command for password reset in this release. Options:

1. Create a new account and migrate the character rows manually.
2. Use a small Go program with `internal/auth.NewVerifier` and the store API.
3. Extend `vxaos-admin` with `account passwd` (recommended future work).

Never store or log plaintext passwords.

## Deleting an account

Not exposed by the CLI. Delete the account row and its characters/state with
the database tools of your choice, after backing up. Foreign keys/cascades may
remove characters, state, bank, friends and guild membership; verify the schema
before doing this on a live database.
