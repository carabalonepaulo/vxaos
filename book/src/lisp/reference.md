# Language reference

This is the quick reference for the server Lisp. The low-level ABI is in
`server-go/docs/lisp.md`.

## Syntax

```lisp
; comment
(if cond then else?)
(begin expr...)
(define name value)
(define (name args...) body...)
(set! name value)
(lambda (args...) body...)
(let ((name value)...) body...)
(cond (test body...)... (else body...))
(when test body...)
(unless test body...)
(and expr...)
(or expr...)
(quote value)
```

- Symbols are bare: `foo`, `atk`.
- Quote a symbol/list with `'`: `'foo`, `'(1 2 3)`.
- Strings use double quotes: `"text"`.
- `nil`/`false` are false; everything else (including `0`) is true.
- `and`/`or` return the actual value.

## Values

`nil`, booleans, integers, floats, strings, symbols, lists, procedures and host
objects.

## Built-ins

| Group | Functions |
|---|---|
| Arithmetic | `+ - * / % ** abs min max floor ceil round rand` |
| Comparison | `= != < <= > >= not` |
| Predicates | `null? number? string? symbol? list? boolean? procedure?` |
| Lists | `list car cdr cons length nth append apply` |
| Strings | `str format print` |

Integer `/` is Ruby-style floor division; `%` follows Ruby's sign rules.

## Host access

```lisp
(get obj 'field)
(call obj 'method arg...)
(set-field obj 'field value)
```

Damage binding:

```lisp
(- (* (get a "atk") 4) (* (get b "def") 2))
```

Event binding:

```lisp
(get client "level")
(call client "quest-in-progress?" 1)
```

## Native functions (event scripts)

`start-quest`, `finish-quest`, `check-point`, `chat-add`, `add-vip-days`,
`change-sex`, `save-original-graphic`, `load-original-graphic`,
`send-player-graphic`, `send-player-sex`, `reset-parameters`, `erase-buff`,
`open-teleport`, `open-bank`, `open-create-guild`, `open-shop`.

See [Event scripts](event-scripts.md) for signatures.

## Damage objects

`a`/`b` fields: `hp mp mhp mmp atk def mat mdf agi luk level`.

`v` exposes player variables by index.

## Limits

- instruction budget per evaluation;
- call-depth limit;
- no `eval`, file, process, network or reflection;
- only registered natives are reachable;
- errors abort the script/formula and are logged.

## Debugging

- Server logs include the script name and error.
- Use `(print value)` to emit a debug line (remove before production).
- Keep formulas small; move shared logic to the prelude.
- Validate event migrations with `vxaos-events check`.
- Validate formulas with `vxaos-formula check`.
- Run `go test ./internal/script ./internal/formula ./internal/eventscript`.

## Common mistakes

| Error | Fix |
|---|---|
| `a.atk` in Lisp | use `(get a "atk")` |
| unquoted symbol list | quote it: `'(1 2)` |
| integer/float mismatch | use `floor`/`round` explicitly |
| unknown native | use a documented native or define a function |
| script runs on client only | mark it `[NG]` |
| quest off by one | scripts are one-based; TOML is zero-based |
| formula returns 0 unexpectedly | check unsupported Ruby in `vxaos-formula check` |
