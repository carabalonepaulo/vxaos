# Extending the server

## Add a client command

1. Add an opcode to `ClientOp` at the end.
2. Add a message struct with `Encode` and `ClientOpcode`.
3. Add the `DecodeClient` case.
4. Add the Ruby constant and `send_*` helper.
5. Add the server dispatch in `internal/net/server.go`.
6. Route it:
   - map-local gameplay → `World.onPlayer`;
   - coordinator/social → `World.do`;
   - admin → coordinator + `enqueueMap` for target actions.
7. Return errors that map to a protocol error code.
8. Add a round-trip test and a net integration test.

## Add a map-local command

```go
func (w *World) MyCommand(ctx context.Context, p *Player, arg int) error {
    return w.onPlayer(ctx, p.ID, func(ms *mapState, p *Player) error {
        // only touch ms/p here
        return nil
    })
}
```

Rules:

- do not read `w.online`, `w.players` or other coordinator state directly;
- if you need social/global data, call `w.coordErr`;
- do not call the database; use `w.queueSave` / `w.persist`;
- send packets through `w.broadcastNear` or AOI helpers;
- update the player's AOI if you move it (`aoiPlayerMoved`) or an entity
  (`aoiEntityChanged` / `aoiMovedEventID`).

## Add a coordinator command

Runs inside `w.do`:

- validate `w.online[p.ID]`;
- never dereference map-owned `*Player`;
- use `onlineEntry` for social state;
- use `w.patchSocial` to update the map cache;
- use `w.enqueueMap` for target-map actions (fire-and-forget);
- persist social changes with the store or `patchSave`.

## Add a persistence field

1. Add the field to `store.Character` or `store.CharacterState`.
2. Add a migration for SQLite/MySQL/Postgres.
3. Include it in clone/snapshot (`cloneCharacterState`) if it is deep.
4. Populate it in `chargen`/gameplay.
5. Add a store round-trip test.
6. Run migrations in a test DB.

## Add a Lisp native

1. Register it in `eventscript.go` or the formula context.
2. Validate arguments and return a typed error.
3. Keep it map-local unless it explicitly calls the coordinator.
4. Add a test that evaluates a script using it.
5. Document it in the event-scripts chapter.

## Add an AOI entity kind

1. Add `aoiKind`.
2. Implement `entityPos`.
3. Implement `aoiSendAdd` / `aoiSendRemove`.
4. Insert static instances in `newAOI`.
5. Call `aoiEntityChanged` on add/move/remove.
6. Add a visibility test.

## Add a protocol field

- append fields at the end of the payload;
- update both Go and Ruby codecs;
- do not reuse removed field slots;
- add a protocol test;
- bump the game version when the client must be regenerated.

## Testing expectations

- Unit: pure helpers and codecs.
- Game: inline map tests for gameplay rules.
- Actor: real goroutine/epoch/AOI/transfer tests.
- Net: real loopback TCP.
- Race: `go test -race ./...` for concurrency changes.
- Load: at least 50 bots for network/actor changes.

## Style

- Prefer small functions with explicit ownership.
- Keep map state closed.
- Snapshot at boundaries.
- Log with structured fields.
- Never log secrets.
- Match existing naming and error style.
