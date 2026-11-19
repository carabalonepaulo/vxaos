# Quests

Quests live in `server-go/content/quests.toml`. The server reads the file at
startup; `vxaos-content build` renders the same data into the client's
`[VS] Quests` module for the UI.

## IDs

The TOML array is zero-based:

```toml
[[quest]]
name = "..."
# index 0
```

Event scripts use one-based ids:

```lisp
(start-quest 1)   ; starts index 0
(finish-quest 1)  ; turns in index 0
```

## Schema

```toml
[[quest]]
name = "Morcegos Malditos"
desc = "Mate 10 Morcegos na Caverna."
switch_id = 0
variable_id = 0
variable_amount = 0
item_id = 0
item_kind = 0
item_amount = 0
enemy_id = 2
enemy_amount = 10
rew_exp = 20
rew_gold = 20
rew_item_id = 2
rew_item_kind = 2
rew_item_amount = 1
repeat = true
```

| Key | Meaning |
|---|---|
| `name` | UI name |
| `desc` | UI description |
| `switch_id` | required switch (0 = none) |
| `variable_id` | required variable (0 = none) |
| `variable_amount` | minimum variable value |
| `item_id` | required turn-in item (0 = none) |
| `item_kind` | 1 item, 2 weapon, 3 armor |
| `item_amount` | required item quantity |
| `enemy_id` | tracked enemy id (0 = none) |
| `enemy_amount` | required kills |
| `rew_exp` | EXP reward |
| `rew_gold` | gold reward |
| `rew_item_id` | item reward (0 = none) |
| `rew_item_kind` | 1 item, 2 weapon, 3 armor |
| `rew_item_amount` | reward quantity |
| `repeat` | remove the finished entry so it can be started again |

Validation rejects empty names, negative ids/amounts, and `item_kind` outside
1..3 (0 is allowed for “none”).

## Lifecycle

1. `start-quest n` adds the quest with state `in progress` (0) if it is not
   already present.
2. Killing a tracked enemy increments `Kills` for the first in-progress quest
   that tracks that enemy.
3. `questRequirementsMet` checks the switch, variable, required item and kill
   count.
4. `finish-quest n` consumes the item, grants gold/EXP/item, marks the quest
   finished (1), and removes the entry when `repeat = true`.
5. Non-repeatable finished quests stay in the state and cannot be restarted.

The client receives `QuestAdd`/`QuestFinish` packets and updates the quest UI.
Item and kill progress messages are server chat lines.

## Requirements and rewards details

- The required item is consumed only when all requirements pass.
- A reward item that does not fit is skipped; the quest still finishes.
- Gold is clamped to the shared `max_gold`.
- EXP uses the normal level-up path (class curve, stats, learnings, client
  packets).
- Switch ids above `max_player_switches` are global switches; others are the
  player's switches.

## Authoring workflow

1. Edit `content/quests.toml`.
2. Run `go run ./cmd/vxaos-content check` to validate.
3. Run `go run ./cmd/vxaos-content build` to update `[VS] Quests`.
4. Restart the server so it reloads `quests.toml`.
5. Test start, kill/item progress, turn-in, rewards and repeat behavior.

## Common mistakes

| Symptom | Cause |
|---|---|
| Quest id off by one | TOML is zero-based; scripts are one-based |
| Turn-in does nothing | Requirement not met, wrong state, or missing item |
| Kill count never grows | `enemy_id` does not match the RPG Maker enemy id |
| Repeat quest cannot restart | `repeat = false` |
| Client shows old text | `vxaos-content build` was not run or client not restarted |
