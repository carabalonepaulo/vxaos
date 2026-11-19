# Damage formulas

Damage formulas are stored as **Lisp inside the `.rvdata2` data**, not converted
at server start. The server compiles and caches them with `internal/script`.

The client never evaluates damage. `RPG::UsableItem::Damage#eval` is overridden
to return 0; the client only displays the result sent by the server.

## Migrating Ruby formulas

```powershell
cd server-go
go run ./cmd/vxaos-formula check   # report what would change
go run ./cmd/vxaos-formula apply   # rewrite Skills.rvdata2 / Items.rvdata2
```

`apply` keeps `Skills.rvdata2.bak` and `Items.rvdata2.bak`. Close the game
before running it.

Example conversion:

```ruby
a.atk * 4 - b.def * 2
```

becomes:

```lisp
(- (* (get a "atk") 4) (* (get b "def") 2))
```

Unsupported Ruby constructs are reported by `check`; they are not silently
converted.

## Bindings

| Name | Meaning |
|---|---|
| `a` | attacker stats |
| `b` | target stats |
| `v` | player variables |

`a` and `b` expose:

```text
hp mp mhp mmp atk def mat mdf agi luk level
```

`v[n]` exposes player variables (1-based in the editor; the Lisp binding uses
the same id semantics as the game).

## Examples

Basic physical:

```lisp
(- (* (get a "atk") 4) (* (get b "def") 2))
```

Magic with level scaling:

```lisp
(+ (get a "mat") (* (get a "level") 2))
```

Clamp to at least 1:

```lisp
(max 1 (- (get a "atk") (get b "def")))
```

Use a variable:

```lisp
(+ (get a "atk") (get v 10))
```

Chance/random:

```lisp
(+ 1 (rand 6))
```

Integer division and modulo follow Ruby semantics. Use `floor`/`round` when you
need explicit rounding.

## How the server applies a formula

1. It computes `battleStats` for attacker/target (base params, equips, states,
   buffs, features).
2. It evaluates the skill/item `Damage.Formula` with `a`, `b`, `v`.
3. It applies variance and critical rules from the data.
4. It applies element/state features and sends the result to clients.

A formula error logs and produces 0 damage; it does not crash the map actor.

## Validation checklist

- `vxaos-formula check` reports zero unsupported formulas.
- The server starts and loads the formula cache.
- Damage is positive for expected stats.
- Healing/negative formulas behave as intended (the item/skill effect decides
  whether the value heals).
- Client shows the server value, not a local preview.

## Notes

- Do not use Ruby methods (`a.atk`, `b.def`) in Lisp; use `(get a "atk")`.
- Do not use `eval`, file or network calls; the VM has none.
- Keep integer arithmetic when the editor data used integers; use floats only
  when needed.
- Formulas are server-only content; a modified client cannot change damage.
