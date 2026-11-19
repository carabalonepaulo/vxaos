# Admin tools

Administrators use the in-game admin panel. The client sends an `AdminCommand`
message; the server validates the account group and executes the action on the
coordinator or on the target's map actor.

## Permission levels

| Group | Commands |
|---:|---|
| 0 | none |
| 1 | `GO`, `PULL`, `MUTE` |
| 2 | all commands |

The group comes from the account row. Reconnect after changing it.

## Commands

| ID | Constant | Fields | Effect |
|---:|---|---|---|
| 0 | `KICK` | `text` = player name | Disconnects the target after a short alert |
| 1 | `TELEPORT` | `text` = name or `all`, `int1` = map, `int2` = x, `int3` = y | Moves targets to a map position |
| 2 | `GO` | `text` = player name | Moves the admin to that player |
| 3 | `PULL` | `text` = name or `all` | Moves targets to the admin |
| 4 | `ITEM` | `text`, `int1` = item id, `int2` = amount | Gives/removes a normal item |
| 5 | `WEAPON` | `text`, `int1` = weapon id, `int2` = amount | Gives/removes a weapon |
| 6 | `ARMOR` | `text`, `int1` = armor id, `int2` = amount | Gives/removes an armor |
| 7 | `GOLD` | `text`, `int2` = amount | Gives/removes gold |
| 8 | `BAN_IP` | `text` = player name, `int1` = days | Bans the target's current IP |
| 9 | `BAN_ACC` | `text` = player name, `int1` = days | Bans the target account |
| 10 | `UNBAN` | `text` = name/account id/IP | Removes the matching ban |
| 11 | `SWITCH` | `text` = switch id, `int1` = 0/1 | Sets a global switch |
| 12 | `MOTD` | `text` | Sets and broadcasts the message of the day |
| 13 | `MUTE` | `text` = player name | Mutes for 30 seconds |
| 14 | `MESSAGE` | `text` | Sends an admin notice to everyone |

The client panel labels come from the generated `[VS] Enums` module. Keep the
numeric ids stable when changing the client.

## Details and safety

### Kick

- Target must be online and must not be an admin.
- The server sends the admin notice and closes the connection after 250 ms so
  the client can display the alert.
- The normal disconnect path performs the leave/save.

### Teleport, Go and Pull

- `TELEPORT` accepts `all`; map bounds are validated by the target map.
- `GO` reads the target's current position on the target's map and moves the
  admin there through the normal transfer path.
- `PULL` reads the admin position and moves each target; `all` skips the admin.
- Cross-map actions are queued to the owning map actor. They are not
  instantaneous acknowledgements; check the target client after the map tick.

### Give item / gold

- `ITEM`, `WEAPON`, `ARMOR` use RPG Maker kind/id values.
- Positive amounts add; negative amounts remove.
- A full inventory reports to the admin and skips that target.
- `GOLD` clamps to the shared `max_gold`.
- Every change is persisted through the normal save path.

### Bans

- `BAN_ACC` resolves the target account; `BAN_IP` requires an online target with
  a known remote IP.
- Days below 1 are treated as 1.
- Matching online non-admin sessions receive the ban notice and are closed.
- `UNBAN` accepts a character name, numeric account id or IP.
- Bans are stored in the database and checked during authentication.

### Global switches

- Only ids above `content/configs.toml` `max_player_switches` are accepted.
- The value is persisted and patched into every active map actor.
- Global switches drive autorun/parallel pages and NPC pages that depend only on
  global state.

### MOTD and global messages

- `MOTD` is stored in `server_state` and shown after `EnterWorld`.
- `MESSAGE` sends a `ServerAdminCommand` notice and a colored chat line.
- Both are broadcast through client send queues; a full queue closes a slow
  session by design.

### Mute

- Sets the account's mute deadline for 30 seconds.
- Admins cannot be muted.
- Muted players receive `ErrMuted` for chat.

## Operational recommendations

- Keep at least two group-2 accounts.
- Do not use `all` for destructive item/gold removal without checking the
  target list.
- Ban by account when possible; IP bans can affect shared networks.
- Use `MOTD` for maintenance notices, not for long text.
- Test admin actions on the load server (`config.load.toml`) before production.
- Remember that admin actions that cross maps are queued; the panel does not
  wait for the map actor.
