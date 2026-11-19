# Economy and item rules

The economy is server-authoritative. The client sends intents; the server
validates ownership, space, distance, level, gold and trade state.

## Item kinds

| Kind | Meaning |
|---:|---|
| 1 | item |
| 2 | weapon |
| 3 | armor |

These values are used by inventory, bank, drops, trade, shops and protocol
messages. They are not RPG Maker's database ids; they are the VXA-OS kind code.

## Inventory

- Maximum item slots: `content/configs.toml` `limits.max_player_items` (30).
- Stack size: `config.toml` `game.max_stack` (999).
- Gold cap: `limits.max_gold` (99,999,999).
- Equipped items occupy equipment slots, not inventory slots.
- `Soulbound=true` items cannot be traded or dropped.
- `VIP=true` equipment requires an active VIP account.
- `Level` note on equipment validates the character level.

Inventory changes are atomic at the command level; a failed validation leaves
the inventory unchanged. Every successful change is persisted (action save).

## Bank

- Account-wide, not character-wide.
- Maximum slots: `limits.max_bank_items` (42).
- Supports items and gold.
- Deposits/withdrawals validate space and ownership.
- Bank state is saved through the persistence worker.
- The bank window is opened by `open-bank` in a 355 script.

## Ground drops

- Maximum simultaneous drops per map: `limits.max_map_drops` (20).
- Drop despawn: `drop_despawn_seconds` (60).
- Pickup delay: `drop_pickup_seconds` (10) before non-owners can pick up.
- Drop ownership is by character name; party members with the same party id may
  pick up before the delay.
- Each drop has a stable `DropID` (`uint32`, slab + generation). The client
  sends the id, not a list index.
- Drops are AOI-scoped: only players whose window contains the drop receive
  `DropAdd`, and they receive `DropRemove` when it expires, is picked up, or
  leaves the window.
- `DropItem` removes the stack from inventory and creates the ground drop
  atomically; a failed add restores the inventory.

## Shops

- Command 302 opens a shop with goods.
- Buy validates gold, item existence, price and inventory space.
- Sell validates ownership, sell price and space.
- The server sends the item/gold deltas and persists the state.
- Shops are map events; the shop state is local to the map actor.

## Trade

- Both players must be on the **same map instance**.
- Maximum distance: 10 tiles (Chebyshev) at invite/accept time.
- Max offered stacks: `max_trade_items` (12).
- Both sides can offer items and gold; offers are checked against ownership.
- Soulbound items cannot be offered.
- Two-step confirmation: one side requests finish, the other accepts.
- Exchange is applied on the map actor; windows close on cancel/logout.
- Persistence snapshots are queued for both players.

## Party loot and EXP

- Party members on the same map share EXP and gold with the party bonus.
- Drops remember the killer's party id, so party members on the map can pick up
  before the delay.
- Party membership and invites are coordinator-owned; loot/EXP are map-local.

## Guild and friends

- Guilds do not have a bank or tax in this release.
- Guild membership is coordinator-owned and persisted; guild name tags are
  visible to AOI viewers.
- Friends are per character and persisted; the list is online-first in the UI.

## Limits quick reference

From `content/configs.toml` defaults:

```text
max_player_items   30
max_bank_items     42
max_trade_items    12
max_map_drops      20
max_stack          999 (config.toml [game])
max_gold           99,999,999
drop_despawn       60 s
drop_pickup        10 s
trade distance     10 tiles
```

Change limits in `content/configs.toml` and rebuild the client content; the
server reads the same file at startup.

## Anti-duplication rules

- Every item move validates the source stack before adding to the destination.
- Shops, bank, drops and trade run on the owning map actor, one command at a
  time.
- Trade is same-map, so both inventories are owned by the same actor; the
  exchange does not race across maps.
- Persistence snapshots are immutable clones; the worker cannot mutate live
  inventory.
