# Glossary

**AOI (area of interest)** — the server-authoritative square window of entities
sent to one player.

**Actor** — a goroutine owning one map runtime and the players on it.

**Attach/detach** — handoff of a player snapshot into/out of a map actor.

**Batch** — a single map actor tick.

**Client patch** — the Ruby scripts applied by `vxaos-clientpatch`.

**Coordinator** — the world goroutine owning online/social/global state and map
lifecycle.

**Epoch** — a per-transfer number that rejects commands addressed to an old
route.

**Formula** — a Lisp damage expression stored in `.rvdata2`.

**Host object** — a Go value exposed to Lisp via `get`/`call`/`set-field`.

**Inline maps** — test-only mode where map commands execute synchronously on the
coordinator; production uses actors.

**Kind** — VXA-OS item kind: 1 item, 2 weapon, 3 armor.

**Lisp** — the server-side script language used for formulas and event 355
blocks.

**MapKey** — `{TemplateID, InstanceID}` identifying a runtime map.

**MapReset** — server packet that clears client-side AOI state before a
snapshot.

**Persistence worker** — goroutine that consumes immutable character/bank
snapshots and writes the DB.

**Protocol v2** — the wire protocol; independent of the product version.

**Route** — `{MapKey, Epoch, Actor}` used to send a command to the owning map.

**Slab** — generation-checked array used for stable drop ids.

**Snapshot** — immutable copy of player/bank state crossing an ownership
boundary.

**VXA-OS** — this project; version 3.0.0 is the first public Go server release.

**`[NG]`** — event script marker for client-only Ruby.

**`[VS]`** — generated client script modules (Protocol, Configs, Enums, Vocab,
Quests, etc.).
